package controllers

import (
	"django_to_go/db"
	"django_to_go/models"
	"django_to_go/service/msg"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

type SnowAddressController struct{}
type QuerySnowAddressRequest struct {
	UserID int `json:"user_id" binding:"required"`
}

type BatchQueryAddressRequest struct {
	Page     int `json:"page" binding:"required,min=1"`
	PageSize int `json:"page_size" binding:"required,min=1,max=100"`
}

type QueryAddressByMobileRequest struct {
	Mobile    string `json:"mobile" binding:"required"`
	DrawBatch int    `json:"draw_batch" binding:"required"`
}

type UpdateAddressByMobileRequest struct {
	Mobile          string `json:"mobile" binding:"required"`
	DrawBatch       int    `json:"draw_batch" binding:"required"`
	Province        string `json:"province" binding:"required"`
	City            string `json:"city" binding:"required"`
	County          string `json:"county" binding:"required"`
	DetailedAddress string `json:"detailed_address" binding:"required"`
	ReceiverName    string `json:"receiver_name" binding:"required"`
	ReceiverPhone   string `json:"receiver_phone" binding:"required"`
}

// 校验用户是否能够填写地址
func (snc *SnowAddressController) QualificationAddressVerification(c *gin.Context) {
	var request QuerySnowAddressRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, msg.ErrResponse("参数错误", err))
		return
	}
	var enduser *models.SnowUser
	db.DB.Where("user_id = ?", 11).First(&enduser)
	begin_time := enduser.RegistrationTime
	end_time := enduser.VerificationCodeExpire
	nowtime := time.Now()

	var user models.SnowUser
	//校验是否存在
	result := db.DB.Where("user_id = ?", request.UserID).First(&user)
	time_info := map[string]any{
		"begin_time": begin_time,
		"end_time":   end_time,
	}
	if result.Error != nil {
		c.JSON(http.StatusBadRequest, msg.ErrResponseStr("用户不存在"))
		return
	}
	//校验是否参与过抽奖
	fmt.Printf("抽奖码:", user.SuccessCode)
	if user.SuccessCode == "{}" {
		c.JSON(http.StatusBadRequest, msg.ErrResponseStr("用户没有参与抽奖"))
		return
	}
	if user.ReceiverName != "" {
		c.JSON(http.StatusBadRequest, msg.ErrResponseStr("已中奖"))
		return
	}
	var snowaddress models.SnowAddress
	result = db.DB.Where("user_id = ?", request.UserID).First(&snowaddress)
	if result.Error != nil {

		snowaddress.UserId = user.UserID
		if err := db.DB.Create(&snowaddress).Error; err != nil {
			c.JSON(http.StatusNotFound, msg.ErrResponseStr("查询失败"))
			return
		}
		if nowtime.Before(begin_time) || nowtime.After(end_time) {
			c.JSON(http.StatusBadRequest, gin.H{
				"msg": "当前未在允许时间内",
				"data": map[string]any{
					"begin_time": begin_time,
					"end_time":   end_time,
				},
				"code": 201,
			})
			return
		}
		c.JSON(http.StatusOK, msg.SuccessResponse("未填写地址", &time_info))
		return
	}
	fmt.Printf("123", snowaddress.ReceiverName)
	switch snowaddress.ReceiverName {

	case "":
		if nowtime.Before(begin_time) || nowtime.After(end_time) {
			c.JSON(http.StatusNotFound, gin.H{
				"msg": "当前未在允许时间内",
				"data": map[string]any{
					"begin_time": begin_time,
					"end_time":   end_time,
				},
				"code": 201,
			})
			return
		}
		c.JSON(http.StatusOK, msg.SuccessResponse("未填写地址", &time_info))
		return
	default:
		c.JSON(http.StatusOK, gin.H{
			"code": 201,
			"msg":  "已填写过地址",
			"data": time_info,
		})
		return
	}
}

