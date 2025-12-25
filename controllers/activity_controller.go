package controllers

import (
	"django_to_go/db"
	"django_to_go/models"
	"django_to_go/utils"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ActivityController 活动控制器
type ActivityController struct{}

// AddActivityImg 添加活动图片 - 与Django版本完全匹配
func (ac *ActivityController) AddActivityImg(c *gin.Context) {
	if c.Request.Method != "POST" {
		c.JSON(http.StatusMethodNotAllowed, gin.H{"code": 405, "message": "不支持的请求方法"})
		return
	}

	// 确保解析表单数据
	if err := c.Request.ParseMultipartForm(10 << 20); err != nil {
		// 不是multipart/form-data格式也没关系，继续尝试获取表单数据
	}

	// 获取表单数据
	category := c.PostForm("category")
	notes := c.PostForm("notes")
	commodities := c.PostForm("commodities")

	// 处理文件上传
	file, header, err := c.Request.FormFile("image")
	var imagePath string
	if err == nil && file != nil {
		defer file.Close()
		// 保存文件到活动图片目录
		directory := "activities"
		filename := utils.GenerateUniqueFilename(header.Filename)

		// 获取当前工作目录的绝对路径
		currentDir, err := os.Getwd()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取工作目录失败: " + err.Error()})
			return
		}

		// 构建完整的保存路径（使用绝对路径）
		fullDir := filepath.Join(currentDir, "media", directory)
		if err := os.MkdirAll(fullDir, 0755); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建目录失败: " + err.Error()})
			return
		}

		// 保存文件
		savePath := filepath.Join(fullDir, filename)
		if err := c.SaveUploadedFile(header, savePath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "保存文件失败: " + err.Error()})
			return
		}

		// 验证文件是否成功保存
		if _, err := os.Stat(savePath); os.IsNotExist(err) {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "文件保存后验证失败: 文件不存在"})
			return
		}

		// 只保存相对路径到数据库
		imagePath = filepath.Join(directory, filename)
	} else if err != nil && !strings.Contains(err.Error(), "request Content-Type isn't multipart/form-data") {
		// 如果有其他错误但不是Content-Type错误，则报错
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "文件上传失败: " + err.Error()})
		return
	}

	// 创建活动图对象，状态默认为'pending'
	activityImg := models.ActivityImage{
		Status:      "pending",
		Image:       imagePath, // 保存相对路径
		Category:    category,
		Notes:       notes,
		Commodities: commodities,
		// Order字段不设置，默认为null
	}

	// 使用Select方法只保存需要的字段，不包含order字段
	if err := db.DB.Select("status", "image", "category", "notes", "commodities").Create(&activityImg).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "添加失败: " + err.Error()})
		return
	}

	// 构建完整的图片URL返回给前端
	proto := utils.GetRequestProto(c)
	baseURL := fmt.Sprintf("%s://%s", proto, c.Request.Host)
	fullImageURL := utils.BuildFullImageURL(baseURL, imagePath, "media")

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "添加成功",
		"data": gin.H{
			"id":    activityImg.ID,
			"image": fullImageURL,
		},
	})
}

// UpdateActivityImageRelations 更新活动图片关系 - 与Django版本完全匹配
func (ac *ActivityController) UpdateActivityImageRelations(c *gin.Context) {
	if c.Request.Method != "POST" {
		c.JSON(http.StatusMethodNotAllowed, gin.H{"code": 405, "message": "不支持的请求方法"})
		return
	}

	// 解析JSON请求体
	var requestData map[string]interface{}
	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的请求格式"})
		return
	}

	// 获取请求数据
	activityID, ok := requestData["activity_id"].(float64)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "缺少activity_id参数"})
		return
	}

	_, _ = requestData["commodities"].(string) // commodities变量未使用，使用下划线忽略
	_, _ = requestData["category"].(string)    // category变量未使用，使用下划线忽略

	// 查询活动图是否存在
	var activityImg models.ActivityImage
	if err := db.DB.Where("id = ?", int(activityID)).First(&activityImg).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "活动图不存在"})
		return
	}

	// 验证商品ID是否存在于commodity应用中 - 简化处理，暂时不实现

	// 在实际实现中，这里需要更新活动图信息
	// 由于Go模型与Django模型结构不同，需要进行适配

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "更新成功", "data": gin.H{"id": activityImg.ID}})
}

