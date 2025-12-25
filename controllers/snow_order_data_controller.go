package controllers

import (
	"django_to_go/db"
	"django_to_go/models"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

// SnowOrderDataController 订单数据控制器
type SnowOrderDataController struct{}

// ImportOrderData 导入订单数据
func (s *SnowOrderDataController) ImportOrderData(c *gin.Context) {
	// 获取上传的文件
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请上传Excel文件",
		})
		return
	}

	// 检查文件类型
	ext := filepath.Ext(file.Filename)
	if ext != ".xlsx" && ext != ".xls" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请上传Excel格式文件",
		})
		return
	}

	// 打开上传的文件
	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "文件打开失败",
		})
		return
	}
	defer src.Close()

	// 读取Excel文件
	xlsx, err := excelize.OpenReader(src)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Excel文件读取失败",
		})
		return
	}

	// 获取第一个工作表
	sheetName := xlsx.GetSheetName(0)
	rows, err := xlsx.GetRows(sheetName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Excel数据读取失败",
		})
		return
	}

	if len(rows) <= 1 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Excel文件内容为空或只有标题行",
		})
		return
	}

	// 解析表头，建立字段映射
	headerMap := make(map[string]int)
	for i, header := range rows[0] {
		headerMap[header] = i
	}

	// 准备导入数据
	successCount := 0
	failCount := 0
	var firstErrorMessage string
	var failedOrders []map[string]interface{} // 记录失败的订单信息和原因

	// 移除事务处理，直接使用DB保存数据
	log.Println("开始处理数据导入...")

	// 遍历数据行
	for rowIndex, row := range rows {
		if rowIndex == 0 {
			continue // 跳过标题行
		}

		orderData := models.SnowOrderData{}

		// 序号 - 设置默认值以满足not null约束
		if idx, ok := headerMap["序号"]; ok && idx < len(row) && row[idx] != "" {
			if serialNum, err := strconv.Atoi(row[idx]); err == nil {
				orderData.SerialNumber = serialNum
			} else {
				orderData.SerialNumber = rowIndex // 使用行号作为默认序号
				log.Printf("第%d行：序号格式不正确，使用默认值: %d", rowIndex+1, orderData.SerialNumber)
			}
		} else {
			orderData.SerialNumber = rowIndex // 使用行号作为默认序号
			log.Printf("第%d行：未找到序号字段或为空，使用默认值: %d", rowIndex+1, orderData.SerialNumber)
		}

		// 线上订单号 - 设置默认值以满足not null约束
		if idx, ok := headerMap["online_order_number"]; ok && idx < len(row) && row[idx] != "" {
			orderData.OnlineOrderNumber = strings.TrimSpace(row[idx])
		} else {
			orderData.OnlineOrderNumber = fmt.Sprintf("ORDER_%d_%d", rowIndex, time.Now().UnixNano())
			log.Printf("第%d行：线上订单号为空，生成默认值: %s", rowIndex+1, orderData.OnlineOrderNumber)
		}

		// 店铺 - 确保设置默认值以满足not null约束
		if idx, ok := headerMap["store"]; ok && idx < len(row) && row[idx] != "" {
			orderData.Store = row[idx]
		} else {
			orderData.Store = "未知店铺"
			log.Printf("第%d行：店铺字段为空或未找到，设置默认值: %s", rowIndex+1, orderData.Store)
		}

		// 订单状态 - 设置默认值以满足not null约束
		if idx, ok := headerMap["order_status"]; ok && idx < len(row) && row[idx] != "" {
			orderData.OrderStatus = row[idx]
		} else {
			orderData.OrderStatus = "未知状态"
			log.Printf("第%d行：订单状态字段为空或未找到，设置默认值: %s", rowIndex+1, orderData.OrderStatus)
		}

		// 订单日期 - 改进日期处理，支持多种日期格式
		if idx, ok := headerMap["order_date"]; ok && idx < len(row) && row[idx] != "" && row[idx] != "0000-00-00" && row[idx] != "0000-00-00 00:00:00" {
			log.Printf("第%d行：尝试解析订单日期: '%s'", rowIndex+1, row[idx])
			parsed := false
			dateFormats := []string{
				"2006-01-02 15:04:05",
				"2006-01-02",
				"2006/01/02 15:04:05",
				"2006/01/02",
				"2006/1/2 15:04:05",
				"2006/1/2",
				"2006-1-2 15:04:05",
				"2006-1-2",
			}
			
			for _, format := range dateFormats {
				if date, err := time.Parse(format, row[idx]); err == nil {
					orderData.OrderDate = &date
					parsed = true
					log.Printf("第%d行：订单日期解析成功，使用格式: '%s'", rowIndex+1, format)
					break
				}
			}
			
			if !parsed {
				log.Printf("第%d行：订单日期格式不正确: %s", rowIndex+1, row[idx])
				altDateStr := strings.ReplaceAll(row[idx], "/", "-")
				log.Printf("第%d行：尝试替换分隔符后解析: '%s'", rowIndex+1, altDateStr)
				if date, err := time.Parse("2006-01-02 15:04:05", altDateStr); err == nil {
					orderData.OrderDate = &date
				} else if date, err := time.Parse("2006-01-02", altDateStr); err == nil {
					orderData.OrderDate = &date
				}
			}
		}

		// 发货日期 - 支持多种日期格式
		if idx, ok := headerMap["ship_date"]; ok && idx < len(row) && row[idx] != "" && row[idx] != "0000-00-00" && row[idx] != "0000-00-00 00:00:00" {
			log.Printf("第%d行：尝试解析发货日期: '%s'", rowIndex+1, row[idx])
			parsed := false
			dateFormats := []string{
				"2006-01-02 15:04:05",
				"2006-01-02",
				"2006/01/02 15:04:05",
				"2006/01/02",
				"2006/1/2 15:04:05",
				"2006/1/2",
				"2006-1-2 15:04:05",
				"2006-1-2",
			}
			
			for _, format := range dateFormats {
				if date, err := time.Parse(format, row[idx]); err == nil {
					orderData.ShipDate = &date
					parsed = true
					break
				}
			}
			
			if !parsed {
				altDateStr := strings.ReplaceAll(row[idx], "/", "-")
				if date, err := time.Parse("2006-01-02 15:04:05", altDateStr); err == nil {
					orderData.ShipDate = &date
				} else if date, err := time.Parse("2006-01-02", altDateStr); err == nil {
					orderData.ShipDate = &date
				}
			}
		}

		// 付款日期 - 支持多种日期格式
		if idx, ok := headerMap["payment_date"]; ok && idx < len(row) && row[idx] != "" && row[idx] != "0000-00-00" && row[idx] != "0000-00-00 00:00:00" {
			log.Printf("第%d行：尝试解析付款日期: '%s'", rowIndex+1, row[idx])
			parsed := false
			dateFormats := []string{
				"2006-01-02 15:04:05",
				"2006-01-02",
				"2006/01/02 15:04:05",
				"2006/01/02",
				"2006/1/2 15:04:05",
				"2006/1/2",
				"2006-1-2 15:04:05",
				"2006-1-2",
			}
			
			for _, format := range dateFormats {
				if date, err := time.Parse(format, row[idx]); err == nil {
					orderData.PaymentDate = &date
					parsed = true
					break
				}
			}
			
			if !parsed {
				altDateStr := strings.ReplaceAll(row[idx], "/", "-")
				if date, err := time.Parse("2006-01-02 15:04:05", altDateStr); err == nil {
					orderData.PaymentDate = &date
				} else if date, err := time.Parse("2006-01-02", altDateStr); err == nil {
					orderData.PaymentDate = &date
				}
			}
		}

		// 卖家id - 设置默认值以满足not null约束
		if idx, ok := headerMap["seller_id"]; ok && idx < len(row) && row[idx] != "" {
			orderData.SellerID = row[idx]
		} else {
			orderData.SellerID = "UNKNOWN_SELLER"
			log.Printf("第%d行：卖家ID字段为空或未找到，设置默认值: %s", rowIndex+1, orderData.SellerID)
		}

		// 确认收货时间 - 支持多种日期格式
		if idx, ok := headerMap["confirm_receipt_time"]; ok && idx < len(row) && row[idx] != "" && row[idx] != "0000-00-00" && row[idx] != "0000-00-00 00:00:00" {
			log.Printf("第%d行：尝试解析确认收货时间: '%s'", rowIndex+1, row[idx])
			parsed := false
			dateFormats := []string{
				"2006-01-02 15:04:05",
				"2006-01-02",
				"2006/01/02 15:04:05",
				"2006/01/02",
				"2006/1/2 15:04:05",
				"2006/1/2",
				"2006-1-2 15:04:05",
				"2006-1-2",
			}
			
			for _, format := range dateFormats {
				if date, err := time.Parse(format, row[idx]); err == nil {
					orderData.ConfirmReceiptTime = &date
					parsed = true
					break
				}
			}
			
			if !parsed {
				altDateStr := strings.ReplaceAll(row[idx], "/", "-")
				if date, err := time.Parse("2006-01-02 15:04:05", altDateStr); err == nil {
					orderData.ConfirmReceiptTime = &date
				} else if date, err := time.Parse("2006-01-02", altDateStr); err == nil {
					orderData.ConfirmReceiptTime = &date
				}
			}
		}

		// 收货人姓名 - 设置默认值以满足not null约束
		if idx, ok := headerMap["consignee_name"]; ok && idx < len(row) && row[idx] != "" {
			orderData.ConsigneeName = row[idx]
		} else {
			orderData.ConsigneeName = "未知收货人"
			log.Printf("第%d行：收货人姓名字段为空或未找到，设置默认值: %s", rowIndex+1, orderData.ConsigneeName)
		}

		// 省 - 设置默认值以满足not null约束
		if idx, ok := headerMap["province"]; ok && idx < len(row) && row[idx] != "" {
			orderData.Province = row[idx]
		} else {
			orderData.Province = "未知省"
			log.Printf("第%d行：省份字段为空或未找到，设置默认值: %s", rowIndex+1, orderData.Province)
		}

		// 市 - 设置默认值以满足not null约束
		if idx, ok := headerMap["city"]; ok && idx < len(row) && row[idx] != "" {
			orderData.City = row[idx]
		} else {
			orderData.City = "未知市"
			log.Printf("第%d行：城市字段为空或未找到，设置默认值: %s", rowIndex+1, orderData.City)
		}

		// 县 - 设置默认值以满足not null约束
		if idx, ok := headerMap["county"]; ok && idx < len(row) && row[idx] != "" {
			orderData.County = row[idx]
		} else {
			orderData.County = "未知县"
			log.Printf("第%d行：区县字段为空或未找到，设置默认值: %s", rowIndex+1, orderData.County)
		}

		// 快递单号
		if idx, ok := headerMap["tracking_number"]; ok && idx < len(row) {
			orderData.TrackingNumber = row[idx]
		}

		// 原始线上订单号
		if idx, ok := headerMap["original_online_order_number"]; ok && idx < len(row) {
			orderData.OriginalOnlineOrderNumber = strings.TrimSpace(row[idx])
		}

		// 实付金额 - 设置默认值以满足not null约束
		if idx, ok := headerMap["actual_payment_amount"]; ok && idx < len(row) && row[idx] != "" {
			if amount, err := strconv.ParseFloat(row[idx], 64); err == nil {
				orderData.ActualPaymentAmount = amount
			} else {
				orderData.ActualPaymentAmount = 0
				log.Printf("第%d行：实付金额格式不正确，使用默认值0: %s", rowIndex+1, row[idx])
			}
		} else {
			orderData.ActualPaymentAmount = 0
			log.Printf("第%d行：实付金额字段为空或未找到，使用默认值0", rowIndex+1)
		}

		// 退货数量
		if idx, ok := headerMap["return_quantity"]; ok && idx < len(row) && row[idx] != "" {
			if quantity, err := strconv.Atoi(row[idx]); err == nil {
				orderData.ReturnQuantity = quantity
			} else {
				log.Printf("第%d行：退货数量格式不正确: %s", rowIndex+1, row[idx])
			}
		}

		// 退货金额
		if idx, ok := headerMap["return_amount"]; ok && idx < len(row) && row[idx] != "" {
			if amount, err := strconv.ParseFloat(row[idx], 64); err == nil {
				orderData.ReturnAmount = amount
			} else {
				log.Printf("第%d行：退货金额格式不正确: %s", rowIndex+1, row[idx])
			}
		}

		// 线上子订单编号
		if idx, ok := headerMap["online_sub_order_number"]; ok && idx < len(row) {
			orderData.OnlineSubOrderNumber = strings.TrimSpace(row[idx])
		}

		// 备注
		if idx, ok := headerMap["remark"]; ok && idx < len(row) {
			orderData.Remark = row[idx]
		}

		// 保存到数据库
		log.Printf("准备保存第%d行数据，订单号：%s，数据完整信息：%+v", rowIndex+1, orderData.OnlineOrderNumber, orderData)
		
		// 移除事务，直接使用DB保存，以便更好地捕获错误
		if err := db.DB.Create(&orderData).Error; err != nil {
			failCount++
			errorMsg := fmt.Sprintf("第%d行导入失败: %v, 数据: %+v", rowIndex+1, err, orderData)
			log.Printf(errorMsg)
			if firstErrorMessage == "" {
				firstErrorMessage = errorMsg
			}
			// 记录失败的订单信息和原因
			failedOrders = append(failedOrders, map[string]interface{}{
				"rowNumber":     rowIndex + 1,
				"serialNumber":  orderData.SerialNumber,
				"onlineOrderNumber": orderData.OnlineOrderNumber,
				"errorReason":   err.Error(),
			})
		} else {
			log.Printf("第%d行数据保存成功，数据库返回ID：%d，订单号：%s", rowIndex+1, orderData.ID, orderData.OnlineOrderNumber)
			successCount++
			
			// 立即验证数据是否真的保存成功
			var verifyData models.SnowOrderData
			if err := db.DB.First(&verifyData, orderData.ID).Error; err != nil {
				log.Printf("警告：第%d行数据保存后验证失败，无法查询到ID：%d，错误：%v", rowIndex+1, orderData.ID, err)
			} else {
				log.Printf("验证成功：第%d行数据ID：%d 确实存在于数据库中，订单号：%s", rowIndex+1, orderData.ID, verifyData.OnlineOrderNumber)
			}
		}
	}

	// 所有数据处理完成
	log.Printf("数据处理完成，成功：%d，失败：%d\n", successCount, failCount)
	
	// 记录失败订单详情
	if len(failedOrders) > 0 {
		log.Printf("失败订单详情 (共%d条):", len(failedOrders))
		for _, order := range failedOrders {
			log.Printf("行号:%v, 序号:%v, 订单号:%v, 失败原因:%v", 
				order["rowNumber"], order["serialNumber"], 
				order["onlineOrderNumber"], order["errorReason"])
		}
	}
	
	// 准备响应消息
	message := "数据导入完成"
	if failCount > 0 {
		message = "数据导入完成，部分记录失败，请检查数据格式"
		if firstErrorMessage != "" {
			message = fmt.Sprintf("%s。第一个错误: %s", message, firstErrorMessage)
		}
	} else {
		message = "订单数据导入成功"
	}
	
	// 返回响应
	c.JSON(http.StatusOK, gin.H{
		"success": successCount > 0,
		"message": message,
		"data": gin.H{
			"total":       successCount + failCount,
			"success":     successCount,
			"fail":        failCount,
			"failedOrders": failedOrders,
		},
	})
	return
}