// QueryAddress 查询地址
func (suc *SnowAddressController) QueryUserAddress(c *gin.Context) {
	var request QuerySnowAddressRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, msg.ErrResponse("参数错误", err))
		return
	}

	var user models.SnowUser
	result := db.DB.Where("user_id = ?", request.UserID).First(&user)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, msg.ErrResponseStr("用户不存在"))
		return

	}
	var snowaddress models.SnowAddress
	result = db.DB.Where("user_id = ?", request.UserID).First(&snowaddress)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, msg.ErrResponseStr("地址用户不存在"))
		return
	}
	info := map[string]any{
		"province":         snowaddress.Province,
		"city":             snowaddress.City,
		"county":           snowaddress.County,
		"detailed_address": snowaddress.DetailedAddress,
		"receiver_name":    snowaddress.ReceiverName,
		"receiver_phone":   snowaddress.ReceiverPhone,
	}
	// 用户存在，返回地址信息
	c.JSON(http.StatusOK, msg.SuccessResponse("查询成功", &info))
}

// UpdateAddressByMobile 根据手机号和波次修改地址信息
func (suc *SnowAddressController) UpdateAddressByMobile(c *gin.Context) {
	var request UpdateAddressByMobileRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, msg.ErrResponse("参数错误", err))
		return
	}

	// 根据手机号和波次查询用户ID
	var userID int
	fmt.Printf("原始波次值: %d\n", request.DrawBatch)
	searchKey := fmt.Sprintf("\"%d\": %s", request.DrawBatch, request.Mobile)
	fmt.Println("searchKey:", searchKey)
	result := db.DB.Table("snow_uesr"). // 注意表名拼写正确
						Where("mobile_batch LIKE ?", "%"+searchKey+"%").
						Limit(1). // 强制只取第一条，优化性能
						Pluck("user_id", &userID)
	fmt.Printf("查询的user_id: %d\n", userID)

	// 处理查询异常
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, msg.ErrResponseStr("查询用户失败"))
		return
	}

	// 检查是否查询到结果（user_id为0表示无匹配）
	if userID == 0 {
		c.JSON(http.StatusNotFound, msg.ErrResponseStr("未找到匹配的用户"))
		return
	}

	// 检查用户是否存在
	var user models.SnowUser
	result = db.DB.Where("user_id = ?", userID).First(&user)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, msg.ErrResponseStr("用户不存在"))
		return
	}

	// 检查地址记录是否存在
	var snowaddress models.SnowAddress
	result = db.DB.Where("user_id = ?", userID).First(&snowaddress)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, msg.ErrResponseStr("地址记录不存在"))
		return
	}

	// 更新地址信息
	time_now := time.Now()
	snowaddress.Province = request.Province
	snowaddress.City = request.City
	snowaddress.County = request.County
	snowaddress.DetailedAddress = request.DetailedAddress
	snowaddress.ReceiverName = request.ReceiverName
	snowaddress.ReceiverPhone = request.ReceiverPhone
	snowaddress.FillTime = &time_now

	// 保存更新
	if err := db.DB.Save(&snowaddress).Error; err != nil {
		c.JSON(http.StatusInternalServerError, msg.ErrResponseStr("地址更新失败"))
		return
	}

	c.JSON(http.StatusOK, msg.SuccessResponseStr("地址更新成功"))
}

func (suc *SnowAddressController) UpdateAddress(c *gin.Context) {
	var request UpdateAddressRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	var snowaddress models.SnowAddress
	result := db.DB.Where("user_id = ?", request.UserID).First(&snowaddress)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, msg.ErrResponseStr("用户不存在"))
		return
	}
	time_now := time.Now()

	// 更新地址
	snowaddress.Province = request.Province
	snowaddress.City = request.City
	snowaddress.County = request.County
	snowaddress.DetailedAddress = request.DetailedAddress
	snowaddress.ReceiverName = request.ReceiverName
	snowaddress.ReceiverPhone = request.ReceiverPhone
	snowaddress.FillTime = &time_now
	db.DB.Save(&snowaddress)

	c.JSON(http.StatusOK, msg.SuccessResponseStr("地址更新成功"))
	return
}

