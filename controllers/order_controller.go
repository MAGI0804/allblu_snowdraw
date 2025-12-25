package controllers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"django_to_go/db"
	"django_to_go/models"

	"github.com/gin-gonic/gin"
)

// OrderController 订单控制器

type OrderController struct{}

// QueryOrdersByUserID 根据用户ID查询订单 - 与Django版本query_by_user_id路由对应
func (oc *OrderController) QueryOrdersByUserID(c *gin.Context) {
	// 绑定请求参数
	var queryData struct {
		Shopname string `json:"shopname" binding:"required"`
		UserID   int    `json:"user_id" binding:"required"`
		Status   string `json:"status" binding:"omitempty"`
		Page     int    `json:"page" binding:"required,min=1"`
		PageSize int    `json:"page_size" binding:"required,min=1,max=50"`
	}

	if err := c.ShouldBindJSON(&queryData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "请求体格式错误"})
		return
	}

	// 验证shopname
	if queryData.Shopname != "youlan_kids" {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "无效的店铺名称"})
		return
	}

	// 限制PageSize最大值为50
	if queryData.PageSize > 50 {
		queryData.PageSize = 50
	}

	// 获取用户ID
	userID := queryData.UserID

	// 构建查询
	var orders []models.Order
	query := db.DB.Table("order_data").Where("user_id = ?", userID)

	// 应用状态过滤
	validStatuses := []string{"pending", "shipped", "delivered", "canceled", "processing", "returning", "exchanging"}
	if queryData.Status != "" {
		statusValid := false
		for _, validStatus := range validStatuses {
			if validStatus == queryData.Status {
				statusValid = true
				break
			}
		}
		if !statusValid {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "订单状态无效"})
			return
		}
		query = query.Where("status = ?", queryData.Status)
	}

	// 计算偏移量
	offset := (queryData.Page - 1) * queryData.PageSize

	// 执行分页查询
	var total int64
	if err := query.Count(&total).Error; err != nil {
		log.Printf("获取订单总数失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "查询订单总数失败: " + err.Error()})
		return
	}

	if err := query.Offset(offset).Limit(queryData.PageSize).Order("order_time DESC").Find(&orders).Error; err != nil {
		log.Printf("查询用户订单失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "查询订单失败: " + err.Error()})
		return
	}

	// 转换订单数据格式
	result := make([]map[string]interface{}, 0, len(orders))
	for _, order := range orders {
		orderMap := convertOrderToMap(order)
		// 确保物流过程信息正确处理
		if order.LogisticsProcess != "" {
			var logisticsProcess []interface{}
			if err := json.Unmarshal([]byte(order.LogisticsProcess), &logisticsProcess); err == nil {
				orderMap["logistics_process"] = logisticsProcess
			}
		}
		result = append(result, orderMap)
	}

	// 返回订单列表 - 与Django版本完全一致的格式
	c.JSON(http.StatusOK, gin.H{
		"status":    "success",
		"data":      result,
		"page":      queryData.Page,
		"page_size": queryData.PageSize,
		"total":     total,
	})
}

// OrderList 获取订单列表 - 与Django版本的orders_query函数对应
func (oc *OrderController) OrderList(c *gin.Context) {
	// 绑定请求参数
	var queryData struct {
		Shopname  string `json:"shopname" binding:"required"`
		UserID    int    `json:"user_id"`
		Status    string `json:"status"`
		Page      int    `json:"page" binding:"required,min=1"`
		PageSize  int    `json:"page_size" binding:"required,min=1,max=50"`
		BeginTime string `json:"begin_time"`
		EndTime   string `json:"end_time"`
	}

	if err := c.ShouldBindJSON(&queryData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请求参数错误"})
		return
	}

	// 验证shopname
	if queryData.Shopname != "youlan_kids" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的店铺名称"})
		return
	}

	// 限制PageSize最大值为50
	if queryData.PageSize > 50 {
		queryData.PageSize = 50
	}

	// 构建查询
	var orders []models.Order
	query := db.DB.Model(&models.Order{}) // 注意这里使用了正确的表名

	// 应用user_id过滤
	if queryData.UserID > 0 {
		query = query.Where("user_id = ?", queryData.UserID)
	}

	// 应用状态过滤
	validStatuses := []string{"pending", "shipped", "delivered", "canceled", "processing"}
	if queryData.Status != "" {
		statusValid := false
		for _, validStatus := range validStatuses {
			if validStatus == queryData.Status {
				statusValid = true
				break
			}
		}
		if !statusValid {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "订单状态无效"})
			return
		}
		query = query.Where("status = ?", queryData.Status)
	}

	// 应用日期过滤
	if queryData.BeginTime != "" {
		beginTime, err := time.Parse("2006-01-02", queryData.BeginTime)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "日期格式必须为YYYY-MM-DD"})
			return
		}
		// 转换为UTC时间（考虑系统时区设置）
		beginTimeUTC := beginTime.In(time.UTC)
		query = query.Where("order_time >= ?", beginTimeUTC)
		log.Printf("订单查询 - 转换后的开始时间(UTC): %v", beginTimeUTC)
	}

	if queryData.EndTime != "" {
		endTime, err := time.Parse("2006-01-02", queryData.EndTime)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "日期格式必须为YYYY-MM-DD"})
			return
		}
		// 转换为UTC时间并加一天（包含当天）
		endTimeUTC := endTime.In(time.UTC).Add(24 * time.Hour)
		query = query.Where("order_time < ?", endTimeUTC)
		log.Printf("订单查询 - 转换后的结束时间(UTC): %v", endTimeUTC)
	}

	// 计算偏移量
	offset := (queryData.Page - 1) * queryData.PageSize

	// 执行分页查询
	var total int64
	if err := query.Count(&total).Error; err != nil {
		log.Printf("获取订单总数失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "查询订单总数失败: " + err.Error()})
		return
	}
	log.Printf("订单查询 - 找到总订单数: %d", total)

	if err := query.Offset(offset).Limit(queryData.PageSize).Order("order_time DESC").Find(&orders).Error; err != nil {
		log.Printf("获取订单列表失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "查询订单列表失败: " + err.Error()})
		return
	}
	log.Printf("订单查询 - 当前页订单数: %d", len(orders))

	// 转换订单数据格式
	result := make([]map[string]interface{}, 0, len(orders))
	for _, order := range orders {
		orderMap := convertOrderToMap(order)
		// 添加物流过程空列表（批量查询时不返回实际物流信息）
		orderMap["logistics_process"] = []interface{}{}
		result = append(result, orderMap)
	}

	// 准备响应数据
	c.JSON(http.StatusOK, gin.H{
		"code":      200,
		"data":      result,
		"page":      queryData.Page,
		"page_size": queryData.PageSize,
		"total":     total,
	})
}

// OrderDetail 获取订单详情 - 与Django版本的query_order_data函数对应
func (oc *OrderController) OrderDetail(c *gin.Context) {
	// 绑定请求参数
	var queryData struct {
		OrderID string `json:"order_id" binding:"required"`
		UserID  int    `json:"user_id"`
	}

	if err := c.ShouldBindJSON(&queryData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "请求体格式错误"})
		return
	}

	// 构建查询条件
	query := db.DB.Model(&models.Order{}).Where("order_id = ?", queryData.OrderID)

	// 如果提供了user_id，则添加到查询条件中
	if queryData.UserID > 0 {
		query = query.Where("user_id = ?", queryData.UserID)
	}

	// 查询订单
	var order models.Order
	if err := query.First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "订单不存在",
		})

		return
	}

	// 准备响应数据
	detailMap := convertOrderToMap(order)

	// 返回订单信息（不含物流更新和物流信息返回）
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "查询订单信息成功",
		"data":    detailMap,
	})
}

