package method

import (
	"bytes"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"django_to_go/config"
	"django_to_go/db"
	"django_to_go/models"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func convertToMap(v interface{}) (map[string]interface{}, bool) {
	// 将结构体转换为JSON
	jsonData, err := json.Marshal(v)
	if err != nil {
		return nil, false
	}

	// 将JSON转换为map
	var result map[string]interface{}
	err = json.Unmarshal(jsonData, &result)
	if err != nil {
		return nil, false
	}

	return result, true
}

// JushuitanOrder 聚水潭订单结构
type JushuitanOrder struct {
	BuyerID      json.Number `json:"buyer_id"` // 使用json.Number处理可能是数字的字段
	Type         string      `json:"type"`
	ShopName     string      `json:"shop_name"`
	SoID         string      `json:"so_id"`
	PayDate      string      `json:"pay_date"`
	Status       string      `json:"status"`
	OrderDate    string      `json:"order_date"`
	SendDate     string      `json:"send_date"`
	EndTime      string      `json:"end_time"`
	PayAmount    interface{} `json:"pay_amount"`
	RawSoID      string      `json:"raw_so_id"`
	OuterOiID    string      `json:"outer_oi_id"`
	LID          string      `json:"l_id"`
	Remark       string      `json:"remark"`
	Consignee    interface{} `json:"consignee"`
	Items        []Item      `json:"items"`
	ReferrerName string      `json:"referrer_name"` // 添加referrer_name字段
	Ts           json.Number `json:"ts"`            // 添加ts字段，用于增量查询
}

// Consignee 收货人信息
type Consignee struct {
	Name     string `json:"name"`
	Province string `json:"province"`
	City     string `json:"city"`
	County   string `json:"county"`
}

// Item 订单项
type Item struct {
	Price              interface{} `json:"price"`
	SellerIncomeAmount interface{} `json:"seller_income_amount"`
	BuyerPaidAmount    interface{} `json:"buyer_paid_amount"`
	OuterOiID          string      `json:"outer_oi_id"`
	OiID               json.Number `json:"oi_id"` // 使用json.Number处理可能是数字的字段
	RawSoID            string      `json:"raw_so_id"`
	IID                string      `json:"i_id"`
	SkuID              string      `json:"sku_id"`
	PropertiesValue    string      `json:"properties_value"`
	Qty                interface{} `json:"qty"`
	Name               string      `json:"name"`
}

// ExtractedOrder 提取的订单数据
type ExtractedOrder struct {
	BuyerID            string      `json:"buyer_id"`
	Type               string      `json:"type"`
	ShopName           string      `json:"shop_name"`
	SoID               string      `json:"so_id"`
	PayDate            string      `json:"pay_date"`
	Price              interface{} `json:"price"`
	SellerIncomeAmount interface{} `json:"seller_income_amount"`
	BuyerPaidAmount    interface{} `json:"buyer_paid_amount"`
	OuterOiID          string      `json:"outer_oi_id"`
	OiID               string      `json:"oi_id"`
	RawSoID            string      `json:"raw_so_id"`
	IID                string      `json:"i_id"`
	SkuID              string      `json:"sku_id"`
	PropertiesValue    string      `json:"properties_value"`
	Qty                interface{} `json:"qty"`
	Name               string      `json:"name"`
}

// JushuitanResponse 聚水潭API响应
type JushuitanResponse struct {
	Data struct {
		Orders []JushuitanOrder `json:"orders"`
	} `json:"data"`
	Msg string `json:"msg"`
}

// BizData 业务参数
type BizData struct {
	PageIndex     string   `json:"page_index"`
	PageSize      string   `json:"page_size"`
	ModifiedBegin string   `json:"modified_begin"`
	ModifiedEnd   string   `json:"modified_end"`
	DateType      string   `json:"date_type"`
	ShopID        string   `json:"shop_id"`
	Status        string   `json:"status"`
	OrderTypes    []string `json:"order_types"`
}

// RequestPayload 请求参数
type RequestPayload struct {
	AppKey      string `json:"app_key"`
	AccessToken string `json:"access_token"`
	Timestamp   string `json:"timestamp"`
	Charset     string `json:"charset"`
	Version     string `json:"version"`
	Biz         string `json:"biz"`
	Sign        string `json:"sign"`
}

// DingTalkMessage 钉钉消息
type DingTalkMessage struct {
	Msgtype  string `json:"msgtype"`
	Markdown struct {
		Title string `json:"title"`
		Text  string `json:"text"`
	} `json:"markdown"`
	At struct {
		IsAtAll bool `json:"isAtAll"`
	} `json:"at"`
}

// FetchJushuitanOrders 获取聚水潭订单数据，支持基于ts时间戳或时间范围的分页查询
// shopID: 店铺ID
// startTime, endTime: 时间范围参数（日期时间字符串）
// start_ts: ts时间戳，sql server中的行版本号，查询条件值是大于等于的关系，可选
// is_get_total: 是否查询总条数，默认true，如果使用start_ts查询，该值需要传false以提高效率
func FetchJushuitanOrders(shopID, startTime, endTime string, start_ts int64, is_get_total bool) ([]ExtractedOrder, []JushuitanOrder, int64, error) {
	// 配置参数
	urlStr := "https://openapi.jushuitan.com/open/orders/single/query"
	appKey := "e50a8f2e66c845c188a04f34ebf4a663"
	accessToken, err := GetToken()
	if err != nil {
		return nil, nil, 0, fmt.Errorf("获取token失败: %v", err)
	}
	secret := "b7a7e5df75ed4ae38c42db4fbe060fb8" // 签名密钥
	charset := "UTF-8"
	version := "2"
	maxPage := 100 // 最大页数限制

	// 使用传入的时间范围参数
	modifiedBegin := startTime
	modifiedEnd := endTime

	// 如果没有传入时间范围，则默认使用昨日时间范围
	if modifiedBegin == "" || modifiedEnd == "" {
		today := time.Now()
		yesterdayBegin := time.Date(today.Year(), today.Month(), today.Day()-1, 0, 0, 0, 0, today.Location())
		yesterdayEnd := time.Date(today.Year(), today.Month(), today.Day()-1, 23, 59, 59, 0, today.Location())
		modifiedBegin = yesterdayBegin.Format("2006-01-02 15:04:05")
		modifiedEnd = yesterdayEnd.Format("2006-01-02 15:04:05")
	}

	// modifiedBegin = "2025-10-01 00:00:00"
	// modifiedEnd = "2025-10-02 23:59:59"
	// 支持三种订单状态
	statuses := []string{"WaitConfirm", "WaitFConfirm", "Sent", "Merged", "Delivering"}
	orderTypes := []string{"普通订单"}

	var allOrders []JushuitanOrder
	var extractedData []ExtractedOrder
	var maxTs int64 = start_ts // 初始化最大ts为传入的start_ts

	// 遍历所有订单状态
	for _, status := range statuses {
		log.Printf("正在获取状态为 %s 的订单\n", status)
		for pageIndex := 1; pageIndex <= maxPage; pageIndex++ {
			// 手动构建biz JSON字符串，根据是否提供start_ts参数决定使用哪种查询方式
			var bizStr string
			if start_ts > 0 {
				// 使用ts时间戳查询方式
				bizStr = fmt.Sprintf(`{"page_index":"%s","page_size":"%s","start_ts":"%d","is_get_total":"%t","shop_id":"%s","status":"%s","order_types":["%s"]}`,
					strconv.Itoa(pageIndex), "100", start_ts, is_get_total, shopID, status, orderTypes[0])
				log.Printf("使用ts查询方式: start_ts=%d, page_index=%d", start_ts, pageIndex)
			} else {
				// 使用时间范围查询方式
				bizStr = fmt.Sprintf(`{"page_index":"%s","page_size":"%s","modified_begin":"%s","modified_end":"%s","date_type":"%s","shop_id":"%s","status":"%s","order_types":["%s"]}`,
					strconv.Itoa(pageIndex), "100", modifiedBegin, modifiedEnd, "2", shopID, status, orderTypes[0])
				log.Printf("使用时间范围查询方式: %s 到 %s, page_index=%d", modifiedBegin, modifiedEnd, pageIndex)
			}

			fmt.Print(bizStr)
			// 获取当前时间戳
			timestamp := strconv.FormatInt(time.Now().Unix(), 10)

			// 构建签名字符串
			signStr := secret + "access_token" + accessToken +
				"app_key" + appKey +
				"biz" + bizStr +
				"charset" + charset +
				"timestamp" + timestamp +
				"version" + version

			// 生成MD5签名
			h := md5.New()
			h.Write([]byte(signStr))
			sign := fmt.Sprintf("%x", h.Sum(nil))

			// 构建请求参数 - 使用表单格式与Python版本保持一致
			data := url.Values{}
			data.Set("app_key", appKey)
			data.Set("access_token", accessToken)
			data.Set("timestamp", timestamp)
			data.Set("charset", charset)
			data.Set("version", version)
			data.Set("biz", bizStr)
			data.Set("sign", sign)

			// 构建表单数据
			formData := data.Encode()

			req, err := http.NewRequest("POST", urlStr, strings.NewReader(formData))

			if err != nil {
				return nil, nil, 0, fmt.Errorf("创建请求失败: %v", err)
			}

			fmt.Printf("请求Body: %s\n", formData)
			// 设置正确的Content-Type
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			client := &http.Client{}
			resp, err := client.Do(req)
			// fmt.Print(resp)
			if err != nil {
				return nil, nil, 0, fmt.Errorf("发送请求失败: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return nil, nil, 0, fmt.Errorf("请求状态异常: %d", resp.StatusCode)
			}

			// 解析响应
			fmt.Printf("响应状态码: %d\n", resp.StatusCode)
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return nil, nil, 0, fmt.Errorf("读取响应失败: %v", err)
			}

			// 打印响应body内容
			fmt.Printf("响应Body: %s\n", string(body))

			var result JushuitanResponse
			if err := json.Unmarshal(body, &result); err != nil {
				return nil, nil, 0, fmt.Errorf("解析响应失败: %v", err)
			}

			// 打印解析后的结果
			// fmt.Printf("解析后的结果: Status=%s, Message=%s\n", result.Status, result.Message)
			fmt.Printf("订单数量: %d\n", len(result.Data.Orders))

			// 提取订单数据
			orders := result.Data.Orders
			allOrders = append(allOrders, orders...)

			// 对于ts查询方式，检查并更新最大ts值
			if start_ts > 0 && len(orders) > 0 {
				// 根据API返回的实际结构，从订单数据中提取ts值
				for _, order := range orders {
					// 从订单结构中直接获取ts值
					if order.Ts.String() != "" {
						tempTs, err := strconv.ParseInt(order.Ts.String(), 10, 64)
						if err == nil {
							// 如果ts值大于当前最大ts，则更新
							if tempTs > maxTs {
								maxTs = tempTs
								log.Printf("更新最大ts值: %d (来自订单 %s)", maxTs, order.SoID)
							}
						}
					}
				}
			}

			// 提取所需字段
			for _, order := range orders {
				// 提取订单级别信息 - 将json.Number转换为string
				orderInfo := ExtractedOrder{
					BuyerID:  order.BuyerID.String(),
					Type:     order.Type,
					ShopName: order.ShopName,
					SoID:     order.SoID,
					PayDate:  order.PayDate,
				}

				// 提取商品级别信息
				for _, item := range order.Items {
					itemData := orderInfo
					itemData.Price = item.Price
					itemData.SellerIncomeAmount = item.SellerIncomeAmount
					itemData.BuyerPaidAmount = item.BuyerPaidAmount
					itemData.OuterOiID = item.OuterOiID
					itemData.RawSoID = item.RawSoID
					itemData.IID = item.IID
					itemData.SkuID = item.SkuID
					itemData.PropertiesValue = item.PropertiesValue
					itemData.Qty = item.Qty
					itemData.Name = item.Name

					itemData.OiID = item.OiID.String() // 将json.Number转换为string
					extractedData = append(extractedData, itemData)
				}
			}

			// 检查是否最后一页
			if len(orders) < 100 {
				break
			}
		}
	}

	return extractedData, allOrders, maxTs, nil
}

// MapToSnowOrderData 将聚水潭原始订单数据映射到SnowOrderData模型
func MapToSnowOrderData(rawOrders []JushuitanOrder, shopID string) []models.SnowOrderData {
	var mappedData []models.SnowOrderData
	log.Printf("开始映射订单数据，总订单数: %d, 店铺ID: %s\n", len(rawOrders), shopID)

	// 统计变量
	// totalOrders := len(rawOrders)
	filteredOrders := 0
	keptOrders := 0

	for _, order := range rawOrders {
		// 对于shop_id为11679528的店铺，需要检查referrer_name
		if shopID == "11679528" {
			referrerName := order.ReferrerName
			log.Printf("处理店铺11679528的订单: %s, referrer_name值: '%s', 长度: %d\n",
				order.SoID, referrerName, len(referrerName))

			// 严格过滤逻辑：只保留referrer_name为'幼岚官方旗舰店'或完全为空的订单
			if referrerName != "幼岚官方旗舰店" && referrerName != "" {
				log.Printf("[过滤] 订单 %s 被过滤，referrer_name: '%s'\n", order.SoID, referrerName)
				filteredOrders++
				continue
			} else {
				log.Printf("[保留] 订单 %s 被保留，referrer_name: '%s'\n", order.SoID, referrerName)
				keptOrders++
			}
		}
		// 记录每个订单的shopID和shopName
		log.Printf("订单 %s 的shopID: %s, shopName: %s\n", order.SoID, shopID, order.ShopName)

		// 解析日期时间
		parseDateTime := func(dateStr string) *time.Time {
			if dateStr != "" {
				// 创建Asia/Shanghai时区
				loc, _ := time.LoadLocation("Asia/Shanghai")
				// 使用带时区的解析函数
				t, err := time.ParseInLocation("2006-01-02 15:04:05", dateStr, loc)
				if err == nil {
					return &t
				}
				// 如果带时区解析失败，尝试不带时区的解析并添加时区
				t, err = time.Parse("2006-01-02 15:04:05", dateStr)
				if err == nil {
					// 将时间转换到Asia/Shanghai时区
					t = t.In(loc)
					return &t
				}
			}
			return nil
		}

		// 从订单数据中提取收货人信息
		var consigneeName, province, city, county string
		if consigneeMap, ok := order.Consignee.(map[string]interface{}); ok {
			if name, ok := consigneeMap["name"].(string); ok {
				consigneeName = name
			}
			if p, ok := consigneeMap["province"].(string); ok {
				province = p
			}
			if c, ok := consigneeMap["city"].(string); ok {
				city = c
			}
			if co, ok := consigneeMap["county"].(string); ok {
				county = co
			}
		}

		// 转换实付金额
		actualPaymentAmount := 0.0
		if payAmount, ok := order.PayAmount.(float64); ok {
			actualPaymentAmount = payAmount
		} else if payAmountStr, ok := order.PayAmount.(string); ok {
			if pa, err := strconv.ParseFloat(payAmountStr, 64); err == nil {
				actualPaymentAmount = pa
			}
		}

		// 根据订单状态设置对应的中文描述
		var orderStatusDesc string
		switch order.Status {
		case "WaitConfirm", "WaitFConfirm":
			orderStatusDesc = "已付款"
		case "Sent":
			orderStatusDesc = "已发货"
		case "Merged", "Delivering":
			orderStatusDesc = "发货中"
		default:
			orderStatusDesc = order.Status // 保留原始状态
		}

		// 映射字段 - 直接对应SnowOrderData模型的每个字段
		snowOrder := models.SnowOrderData{
			SerialNumber:              0, // 将在数据库中自增
			OnlineOrderNumber:         order.SoID,
			OrderStatus:               orderStatusDesc,
			Store:                     order.ShopName,
			OrderDate:                 parseDateTime(order.OrderDate),
			ShipDate:                  parseDateTime(order.SendDate),
			PaymentDate:               parseDateTime(order.PayDate),
			SellerID:                  order.BuyerID.String(),
			ConfirmReceiptTime:        parseDateTime(order.EndTime),
			ConsigneeName:             consigneeName,
			Province:                  province,
			City:                      city,
			County:                    county,
			TrackingNumber:            order.LID,
			OriginalOnlineOrderNumber: order.RawSoID,
			ActualPaymentAmount:       actualPaymentAmount,
			ReturnQuantity:            0,
			ReturnAmount:              0.0,
			OnlineSubOrderNumber:      order.OuterOiID,
			Remark:                    "聚水潭",
		}

		// 为每个订单项创建一条记录
		for _, item := range order.Items {
			// 创建订单项的副本
			itemOrder := snowOrder
			// 如果订单项有子订单号，则使用订单项的子订单号
			if item.OuterOiID != "" {
				itemOrder.OnlineSubOrderNumber = item.OuterOiID
			}
			// 添加商品名称到备注
			// if item.Name != "" {
			// 	if itemOrder.Remark != "" {
			// 		itemOrder.Remark += " | 商品: " + item.Name
			// 	} else {
			// 		itemOrder.Remark = item.Name
			// 	}
			// }

			mappedData = append(mappedData, itemOrder)
		}
	}

	return mappedData
}

// SendDingTalkMessage 发送钉钉通知
func SendDingTalkMessage(shopName string, ordersResult []int) error {
	// 钉钉机器人配置
	webhook := "https://oapi.dingtalk.com/robot/send?access_token=90f3fae0aa0e03a8ca113f6e99f97998700a0d769cca3340f881db7d873345d6"
	appSecret := "SEC4a5d4c9477980ad0e78fe62b47b44629b9dc5cedb02c0c6e541ac53e2bc52ad1"

	// 获取当前时间戳（毫秒）
	timestamp := strconv.FormatInt(time.Now().UnixNano()/1000000, 10)

	// 生成签名
	signStr := timestamp + "\n" + appSecret
	h := hmac.New(sha256.New, []byte(appSecret))
	h.Write([]byte(signStr))
	sign := url.QueryEscape(base64.StdEncoding.EncodeToString(h.Sum(nil)))

	// 构造最终URL
	fullURL := webhook + "&timestamp=" + timestamp + "&sign=" + sign

	// 解析处理结果
	originalCount, filteredCount, insertCount := 0, 0, 0
	if len(ordersResult) >= 1 {
		originalCount = ordersResult[0]
	}
	if len(ordersResult) >= 2 {
		filteredCount = ordersResult[1]
	}
	if len(ordersResult) >= 3 {
		insertCount = ordersResult[2]
	}

	// 创建消息内容
	message := DingTalkMessage{
		Msgtype: "markdown",
	}
	message.Markdown.Title = "聚水潭数据同步报告"
	message.Markdown.Text = fmt.Sprintf(
		"聚水潭数据同步至数据库中\n### 🏪 %s 数据同步完成\n\n**订单数据**:\n- ✅ 原始订单: %d 条\n- ✅ 过滤订单: %d 条\n- ⚠️ 插入订单: %d 条\n\n**处理时间**: %s",
		shopName, originalCount, filteredCount, insertCount, time.Now().Format("2006-01-02 15:04:05"),
	)
	message.At.IsAtAll = false

	// 发送请求
	messageJSON, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %v", err)
	}

	req, err := http.NewRequest("POST", fullURL, bytes.NewBuffer(messageJSON))
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("请求状态异常: %d", resp.StatusCode)
	}

	return nil
}