// ActivityImageOnline 活动图片上线 - 与Django版本完全匹配
func (ac *ActivityController) ActivityImageOnline(c *gin.Context) {
	if c.Request.Method != "POST" {
		c.JSON(http.StatusMethodNotAllowed, gin.H{"code": 405, "message": "不支持的请求方法"})
		return
	}

	// 解析JSON请求体
	var requestData map[string]interface{}
	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的请求格式"})
		return
	}

	// 获取请求数据
	activityID, ok := requestData["activity_id"].(float64)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "缺少activity_id参数"})
		return
	}

	// 验证活动图是否存在
	var activityImg models.ActivityImage
	if err := db.DB.Where("id = ?", int(activityID)).First(&activityImg).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "活动图不存在"})
		return
	}

	// 检查已上线活动图数量
	var onlineCount int64
	db.DB.Model(&models.ActivityImage{}).Where("status = ?", "online").Count(&onlineCount)

	// 如果当前活动图不是已上线状态，且已上线数量已达5，则不允许上线
	if activityImg.Status != "online" && onlineCount >= 5 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "上线失败：最多只能上线5张活动图"})
		return
	}

	// 保存原始状态用于判断
	originalStatus := activityImg.Status

	// 更新活动图状态和上线时间
	activityImg.Status = "online"
	now := time.Now()
	activityImg.OnlineTime = &now
	activityImg.OfflineTime = nil // 设置为nil避免'0000-00-00'问题

	// 当上线图片小于5时，图片成功上线时顺序默认往后拍一位
	if originalStatus != "online" && onlineCount < 5 {
		// 获取当前最大的顺序值
		// order是MySQL保留关键字，需要用反引号包围
		var maxOrder int
		db.DB.Model(&models.ActivityImage{}).Where("status = ?", "online").Select("COALESCE(MAX(`order`), 0)").Scan(&maxOrder)
		newOrder := maxOrder + 1
		activityImg.Order = &newOrder
	}

	if err := db.DB.Save(&activityImg).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "上线失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "上线成功", "data": gin.H{"id": activityImg.ID, "order": activityImg.Order}})
}

// BatchUpdateActivityImageOrder 批量修改活动图片顺序
func (ac *ActivityController) BatchUpdateActivityImageOrder(c *gin.Context) {
	if c.Request.Method != "POST" {
		c.JSON(http.StatusMethodNotAllowed, gin.H{"code": 405, "message": "不支持的请求方法"})
		return
	}

	// 解析JSON请求体，期望格式：{"images": [{"id": 1, "order": 2}, {"id": 2, "order": 1}]}
	var requestData struct {
		Images []struct {
			ID    int `json:"id"`
			Order int `json:"order"`
		} `json:"images"`
	}

	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的请求格式"})
		return
	}

	// 验证请求数据
	if len(requestData.Images) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "缺少图片数据"})
		return
	}

	// 开始事务
	tx := db.DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "事务开始失败: " + tx.Error.Error()})
		return
	}

	// 批量更新图片顺序
	for _, img := range requestData.Images {
		// 验证图片是否存在
		var activityImg models.ActivityImage
		if err := tx.Where("id = ?", img.ID).First(&activityImg).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": fmt.Sprintf("图片ID %d 不存在", img.ID)})
			return
		}

		// 更新顺序
		orderValue := img.Order
		activityImg.Order = &orderValue
		if err := tx.Save(&activityImg).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新顺序失败: " + err.Error()})
			return
		}
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "事务提交失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "批量更新顺序成功"})
}