// ChangeStatus 修改订单状态 - 与Django版本的change_status函数对应
func (oc *OrderController) ChangeStatus(c *gin.Context) {
	// 绑定请求数据
	var requestData struct {
		OrderID          string      `json:"order_id" binding:"required"`
		Status           string      `json:"status" binding:"required"`
		ExpressCompany   string      `json:"express_company"`
		ExpressNumber    string      `json:"express_number"`
		LogisticsProcess interface{} `json:"logistics_process"`
	}

	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求数据无效",
		})
		return
	}

	// 验证状态值
	validStatus := map[string]bool{
		"pending":    true,
		"processing": true,
		"shipped":    true,
		"delivered":  true,
		"canceled":   true,
	}

	if !validStatus[requestData.Status] {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的订单状态",
		})
		return
	}

	// 查询订单
	var order models.Order
	if err := db.DB.Where("order_id = ?", requestData.OrderID).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "订单不存在",
		})
		return
	}

	// 记录旧状态
	oldStatus := order.Status

	// 更新订单状态和物流信息
	order.Status = requestData.Status
	updatedExpressInfo := make(map[string]interface{})

	if requestData.ExpressCompany != "" {
		order.ExpressCompany = requestData.ExpressCompany
		updatedExpressInfo["express_company"] = requestData.ExpressCompany
	}

	if requestData.ExpressNumber != "" {
		order.ExpressNumber = requestData.ExpressNumber
		updatedExpressInfo["express_number"] = requestData.ExpressNumber
	}

	if requestData.LogisticsProcess != nil {
		// 验证logistics_process是否为有效的JSON
		logisticsJSON, err := json.Marshal(requestData.LogisticsProcess)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "物流过程数据格式无效",
			})
			return
		}
		order.LogisticsProcess = string(logisticsJSON)
		updatedExpressInfo["logistics_process"] = requestData.LogisticsProcess
	}

	// 保存更新，使用Select明确指定需要保存的字段
	if err := db.DB.Select("status", "express_company", "express_number", "logistics_process").Save(&order).Error; err != nil {
		log.Printf("修改订单状态失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "修改订单状态失败",
		})
		return
	}

	// 记录状态变更日志
	log.Printf("订单状态变更: order_id=%s, 旧状态=%s, 新状态=%s", requestData.OrderID, oldStatus, requestData.Status)

	// 解析商品列表用于返回
	var productList []interface{}
	if order.ProductList != "" {
		if err := json.Unmarshal([]byte(order.ProductList), &productList); err != nil {
			productList = []interface{}{}
		}
	}

	// 转换下单时间为UTC+8并格式化
	orderTimeCN := order.OrderTime.Add(8 * time.Hour)
	formattedTime := orderTimeCN.Format("2006-01-02 15:04:05")

	// 返回更新后的订单信息
	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "订单状态更新成功",
		"data": map[string]interface{}{
			"order_id":          order.OrderID,
			"status":            order.Status,
			"old_status":        oldStatus,
			"receiver_name":     order.ReceiverName,
			"receiver_phone":    order.ReceiverPhone,
			"province":          order.Province,
			"city":              order.City,
			"county":            order.County,
			"detailed_address":  order.DetailedAddress,
			"order_amount":      order.OrderAmount,
			"product_list":      productList,
			"order_time":        formattedTime,
			"express_company":   order.ExpressCompany,
			"express_number":    order.ExpressNumber,
			"shipped_time":      order.ShippedTime,
			"delivered_time":    order.DeliveredTime,
			"canceled_time":     order.CanceledTime,
			"processing_time":   order.ProcessingTime,
			"logistics_process": []interface{}{}, // 批量查询时不返回实际物流信息
		},
	})
}

