package controllers

import (
	"log"
	"net/http"
	"strconv"

	"django_to_go/db"
	"django_to_go/models"

	"github.com/gin-gonic/gin"
)

// AddressController 地址控制器
type AddressController struct{}

// AddAddress 新增用户地址 - 与Django版本的add_address函数完全匹配
func (ac *AddressController) AddAddress(c *gin.Context) {
	var requestData struct {
		UserID          int    `json:"user_id" binding:"required"`
		Province        string `json:"province" binding:"required"`
		City            string `json:"city" binding:"required"`
		County          string `json:"county" binding:"required"`
		DetailedAddress string `json:"detailed_address" binding:"required"`
		ReceiverName    string `json:"receiver_name" binding:"required"`
		PhoneNumber     string `json:"phone_number" binding:"required"`
		IsDefault       bool   `json:"is_default"`
		Remark          string `json:"remark"`
	}

	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "缺少必填字段",
		})
		return
	}

	// 检查用户是否存在
	var user models.User
	if err := db.DB.Where("user_id = ?", requestData.UserID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "用户不存在",
		})
		return
	}

	// 如果设置为默认地址，先将该用户的其他默认地址取消
	if requestData.IsDefault {
		// 取消该用户的所有默认地址
		db.DB.Model(&models.Address{}).
			Where("user_id = ? AND is_default = ?", requestData.UserID, true).
			Update("is_default", false)
	}

	// 创建新地址
	address := models.Address{
		UserID:          requestData.UserID,
		ReceiverName:    requestData.ReceiverName,
		PhoneNumber:     requestData.PhoneNumber,
		Province:        requestData.Province,
		City:            requestData.City,
		County:          requestData.County,
		DetailedAddress: requestData.DetailedAddress,
		IsDefault:       requestData.IsDefault,
	}

	if err := db.DB.Create(&address).Error; err != nil {
		log.Printf("新增地址失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "新增地址失败",
		})
		return
	}

	log.Printf("用户 %d 新增地址成功: %d", requestData.UserID, address.AddressID)
	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"message":    "新增地址成功",
		"address_id": address.AddressID,
	})
}

// DeleteAddress 删除用户地址 - 与Django版本的delete_address函数完全匹配
func (ac *AddressController) DeleteAddress(c *gin.Context) {
	var requestData struct {
		AddressID int `json:"address_id" binding:"required"`
		UserID    int `json:"user_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "缺少必填字段",
		})
		return
	}

	// 验证地址是否存在且属于该用户
	var address models.Address
	if err := db.DB.Where("address_id = ? AND user_id = ?", requestData.AddressID, requestData.UserID).First(&address).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "地址不存在或不属于该用户",
		})
		return
	}

	// 删除地址
	if err := db.DB.Delete(&address).Error; err != nil {
		log.Printf("删除地址失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "删除地址失败",
		})
		return
	}

	log.Printf("用户 %d 删除地址成功: %d", requestData.UserID, requestData.AddressID)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "删除地址成功",
	})
}

// UpdateAddress 更新用户地址 - 与Django版本的update_address函数完全匹配
func (ac *AddressController) UpdateAddress(c *gin.Context) {
	var requestData struct {
		AddressID       int    `json:"address_id" binding:"required"`
		UserID          int    `json:"user_id" binding:"required"`
		Province        string `json:"province"`
		City            string `json:"city"`
		County          string `json:"county"`
		DetailedAddress string `json:"detailed_address"`
		ReceiverName    string `json:"receiver_name"`
		PhoneNumber     string `json:"phone_number"`
		IsDefault       bool   `json:"is_default"`
		Remark          string `json:"remark"`
	}

	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "缺少必填字段",
		})
		return
	}

	// 验证地址是否存在且属于该用户
	var address models.Address
	if err := db.DB.Where("address_id = ? AND user_id = ?", requestData.AddressID, requestData.UserID).First(&address).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "地址不存在或不属于该用户",
		})
		return
	}

	// 如果设置为默认地址，先将该用户的其他默认地址取消
	if requestData.IsDefault {
		db.DB.Model(&models.Address{}).
			Where("user_id = ? AND is_default = ? AND address_id != ?", address.UserID, true, requestData.AddressID).
			Update("is_default", false)
	}

	// 更新地址信息
	hasChanges := false
	if requestData.Province != "" {
		address.Province = requestData.Province
		hasChanges = true
	}
	if requestData.City != "" {
		address.City = requestData.City
		hasChanges = true
	}
	if requestData.County != "" {
		address.County = requestData.County
		hasChanges = true
	}
	if requestData.DetailedAddress != "" {
		address.DetailedAddress = requestData.DetailedAddress
		hasChanges = true
	}
	if requestData.ReceiverName != "" {
		address.ReceiverName = requestData.ReceiverName
		hasChanges = true
	}
	if requestData.PhoneNumber != "" {
		address.PhoneNumber = requestData.PhoneNumber
		hasChanges = true
	}
	address.IsDefault = requestData.IsDefault
	hasChanges = true
	if requestData.Remark != "" {
		// 这里不做处理，因为模型中没有remark字段
	}

	// 保存更新后的地址
	if hasChanges {
		if err := db.DB.Save(&address).Error; err != nil {
			log.Printf("更新地址失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "更新地址失败",
			})
			return
		}
	}

	log.Printf("用户 %d 更新地址成功: %d", requestData.UserID, requestData.AddressID)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "更新地址成功",
	})
}

// SetDefaultAddress 设置默认地址 - 与Django版本的set_default_address函数完全匹配
func (ac *AddressController) SetDefaultAddress(c *gin.Context) {
	var requestData struct {
		AddressID int `json:"address_id" binding:"required"`
		UserID    int `json:"user_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "缺少必填字段",
		})
		return
	}

	// 检查地址是否存在且属于该用户
	var address models.Address
	if err := db.DB.Where("address_id = ? AND user_id = ?", requestData.AddressID, requestData.UserID).First(&address).Error; err != nil {
		log.Printf("地址不存在或不属于该用户: user_id=%d, address_id=%d", requestData.UserID, requestData.AddressID)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "地址不存在或不属于该用户",
		})
		return
	}

	// 先取消该用户的其他默认地址
	db.DB.Model(&models.Address{}).
		Where("user_id = ? AND is_default = ?", requestData.UserID, true).
		Update("is_default", false)

	// 设置新的默认地址
	address.IsDefault = true
	if err := db.DB.Save(&address).Error; err != nil {
		log.Printf("设置默认地址失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "设置默认地址失败",
		})
		return
	}

	log.Printf("用户 %d 设置默认地址成功: %d", requestData.UserID, requestData.AddressID)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "设置默认地址成功",
	})
}