// SaveSnowOrderDataToDB 将数据保存到数据库
func SaveSnowOrderDataToDB(data []models.SnowOrderData) (int, error) {
	// 检查数据库连接是否初始化（正式环境）
	if db.DB == nil {
		return 0, fmt.Errorf("数据库连接未初始化，无法保存数据到数据库")
	}

	// 检查数据切片是否为空
	if len(data) == 0 {
		log.Println("提示: 没有数据需要保存到SnowOrderData模型")
		return 0, nil
	}

	// 分批插入数据，每批100条，避免MySQL占位符数量限制
	batchSize := 100
	totalInserted := 0

	for i := 0; i < len(data); i += batchSize {
		end := i + batchSize
		if end > len(data) {
			end = len(data)
		}

		batchData := data[i:end]
		result := db.DB.Create(&batchData)
		if result.Error != nil {
			return totalInserted, fmt.Errorf("保存数据到数据库失败: %v", result.Error)
		}

		totalInserted += int(result.RowsAffected)
		log.Printf("已保存第 %d-%d 条记录，当前累计保存 %d 条", i+1, end, totalInserted)
	}

	// 记录成功保存的记录数
	log.Printf("成功保存 %d 条记录到SnowOrderData模型", totalInserted)

	return totalInserted, nil
}

