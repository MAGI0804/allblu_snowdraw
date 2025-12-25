package taobao

import (
	"bytes"
	"crypto/md5"
	"django_to_go/config"
	"django_to_go/db"
	"django_to_go/models"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// TaobaoTradeResponse 淘宝交易API响应结构
type TaobaoTradeResponse struct {
	TradesSoldGetResponse struct {
		TotalResults int  `json:"total_results"`
		HasNext      bool `json:"has_next"`
		Trades       struct {
			Trade []TaobaoTrade `json:"trade"`
		} `json:"trades"`
	} `json:"trades_sold_get_response"`
}

// TaobaoTrade 淘宝交易信息
type TaobaoTrade struct {
	Tid         string `json:"tid"`
	Status      string `json:"status"`
	Created     string `json:"created"`
	PayTime     string `json:"pay_time"`
	ConsignTime string `json:"consign_time"`
	EndTime     string `json:"end_time"`
	Payment     string `json:"payment"`
	Oaid        string `json:"oaid"`
	NoShipping  bool   `json:"no_shipping"`
	SignTime    string `json:"sign_time"`
}

// generateMD5Signature 生成淘宝API签名（与Python版本完全一致）
func generateMD5Signature(params map[string]string) string {
	// 使用与Python版本相同的app_secret
	appSecret := "a9f5cd5174b0007500ecd99b3bbe6daf"

	// 1. 排序参数（按照Python版本的逻辑，按键名升序排序）
	var keys []string
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 2. 拼接参数 - 先拼接所有键值对，然后用app_secret前后包裹
	var paramStr strings.Builder
	// 先拼接所有键值对
	for _, k := range keys {
		paramStr.WriteString(k + params[k])
	}
	// 用app_secret前后包裹
	finalString := appSecret + paramStr.String() + appSecret

	// 3. 计算MD5并转为大写
	md5Hash := md5.Sum([]byte(finalString))
	return strings.ToUpper(fmt.Sprintf("%x", md5Hash))
}

// ParseDateTime 解析淘宝API返回的日期时间字符串
func ParseDateTime(dateStr string) *time.Time {
	if dateStr == "" {
		return nil
	}

	// 淘宝API返回的时间格式处理（移除可能存在的额外空格）
	dateStr = strings.ReplaceAll(dateStr, " ", "")

	// 创建+8h固定时区（Asia/Shanghai）
	loc, _ := time.LoadLocation("Asia/Shanghai")

	formats := []string{
		"2006-01-0215:04:05", // 淘宝API常见格式
		"2006-01-02T15:04:05Z",
		"2006-01-02",
	}

	for _, format := range formats {
		if t, err := time.ParseInLocation(format, dateStr, loc); err == nil {
			// 确保返回的时间使用Asia/Shanghai时区
			return &t
		}
	}

	// 尝试不指定时区解析，如果成功则转换到Asia/Shanghai时区
	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			// 将UTC或本地时间转换到Asia/Shanghai时区
			shanghaiTime := t.In(loc)
			return &shanghaiTime
		}
	}

	log.Printf("无法解析时间字符串: %s", dateStr)
	return nil
}

