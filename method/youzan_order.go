package method

import (
	"bytes"
	"django_to_go/config"
	"django_to_go/db"
	"django_to_go/models"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"
)

// YouzanOrderResponse 有赞API响应结构
type YouzanOrderResponse struct {
	TraceID string `json:"trace_id"`
	Code    int    `json:"code"`
	Data    struct {
		FullOrderInfoList []struct {
			FullOrderInfo YouzanFullOrderInfo `json:"full_order_info"`
		} `json:"full_order_info_list"`
		TotalResults int `json:"total_results"`
	} `json:"data"`
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// YouzanFullOrderInfo 有赞完整订单信息
type YouzanFullOrderInfo struct {
	OrderInfo struct {
		ConsignTime string `json:"consign_time"`
		Created     string `json:"created"`
		StatusStr   string `json:"status_str"`
		SuccessTime string `json:"success_time"`
		Type        int    `json:"type"`
		ShopName    string `json:"shop_name"`
		Tid         string `json:"tid"`
		PayTime     string `json:"pay_time"`
		Status      string `json:"status"`
	} `json:"order_info"`
	AddressInfo struct {
		ReceiverName     string `json:"receiver_name"`
		DeliveryProvince string `json:"delivery_province"`
		DeliveryCity     string `json:"delivery_city"`
		DeliveryDistrict string `json:"delivery_district"`
	} `json:"address_info"`
	PayInfo struct {
		Payment string `json:"payment"`
	} `json:"pay_info"`
}

// GetYouzanAccessToken 获取有赞open平台access_token
func GetYouzanAccessToken() (string, error) {
	url := "https://open.youzanyun.com/auth/token"
	data := map[string]interface{}{
		"authorize_type": "silent",
		"client_id":      "379981eff640bbb278",
		"client_secret":  "1ef6d04d42b03784bd75fc1b74493c06",
		"grant_id":       "15707004",
		"refresh":        false,
	}

	dataBytes, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("序列化请求数据失败: %v", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(dataBytes))
	if err != nil {
		return "", fmt.Errorf("请求access_token失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("解析响应失败: %v", err)
	}

	if success, ok := result["success"].(bool); ok && success {
		if code, ok := result["code"].(float64); ok && code == 200 {
			if dataMap, ok := result["data"].(map[string]interface{}); ok {
				if accessToken, ok := dataMap["access_token"].(string); ok {
					return accessToken, nil
				}
			}
		}
	}

	message := "未知错误"
	if msg, ok := result["message"].(string); ok {
		message = msg
	}
	return "", fmt.Errorf("获取access_token失败: %s", message)
}

// GetYouzanOrders 获取有赞订单数据（分页获取，不按小时拆分），接受时间范围参数
func GetYouzanOrders(accessToken, startTime, endTime string) ([]YouzanFullOrderInfo, error) {
	url := "https://open.youzanyun.com/api/youzan.trades.sold.get/4.0.4"

	// 如果没有传入时间范围，则默认使用昨日时间范围
	if startTime == "" || endTime == "" {
		today := time.Now()
		yesterdayBegin := time.Date(today.Year(), today.Month(), today.Day()-1, 0, 0, 0, 0, today.Location())
		yesterdayEnd := time.Date(today.Year(), today.Month(), today.Day()-1, 23, 59, 59, 0, today.Location())
		startTime = yesterdayBegin.Format("2006-01-02 15:04:05")
		endTime = yesterdayEnd.Format("2006-01-02 15:04:05")
	}

	// 存储所有订单
	var allOrders []YouzanFullOrderInfo
	pageSize := 100
	const maxPageNo = 100 // API限制最大页码为100

	log.Printf("开始获取有赞订单，时间范围: %s 到 %s\n", startTime, endTime)

	// 定义需要获取的订单状态
	statuses := []string{"WAIT_BUYER_CONFIRM_GOODS", "TRADE_SUCCESS", "WAIT_BUYER_PAY", "WAIT_SELLER_SEND_GOODS"}

	// 分别获取不同状态的订单
	for _, status := range statuses {
		log.Printf("开始获取状态为 %s 的订单\n", status)
		pageNo := 1

		for pageNo <= maxPageNo {
			params := map[string]interface{}{
				"page_no":       strconv.Itoa(pageNo),
				"page_size":     strconv.Itoa(pageSize),
				"start_created": startTime,
				"end_created":   endTime,
				"status":        status, // 添加状态参数
			}

			// 构建请求URL
			requestURL := fmt.Sprintf("%s?access_token=%s", url, accessToken)

			paramsBytes, err := json.Marshal(params)
			if err != nil {
				return nil, fmt.Errorf("序列化请求参数失败: %v", err)
			}

			resp, err := http.Post(requestURL, "application/json", bytes.NewBuffer(paramsBytes))
			fmt.Printf("请求体: %s\n", string(paramsBytes))
			if err != nil {
				return nil, fmt.Errorf("请求订单数据失败: %v", err)
			}

			body, err := ioutil.ReadAll(resp.Body)
			resp.Body.Close() // 读取完成后关闭响应体
			fmt.Printf("响应Body: %s\n", string(body))
			if err != nil {
				return nil, fmt.Errorf("读取订单响应失败: %v", err)
			}

			var orderResponse YouzanOrderResponse
			if err := json.Unmarshal(body, &orderResponse); err != nil {
				return nil, fmt.Errorf("解析订单响应失败: %v", err)
			}

			if !orderResponse.Success {
				return nil, fmt.Errorf("API请求失败: %s", orderResponse.Message)
			}

			// 添加订单
			pageOrders := orderResponse.Data.FullOrderInfoList
			if len(pageOrders) == 0 {
				log.Printf("状态 %s 的第 %d 页没有返回订单数据\n", status, pageNo)
				break
			}

			for _, orderItem := range pageOrders {
				log.Printf("获取到订单: TID=%s, 状态=%s, 商店=%s\n",
					orderItem.FullOrderInfo.OrderInfo.Tid,
					orderItem.FullOrderInfo.OrderInfo.Status,
					orderItem.FullOrderInfo.OrderInfo.ShopName)
				allOrders = append(allOrders, orderItem.FullOrderInfo)
			}

			// 计算是否还有下一页数据
			totalResults := orderResponse.Data.TotalResults
			log.Printf("已获取状态 %s 的第 %d 页订单，当前累计获取 %d 条，该状态总记录数 %d\n",
				status, pageNo, len(allOrders), totalResults)

			// 如果当前页数据少于pageSize，或者已经获取了所有数据，则结束分页
			if len(pageOrders) < pageSize {
				log.Printf("已获取状态 %s 的所有订单\n", status)
				break
			}

			// 增加页码，准备获取下一页
			pageNo++
			log.Printf("准备获取状态 %s 的第 %d 页订单数据\n", status, pageNo)

			// 避免请求过于频繁
			time.Sleep(1 * time.Second)
		}

		// 不同状态之间也加入延迟
		time.Sleep(1 * time.Second)
	}

	log.Printf("订单获取完成，共获取 %d 条订单\n", len(allOrders))
	return allOrders, nil
}

// ParseDateTime 解析日期时间字符串为time.Time指针，显式使用东八区(Asia/Shanghai)解析，避免时区转换
func ParseDateTime(dateStr string) *time.Time {
	if dateStr == "" {
		return nil
	}

	// 加载东八区时区（北京时间）
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		// 若加载时区失败，默认使用UTC但打印警告（实际生产环境应确保时区文件存在）
		log.Printf("警告：无法加载Asia/Shanghai时区，使用UTC替代: %v", err)
		loc = time.UTC
	}

	// 尝试解析不同格式的时间字符串，显式指定东八区
	formats := []string{
		"2006-01-02 15:04:05",  // 有赞常见的时间格式（无时区，默认北京时间）
		"2006-01-02T15:04:05Z", // ISO格式（带Z表示UTC，会自动解析为UTC时间）
		"2006-01-02",           // 仅日期格式
	}

	for _, format := range formats {
		// 使用ParseInLocation显式指定时区解析，确保无时区信息的时间被当作东八区处理
		t, err := time.ParseInLocation(format, dateStr, loc)
		if err == nil {
			return &t
		}
	}

	log.Printf("无法解析时间字符串: %s", dateStr)
	return nil
}

// MapYouzanToSnowOrderData 将有赞订单数据映射到SnowOrderData模型
func MapYouzanToSnowOrderData(orders []YouzanFullOrderInfo) []models.SnowOrderData {
	var snowOrderDataList []models.SnowOrderData
	log.Printf("开始映射有赞订单数据，共 %d 条\n", len(orders))

	for _, order := range orders {
		// 解析支付金额
		paymentAmount := 0.0
		if order.PayInfo.Payment != "" {
			if amount, err := strconv.ParseFloat(order.PayInfo.Payment, 64); err == nil {
				paymentAmount = amount
			}
		}

		// 生成备注 - 根据店铺名称设置
		remark := "线下店"
		if order.OrderInfo.ShopName == "allblu幼岚" {
			remark = "小程序"
		}

		// 转换订单状态为中文描述
		orderStatusCN := order.OrderInfo.Status // 默认保留原状态
		switch order.OrderInfo.Status {
		case "WAIT_BUYER_CONFIRM_GOODS":
			orderStatusCN = "已发货"
		case "TRADE_SUCCESS":
			orderStatusCN = "交易完成"
		case "WAIT_BUYER_PAY":
			orderStatusCN = "已付款"
		case "WAIT_SELLER_SEND_GOODS":
			orderStatusCN = "待发货"
		}

		// 创建SnowOrderData实例，按照用户要求的字段映射关系
		snowOrderData := models.SnowOrderData{
			// Number为tid
			OnlineOrderNumber: order.OrderInfo.Tid,
			// 订单状态转换为中文
			OrderStatus: orderStatusCN,
			// 店铺为shop_name
			Store: order.OrderInfo.ShopName,
			// 订单日期为created
			OrderDate: ParseDateTime(order.OrderInfo.Created),
			// 发货日期consign_time
			ShipDate: ParseDateTime(order.OrderInfo.ConsignTime),
			// 付款日期为pay_time
			PaymentDate: ParseDateTime(order.OrderInfo.PayTime),
			// 卖家ID使用root_kdt_id
			SellerID: "15707004", // 有赞root_kdt_id
			// 确认收货时间为success_time
			ConfirmReceiptTime: ParseDateTime(order.OrderInfo.SuccessTime),
			// 收货人姓名receiver_name
			ConsigneeName: order.AddressInfo.ReceiverName,
			// 省delivery_province
			Province: order.AddressInfo.DeliveryProvince,
			// 市delivery_city
			City: order.AddressInfo.DeliveryCity,
			// 县delivery_district
			County: order.AddressInfo.DeliveryDistrict,
			// 原始线上订单号为tid
			OriginalOnlineOrderNumber: order.OrderInfo.Tid,
			// 实付金额payment
			ActualPaymentAmount: paymentAmount,
			// 备注为当shopname为allblu幼岚时写为小程序，其它的写为线下店
			Remark: remark,
		}

		log.Printf("映射订单: TID=%s, 店铺=%s, 状态=%s(原状态:%s), 备注=%s, 实付金额=%.2f\n",
			order.OrderInfo.Tid, order.OrderInfo.ShopName, orderStatusCN, order.OrderInfo.Status, remark, paymentAmount)
		snowOrderDataList = append(snowOrderDataList, snowOrderData)
	}

	log.Printf("订单数据映射完成，共生成 %d 条SnowOrderData记录\n", len(snowOrderDataList))
	return snowOrderDataList
}

// SaveYouzanOrdersToDB 保存有赞订单数据到数据库
func SaveYouzanOrdersToDB(orders []models.SnowOrderData) (int, error) {
	if len(orders) == 0 {
		return 0, nil
	}

	log.Printf("开始保存有赞订单数据到数据库，共 %d 条\n", len(orders))

	// 获取数据库连接
	database := db.DB
	if database == nil {
		log.Printf("数据库连接未初始化\n")
		return 0, fmt.Errorf("database connection is nil")
	}

	// 批量插入数据
	result := database.CreateInBatches(orders, 100)
	if result.Error != nil {
		return 0, fmt.Errorf("保存数据到数据库失败: %v", result.Error)
	}

	savedCount := int(result.RowsAffected)
	log.Printf("成功保存 %d 条记录到SnowOrderData模型\n", savedCount)
	return savedCount, nil
}

// MainYouzanOrder 主函数，协调整个流程，接受时间范围参数
func MainYouzanOrder(startTime, endTime string) {
	log.Println("开始执行有赞订单导入任务")

	// 初始化配置
	appConfig := config.LoadConfig()

	// 初始化数据库连接
	db.InitDB(appConfig)
	log.Println("数据库连接初始化完成")

	// 获取access_token
	accessToken, err := GetYouzanAccessToken()
	if err != nil {
		log.Printf("获取access_token失败: %v\n", err)
		return
	}
	log.Println("成功获取access_token")

	// 获取订单数据，传入时间范围参数
	orders, err := GetYouzanOrders(accessToken, startTime, endTime)
	if err != nil {
		log.Printf("获取订单数据失败: %v\n", err)
		return
	}
	log.Printf("成功获取 %d 条订单数据\n", len(orders))

	// 映射到SnowOrderData模型
	snowOrderDataList := MapYouzanToSnowOrderData(orders)

	// 保存到数据库
	savedCount, err := SaveYouzanOrdersToDB(snowOrderDataList)
	if err != nil {
		log.Printf("保存订单数据失败: %v\n", err)
		return
	}

	log.Printf("有赞订单导入任务完成，成功导入 %d 条记录\n", savedCount)
}
