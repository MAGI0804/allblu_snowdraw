package controllers

import (
	"net/http"
	"strconv"
	"time"

	"django_to_go/db"
	"django_to_go/models"

	"github.com/gin-gonic/gin"
)

// CartController 购物车控制器
type CartController struct{}

// AddToCart 添加商品到购物车 - 与Django版本完全匹配
func (cc *CartController) AddToCart(c *gin.Context) {
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
	userID, ok := requestData["user_id"].(float64)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误：user_id不能为空"})
		return
	}

	// 确保商品编码是字符串类型
	commodityCode := ""
	if codeStr, ok := requestData["commodity_code"].(string); ok {
		commodityCode = codeStr
	} else if codeFloat, ok := requestData["commodity_code"].(float64); ok {
		commodityCode = strconv.FormatFloat(codeFloat, 'f', -1, 64)
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误：commodity_code不能为空"})
		return
	}

	// 处理数量参数，默认为1
	quantity := 1
	if quantityVal, ok := requestData["quantity"].(float64); ok {
		quantity = int(quantityVal)
	}

	// 验证数量参数
	if quantity <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误：quantity必须为正整数"})
		return
	}

	// 检查用户是否存在
	var user models.User
	if err := db.DB.Where("user_id = ?", int(userID)).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "用户不存在"})
		return
	}

	// 检查商品是否存在
	var commodity models.Commodity
	if err := db.DB.Where("commodity_id = ?", commodityCode).First(&commodity).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "商品不存在"})
		return
	}

	// 获取或创建购物车
	var cart models.Cart
	if err := db.DB.Where("user_id = ?", int(userID)).First(&cart).Error; err != nil {
		// 购物车不存在，创建新的购物车
		cart = models.Cart{
			UserID:    int(userID),
			CartItems: make(models.CartItemsMap),
		}
	}

	// 添加商品到购物车
	currentTime := time.Now().Format("2006-01-02 15:04:05")
	if item, exists := cart.CartItems[commodityCode]; exists {
		// 商品已存在，增加数量
		item.Quantity += quantity
		item.AddedTime = currentTime
		cart.CartItems[commodityCode] = item
	} else {
		// 商品不存在，添加新商品
		cart.CartItems[commodityCode] = models.CartItemJSON{
			Quantity:  quantity,
			AddedTime: currentTime,
		}
	}

	// 保存购物车
	if err := db.DB.Save(&cart).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "添加到购物车失败: " + err.Error()})
		return
	}

	// 计算购物车总数量
	totalQuantity := 0
	for _, item := range cart.CartItems {
		totalQuantity += item.Quantity
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "商品添加到购物车成功",
		"data": gin.H{
			"commodity_code": commodityCode,
			"quantity":       cart.CartItems[commodityCode].Quantity,
			"total_items":    totalQuantity,
		},
	})
}