// MapTaobaoToSnowOrderData 将淘宝订单数据映射到SnowOrderData模型
func MapTaobaoToSnowOrderData(taobaoTrades []TaobaoTrade) []models.SnowOrderData {
	var snowOrders []models.SnowOrderData

	for _, trade := range taobaoTrades {
		// 转换订单状态
		var orderStatus string
		switch trade.Status {
		case "TRADE_FINISHED":
			orderStatus = "交易完成"
		case "WAIT_BUYER_CONFIRM_GOODS":
			orderStatus = "已发货"
		case "ELLER_CONSIGNED_PART":
			orderStatus = "已付款部分发货"
		case "WAIT_SELLER_SEND_GOODS":
			orderStatus = "已付款待发货"
		default:
			orderStatus = "未知状态"
		}

		// 解析实付金额
		paymentAmount := 0.0
		if payment, err := strconv.ParseFloat(trade.Payment, 64); err == nil {
			paymentAmount = payment
		}

		// 创建SnowOrderData模型实例
		snowOrder := models.SnowOrderData{
			SerialNumber:              0, // 将在保存时自动生成或设置
			OnlineOrderNumber:         trade.Tid,
			OrderStatus:               orderStatus,
			Store:                     "天猫淘宝旗舰店",
			OrderDate:                 ParseDateTime(trade.Created),
			ShipDate:                  ParseDateTime(trade.ConsignTime),
			PaymentDate:               ParseDateTime(trade.PayTime),
			SellerID:                  "", // API响应中没有buyer_open_uid，暂时留空
			ConfirmReceiptTime:        ParseDateTime(trade.EndTime),
			ConsigneeName:             trade.Oaid,
			Province:                  "未知省", // 设置默认值以满足非空约束
			City:                      "未知市", // 设置默认值以满足非空约束
			County:                    "未知县", // 设置默认值以满足非空约束
			OriginalOnlineOrderNumber: trade.Tid,
			ActualPaymentAmount:       paymentAmount,
		}

		log.Printf("映射订单: TID=%s, 店铺=%s, 订单状态=%s, 实付金额=%.2f\n",
			trade.Tid, snowOrder.Store, snowOrder.OrderStatus, snowOrder.ActualPaymentAmount)

		snowOrders = append(snowOrders, snowOrder)
	}

	return snowOrders
}

// GetTaobaoOrders 获取并处理淘宝订单数据，接受时间范围参数
func GetTaobaoOrders(startTime, endTime string, processPage func([]TaobaoTrade) (int, error)) (int, error) {
	baseURL := "http://gw.api.taobao.com/router/rest"

	// 设置日期范围
	startCreated := startTime
	endCreated := endTime

	// 如果没有传入时间范围，则默认使用昨日时间范围
	if startCreated == "" || endCreated == "" {
		today := time.Now()
		yesterdayBegin := time.Date(today.Year(), today.Month(), today.Day()-1, 0, 0, 0, 0, today.Location())
		yesterdayEnd := time.Date(today.Year(), today.Month(), today.Day()-1, 23, 59, 59, 0, today.Location())
		startCreated = yesterdayBegin.Format("2006-01-02 15:04:05")
		endCreated = yesterdayEnd.Format("2006-01-02 15:04:05")
	}

	// 需要查询的字段
	fields := []string{"tid", "created", "status", "payment", "end_time", "pay_time", "consign_time", "sign_time", "invoice_no", "oaid"}

	// 查询四种状态的订单
	statuses := []string{"TRADE_FINISHED", "WAIT_BUYER_CONFIRM_GOODS", "SELLER_CONSIGNED_PART", "WAIT_SELLER_SEND_GOODS"}

	// 速率限制设置：每秒最多发送5个请求
	maxRequestsPerSecond := 8
	requestInterval := time.Second / time.Duration(maxRequestsPerSecond)
	lastRequestTime := time.Now().Add(-requestInterval) // 初始化以允许立即发送第一个请求

	totalProcessed := 0

	for _, status := range statuses {
		pageNo := 1
		for {
			// 检查速率限制
			elapsed := time.Since(lastRequestTime)
			if elapsed < requestInterval {
				time.Sleep(requestInterval - elapsed)
			}
			lastRequestTime = time.Now()

			// 准备参数（与Python版本一致）
			params := map[string]string{
				"method":        "taobao.trades.sold.get",
				"app_key":       "34101613",
				"session":       "6101607ec53f58e15fdbc10cc7baa1d2c3c4e292fce19183948263976",
				"timestamp":     time.Now().Format("2006-01-02 15:04:05"),
				"v":             "2.0",
				"sign_method":   "md5",
				"format":        "json",
				"start_created": startCreated,
				"end_created":   endCreated,
				"page_no":       strconv.Itoa(pageNo),
				"page_size":     "100",
				"use_has_next":  "true",
				"status":        status,
				"fields":        strings.Join(fields, ","),
				"real_payment":  "淘宝",
			}

			// 生成签名
			params["sign"] = generateMD5Signature(params)

			// 构建请求URL
			queryString := url.Values{}
			for k, v := range params {
				queryString.Add(k, v)
			}
			fullURL := baseURL + "?" + queryString.Encode()

			// 创建HTTP客户端
			client := &http.Client{
				Timeout: 10 * time.Second,
			}

			// 发送请求
			resp, err := client.Get(fullURL)
			if err != nil {
				return totalProcessed, fmt.Errorf("请求第 %d 页时出错: %v", pageNo, err)
			}

			// 打印响应状态码
			log.Printf("第 %d 页请求响应状态码: %d", pageNo, resp.StatusCode)

			// 读取响应内容
			responseBody, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Printf("读取响应内容失败: %v", err)
			} else {
				// 重新创建读取器以用于后续解析
				resp.Body = io.NopCloser(bytes.NewBuffer(responseBody))
			}

			defer resp.Body.Close()

			// 解析响应
			var tradeResponse TaobaoTradeResponse
			if err := json.NewDecoder(resp.Body).Decode(&tradeResponse); err != nil {
				return totalProcessed, fmt.Errorf("解析响应失败: %v", err)
			}

			// 处理当前页数据
			trades := tradeResponse.TradesSoldGetResponse.Trades.Trade
			log.Printf("已获取状态 %s 的第 %d 页订单，共 %d 条", status, pageNo, len(trades))

			// 立即处理并写入当前页数据
			if len(trades) > 0 {
				processed, err := processPage(trades)
				if err != nil {
					return totalProcessed, fmt.Errorf("处理第 %d 页数据失败: %v", pageNo, err)
				}
				totalProcessed += processed
			}

			// 检查是否有下一页
			if !tradeResponse.TradesSoldGetResponse.HasNext {
				break
			}

			pageNo++
		}
	}

	log.Printf("总共处理了 %d 条淘宝订单数据\n", totalProcessed)
	return totalProcessed, nil
}

