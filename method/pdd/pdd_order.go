package pdd

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"django_to_go/config"
	"django_to_go/db"
	"django_to_go/models"
)

const (
	// ClientID 固定的client_id
	ClientID = "4b415953a5294085b1559afc0c453cb7"
	// APIURL 拼多多API网关地址
	APIURL = "https://gw-api.pinduoduo.com/api/router"
)

// PDDPaymentItem 拼多多支付商品项

type PDDPaymentItem struct {
	GoodsCount   int     `json:"goods_count"`
	GoodsID      int64   `json:"goods_id"`
	GoodsName    string  `json:"goods_name"`
	GoodsPrice   float64 `json:"goods_price"`
	OuterGoodsID string  `json:"outer_goods_id"`
	OuterID      string  `json:"outer_id"`
	GoodsSpec    string  `json:"goods_spec"`
	SkuID        int64   `json:"sku_id"`
}

// PDDSimplifiedOrder 拼多多简化订单结构

type PDDSimplifiedOrder struct {
	ShopName  string           `json:"shop_name"`
	OrderSN   string           `json:"order_sn"`
	PayAmount float64          `json:"pay_amount"`
	PayTime   string           `json:"pay_time"`
	ItemList  []PDDPaymentItem `json:"item_list"`
}

// TimePeriod 时间段结构

type TimePeriod struct {
	Name  string
	Start int
	End   int
}