// BatchDeleteFromCart 批量删除购物车商品 - 与Django版本完全匹配
func (cc *CartController) BatchDeleteFromCart(c *gin.Context) {
	// 支持DELETE和POST方法
	var requestData map[string]interface{}

	// 根据请求方法解析参数
	if c.Request.Method == "DELETE" {
		// DELETE方法从查询参数获取user_id
		userIDStr := c.Query("user_id")
		if userIDStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误：user_id不能为空"})
			return
		}
		userID, err := strconv.Atoi(userIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误：user_id必须是整数"})
			return
		}
		// DELETE方法视为清空购物车
		requestData = map[string]interface{}{"user_id": float64(userID)}
	} else {
		// POST方法从JSON请求体获取参数
		if err := c.ShouldBindJSON(&requestData); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的请求格式"})
			return
		}
	}

	// 获取用户ID
	userIDFloat, ok := requestData["user_id"].(float64)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误：user_id不能为空"})
		return
	}
	userID := int(userIDFloat)

	// 检查用户是否存在
	var user models.User
	if err := db.DB.Where("user_id = ?", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "用户不存在"})
		return
	}

	// 获取购物车
	var cart models.Cart
	if err := db.DB.Where("user_id = ?", userID).First(&cart).Error; err != nil {
		// 购物车不存在，返回空数据
		c.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": "删除成功",
			"data": gin.H{
				"deleted_count": 0,
			},
		})
		return
	}

	// 定义删除统计
	var deletedCount int64
	var notExistCodes []string

	// 检查是否有commodity_codes参数
	if commodityCodes, ok := requestData["commodity_codes"].([]interface{}); ok && len(commodityCodes) > 0 {
		// 构建商品编码字符串列表
		var codeStrings []string
		for _, code := range commodityCodes {
			if codeStr, ok := code.(string); ok {
				codeStrings = append(codeStrings, codeStr)
			} else if codeInt, ok := code.(float64); ok {
				codeStrings = append(codeStrings, strconv.FormatFloat(codeInt, 'f', -1, 64))
			}
		}

		// 记录实际删除的商品数量
		deletedCount = 0

		// 检查哪些商品编码不存在于购物车中
		for _, code := range codeStrings {
			if _, exists := cart.CartItems[code]; exists {
				// 商品存在，删除它
				delete(cart.CartItems, code)
				deletedCount++
			} else {
				// 商品不存在，添加到不存在列表
				notExistCodes = append(notExistCodes, code)
			}
		}
	} else {
		// 清空购物车
		deletedCount = int64(len(cart.CartItems))
		cart.CartItems = make(models.CartItemsMap)
	}

	// 保存更新后的购物车
	if err := db.DB.Save(&cart).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "删除购物车商品失败: " + err.Error(),
		})
		return
	}

	// 构建响应
	data := gin.H{
		"deleted_count": deletedCount,
	}
	if len(notExistCodes) > 0 {
		data["not_exist_codes"] = notExistCodes
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "删除成功",
		"data":    data,
	})
}

// QueryCartItems 查询购物车商品 - 与Django版本完全匹配
func (cc *CartController) QueryCartItems(c *gin.Context) {
	// 支持GET和POST方法
	var userID int
	var err error

	// 根据请求方法获取user_id
	if c.Request.Method == "GET" {
		// GET方法从查询参数获取user_id
		userIDStr := c.Query("user_id")
		if userIDStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误：user_id不能为空"})
			return
		}
		userID, err = strconv.Atoi(userIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误：user_id必须是整数"})
			return
		}
	} else {
		// POST方法从JSON请求体获取参数
		var requestData map[string]interface{}
		if err := c.ShouldBindJSON(&requestData); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的请求格式"})
			return
		}
		userIDFloat, ok := requestData["user_id"].(float64)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误：user_id不能为空"})
			return
		}
		userID = int(userIDFloat)
	}

	// 检查用户是否存在
	var user models.User
	if err := db.DB.Where("user_id = ?", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "用户不存在"})
		return
	}

	// 查询购物车
	var cart models.Cart
	if err := db.DB.Where("user_id = ?", userID).First(&cart).Error; err != nil {
		// 购物车不存在，返回空数据 - 符合Django返回格式
		c.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": "查询成功",
			"data": gin.H{
				"cart_items":   []interface{}{},
				"items_count":  0,
				"total_quantity": 0,
			},
		})
		return
	}

	// 计算总数量
	var totalQuantity int64

	// 遍历购物车项
	var cartItems []map[string]interface{}
	for commodityCode, item := range cart.CartItems {
		// 查询商品信息
		var commodity models.Commodity
		if err := db.DB.Where("commodity_id = ?", commodityCode).First(&commodity).Error; err != nil {
			// 商品不存在，跳过此项
			continue
		}

		totalQuantity += int64(item.Quantity)

		// 构建商品项数据 - 符合Django返回格式
		itemData := map[string]interface{}{
			"commodity_code": commodityCode,
			"quantity":       item.Quantity,
			"added_time":     item.AddedTime,
		}
		cartItems = append(cartItems, itemData)
	}

	// 构建data对象
	dataObject := gin.H{
		"cart_items":   cartItems,
		"items_count":  len(cartItems),
		"total_quantity": totalQuantity,
	}

	// 返回结果 - 符合Django返回格式
	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "查询成功",
		"data":    dataObject,
	})
}