// OrderCreate 创建订单 - 与Django版本的add_order函数对应
func (oc *OrderController) OrderCreate(c *gin.Context) {
	// 绑定请求参数
	var orderData struct {
		ReceiverName    string        `json:"receiver_name" binding:"required"`
		Province        string        `json:"province" binding:"required"`
		City            string        `json:"city" binding:"required"`
		County          string        `json:"county" binding:"required"`
		DetailedAddress string        `json:"detailed_address" binding:"required"`
		OrderAmount     float64       `json:"order_amount" binding:"required"`
		ProductList     []interface{} `json:"product_list" binding:"required,dive"`
		UserID          int           `json:"user_id" binding:"required"`
		ReceiverPhone   interface{}   `json:"receiver_phone"` // 支持数字或字符串类型的手机号
		ExpressCompany  string        `json:"express_company"`
		ExpressNumber   string        `json:"express_number"`
		Remark          string        `json:"remark"` // 新增备注字段
	}

	// 先绑定JSON数据
	if err := c.ShouldBindJSON(&orderData); err != nil {
		// 提供更详细的错误信息
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "请求参数错误: " + err.Error()})
		return
	}

	// 处理手机号类型转换
	var receiverPhoneStr string
	if orderData.ReceiverPhone != nil {
		switch v := orderData.ReceiverPhone.(type) {
		case string:
			receiverPhoneStr = v
		case float64:
			// 将数字类型的手机号转换为字符串
			receiverPhoneStr = strconv.FormatFloat(v, 'f', 0, 64)
		default:
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "手机号格式不正确"})
			return
		}
	}
	orderData.ReceiverPhone = receiverPhoneStr

	// 验证product_list是否为列表
	if len(orderData.ProductList) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "product_list不能为空列表"})
		return
	}

	// 生成订单号
	orderID := generateOrderNo()

	// 将ProductList转换为JSON字符串
	productListJSON, err := json.Marshal(orderData.ProductList)
	if err != nil {
		log.Printf("转换product_list失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "服务器内部错误"})
		return
	}

	// 创建订单，只包含数据库实际存在的字段
	order := models.Order{
		OrderID:         orderID,
		UserID:          orderData.UserID,
		ReceiverName:    orderData.ReceiverName,
		ReceiverPhone:   receiverPhoneStr, // 使用转换后的字符串类型手机号
		Province:        orderData.Province,
		City:            orderData.City,
		County:          orderData.County,
		DetailedAddress: orderData.DetailedAddress,
		OrderAmount:     orderData.OrderAmount,
		ProductList:     string(productListJSON),
		ExpressCompany:  orderData.ExpressCompany,
		ExpressNumber:   orderData.ExpressNumber,
		Status:          "pending",
		OrderTime:       time.Now(),
		Remarks:         orderData.Remark, // 使用Remarks字段
	}

	// 保存订单，使用Select明确指定需要保存的字段，避免包含数据库中不存在的字段
	if err := db.DB.Select("order_id", "user_id", "receiver_name", "receiver_phone", "province", "city", "county", "detailed_address", "order_amount", "product_list", "express_company", "express_number", "status", "order_time", "remarks").Create(&order).Error; err != nil {
		log.Printf("创建订单失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "服务器内部错误"})
		return
	}

	// 准备响应数据
	responseData := map[string]interface{}{
		"order_id":        orderID,
		"user_id":         orderData.UserID,
		"receiver_name":   orderData.ReceiverName,
		"receiver_phone":  receiverPhoneStr, // 返回转换后的字符串类型手机号
		"express_company": orderData.ExpressCompany,
		"express_number":  orderData.ExpressNumber,
		"remark":          orderData.Remark, // 包含备注信息
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "success",
		"message":  "订单创建成功",
		"order_id": orderID,
		"data":     responseData,
	})
}

// OrderUpdate 更新订单
func (oc *OrderController) OrderUpdate(c *gin.Context) {
	orderIDStr := c.Param("id")
	orderID, err := strconv.Atoi(orderIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的订单ID"})
		return
	}

	// 查询订单
	var order models.Order
	if err := db.DB.Where("order_id = ?", orderID).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "订单不存在"})
		return
	}

	// 绑定请求数据
	var updateData map[string]interface{}
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的JSON格式"})
		return
	}

	// 更新字段
	if status, ok := updateData["status"].(string); ok && status != "" {
		order.Status = status
	}

	if paymentMethod, ok := updateData["payment_method"].(string); ok && paymentMethod != "" {
		order.PaymentMethod = paymentMethod
	}

	if paymentTime, ok := updateData["payment_time"].(string); ok && paymentTime != "" {
		parsedTime, timeErr := time.Parse("2006-01-02 15:04:05", paymentTime)
		if timeErr == nil {
			order.PaymentTime = parsedTime
		}
	}

	// 根据Order结构体，以下字段不存在：DeliveryMethod, ExpressCompany, ExpressNumber
	if deliveryMethod, ok := updateData["delivery_method"].(string); ok && deliveryMethod != "" {
		order.DeliveryMethod = deliveryMethod
	}

	if expressCompany, ok := updateData["express_company"].(string); ok && expressCompany != "" {
		order.ExpressCompany = expressCompany
	}

	if expressNumber, ok := updateData["express_number"].(string); ok && expressNumber != "" {
		order.ExpressNumber = expressNumber
	}

	// 保存更新
	if err := db.DB.Save(&order).Error; err != nil {
		log.Printf("更新订单失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "订单更新成功",
		"data":    convertOrderToMap(order),
	})
}