// BatchQueryAddress 批量查询地址信息，支持分页
func (suc *SnowAddressController) BatchQueryAddress(c *gin.Context) {
	var request BatchQueryAddressRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, msg.ErrResponse("参数错误", err))
		return
	}

	page := request.Page
	pageSize := request.PageSize

	var total int64
	var addresses []models.SnowAddress

	// 计算总数
	db.DB.Model(&models.SnowAddress{}).Count(&total)

	// 分页查询
	offset := (page - 1) * pageSize
	db.DB.Offset(offset).Limit(pageSize).Find(&addresses)

	// 准备响应数据
	response := map[string]interface{}{
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
		"data":        addresses,
	}

	c.JSON(http.StatusOK, msg.SuccessResponse("查询成功", &response))
}

// ExportAllAddress 导出全部地址到Excel
func (suc *SnowAddressController) ExportAllAddress(c *gin.Context) {
	// 查询所有地址数据
	var addresses []models.SnowAddress
	db.DB.Find(&addresses)

	// 创建Excel文件
	f := excelize.NewFile()

	// 设置工作表名称
	sheetName := "地址信息"
	f.SetSheetName("Sheet1", sheetName)

	// 设置表头
	header := []string{"序号", "收货人姓名", "手机号", "省", "市", "县", "具体地址", "填写时间"}
	for i, h := range header {
		cell := fmt.Sprintf("%s%d", string(rune('A'+i)), 1)
		f.SetCellValue(sheetName, cell, h)
	}

	// 填充数据
	for i, addr := range addresses {
		row := i + 2
		fillTime := ""
		if addr.FillTime != nil {
			fillTime = addr.FillTime.Format("2006-01-02 15:04:05")
		}

		// 设置行数据
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), i+1)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), addr.ReceiverName)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), addr.ReceiverPhone)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), addr.Province)
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), addr.City)
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), addr.County)
		f.SetCellValue(sheetName, fmt.Sprintf("G%d", row), addr.DetailedAddress)
		f.SetCellValue(sheetName, fmt.Sprintf("H%d", row), fillTime)
	}

	// 设置响应头
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=addresses_%s.xlsx", time.Now().Format("20060102_150405")))

	// 导出文件
	if err := f.Write(c.Writer); err != nil {
		c.JSON(http.StatusInternalServerError, msg.ErrResponseStr("导出失败"))
		return
	}
}

func (suc *SnowAddressController) QueryAddressByMobile(c *gin.Context) {
	var request QueryAddressByMobileRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, msg.ErrResponse("参数错误", err))
		return
	}
	var userID int
	fmt.Printf("原始波次值", request.DrawBatch)
	searchKey := fmt.Sprintf("\"%d\": %s", request.DrawBatch, request.Mobile)
	fmt.Println("searchKey:", searchKey)
	result := db.DB.Table("snow_uesr").
		Where("mobile_batch LIKE ?", "%"+searchKey+"%").
		Limit(1). // 强制只取第一条，优化性能
		Pluck("user_id", &userID)
	fmt.Printf("查询的user_id: %d\n", userID)

	// 5. 处理查询异常
	if result.Error != nil {
		c.JSON(http.StatusBadRequest, msg.ErrResponseStr("查询失败"))
		return
	}

	// 6. 检查是否查询到结果（user_id为0表示无匹配）
	if userID == 0 {
		c.JSON(http.StatusBadRequest, msg.ErrResponseStr("查询失败"))
		return
	}
	var snowaddress models.SnowAddress
	result = db.DB.Where("user_id = ?", userID).First(&snowaddress)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, msg.ErrResponseStr("未填写地址"))
		return
	}
	info := map[string]any{
		"province":         snowaddress.Province,
		"city":             snowaddress.City,
		"county":           snowaddress.County,
		"detailed_address": snowaddress.DetailedAddress,
		"receiver_name":    snowaddress.ReceiverName,
		"receiver_phone":   snowaddress.ReceiverPhone,
	}
	// 用户存在，返回地址信息
	c.JSON(http.StatusOK, msg.SuccessResponse("查询成功", &info))
}