// UpdateCartItemQuantity 更新购物车商品数量 - 与Django版本完全匹配
func (cc *CartController) UpdateCartItemQuantity(c *gin.Context) {
	var requestData map[string]interface{}

	// 解析JSON请求体
	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的请求格式"})
		return
	}

	// 获取用户ID
	userIDFloat, ok := requestData["user_id"].(float64)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误：user_id不能为空"})
		return
	}
	userID := int(userIDFloat)

	// 获取商品编码
	commodityCode, ok := requestData["commodity_code"].(string)
	if !ok {
		// 尝试转换数字类型的商品编码
		codeFloat, ok := requestData["commodity_code"].(float64)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误：commodity_code不能为空"})
			return
		}
		commodityCode = strconv.FormatFloat(codeFloat, 'f', -1, 64)
	}

	// 获取新数量
	quantityFloat, ok := requestData["quantity"].(float64)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误：quantity不能为空"})
		return
	}
	quantity := int(quantityFloat)
	if quantity < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误：quantity必须大于0"})
		return
	}

	// 检查用户是否存在
	var user models.User
	if err := db.DB.Where("user_id = ?", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "用户不存在"})
		return
	}

	// 查询购物车
	var cart models.Cart
	if err := db.DB.Where("user_id = ?", userID).First(&cart).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "购物车不存在"})
		return
	}

	// 检查商品是否在购物车中
	cartItem, exists := cart.CartItems[commodityCode]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "购物车商品不存在"})
		return
	}

	// 查询商品信息以验证库存
	var commodity models.Commodity
	if err := db.DB.Where("commodity_id = ?", commodityCode).First(&commodity).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "商品不存在"})
		return
	}

	// 更新数量
	cartItem.Quantity = quantity
	cart.CartItems[commodityCode] = cartItem

	// 保存更新后的购物车
	if err := db.DB.Save(&cart).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "更新购物车商品数量失败: " + err.Error(),
		})
		return
	}

	// 转换商品信息为map
	commodityMap := make(map[string]interface{})
	commodityMap["commodity_code"] = commodity.CommodityID
	commodityMap["commodity_name"] = commodity.Name
	commodityMap["price"] = commodity.Price
	commodityMap["main_image"] = commodity.Image

	// 构建响应数据
	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "更新成功",
		"data": gin.H{
			"commodity_code": commodityCode,
			"quantity":       quantity,
			"commodity":      commodityMap,
		},
	})
}

// IncreaseCartItemQuantity 增加购物车商品数量 - 与Django版本完全匹配
func (cc *CartController) IncreaseCartItemQuantity(c *gin.Context) {
	var requestData map[string]interface{}

	// 解析JSON请求体
	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的请求格式"})
		return
	}

	// 获取用户ID
	userIDFloat, ok := requestData["user_id"].(float64)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误：user_id不能为空"})
		return
	}
	userID := int(userIDFloat)

	// 获取商品编码
	commodityCode, ok := requestData["commodity_code"].(string)
	if !ok {
		// 尝试转换数字类型的商品编码
		codeFloat, ok := requestData["commodity_code"].(float64)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误：commodity_code不能为空"})
			return
		}
		commodityCode = strconv.FormatFloat(codeFloat, 'f', -1, 64)
	}

	// 检查用户是否存在
	var user models.User
	if err := db.DB.Where("user_id = ?", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "用户不存在"})
		return
	}

	// 查询购物车
	var cart models.Cart
	if err := db.DB.Where("user_id = ?", userID).First(&cart).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "购物车不存在"})
		return
	}

	// 检查商品是否在购物车中
	cartItem, exists := cart.CartItems[commodityCode]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "购物车商品不存在"})
		return
	}

	// 查询商品信息以验证库存
	var commodity models.Commodity
	if err := db.DB.Where("commodity_id = ?", commodityCode).First(&commodity).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "商品不存在"})
		return
	}

	// 增加数量
	cartItem.Quantity += 1
	cart.CartItems[commodityCode] = cartItem

	// 保存更新后的购物车
	if err := db.DB.Save(&cart).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "增加购物车商品数量失败: " + err.Error(),
		})
		return
	}

	// 转换商品信息为map
	commodityMap := make(map[string]interface{})
	commodityMap["commodity_code"] = commodity.CommodityID
	commodityMap["commodity_name"] = commodity.Name
	commodityMap["price"] = commodity.Price
	commodityMap["main_image"] = commodity.Image

	// 构建响应数据
	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "增加成功",
		"data": gin.H{
			"commodity_code": commodityCode,
			"quantity":       cartItem.Quantity,
			"commodity":      commodityMap,
		},
	})
}