// generateSignAdvanced 生成拼多多API签名
func generateSignAdvanced(params map[string]string, clientSecret string, excludeKeys ...string) string {
	// 默认排除sign参数
	if len(excludeKeys) == 0 {
		excludeKeys = []string{"sign"}
	}

	// 创建排除键的映射以提高查找效率
	excludeMap := make(map[string]bool)
	for _, key := range excludeKeys {
		excludeMap[key] = true
	}

	// 过滤掉需要排除的参数
	filteredParams := make(map[string]string)
	for k, v := range params {
		if !excludeMap[k] && v != "" {
			filteredParams[k] = v
		}
	}

	// 步骤1：参数排序
	var keys []string
	for k := range filteredParams {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 步骤2：字符串拼接
	var baseString strings.Builder
	for _, key := range keys {
		baseString.WriteString(key)
		baseString.WriteString(filteredParams[key])
	}

	finalString := clientSecret + baseString.String() + clientSecret

	// 步骤3：生成sign值
	h := md5.New()
	h.Write([]byte(finalString))
	sign := hex.EncodeToString(h.Sum(nil))

	return strings.ToUpper(sign)
}

// parsePayTime 解析支付时间字符串为time.Time
func parsePayTime(payTimeStr string) (*time.Time, error) {
	// 移除时间字符串中的多余空格
	payTimeStr = strings.ReplaceAll(payTimeStr, " ", "")
	log.Printf("解析支付时间: %s", payTimeStr)

	// 尝试不同的时间格式 - 解析为本地时间后直接减去8小时
	formats := []string{
		"2006-01-0215:04:05",
		"2006-01-0215:04:05",
		"2006-01-0215:04",
	}

	for _, format := range formats {
		t, err := time.Parse(format, payTimeStr)
		if err == nil {
			// 直接减去8小时
			adjustedTime := t.Add(-8 * time.Hour)
			log.Printf("成功解析支付时间并减去8小时: %s (原时间: %s), Unix时间戳: %d", 
				adjustedTime.Format("2006-01-02 15:04:05"), 
				t.Format("2006-01-02 15:04:05"), 
				adjustedTime.Unix())
			return &adjustedTime, nil
		}
	}

	return nil, fmt.Errorf("无法解析支付时间: %s", payTimeStr)
}

// ConvertToSnowOrderData 将PDDSimplifiedOrder转换为SnowOrderData
func ConvertToSnowOrderData(pddOrder PDDSimplifiedOrder) (*models.SnowOrderData, error) {
	// 解析支付时间
	payTime, err := parsePayTime(pddOrder.PayTime)
	if err != nil {
		log.Printf("解析支付时间失败: %v, 订单号: %s", err, pddOrder.OrderSN)
		// 使用当前时间作为备选，直接减去8小时
		now := time.Now()
		adjustedTime := now.Add(-8 * time.Hour)
		log.Printf("使用当前时间并减去8小时作为备选: %s (原时间: %s), Unix时间戳: %d", 
			adjustedTime.Format("2006-01-02 15:04:05"), 
			now.Format("2006-01-02 15:04:05"), 
			adjustedTime.Unix())
		payTime = &adjustedTime
	} else {
		log.Printf("订单 %s 的支付时间: %s, Unix时间戳: %d",
			pddOrder.OrderSN,
			payTime.Format("2006-01-02 15:04:05"),
			payTime.Unix())
	}

	// 创建SnowOrderData对象
	// 注意：SerialNumber、SellerID、ConsigneeName、Province、City、County等必需字段需要设置默认值或通过其他方式获取
	snowOrder := &models.SnowOrderData{
		OnlineOrderNumber:         pddOrder.OrderSN,   // 对应order_sn
		OrderStatus:               "已付款",              // 统一设置为"已付款"
		Store:                     pddOrder.ShopName,  // 对应shop_name
		OrderDate:                 payTime,            // 对应pay_time
		ShipDate:                  payTime,            // 对应pay_time
		PaymentDate:               payTime,            // 对应pay_time
		SellerID:                  "0",                // 默认值，实际应用中可能需要从其他地方获取
		ConsigneeName:             "",                 // 空字符串，实际应用中可能需要从其他地方获取
		Province:                  "",                 // 空字符串，实际应用中可能需要从其他地方获取
		City:                      "",                 // 空字符串，实际应用中可能需要从其他地方获取
		County:                    "",                 // 空字符串，实际应用中可能需要从其他地方获取
		ActualPaymentAmount:       pddOrder.PayAmount, // 对应pay_amount
		OriginalOnlineOrderNumber: pddOrder.OrderSN,   // 原始线上订单号也使用order_sn
	}

	log.Printf("创建订单数据: 订单号=%s, 店铺=%s, 准备写入数据库", pddOrder.OrderSN, pddOrder.ShopName)
	return snowOrder, nil
}

// GetPDOrders 获取拼多多订单列表
func GetPDOrders(startTimeStr, endTimeStr string) ([]models.SnowOrderData, error) {
	// 从dingdan_test.go中固定的access_tokens和CLIENT_SECRET配置
	accessTokens := map[string]string{
		"拼多多官方旗舰店": "3bab67a2267e469b8af8c650f3f01c1f0f68d26b",
		"拼多多童装旗舰店": "94049dfee30044b7ac5632bbe0163ff3480e0199",
		"拼多多户外旗舰店": "4b143a73fc2346eaa226c10672b05377905087df",
	}
	const CLIENT_SECRET = "c584c4924f5ed15e393f1f16cb30993c12a655ad"

	// 记录原始输入时间
	log.Printf("原始输入时间范围: startTimeStr=%s, endTimeStr=%s", startTimeStr, endTimeStr)

	// 解析开始时间和结束时间 - 使用本地时区，与Python版本保持一致
	startTime, err := time.Parse("2006-01-02 15:04:05", startTimeStr)
	if err != nil {
		return nil, fmt.Errorf("解析开始时间失败: %v", err)
	}
	log.Printf("解析后startTime (本地时间): %s, Unix时间戳: %d", startTime.Format("2006-01-02 15:04:05"), startTime.Unix())

	endTime, err := time.Parse("2006-01-02 15:04:05", endTimeStr)
	if err != nil {
		return nil, fmt.Errorf("解析结束时间失败: %v", err)
	}
	log.Printf("解析后endTime (本地时间): %s, Unix时间戳: %d", endTime.Format("2006-01-02 15:04:05"), endTime.Unix())

	// 存储所有获取到的订单
	var allPDOrders []PDDSimplifiedOrder

	// 遍历所有店铺
	for shopName, token := range accessTokens {
		log.Printf("正在获取 %s 的订单，时间范围：%s 到 %s (本地时间)", shopName, startTime.Format("2006-01-02 15:04:05"), endTime.Format("2006-01-02 15:04:05"))

		// 直接将本地时间转换为UTC时间（减去8小时）以确保时间戳正确
		// 这是为了解决API期望的时间戳格式问题
		timeZoneOffset := 8 * time.Hour // 8小时时区偏移
		startTimeUTC := startTime.Add(-timeZoneOffset)
		endTimeUTC := endTime.Add(-timeZoneOffset)
		startConfirmAt := strconv.FormatInt(startTimeUTC.Unix(), 10)
		endConfirmAt := strconv.FormatInt(endTimeUTC.Unix(), 10)
		log.Printf("API请求时间参数: start_confirm_at=%s (原始本地时间: %s, 转换后UTC时间: %s)",
			startConfirmAt, startTime.Format("2006-01-02 15:04:05"), startTimeUTC.Format("2006-01-02 15:04:05 UTC"))
		log.Printf("API请求时间参数: end_confirm_at=%s (原始本地时间: %s, 转换后UTC时间: %s)",
			endConfirmAt, endTime.Format("2006-01-02 15:04:05"), endTimeUTC.Format("2006-01-02 15:04:05 UTC"))

		page := 1
		pageSize := 100
		maxPages := 10

		// 分页获取订单
		for page <= maxPages {
			// 准备请求参数
			timestamp := strconv.FormatInt(time.Now().Unix(), 10)

			params := map[string]string{
				"type":             "pdd.order.list.get",
				"timestamp":        timestamp,
				"client_id":        ClientID,
				"data_type":        "json",
				"access_token":     token,
				"page":             strconv.Itoa(page),
				"page_size":        strconv.Itoa(pageSize),
				"order_status":     "5",
				"refund_status":    "5",
				"start_confirm_at": startConfirmAt,
				"end_confirm_at":   endConfirmAt,
			}

			// 生成签名
			sign := generateSignAdvanced(params, CLIENT_SECRET)
			params["sign"] = sign

			// 构建POST请求体
			var formData bytes.Buffer
			for k, v := range params {
				if formData.Len() > 0 {
					formData.WriteByte('&')
				}
				formData.WriteString(fmt.Sprintf("%s=%s", k, v))
			}

			log.Printf("请求参数: %s", formData.String())
			// 发送POST请求
			req, err := http.NewRequest("POST", APIURL, &formData)
			if err != nil {
				log.Printf("创建请求失败: %v", err)
				break
			}

			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			client := &http.Client{Timeout: 30 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				log.Printf("发送请求失败: %v", err)
				break
			}
			defer resp.Body.Close()

			// 读取响应内容
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Printf("读取响应失败: %v", err)
				break
			}
			// log.Printf("原始响应内容: %s", string(body))
			// 解析响应
			var result map[string]interface{}
			if err := json.Unmarshal(body, &result); err != nil {
				log.Printf("解析响应失败: %v", err)
				break
			}

			// 检查是否有错误
			if errorResponse, exists := result["error_response"].(map[string]interface{}); exists {
				errorMsg := "未知错误"
				if msg, ok := errorResponse["error_msg"].(string); ok {
					errorMsg = msg
				}
				log.Printf("获取订单失败: %s", errorMsg)
				break
			}

			// 处理订单数据
			if orderResponse, exists := result["order_list_get_response"].(map[string]interface{}); exists {
				if orderList, ok := orderResponse["order_list"].([]interface{}); ok {
					if len(orderList) > 0 {
						// 提取关键字段并添加店铺名
						for _, order := range orderList {
							if orderMap, ok := order.(map[string]interface{}); ok {
								simplifiedOrder := PDDSimplifiedOrder{
									ShopName: shopName,
								}

								// 提取订单信息
								if orderSN, ok := orderMap["order_sn"].(string); ok {
									simplifiedOrder.OrderSN = orderSN
								}

								if payAmount, ok := orderMap["pay_amount"].(float64); ok {
									simplifiedOrder.PayAmount = payAmount
								}

								if payTime, ok := orderMap["pay_time"].(string); ok {
									simplifiedOrder.PayTime = payTime
								}

								// 处理商品列表
								if itemList, ok := orderMap["item_list"].([]interface{}); ok {
									for _, item := range itemList {
										if itemMap, ok := item.(map[string]interface{}); ok {
											simplifiedItem := PDDPaymentItem{}

											if goodsCount, ok := itemMap["goods_count"].(float64); ok {
												simplifiedItem.GoodsCount = int(goodsCount)
											}

											if goodsID, ok := itemMap["goods_id"].(float64); ok {
												simplifiedItem.GoodsID = int64(goodsID)
											}

											if goodsName, ok := itemMap["goods_name"].(string); ok {
												simplifiedItem.GoodsName = goodsName
											}

											if goodsPrice, ok := itemMap["goods_price"].(float64); ok {
												simplifiedItem.GoodsPrice = goodsPrice
											}

											if outerGoodsID, ok := itemMap["outer_goods_id"].(string); ok {
												simplifiedItem.OuterGoodsID = outerGoodsID
											}

											if outerID, ok := itemMap["outer_id"].(string); ok {
												simplifiedItem.OuterID = outerID
											}

											if goodsSpec, ok := itemMap["goods_spec"].(string); ok {
												simplifiedItem.GoodsSpec = goodsSpec
											}

											if skuID, ok := itemMap["sku_id"].(float64); ok {
												simplifiedItem.SkuID = int64(skuID)
											}

											simplifiedOrder.ItemList = append(simplifiedOrder.ItemList, simplifiedItem)
										}
									}
								}

								allPDOrders = append(allPDOrders, simplifiedOrder)
							}
						}

						log.Printf("  获取到 %d 条订单", len(orderList))

						// 如果返回的订单数量小于page_size，说明已经是最后一页
						if len(orderList) < pageSize {
							break
						}
					} else {
						// 没有更多订单
						break
					}
				} else {
					log.Printf("  响应中没有找到订单列表")
					break
				}
			} else {
				log.Printf("  响应格式不正确")
				break
			}

			page++
			// 添加延迟避免请求过于频繁
			time.Sleep(500 * time.Millisecond)
		}

		// 计算该店铺的订单数量
		shopOrderCount := 0
		for _, order := range allPDOrders {
			if order.ShopName == shopName {
				shopOrderCount++
			}
		}
		log.Printf("%s 订单获取完成，共获取 %d 条订单\n", shopName, shopOrderCount)
	}

	// 转换为SnowOrderData模型并写入数据库
	var snowOrderDataList []models.SnowOrderData

	// 检查数据库连接是否初始化
	if db.DB == nil {
		log.Printf("警告: 数据库连接未初始化，尝试初始化数据库连接...")

		// 尝试加载配置并初始化数据库
		appConfig := config.LoadConfig()
		db.InitDB(appConfig)
		log.Printf("数据库连接初始化完成")

		// 再次检查数据库连接
		if db.DB == nil {
			log.Printf("错误: 数据库连接初始化失败，仅返回转换后的订单数据，不写入数据库")

			// 只转换数据，不写入数据库
			for _, pddOrder := range allPDOrders {
				snowOrder, err := ConvertToSnowOrderData(pddOrder)
				if err != nil {
					log.Printf("转换订单失败: %v, 订单号: %s", err, pddOrder.OrderSN)
					continue
				}
				snowOrderDataList = append(snowOrderDataList, *snowOrder)
			}

			return snowOrderDataList, nil
		}
	}

	tx := db.DB.Begin() // 开启事务
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			log.Printf("数据库操作发生panic，已回滚事务: %v", r)
		}
	}()

	// 用于跟踪成功写入的订单数
	successCount := 0
	errorCount := 0

	for _, pddOrder := range allPDOrders {
		log.Printf("开始处理订单: %s, 原始支付时间: %s", pddOrder.OrderSN, pddOrder.PayTime)
		snowOrder, err := ConvertToSnowOrderData(pddOrder)
		if err != nil {
			log.Printf("转换订单失败: %v, 订单号: %s", err, pddOrder.OrderSN)
			errorCount++
			continue
		}

		// 记录将要写入数据库的时间字段值
		if snowOrder.OrderDate != nil {
			log.Printf("订单 %s 数据库写入时间 (本地时间): OrderDate=%s (Unix: %d), ShipDate=%s (Unix: %d), PaymentDate=%s (Unix: %d)",
				snowOrder.OnlineOrderNumber,
				snowOrder.OrderDate.Format("2006-01-02 15:04:05"), snowOrder.OrderDate.Unix(),
				snowOrder.ShipDate.Format("2006-01-02 15:04:05"), snowOrder.ShipDate.Unix(),
				snowOrder.PaymentDate.Format("2006-01-02 15:04:05"), snowOrder.PaymentDate.Unix())
		}

		// 检查订单是否已存在
		var existingOrder models.SnowOrderData
		result := tx.Where("online_order_number = ?", snowOrder.OnlineOrderNumber).First(&existingOrder)

		if result.Error != nil {
			// 订单不存在，创建新订单
			if err := tx.Create(snowOrder).Error; err != nil {
				log.Printf("创建订单失败: %v, 订单号: %s", err, snowOrder.OnlineOrderNumber)
				tx.Rollback()
				return nil, fmt.Errorf("创建订单失败: %v", err)
			}
			log.Printf("成功创建新订单，订单号: %s", snowOrder.OnlineOrderNumber)
		} else {
			// 订单已存在，更新订单信息
			if err := tx.Model(&existingOrder).Updates(snowOrder).Error; err != nil {
				log.Printf("更新订单失败: %v, 订单号: %s", err, snowOrder.OnlineOrderNumber)
				tx.Rollback()
				return nil, fmt.Errorf("更新订单失败: %v", err)
			}
			log.Printf("成功更新订单，订单号: %s", snowOrder.OnlineOrderNumber)
		}

		snowOrderDataList = append(snowOrderDataList, *snowOrder)
		successCount++
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		log.Printf("提交事务失败: %v", err)
		return nil, fmt.Errorf("提交事务失败: %v", err)
	}

	log.Printf("订单数据写入数据库完成，成功写入: %d 条，失败: %d 条", successCount, errorCount)

	return snowOrderDataList, nil
}

// MainPDDOrder 拼多多订单主函数，与dingdan.go中的Main函数保持一致的签名
// startTime: 开始时间字符串，格式为"2006-01-02 15:04:05"
// endTime: 结束时间字符串，格式为"2006-01-02 15:04:05"
func MainPDDOrder(startTime, endTime string) {
	// 初始化数据库连接
	appConfig := config.LoadConfig()
	log.Println("正在初始化数据库连接...")
	db.InitDB(appConfig)
	log.Println("数据库连接初始化完成")

	log.Printf("开始获取拼多多订单，时间范围：%s 到 %s\n", startTime, endTime)

	// 调用GetPDOrders函数获取订单数据
	snowOrderData, err := GetPDOrders(startTime, endTime)
	if err != nil {
		log.Printf("获取拼多多订单失败: %v\n", err)
		return
	}

	log.Printf("拼多多订单获取完成，共获取 %d 条订单记录\n", len(snowOrderData))
}