// GetToken 获取聚水潭访问令牌
// 实现与Python版本相同的token获取逻辑
func GetToken() (string, error) {
	// 配置参数 - 与Python版本保持一致
	appKey := "e50a8f2e66c845c188a04f34ebf4a663"
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	charset := "utf-8"
	appSecret := "b7a7e5df75ed4ae38c42db4fbe060fb8"
	grantType := "authorization_code"
	code := "4xFIOC"

	// 构建签名字符串
	signStr := appSecret + "app_key" + appKey + "charset" + charset + "code" + code + "grant_type" + grantType + "timestamp" + timestamp

	// 生成MD5签名
	h := md5.New()
	h.Write([]byte(signStr))
	sign := fmt.Sprintf("%x", h.Sum(nil))

	// 构建请求参数
	data := url.Values{}
	data.Set("app_key", appKey)
	data.Set("grant_type", grantType)
	data.Set("timestamp", timestamp)
	data.Set("code", code)
	data.Set("charset", charset)
	data.Set("sign", sign)

	// 发送请求
	req, err := http.NewRequest("POST", "https://openapi.jushuitan.com/openWeb/auth/getInitToken", strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("创建token请求失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("发送token请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取token响应失败: %v", err)
	}

	// 解析响应
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("解析token响应失败: %v", err)
	}

	// 提取access_token
	dataMap, ok := result["data"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("响应中没有data字段或格式错误")
	}

	token, ok := dataMap["access_token"].(string)
	if !ok {
		return "", fmt.Errorf("响应中没有access_token字段或格式错误")
	}

	return token, nil
}