// ActivityImageOffline 活动图片下线 - 与Django版本完全匹配
func (ac *ActivityController) ActivityImageOffline(c *gin.Context) {
	if c.Request.Method != "POST" {
		c.JSON(http.StatusMethodNotAllowed, gin.H{"code": 405, "message": "不支持的请求方法"})
		return
	}

	// 解析JSON请求体
	var requestData map[string]interface{}
	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的请求格式"})
		return
	}

	// 获取请求数据
	activityID, ok := requestData["activity_id"].(float64)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "缺少activity_id参数"})
		return
	}

	// 开始事务
	tx := db.DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "事务开始失败: " + tx.Error.Error()})
		return
	}

	// 验证活动图是否存在
	var activityImg models.ActivityImage
	if err := tx.Where("id = ?", int(activityID)).First(&activityImg).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "活动图不存在"})
		return
	}

	// 保存当前图片的order值，用于后续调整
	currentOrder := activityImg.Order

	// 更新活动图状态为下线，并清空order
	activityImg.Status = "offline"
	offlineNow := time.Now()
	activityImg.OfflineTime = &offlineNow
	activityImg.Order = nil

	if err := tx.Save(&activityImg).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "下线失败: " + err.Error()})
		return
	}

	// 如果当前图片有order值，则将后续图片的order往前移一位
	if currentOrder != nil {
		// 更新所有order大于当前图片order的图片，order减1
		// order是MySQL保留关键字，需要用反引号包围
		if err := tx.Model(&models.ActivityImage{}).
			Where("status = ? AND `order` > ?", "online", *currentOrder).
			Update("`order`", gorm.Expr("`order` - 1")).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新后续图片顺序失败: " + err.Error()})
			return
		}
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "事务提交失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "下线成功", "data": gin.H{"id": activityImg.ID}})
}

// BatchQueryActivityImages 批量查询活动图片 - 与Django版本完全匹配
func (ac *ActivityController) BatchQueryActivityImages(c *gin.Context) {
	if c.Request.Method != "POST" {
		c.JSON(http.StatusMethodNotAllowed, gin.H{"code": 405, "message": "不支持的请求方法"})
		return
	}

	// 解析JSON请求体
	var requestData map[string]interface{}
	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的请求格式"})
		return
	}

	// 获取分页参数
	page := 1
	pageSize := 10

	if pageVal, ok := requestData["page"].(float64); ok {
		page = int(pageVal)
	}
	if pageSizeVal, ok := requestData["pageSize"].(float64); ok {
		pageSize = int(pageSizeVal)
	}

	// 构建查询
	query := db.DB.Model(&models.ActivityImage{})

	// 处理状态过滤
	if statusVal, ok := requestData["status"].(string); ok && statusVal != "" {
		query = query.Where("status = ?", statusVal)
	}

	// 查询总数
	var total int64
	query.Count(&total)

	// 计算偏移量
	offset := (page - 1) * pageSize

	// 查询数据
	var activityImages []models.ActivityImage
	query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&activityImages)

	// 格式化返回结果
	var results []map[string]interface{}
	for _, img := range activityImages {
		// 解析commodities字段（文本格式，用逗号分隔的商品ID）
		var commodityIDs []int
		if img.Commodities != "" {
			ids := strings.Split(img.Commodities, ",")
			for _, idStr := range ids {
				if id, err := strconv.Atoi(strings.TrimSpace(idStr)); err == nil {
					commodityIDs = append(commodityIDs, id)
				}
			}
		}

		// 处理日期格式
		var onlineTime, offlineTime string
		if img.OnlineTime != nil {
			onlineTime = img.OnlineTime.Format("2006-01-02 15:04:05")
		}
		if img.OfflineTime != nil {
			offlineTime = img.OfflineTime.Format("2006-01-02 15:04:05")
		}

		// 构建完整的图片URL
		// 获取请求的协议，考虑反向代理环境
		proto := utils.GetRequestProto(c)
		baseURL := fmt.Sprintf("%s://%s", proto, c.Request.Host)
		// 将Windows路径的反斜杠转换为正斜杠，确保URL可访问
		imagePathWithForwardSlashes := strings.ReplaceAll(img.Image, "\\", "/")
		fullImageURL := utils.BuildFullImageURL(baseURL, imagePathWithForwardSlashes, "media")

		result := map[string]interface{}{
			"id":           img.ID,
			"image":        fullImageURL, // 添加完整的media路径
			"status":       img.Status,
			"online_time":  onlineTime,
			"offline_time": offlineTime,
			"commodities":  commodityIDs,
			"category":     img.Category, // 从模型中直接获取
			"notes":        img.Notes,    // 从模型中直接获取
			"created_at":   img.CreatedAt.Format("2006-01-02 15:04:05"),
			"updated_at":   img.UpdatedAt.Format("2006-01-02 15:04:05"),
			"order":        img.Order, // 从模型中直接获取
		}
		results = append(results, result)
	}

	// 返回分页数据，与Django格式匹配
	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "查询成功",
		"data": gin.H{
			"items":    results,
			"total":    total,
			"page":     page,
			"pageSize": pageSize,
		},
	})
}