// DecreaseCartItemQuantity 减少购物车商品数量 - 与Django版本完全匹配
func (cc *CartController) DecreaseCartItemQuantity(c *gin.Context) {
	var requestData map[string]interface{}

	// 解析JSON请求体
	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的请求格式"})
		return
	}

	// 获取用户ID
	userIDFloat, ok := requestData["user_id"].(float64)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误：user_id不能为空"})
		return
	}
	userID := int(userIDFloat)

	// 获取商品编码
	commodityCode, ok := requestData["commodity_code"].(string)
	if !ok {
		// 尝试转换数字类型的商品编码
		codeFloat, ok := requestData["commodity_code"].(float64)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误：commodity_code不能为空"})
			return
		}
		commodityCode = strconv.FormatFloat(codeFloat, 'f', -1, 64)
	}

	// 检查用户是否存在
	var user models.User
	if err := db.DB.Where("user_id = ?", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "用户不存在"})
		return
	}

	// 查询购物车
	var cart models.Cart
	if err := db.DB.Where("user_id = ?", userID).First(&cart).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "购物车不存在"})
		return
	}

	// 检查商品是否在购物车中
	cartItem, exists := cart.CartItems[commodityCode]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "购物车商品不存在"})
		return
	}

	// 查询商品信息
	var commodity models.Commodity
	if err := db.DB.Where("commodity_id = ?", commodityCode).First(&commodity).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "商品不存在"})
		return
	}

	// 减少数量或删除商品
	var quantity int
	if cartItem.Quantity > 1 {
		cartItem.Quantity -= 1
		cart.CartItems[commodityCode] = cartItem
		quantity = cartItem.Quantity
	} else {
		// 数量为1时删除商品
		delete(cart.CartItems, commodityCode)
		quantity = 0
	}

	// 保存更新后的购物车
	if err := db.DB.Save(&cart).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "操作购物车商品失败: " + err.Error(),
		})
		return
	}

	// 转换商品信息为map
	commodityMap := make(map[string]interface{})
	commodityMap["commodity_code"] = commodity.CommodityID
	commodityMap["commodity_name"] = commodity.Name
	commodityMap["price"] = commodity.Price
	commodityMap["main_image"] = commodity.Image

	// 构建响应数据
	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "减少成功",
		"data": gin.H{
			"commodity_code": commodityCode,
			"quantity":       quantity,
			"commodity":      commodityMap,
		},
	})
}

// ClearCart 清空购物车 - 与Django版本完全匹配
func (cc *CartController) ClearCart(c *gin.Context) {
	// 支持DELETE和POST方法
	var userID int
	var err error

	// 根据请求方法获取user_id
	if c.Request.Method == "DELETE" {
		// DELETE方法从查询参数获取user_id
		userIDStr := c.Query("user_id")
		if userIDStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误：user_id不能为空"})
			return
		}
		userID, err = strconv.Atoi(userIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误：user_id必须是整数"})
			return
		}
	} else {
		// POST方法从JSON请求体获取参数
		var requestData map[string]interface{}
		if err := c.ShouldBindJSON(&requestData); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的请求格式"})
			return
		}
		userIDFloat, ok := requestData["user_id"].(float64)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误：user_id不能为空"})
			return
		}
		userID = int(userIDFloat)
	}

	// 检查用户是否存在
	var user models.User
	if err := db.DB.Where("user_id = ?", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "用户不存在"})
		return
	}

	// 查询购物车
	var cart models.Cart
	if err := db.DB.Where("user_id = ?", userID).First(&cart).Error; err != nil {
		// 购物车不存在，视为清空成功
		c.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": "购物车已清空",
			"data": gin.H{
				"cleared_count": 0,
				"total_items":   0,
			},
		})
		return
	}

	// 获取清空前的商品数量
	clearedCount := int64(len(cart.CartItems))

	// 清空购物车
	cart.CartItems = make(models.CartItemsMap)

	// 保存更新后的购物车
	if err := db.DB.Save(&cart).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "清空购物车失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "购物车已清空",
		"data": gin.H{
			"cleared_count": clearedCount,
			"total_items":   0,
		},
	})
}