// GetAddresses 获取用户所有地址 - 与Django版本的get_addresses函数完全匹配
func (ac *AddressController) GetAddresses(c *gin.Context) {
	var requestData struct {
		UserID int `json:"user_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "缺少必填字段",
		})
		return
	}

	var addresses []models.Address
	if err := db.DB.Where("user_id = ?", requestData.UserID).Order("is_default DESC, created_at DESC").Find(&addresses).Error; err != nil {
		log.Printf("获取用户 %d 的地址列表失败: %v", requestData.UserID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取地址失败",
		})
		return
	}

	log.Printf("成功获取用户 %d 的地址列表，共 %d 条记录", requestData.UserID, len(addresses))
	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   "获取地址成功",
		"addresses": addresses,
	})
}

// GetAddressByID 根据ID获取地址详情 - 与Django版本的get_address_by_id函数完全匹配
func (ac *AddressController) GetAddressByID(c *gin.Context) {
	// 使用map来灵活处理不同类型的参数
	var requestMap map[string]interface{}
	if err := c.ShouldBindJSON(&requestMap); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "请求格式错误",
		})
		return
	}

	// 解析AddressID，支持字符串和整数类型
	addressID, ok := requestMap["address_id"]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "缺少address_id字段",
		})
		return
	}

	// 尝试将AddressID转换为int
	var addressIDInt int
	switch v := addressID.(type) {
	case string:
		// 字符串类型，尝试转换为整数
		id, err := strconv.Atoi(v)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "address_id格式错误，需为整数",
			})
			return
		}
		addressIDInt = id
	case float64:
		// JSON数字默认解析为float64
		addressIDInt = int(v)
	case int:
		addressIDInt = v
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "address_id格式错误",
		})
		return
	}

	// 解析UserID
	userID, ok := requestMap["user_id"]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "缺少user_id字段",
		})
		return
	}

	var userIDInt int
	switch v := userID.(type) {
	case string:
		id, err := strconv.Atoi(v)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "user_id格式错误，需为整数",
			})
			return
		}
		userIDInt = id
	case float64:
		userIDInt = int(v)
	case int:
		userIDInt = v
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "user_id格式错误",
		})
		return
	}

	var address models.Address
	if err := db.DB.Where("address_id = ? AND user_id = ?", addressIDInt, userIDInt).First(&address).Error; err != nil {
		log.Printf("地址不存在或不属于该用户: user_id=%d, address_id=%d", userIDInt, addressIDInt)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "地址不存在或不属于该用户",
		})
		return
	}

	log.Printf("成功获取用户 %d 的地址详情: %d", userIDInt, addressIDInt)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "获取地址成功",
		"address": address,
	})
}