// OrderCancel 取消订单 - 与Django版本的功能保持一致
func (oc *OrderController) OrderCancel(c *gin.Context) {
	// 获取订单ID
	var requestData struct {
		OrderID string `json:"order_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "请求数据无效",
		})
		return
	}

	orderID := requestData.OrderID

	// 查询订单
	var order models.Order
	if err := db.DB.Where("order_id = ?", orderID).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "订单不存在",
		})
		return
	}

	// 检查订单状态 - 只有未发货状态（pending/paid）才能取消
	if order.Status != "pending" && order.Status != "paid" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "只有待支付和已支付状态的订单才能取消",
		})
		return
	}

	// 更新订单状态和取消时间
	order.Status = "canceled"
	order.CanceledTime = time.Now()
	if err := db.DB.Select("status", "canceled_time").Save(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "取消订单失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "订单取消成功",
		"data":    gin.H{"order_id": orderID},
	})
}

// OrderPay 支付订单 - 与Django版本的功能保持一致
func (oc *OrderController) OrderPay(c *gin.Context) {
	// 获取订单ID
	orderID := c.Query("order_id")
	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "订单ID不能为空",
		})
		return
	}

	// 查询订单
	var order models.Order
	if err := db.DB.Where("order_id = ?", orderID).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "订单不存在",
		})
		return
	}

	// 检查订单状态是否允许支付
	if order.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "订单状态不允许支付",
		})
		return
	}

	// 更新订单状态
	order.Status = "paid"
	order.PaymentTime = time.Now()

	// 保存更新
	if err := db.DB.Save(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "支付订单失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "订单支付成功",
		"data":    gin.H{"order_id": orderID},
	})
}

// OrderDeliver 发货 - 与Django版本update_express_info函数对应
func (oc *OrderController) OrderDeliver(c *gin.Context) {
	// 绑定请求数据
	var deliverData struct {
		OrderID        string `json:"order_id" binding:"required"`
		ExpressCompany string `json:"express_company" binding:"required"`
		ExpressNumber  string `json:"express_number" binding:"required"`
	}

	if err := c.ShouldBindJSON(&deliverData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "请求数据无效",
		})
		return
	}

	// 查询订单
	var order models.Order
	if err := db.DB.Where("order_id = ?", deliverData.OrderID).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "订单不存在",
		})
		return
	}

	// 检查订单状态是否允许发货
	if order.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "订单状态不允许发货",
		})
		return
	}

	// 更新订单状态、物流信息和发货时间
	order.Status = "shipped"
	order.ExpressCompany = deliverData.ExpressCompany
	order.ExpressNumber = deliverData.ExpressNumber
	order.ShippedTime = time.Now()
	// 使用Select明确指定需要保存的字段，包含新添加的shipped_time
	if err := db.DB.Select("status", "express_company", "express_number", "shipped_time").Save(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "更新订单物流信息失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "订单发货成功",
		"data":    gin.H{"order_id": deliverData.OrderID},
	})
}

// OrderReceive 签收订单 - 处理订单签收逻辑并设置delivered_time
func (oc *OrderController) OrderReceive(c *gin.Context) {
	// 绑定请求数据
	var requestData struct {
		OrderID string `json:"order_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "请求数据无效",
		})
		return
	}

	// 查询订单
	var order models.Order
	if err := db.DB.Where("order_id = ?", requestData.OrderID).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "订单不存在",
		})
		return
	}

	// 检查订单状态是否允许签收
	if order.Status != "shipped" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "订单状态不允许签收",
		})
		return
	}

	// 更新订单状态和签收时间
	order.Status = "delivered"
	order.DeliveredTime = time.Now()
	if err := db.DB.Select("status", "delivered_time").Save(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "签收订单失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "订单签收成功",
		"data":    gin.H{"order_id": requestData.OrderID},
	})
}