// Main 主函数，接受起始时间和结束时间参数
func Main(startTime, endTime string) {
	// 初始化数据库连接
	appConfig := config.LoadConfig()
	log.Println("正在初始化数据库连接...")
	db.InitDB(appConfig)
	log.Println("数据库连接初始化完成")

	shopDict := map[string]string{
		"幼岚-抖音官方旗舰店":    "11679528",
		"幼岚-抖音童装旗舰店":    "16540940",
		"幼岚-抖音旗舰店":      "11425575",
		"幼岚-京东":         "12573089",
		"幼岚-唯品会":        "12597654",
		"幼岚-微信视频号小店":    "14395031",
		"幼岚-上生新所 视频号小店": "17380919",
		// 其他店铺可以在这里添加
	}

	// shopDict := map[string]string{
	// 	"幼岚-京东": "12573089",
	// }

	for shopName, shopID := range shopDict {
		fmt.Printf("正在处理 %s 的数据，时间范围：%s 到 %s\n", shopName, startTime, endTime)

		// 获取订单数据，传入时间范围参数
		// 对于基于时间范围的查询，start_ts设为0，is_get_total设为true
		extractedOrders, rawOrders, maxTs, err := FetchJushuitanOrders(shopID, startTime, endTime, 0, true)
		if err != nil {
			fmt.Printf("获取订单数据失败: %v\n", err)
			continue
		}

		// 记录本次查询的最大ts值（如果有）
		if maxTs > 0 {
			fmt.Printf("本次查询的最大ts值: %d\n", maxTs)
			fmt.Printf("下次基于ts的查询可以使用此值作为start_ts参数\n")
		}

		// 将原始订单数据映射到SnowOrderData模型
		snowOrderData := MapToSnowOrderData(rawOrders, shopID)
		fmt.Printf("已映射 %d 条SnowOrderData模型格式数据\n", len(snowOrderData))

		// 将数据保存到数据库
		insertedCount, err := SaveSnowOrderDataToDB(snowOrderData)
		if err != nil {
			fmt.Printf("保存数据到数据库失败: %v\n", err)
			continue
		}
		fmt.Printf("数据已成功保存到SnowOrderData模型，共 %d 条记录\n", insertedCount)

		// 构建订单处理结果，用于钉钉通知
		ordersResult := []int{len(extractedOrders), 0, insertedCount}

		// 发送钉钉通知
		if err := SendDingTalkMessage(shopName, ordersResult); err != nil {
			fmt.Printf("发送钉钉通知失败: %v\n", err)
		}
	}
}