// SaveSinglePageOrders 处理单页订单数据并保存到数据库
func SaveSinglePageOrders(taobaoTrades []TaobaoTrade) (int, error) {
	if len(taobaoTrades) == 0 {
		return 0, nil
	}

	// 映射到SnowOrderData模型
	snowOrders := MapTaobaoToSnowOrderData(taobaoTrades)
	log.Printf("正在处理 %d 条SnowOrderData模型格式数据\n", len(snowOrders))

	// 为每个订单设置序号（使用时间戳+索引）
	baseSerialNum := int(time.Now().Unix())
	for i := range snowOrders {
		snowOrders[i].SerialNumber = baseSerialNum + i
	}

	// 使用较小的批处理大小
	batchSize := 100
	totalInserted := 0

	for i := 0; i < len(snowOrders); i += batchSize {
		end := i + batchSize
		if end > len(snowOrders) {
			end = len(snowOrders)
		}

		batchData := snowOrders[i:end]
		result := db.DB.Create(&batchData)
		if result.Error != nil {
			return totalInserted, fmt.Errorf("保存数据到数据库失败: %v", result.Error)
		}

		totalInserted += int(result.RowsAffected)
		log.Printf("已保存第 %d-%d 条记录，当前累计保存 %d 条", i+1, end, totalInserted)

		// 在批处理之间添加延迟
		if i+batchSize < len(snowOrders) {
			time.Sleep(30 * time.Millisecond) // 减少延迟时间以加快处理
		}
	}

	return totalInserted, nil
}

// MainTaobaoOrder 主函数，处理淘宝订单数据，接受时间范围参数
func MainTaobaoOrder(startTime, endTime string) error {
	// 初始化配置
	appConfig := config.LoadConfig()
	log.Println("正在初始化数据库连接...")

	// 初始化数据库连接
	db.InitDB(appConfig)
	log.Println("数据库连接初始化完成")

	// 获取并处理淘宝订单数据 - 使用回调函数处理每页数据
	log.Printf("开始获取淘宝订单数据，时间范围：%s 到 %s...\n", startTime, endTime)
	savedCount, err := GetTaobaoOrders(startTime, endTime, func(trades []TaobaoTrade) (int, error) {
		return SaveSinglePageOrders(trades)
	})

	if err != nil {
		return fmt.Errorf("获取和处理淘宝订单数据失败: %v", err)
	}

	log.Printf("淘宝订单数据处理完成，共保存 %d 条记录\n", savedCount)
	return nil
}