// OrderRequestReturn 申请退换货 - 处理订单退换货申请逻辑并设置processing_time
func (oc *OrderController) OrderRequestReturn(c *gin.Context) {
	// 绑定请求数据
	var requestData struct {
		OrderID       string   `json:"order_id" binding:"required"`
		Type          string   `json:"type" binding:"omitempty,oneof=return exchange"` // return:退货, exchange:换货，默认为return
		Reason        string   `json:"reason" binding:"required"`
		BuyerProvince string   `json:"buyer_province" binding:"required"`
		BuyerCity     string   `json:"buyer_city" binding:"required"`
		BuyerCounty   string   `json:"buyer_county" binding:"required"`
		BuyerAddress  string   `json:"buyer_address" binding:"required"`
		BuyerPhone    string   `json:"buyer_phone" binding:"required"`
		ProductIDs    []string `json:"product_ids" binding:"omitempty"` // 要退换货的商品ID列表，如果不提供则使用订单所有商品
	}

	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "请求数据无效: " + err.Error(),
		})
		return
	}

	// 设置默认值为return
	if requestData.Type == "" {
		requestData.Type = "return"
	}

	// 查询订单
	var order models.Order
	if err := db.DB.Where("order_id = ?", requestData.OrderID).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "订单不存在",
		})
		return
	}

	// 检查订单状态是否允许申请退换货
	// 通常只有已完成的订单才能申请退换货
	if order.Status != "delivered" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "订单状态不允许申请退换货",
		})
		return
	}

	// 生成退货订单号：T+YYYYMMDD+8位随机数字
	returnOrderID := generateReturnOrderNo()

	// 更新订单状态为processing，备注写入退货订单号
	order.Status = "processing"

	// 添加备注说明退货订单号
	returnRemark := fmt.Sprintf("%s申请订单号: %s", map[string]string{"return": "退货", "exchange": "换货"}[requestData.Type], returnOrderID)
	if order.Remarks != "" {
		order.Remarks += "\n" + returnRemark
	} else {
		order.Remarks = returnRemark
	}

	if err := db.DB.Select("status", "remarks").Save(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "申请退换货失败",
		})
		return
	}

	// 处理商品列表 - 只写入该订单下的指定商品
	var returnProductList string
	if len(requestData.ProductIDs) > 0 {
		// 如果请求中提供了商品ID列表，需要从订单的完整商品列表中过滤出这些商品
		// 解析订单商品列表
		var orderProducts []map[string]interface{}
		if err := json.Unmarshal([]byte(order.ProductList), &orderProducts); err == nil {
			// 创建商品ID映射以便快速查找
			productIDMap := make(map[string]bool)
			for _, productID := range requestData.ProductIDs {
				productIDMap[productID] = true
			}

			// 过滤商品
			var filteredProducts []map[string]interface{}
			for _, product := range orderProducts {
				// 检查商品ID是否在请求列表中，支持不同的字段名称
				if productID, ok := product["product_id"].(string); ok && productIDMap[productID] {
					filteredProducts = append(filteredProducts, product)
				} else if productID, ok := product["id"].(string); ok && productIDMap[productID] {
					filteredProducts = append(filteredProducts, product)
				}
			}

			// 将过滤后的商品列表转换回字符串
			if len(filteredProducts) > 0 {
				if filteredProductData, err := json.Marshal(filteredProducts); err == nil {
					returnProductList = string(filteredProductData)
				}
			}
		}

		// 如果过滤失败或没有匹配的商品，使用原始订单商品列表
		if returnProductList == "" {
			returnProductList = order.ProductList
		}
	} else {
		// 如果没有提供商品ID列表，使用订单所有商品
		returnProductList = order.ProductList
	}

	// 创建退货订单记录
	returnOrder := models.ReturnOrder{
		ReturnID:      returnOrderID,
		OrderID:       requestData.OrderID,
		ProductList:   returnProductList, // 写入过滤后的商品列表
		Type:          requestData.Type,
		Status:        "pending",
		RequestTime:   time.Now(),
		Reason:        requestData.Reason,
		BuyerProvince: requestData.BuyerProvince,
		BuyerCity:     requestData.BuyerCity,
		BuyerCounty:   requestData.BuyerCounty,
		BuyerAddress:  requestData.BuyerAddress,
		BuyerPhone:    requestData.BuyerPhone,
	}

	// 保存退货订单，只插入我们设置的字段，避免未初始化的时间字段导致MySQL错误
	if err := db.DB.Select("return_id", "order_id", "product_list", "type", "status", "request_time", "reason", "buyer_province", "buyer_city", "buyer_county", "buyer_address", "buyer_phone").Create(&returnOrder).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "创建退货订单失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": fmt.Sprintf("%s申请提交成功", map[string]string{"return": "退货", "exchange": "换货"}[requestData.Type]),
		"data": gin.H{
			"order_id":        requestData.OrderID,
			"type":            requestData.Type,
			"return_order_id": returnOrderID,
		},
	})
}

// 生成退货订单号 - 格式为T+YYYYMMDD+8位随机数字
func generateReturnOrderNo() string {
	currentDate := time.Now().Format("20060102")
	maxRetries := 5
	var returnOrderID string

	for retry := 0; retry < maxRetries; retry++ {
		// 生成8位随机数字
		var randomNum string
		for i := 0; i < 8; i++ {
			randomNum += fmt.Sprintf("%d", time.Now().UnixNano()%10)
			// 添加小延迟以确保随机性
			time.Sleep(time.Nanosecond)
		}

		returnOrderID = fmt.Sprintf("T%s%s", currentDate, randomNum)

		// 检查退货订单号是否已存在
		var count int64
		err := db.DB.Model(&models.ReturnOrder{}).Where("return_id = ?", returnOrderID).Count(&count).Error
		if err == nil && count == 0 {
			return returnOrderID
		}
	}

	// 如果重试多次仍失败，使用时间戳作为后备方案
	return fmt.Sprintf("T%s%d", currentDate, time.Now().UnixNano()%100000000)
}

// 工具函数：生成订单号 - 格式为Y+YYYYMMDD+8位随机数字
func generateOrderNo() string {
	currentDate := time.Now().Format("20060102")
	maxRetries := 5
	var orderID string

	for retry := 0; retry < maxRetries; retry++ {
		// 生成8位随机数字
		var randomNum string
		for i := 0; i < 8; i++ {
			randomNum += fmt.Sprintf("%d", time.Now().UnixNano()%10)
			// 添加小延迟以确保随机性
			time.Sleep(time.Nanosecond)
		}

		orderID = fmt.Sprintf("Y%s%s", currentDate, randomNum)

		// 检查订单号是否已存在
		var count int64
		err := db.DB.Model(&models.Order{}).Where("order_id = ?", orderID).Count(&count).Error
		if err == nil && count == 0 {
			return orderID
		}
	}

	// 如果重试多次仍失败，使用时间戳作为后备方案
	return fmt.Sprintf("Y%s%d", currentDate, time.Now().UnixNano()%100000000)
}

// 工具函数：将订单对象转换为map - 与Django版本返回格式一致
func convertOrderToMap(order models.Order) map[string]interface{} {
	result := make(map[string]interface{})
	result["order_id"] = order.OrderID
	result["user_id"] = order.UserID
	result["receiver_name"] = order.ReceiverName
	result["receiver_phone"] = order.ReceiverPhone
	result["province"] = order.Province
	result["city"] = order.City
	result["county"] = order.County
	result["detailed_address"] = order.DetailedAddress
	result["order_amount"] = order.OrderAmount
	result["status"] = order.Status

	// 正确解析product_list为数组
	if order.ProductList != "" {
		var productList []string
		if err := json.Unmarshal([]byte(order.ProductList), &productList); err == nil {
			result["product_list"] = productList
		} else {
			// 如果解析失败，设置为空数组
			result["product_list"] = []string{}
		}
	} else {
		result["product_list"] = []string{}
	}

	// 直接使用数据库中的时间，不再添加8小时偏移
	result["order_time"] = order.OrderTime.Format("2006-01-02 15:04:05")

	// 添加物流相关字段，确保空值返回空字符串
	result["express_company"] = order.ExpressCompany
	result["express_number"] = order.ExpressNumber

	// 添加新的时间字段
	if !order.ShippedTime.IsZero() {
		result["shipped_time"] = order.ShippedTime.Format("2006-01-02 15:04:05")
	} else {
		result["shipped_time"] = ""
	}

	if !order.DeliveredTime.IsZero() {
		result["delivered_time"] = order.DeliveredTime.Format("2006-01-02 15:04:05")
	} else {
		result["delivered_time"] = ""
	}

	if !order.CanceledTime.IsZero() {
		result["canceled_time"] = order.CanceledTime.Format("2006-01-02 15:04:05")
	} else {
		result["canceled_time"] = ""
	}

	if !order.ProcessingTime.IsZero() {
		result["processing_time"] = order.ProcessingTime.Format("2006-01-02 15:04:05")
	} else {
		result["processing_time"] = ""
	}

	// 初始化空物流过程列表
	result["logistics_process"] = []interface{}{}

	return result
}

// 工具函数：将订单列表转换为map数组
func convertOrdersToMap(orders []models.Order) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(orders))
	for _, order := range orders {
		result = append(result, convertOrderToMap(order))
	}
	return result
}

// BatchOrdersQuery 批量查询订单 - 与Django版本的batch_orders_query函数对应
func (oc *OrderController) BatchOrdersQuery(c *gin.Context) {
	// 绑定请求参数
	var queryData struct {
		Shopname  string `json:"shopname" binding:"required"`
		UserID    int    `json:"user_id" binding:"required"`
		Status    string `json:"status"`
		Page      int    `json:"page" binding:"required,min=1"`
		PageSize  int    `json:"page_size" binding:"required,min=1,max=50"`
		BeginTime string `json:"begin_time"`
		EndTime   string `json:"end_time"`
	}

	if err := c.ShouldBindJSON(&queryData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请求参数错误"})
		return
	}

	// 验证shopname
	if queryData.Shopname != "youlan_kids" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的店铺名称"})
		return
	}

	// 限制PageSize最大值为50
	if queryData.PageSize > 50 {
		queryData.PageSize = 50
	}

	// 构建查询
	var orders []models.Order
	// 直接指定表名，避免可能的模型映射问题
	query := db.DB.Table("order_data")

	// 应用user_id过滤（必填）
	query = query.Where("user_id = ?", queryData.UserID)
	log.Printf("订单查询 - 用户ID: %d, 状态: %s, 开始时间: %s, 结束时间: %s", queryData.UserID, queryData.Status, queryData.BeginTime, queryData.EndTime)

	// 应用状态过滤
	validStatuses := []string{"pending", "shipped", "delivered", "canceled", "processing"}
	if queryData.Status != "" {
		statusValid := false
		for _, validStatus := range validStatuses {
			if validStatus == queryData.Status {
				statusValid = true
				break
			}
		}
		if !statusValid {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "订单状态无效"})
			return
		}
		query = query.Where("status = ?", queryData.Status)
	}

	// 应用日期过滤
	if queryData.BeginTime != "" {
		beginTime, err := time.Parse("2006-01-02", queryData.BeginTime)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "日期格式必须为YYYY-MM-DD"})
			return
		}
		// 转换为UTC时间（Django默认存储UTC时间）
		beginTime = beginTime.Add(-8 * time.Hour)
		query = query.Where("order_time >= ?", beginTime)
	}

	if queryData.EndTime != "" {
		endTime, err := time.Parse("2006-01-02", queryData.EndTime)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "日期格式必须为YYYY-MM-DD"})
			return
		}
		// 转换为UTC时间并加一天（包含当天）
		endTime = endTime.Add(-8*time.Hour + 24*time.Hour)
		query = query.Where("order_time < ?", endTime)
	}

	// 计算偏移量
	offset := (queryData.Page - 1) * queryData.PageSize

	// 执行分页查询
	var total int64
	if err := query.Count(&total).Error; err != nil {
		log.Printf("获取订单总数失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "服务器内部错误"})
		return
	}

	if err := query.Offset(offset).Limit(queryData.PageSize).Order("order_time DESC").Find(&orders).Error; err != nil {
		log.Printf("获取订单列表失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "服务器内部错误"})
		return
	}

	// 转换订单数据格式
	result := make([]map[string]interface{}, 0, len(orders))
	for _, order := range orders {
		orderMap := convertOrderToMap(order)
		// 如果有物流过程信息，解析并返回
		if order.LogisticsProcess != "" {
			var logisticsProcess []interface{}
			if err := json.Unmarshal([]byte(order.LogisticsProcess), &logisticsProcess); err == nil {
				orderMap["logistics_process"] = logisticsProcess
			}
		}
		result = append(result, orderMap)
	}

	// 准备响应数据
	c.JSON(http.StatusOK, gin.H{
		"code":      200,
		"data":      result,
		"page":      queryData.Page,
		"page_size": queryData.PageSize,
		"total":     total,
	})
}

// SyncLogisticsInfo 同步物流信息 - 与Django版本的sync_logistics_info函数对应
func (oc *OrderController) SyncLogisticsInfo(c *gin.Context) {
	// 绑定请求参数
	var requestData struct {
		OrderID string `json:"order_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请求参数错误"})
		return
	}

	// 查询订单
	var order models.Order
	if err := db.DB.Where("order_id = ?", requestData.OrderID).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "订单不存在"})
		return
	}

	// 检查订单是否有物流信息
	if order.ExpressCompany == "" || order.ExpressNumber == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "订单没有物流信息"})
		return
	}

	// 这里应该调用物流API获取最新物流信息
	// 为了与Django版本保持一致，我们暂时使用模拟数据
	logisticsProcess := []map[string]interface{}{
		{
			"time":        "2023-01-01 12:00:00",
			"location":    "上海市",
			"description": "包裹已发出",
		},
		{
			"time":        "2023-01-02 10:00:00",
			"location":    "北京市",
			"description": "包裹已到达中转中心",
		},
		{
			"time":        "2023-01-03 08:00:00",
			"location":    "广州市",
			"description": "包裹已派送",
		},
	}

	// 更新订单的物流过程信息
	logisticsJSON, err := json.Marshal(logisticsProcess)
	if err != nil {
		log.Printf("转换物流过程数据失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "服务器内部错误"})
		return
	}

	order.LogisticsProcess = string(logisticsJSON)

	// 如果物流状态显示已送达，更新订单状态
	if len(logisticsProcess) > 0 {
		lastStatus := logisticsProcess[len(logisticsProcess)-1]
		if desc, ok := lastStatus["description"].(string); ok {
			if strings.Contains(strings.ToLower(desc), "已签收") ||
				strings.Contains(strings.ToLower(desc), "已送达") {
				order.Status = "delivered"
			}
		}
	}

	// 保存更新
	if err := db.DB.Save(&order).Error; err != nil {
		log.Printf("更新物流信息失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "服务器内部错误"})
		return
	}

	// 返回同步后的物流信息
	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "物流信息同步成功",
		"data": map[string]interface{}{
			"order_id":          requestData.OrderID,
			"express_company":   order.ExpressCompany,
			"express_number":    order.ExpressNumber,
			"logistics_process": logisticsProcess,
		},
	})
}

// ChangeReceivingData 修改收货信息 - 与Django版本的change_receiving_data函数对应
func (oc *OrderController) ChangeReceivingData(c *gin.Context) {
	// 绑定请求数据
	var requestData struct {
		OrderID         string `json:"order_id" binding:"required"`
		ReceiverName    string `json:"receiver_name" binding:"required"`
		ReceiverPhone   string `json:"receiver_phone"` // 改为可选字段
		Province        string `json:"province" binding:"required"`
		City            string `json:"city" binding:"required"`
		County          string `json:"county" binding:"required"`
		DetailedAddress string `json:"detailed_address" binding:"required"`
	}

	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "请求数据无效",
		})
		return
	}

	// 查询订单
	var order models.Order
	if err := db.DB.Where("order_id = ?", requestData.OrderID).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "订单不存在",
		})
		return
	}

	// 检查订单状态是否允许修改收货信息
	if order.Status == "shipped" || order.Status == "delivered" || order.Status == "cancelled" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "订单状态不允许修改收货信息",
		})
		return
	}

	// 更新收货信息
	order.ReceiverName = requestData.ReceiverName
	// 只有当提供了receiver_phone时才更新
	if requestData.ReceiverPhone != "" {
		order.ReceiverPhone = requestData.ReceiverPhone
	}
	order.Province = requestData.Province
	order.City = requestData.City
	order.County = requestData.County
	order.DetailedAddress = requestData.DetailedAddress

	if err := db.DB.Save(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "修改收货信息失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "收货信息修改成功",
		"data":    gin.H{"order_id": requestData.OrderID},
	})
}

// ReturnOrderDeliver 退货订单发货
func (oc *OrderController) ReturnOrderDeliver(c *gin.Context) {
	// 绑定请求数据
	var requestData struct {
		ReturnOrderID  string `json:"return_order_id" binding:"required"`
		ExpressCompany string `json:"express_company" binding:"required"`
		ExpressNumber  string `json:"express_number" binding:"required"`
	}

	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "请求数据无效: " + err.Error(),
		})
		return
	}

	// 查询退货订单
	var returnOrder models.ReturnOrder
	if err := db.DB.Where("return_id = ?", requestData.ReturnOrderID).First(&returnOrder).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "退货订单不存在",
		})
		return
	}

	// 检查退货订单状态是否允许发货
	if returnOrder.Status != "pending" && returnOrder.Status != "processing" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "退货订单状态不允许发货",
		})
		return
	}

	// 更新退货订单信息
	returnOrder.Status = "shipped"
	returnOrder.ShippedTime = time.Now()
	returnOrder.ExpressCompany = requestData.ExpressCompany
	returnOrder.ExpressNumber = requestData.ExpressNumber

	if err := db.DB.Save(&returnOrder).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "退货订单发货失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "退货订单发货成功",
		"data": gin.H{
			"return_order_id": requestData.ReturnOrderID,
			"express_company": requestData.ExpressCompany,
			"express_number":  requestData.ExpressNumber,
		},
	})
}

// ReturnOrderReceive 退货订单签收
func (oc *OrderController) ReturnOrderReceive(c *gin.Context) {
	// 绑定请求数据
	var requestData struct {
		ReturnOrderID string `json:"return_order_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "请求数据无效: " + err.Error(),
		})
		return
	}

	// 查询退货订单
	var returnOrder models.ReturnOrder
	if err := db.DB.Where("return_id = ?", requestData.ReturnOrderID).First(&returnOrder).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "退货订单不存在",
		})
		return
	}

	// 检查退货订单状态是否允许签收
	if returnOrder.Status != "shipped" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "退货订单状态不允许签收",
		})
		return
	}

	// 更新退货订单状态为已完成
	returnOrder.Status = "completed"
	returnOrder.CompletedTime = time.Now()

	if err := db.DB.Save(&returnOrder).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "退货订单签收失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "退货订单签收成功",
		"data":    gin.H{"return_order_id": requestData.ReturnOrderID},
	})
}

// ReturnOrderCancel 退货订单取消
func (oc *OrderController) ReturnOrderCancel(c *gin.Context) {
	// 绑定请求数据
	var requestData struct {
		ReturnOrderID string `json:"return_order_id" binding:"required"`
		Reason        string `json:"reason" binding:"required"`
	}

	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "请求数据无效: " + err.Error(),
		})
		return
	}

	// 查询退货订单
	var returnOrder models.ReturnOrder
	if err := db.DB.Where("return_id = ?", requestData.ReturnOrderID).First(&returnOrder).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "退货订单不存在",
		})
		return
	}

	// 检查退货订单状态是否允许取消
	if returnOrder.Status == "completed" || returnOrder.Status == "canceled" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "退货订单状态不允许取消",
		})
		return
	}

	// 更新退货订单状态为已取消
	returnOrder.Status = "canceled"
	returnOrder.CanceledTime = time.Now()
	// 添加取消原因到备注
	if returnOrder.Remarks != "" {
		returnOrder.Remarks += "\n取消原因: " + requestData.Reason
	} else {
		returnOrder.Remarks = "取消原因: " + requestData.Reason
	}

	if err := db.DB.Save(&returnOrder).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "退货订单取消失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "退货订单取消成功",
		"data":    gin.H{"return_order_id": requestData.ReturnOrderID},
	})
}

// ReturnOrderUpdateBuyerInfo 退货订单修改买家信息
func (oc *OrderController) ReturnOrderUpdateBuyerInfo(c *gin.Context) {
	// 绑定请求数据
	var requestData struct {
		ReturnOrderID string `json:"return_order_id" binding:"required"`
		BuyerProvince string `json:"buyer_province" binding:"required"`
		BuyerCity     string `json:"buyer_city" binding:"required"`
		BuyerCounty   string `json:"buyer_county" binding:"required"`
		BuyerAddress  string `json:"buyer_address" binding:"required"`
		BuyerPhone    string `json:"buyer_phone" binding:"required"`
	}

	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "请求数据无效: " + err.Error(),
		})
		return
	}

	// 查询退货订单
	var returnOrder models.ReturnOrder
	if err := db.DB.Where("return_id = ?", requestData.ReturnOrderID).First(&returnOrder).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "退货订单不存在",
		})
		return
	}

	// 检查退货订单状态是否允许修改买家信息
	if returnOrder.Status == "shipped" || returnOrder.Status == "completed" || returnOrder.Status == "canceled" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "退货订单状态不允许修改买家信息",
		})
		return
	}

	// 更新买家信息
	returnOrder.BuyerProvince = requestData.BuyerProvince
	returnOrder.BuyerCity = requestData.BuyerCity
	returnOrder.BuyerCounty = requestData.BuyerCounty
	returnOrder.BuyerAddress = requestData.BuyerAddress
	returnOrder.BuyerPhone = requestData.BuyerPhone

	if err := db.DB.Save(&returnOrder).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "修改买家信息失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "买家信息修改成功",
		"data":    gin.H{"return_order_id": requestData.ReturnOrderID},
	})
}
