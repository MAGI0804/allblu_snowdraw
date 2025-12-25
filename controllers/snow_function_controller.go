package controllers

import (
	"crypto/rand"
	"django_to_go/db"
	vip "django_to_go/method/vip"
	"django_to_go/models"
	"encoding/json"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"log"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
	"golang.org/x/image/webp"
)

// SnowFunctionController 抽奖功能控制器
type SnowFunctionController struct{}

// NewSnowFunctionController 创建抽奖功能控制器实例
func NewSnowFunctionController() *SnowFunctionController {
	return &SnowFunctionController{}
}

// QueryUserByCodeRequest 根据抽奖码或手机号和轮次查询用户信息请求结构体
type QueryUserByCodeRequest struct {
	DrawBatch   int    `json:"draw_batch" binding:"required"`   // 抽奖轮次
	SearchValue string `json:"search_value" binding:"required"` // 查询值（抽奖码或手机号）
	QueryType   string `json:"query_type" binding:"required"`   // 查询方式："code"表示抽奖码，"mobile"表示手机号
}

// ValidateOrderRequest 校验订单号请求结构体
type ValidateOrderRequest struct {
	OrderNum  string `json:"order_num" binding:"required"`  // 订单号
	DrawBatch int    `json:"draw_batch" binding:"required"` // 抽奖轮次
}

// QueryParticipationCountRequest 查询参与人数请求结构体
type QueryParticipationCountRequest struct {
	DrawBatch int `json:"draw_batch" binding:"required"` // 抽奖轮次
}

// UpgradeUserVipRequest 升级用户会员请求结构体
type UpgradeUserVipRequest struct {
	Mobile string `json:"mobile" binding:"required"` // 用户手机号
}

// QueryUserVipInfoRequest 查询用户会员信息请求结构体
type QueryUserVipInfoRequest struct {
	Mobile string `json:"mobile" binding:"required"` // 用户手机号
}

// LotteryDrawRequest 抽奖请求结构体
type LotteryDrawRequest struct {
	DrawBatch int `json:"draw_batch" binding:"required"` // 抽奖轮次
}

// ExportWinnersRequest 导出中奖名单请求结构体
type ExportWinnersRequest struct {
	DrawBatch int `json:"draw_batch" binding:"required"` // 抽奖轮次
}

// ImportWinnersRequest 导入中奖名单请求结构体
type ImportWinnersRequest struct {
	DrawBatch int `json:"draw_batch" binding:"required"` // 抽奖轮次
}

// DrawSingleWinnerRequest 抽取单个中奖者请求结构体
type DrawSingleWinnerRequest struct {
	DrawBatch int `json:"draw_batch" binding:"required"` // 抽奖轮次
}

// QueryEligibleUsersRequest 查询符合抽奖条件用户请求结构体
type QueryEligibleUsersRequest struct {
	DrawBatch int `json:"draw_batch" binding:"required"` // 抽奖轮次
}

// DrawSingleWinner 抽取单个中奖者
func (sfc *SnowFunctionController) DrawSingleWinner(c *gin.Context) {
	log.Printf("===================== 开始执行单次抽奖操作 =====================")
	var request DrawSingleWinnerRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		log.Printf("请求参数绑定失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}
	log.Printf("单次抽奖请求参数绑定成功，抽奖轮次: %d", request.DrawBatch)

	// 1. 获取抽奖信息
	var lotteryDraw models.SnowLotteryDraw
	if err := db.DB.Where("draw_batch = ?", request.DrawBatch).First(&lotteryDraw).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "抽奖轮次不存在"})
		return
	}

	// 2. 解析当前中奖名单
	var currentWinners []map[string]interface{}
	if lotteryDraw.WinnersList != "" && lotteryDraw.WinnersList != "[]" {
		// 尝试修复WinnersList格式，使其成为有效的JSON数组
		formattedWinnersList := "[" + lotteryDraw.WinnersList + "]"
		if err := json.Unmarshal([]byte(formattedWinnersList), &currentWinners); err != nil {
			// 如果解析失败，尝试直接解析原字符串
			if err := json.Unmarshal([]byte(lotteryDraw.WinnersList), &currentWinners); err != nil {
				log.Printf("解析当前中奖名单失败: %v", err)
				currentWinners = []map[string]interface{}{}
			}
		}
	}

	// 3. 检查是否还能继续抽奖
	if len(currentWinners) >= lotteryDraw.TotalDrawers {
		log.Printf("中奖人数已达上限: %d/%d", len(currentWinners), lotteryDraw.TotalDrawers)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":                 "中奖人数已达上限，无法继续抽奖",
			"current_winners_count": len(currentWinners),
			"total_drawers":         lotteryDraw.TotalDrawers,
		})
		return
	}

	// 4. 获取参与者列表
	var eligibleUsers []models.SnowSuccessUser
	if lotteryDraw.ParticipantsList != "" && lotteryDraw.ParticipantsList != "[]" {
		var userIDs []string
		if err := json.Unmarshal([]byte(lotteryDraw.ParticipantsList), &userIDs); err == nil {
			// 转换为整数数组
			var userIDInts []int
			for _, idStr := range userIDs {
				if id, err := strconv.Atoi(idStr); err == nil {
					userIDInts = append(userIDInts, id)
				}
			}

			// 查询符合条件的用户（排除已中奖的用户）
			for _, userID := range userIDInts {
				var snowUser models.SnowUser
				if err := db.DB.Where("user_id = ?", userID).First(&snowUser).Error; err != nil {
					continue
				}

				var successUser models.SnowSuccessUser
				if err := db.DB.Where("mobile = ?", snowUser.Mobile).First(&successUser).Error; err != nil {
					continue
				}

				// 检查是否已中奖
				isAlreadyWinner := false
				for _, winner := range currentWinners {
					if mobile, ok := winner["mobile"]; ok {
						var winnerMobile int
						switch v := mobile.(type) {
						case string:
							fmt.Sscanf(v, "%d", &winnerMobile)
						case float64:
							winnerMobile = int(v)
						case int:
							winnerMobile = v
						default:
							fmt.Sscanf(fmt.Sprintf("%v", v), "%d", &winnerMobile)
						}
						if winnerMobile == successUser.Mobile {
							isAlreadyWinner = true
							break
						}
					}
				}

				if !isAlreadyWinner {
					eligibleUsers = append(eligibleUsers, successUser)
				}
			}
		}
	}

	// 5. 随机选择一个用户
	if len(eligibleUsers) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "没有可抽奖的用户"})
		return
	}

	// 使用随机数选择用户
	randomIndex, err := rand.Int(rand.Reader, big.NewInt(int64(len(eligibleUsers))))
	if err != nil {
		log.Printf("生成随机数失败: %v", err)
		// 备用随机方法
		index := time.Now().UnixNano() % int64(len(eligibleUsers))
		if index < 0 {
			index = -index
		}
		randomIndex = big.NewInt(index)
	}
	selectedIndex := int(randomIndex.Int64())
	selectedUser := eligibleUsers[selectedIndex]

	// 6. 构建中奖者信息
	drawBatchStr := strconv.Itoa(request.DrawBatch)

	// 获取订单号
	orderNums, err := selectedUser.GetOrderNUM()
	var orderNum string
	if err == nil {
		if num, exists := orderNums[drawBatchStr]; exists {
			orderNum = num
		}
	}

	// 获取抽奖码
	successCodes, err := selectedUser.GetSuccessCode()
	var drawCode string
	if err == nil {
		if code, exists := successCodes[drawBatchStr]; exists && code != "" {
			drawCode = code
		}
	}

	// 如果没有找到抽奖码，尝试从SnowUser表获取
	if drawCode == "" {
		var snowUser models.SnowUser
		if err := db.DB.Where("mobile = ?", selectedUser.Mobile).First(&snowUser).Error; err == nil {
			if snowUser.SuccessCodeMap != nil {
				if code, exists := snowUser.SuccessCodeMap[drawBatchStr]; exists && code != "" {
					drawCode = code
				}
			}
		}
	}

	// 解析奖品信息，只获取奖品名称
	var prizeName string
	if lotteryDraw.Prizes != "" && lotteryDraw.Prizes != "{}" {
		var prizesMap map[string]interface{}
		if err := json.Unmarshal([]byte(lotteryDraw.Prizes), &prizesMap); err == nil {
			// 获取第一个奖品名称
			for name := range prizesMap {
				prizeName = name
				break // 只取第一个
			}
		}
	}

	// 构建中奖者信息
	winnerInfo := gin.H{
		"draw_code":  drawCode,
		"mobile":     selectedUser.Mobile,
		"nickname":   selectedUser.Nickname,
		"order_num":  orderNum,
		"prize_name": prizeName,
	}

	// 7. 将中奖者添加到WinnersList
	currentWinners = append(currentWinners, winnerInfo)

	// 序列化为JSON
	winnersJSON, err := json.Marshal(currentWinners)
	if err != nil {
		log.Printf("序列化中奖者信息失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "添加中奖者失败"})
		return
	}

	// 8. 更新数据库
	lotteryDraw.WinnersList = string(winnersJSON)
	if err := db.DB.Save(&lotteryDraw).Error; err != nil {
		log.Printf("更新中奖名单失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存中奖者失败"})
		return
	}

	// 9. 更新用户的中奖状态
	winningStatus, err := selectedUser.GetWinningStatus()
	if err != nil {
		winningStatus = make(map[string]bool)
	}
	winningStatus[drawBatchStr] = true
	if err := selectedUser.SetWinningStatus(winningStatus); err != nil {
		log.Printf("设置用户中奖状态失败: %v", err)
	}

	if err := db.DB.Save(&selectedUser).Error; err != nil {
		log.Printf("保存用户信息失败: %v", err)
	}

	// 10. 返回结果
	eligibleCount := len(eligibleUsers)
	currentWinnersCount := len(currentWinners)
	canStillDraw := currentWinnersCount < lotteryDraw.TotalDrawers
	pendingCount := lotteryDraw.TotalDrawers - currentWinnersCount

	log.Printf("===================== 单次抽奖操作完成 =====================")
	log.Printf("抽奖结果: 轮次=%d, 当前中奖人数=%d, 可抽人数=%d, 待抽出=%d, 中奖者手机号=%d",
		request.DrawBatch, currentWinnersCount, eligibleCount, pendingCount, selectedUser.Mobile)

	c.JSON(http.StatusOK, gin.H{
		"success":               true,
		"message":               "抽奖成功",
		"draw_batch":            request.DrawBatch,
		"current_winners_count": currentWinnersCount,
		"eligible_count":        eligibleCount,
		"pending_count":         pendingCount,
		"can_still_draw":        canStillDraw,
		"total_drawers":         lotteryDraw.TotalDrawers,
		"winner_info":           winnerInfo,
	})
}

// convertImageToJPG 将图片转换为JPG格式
func (sfc *SnowFunctionController) convertImageToJPG(inputPath, outputPath, fileType string) error {
	// 打开原始图片文件
	inFile, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("打开原始图片文件失败: %v", err)
	}
	defer inFile.Close()

	// 解码图片
	var img image.Image
	var decodeErr error

	switch fileType {
	case "image/jpeg", "image/jpg":
		img, decodeErr = jpeg.Decode(inFile)
	case "image/png":
		inFile.Seek(0, 0) // 重新定位文件指针
		img, decodeErr = png.Decode(inFile)
	case "image/gif":
		inFile.Seek(0, 0) // 重新定位文件指针
		img, decodeErr = gif.Decode(inFile)
	case "image/webp":
		inFile.Seek(0, 0) // 重新定位文件指针
		img, decodeErr = webp.Decode(inFile)
	default:
		return fmt.Errorf("不支持的图片格式: %s", fileType)
	}

	if decodeErr != nil {
		return fmt.Errorf("解码图片失败: %v", decodeErr)
	}

	// 创建输出文件
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("创建输出文件失败: %v", err)
	}
	defer outFile.Close()

	// 将图片转换为JPG格式（质量85）
	options := &jpeg.Options{Quality: 85}
	if err := jpeg.Encode(outFile, img, options); err != nil {
		return fmt.Errorf("编码JPG图片失败: %v", err)
	}

	log.Printf("图片转换完成: %s -> %s", inputPath, outputPath)
	return nil
}

// QueryEligibleUsers 查询符合抽奖条件的用户信息
func (sfc *SnowFunctionController) QueryEligibleUsers(c *gin.Context) {
	var request QueryEligibleUsersRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	log.Printf("查询符合抽奖条件用户，轮次: %d", request.DrawBatch)

	// 1. 获取抽奖信息
	var lotteryDraw models.SnowLotteryDraw
	if err := db.DB.Where("draw_batch = ?", request.DrawBatch).First(&lotteryDraw).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "抽奖轮次不存在"})
		return
	}

	// 2. 解析当前中奖名单
	var currentWinners []map[string]interface{}
	if lotteryDraw.WinnersList != "" && lotteryDraw.WinnersList != "[]" {
		// 尝试修复WinnersList格式
		formattedWinnersList := "[" + lotteryDraw.WinnersList + "]"
		if err := json.Unmarshal([]byte(formattedWinnersList), &currentWinners); err != nil {
			if err := json.Unmarshal([]byte(lotteryDraw.WinnersList), &currentWinners); err != nil {
				log.Printf("解析当前中奖名单失败: %v", err)
				currentWinners = []map[string]interface{}{}
			}
		}
	}

	// 3. 获取参与者列表
	var eligibleUsers []gin.H
	if lotteryDraw.ParticipantsList != "" && lotteryDraw.ParticipantsList != "[]" {
		var userIDs []string
		if err := json.Unmarshal([]byte(lotteryDraw.ParticipantsList), &userIDs); err == nil {
			// 转换为整数数组
			var userIDInts []int
			for _, idStr := range userIDs {
				if id, err := strconv.Atoi(idStr); err == nil {
					userIDInts = append(userIDInts, id)
				}
			}

			// 查询符合条件的用户（排除已中奖的用户）
			for _, userID := range userIDInts {
				var snowUser models.SnowUser
				if err := db.DB.Where("user_id = ?", userID).First(&snowUser).Error; err != nil {
					continue
				}

				var successUser models.SnowSuccessUser
				if err := db.DB.Where("mobile = ?", snowUser.Mobile).First(&successUser).Error; err != nil {
					continue
				}

				// 检查是否已中奖
				isAlreadyWinner := false
				for _, winner := range currentWinners {
					if mobile, ok := winner["mobile"]; ok {
						var winnerMobile int
						switch v := mobile.(type) {
						case string:
							fmt.Sscanf(v, "%d", &winnerMobile)
						case float64:
							winnerMobile = int(v)
						case int:
							winnerMobile = v
						default:
							fmt.Sscanf(fmt.Sprintf("%v", v), "%d", &winnerMobile)
						}
						if winnerMobile == successUser.Mobile {
							isAlreadyWinner = true
							break
						}
					}
				}

				if !isAlreadyWinner {
					// 获取订单号和抽奖码
					drawBatchStr := strconv.Itoa(request.DrawBatch)

					orderNums, _ := successUser.GetOrderNUM()
					var orderNum string
					if num, exists := orderNums[drawBatchStr]; exists {
						orderNum = num
					}

					successCodes, _ := successUser.GetSuccessCode()
					var drawCode string
					if code, exists := successCodes[drawBatchStr]; exists {
						drawCode = code
					}

					// 添加到结果列表
					eligibleUsers = append(eligibleUsers, gin.H{
						"draw_code": drawCode,
						"mobile":    successUser.Mobile,
						"nickname":  successUser.Nickname,
						"order_num": orderNum,
					})
				}
			}
		}
	}

	// 4. 返回结果
	currentWinnersCount := len(currentWinners)
	eligibleCount := len(eligibleUsers)
	pendingCount := lotteryDraw.TotalDrawers - currentWinnersCount
	canStillDraw := currentWinnersCount < lotteryDraw.TotalDrawers

	log.Printf("查询符合抽奖条件用户完成: 轮次=%d, 当前中奖人数=%d, 可抽奖人数=%d, 待抽出=%d",
		request.DrawBatch, currentWinnersCount, eligibleCount, pendingCount)

	c.JSON(http.StatusOK, gin.H{
		"success":               true,
		"message":               "查询成功",
		"draw_batch":            request.DrawBatch,
		"current_winners_count": currentWinnersCount,
		"eligible_count":        eligibleCount,
		"pending_count":         pendingCount,
		"can_still_draw":        canStillDraw,
		"total_drawers":         lotteryDraw.TotalDrawers,
		"eligible_users":        eligibleUsers,
	})
}

// QueryUserByCode 根据抽奖码或手机号和轮次查询用户信息
func (sfc *SnowFunctionController) QueryUserByCode(c *gin.Context) {
	var request QueryUserByCodeRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	// 验证查询方式
	if request.QueryType != "code" && request.QueryType != "mobile" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的查询方式，支持 'code' 或 'mobile'"})
		return
	}

	// 转换抽奖轮次为字符串
	drawBatchStr := strconv.Itoa(request.DrawBatch)

	// 直接查询所有SnowUser
	var snowUsers []models.SnowUser
	if err := db.DB.Find(&snowUsers).Error; err != nil {
		log.Printf("查询SnowUser列表失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询用户信息失败"})
		return
	}

	// 根据查询方式查找对应的SnowUser
	var matchedSnowUser *models.SnowUser

	for i := range snowUsers {
		user := &snowUsers[i]

		if request.QueryType == "code" {
			// 根据抽奖码查询
			// 直接从SuccessCodeMap获取
			var matched bool
			if user.SuccessCodeMap != nil {
				matched = user.SuccessCodeMap[drawBatchStr] == request.SearchValue
			} else {
				// 解析JSON字符串
				var successCodeMap map[string]string
				if err := json.Unmarshal([]byte(user.SuccessCode), &successCodeMap); err == nil {
					matched = successCodeMap[drawBatchStr] == request.SearchValue
				}
			}

			// 检查当前轮次的抽奖码是否匹配
			if matched {
				matchedSnowUser = user
				break
			}
		} else if request.QueryType == "mobile" {
			// 根据手机号查询
			mobileBatch, err := user.GetMobileBatch()
			if err != nil {
				continue
			}

			// 检查当前轮次的手机号是否匹配
			if mobile, exists := mobileBatch[drawBatchStr]; exists && strconv.Itoa(int(mobile)) == request.SearchValue {
				matchedSnowUser = user
				break
			}
		}
	}

	// 如果没有找到匹配的用户
	if matchedSnowUser == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到匹配的用户信息"})
		return
	}

	// 从SnowUser中获取所需信息
	// 1. 获取对应波次的手机号
	var batchMobile string
	mobileBatch, err := matchedSnowUser.GetMobileBatch()
	if err == nil {
		if mobile, exists := mobileBatch[drawBatchStr]; exists {
			batchMobile = strconv.Itoa(int(mobile))
		}
	}

	// 2. 获取对应波次的订单号
	// var orderNum string
	// // 直接从OrderNumbersMap获取
	// if matchedSnowUser.OrderNumbersMap != nil {
	// 	orderNum = matchedSnowUser.OrderNumbersMap[drawBatchStr]
	// } else if matchedSnowUser.OrderNumbers != "" {
	// 	var orderNumbersMap map[string]string
	// 	if err := json.Unmarshal([]byte(matchedSnowUser.OrderNumbers), &orderNumbersMap); err == nil {
	// 		orderNum = orderNumbersMap[drawBatchStr]
	// 	}
	// }

	// 4. 获取对应波次的参与状态
	var participationStatus bool
	// 直接从ParticipationStatusMap获取
	if matchedSnowUser.ParticipationStatusMap != nil {
		participationStatus = matchedSnowUser.ParticipationStatusMap[drawBatchStr]
	} else if matchedSnowUser.ParticipationStatus != "" {
		var statusMap map[string]bool
		if err := json.Unmarshal([]byte(matchedSnowUser.ParticipationStatus), &statusMap); err == nil {
			participationStatus = statusMap[drawBatchStr]
		}
	}

	// 5. 获取对应波次的抽奖时间
	var drawTime string
	if matchedSnowUser.DrawTimesMap != nil {
		if timeValue, exists := matchedSnowUser.DrawTimesMap[drawBatchStr]; exists {
			drawTime = timeValue.Format("2006-01-02 15:04:05")
		}
	} else if matchedSnowUser.DrawTimes != "" {
		var drawTimesMap map[string]time.Time
		if err := json.Unmarshal([]byte(matchedSnowUser.DrawTimes), &drawTimesMap); err == nil {
			if timeValue, exists := drawTimesMap[drawBatchStr]; exists {
				drawTime = timeValue.Format("2006-01-02 15:04:05")
			}
		}
	}

	// 获取对应波次的抽奖码
	var batchCode string
	if matchedSnowUser.SuccessCodeMap != nil {
		batchCode = matchedSnowUser.SuccessCodeMap[drawBatchStr]
	} else {
		// 解析JSON字符串
		var successCodeMap map[string]string
		if err := json.Unmarshal([]byte(matchedSnowUser.SuccessCode), &successCodeMap); err == nil {
			batchCode = successCodeMap[drawBatchStr]
		}
	}

	// 返回用户信息（所有信息都来自SnowUser）
	response := gin.H{
		"mobile":               batchMobile,
		"user_id":              matchedSnowUser.UserID,
		"nickname":             matchedSnowUser.Nickname,
		"participation_status": participationStatus,
		"draw_time":            drawTime,
		"code":                 batchCode,
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "查询成功",
		"data":    response,
	})
}

// ValidateOrder 校验订单号是否满足抽奖轮次要求
func (sfc *SnowFunctionController) ValidateOrder(c *gin.Context) {
	var request ValidateOrderRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	// 1. 根据抽奖波次从SnowLotteryDraw获取抽奖时间范围
	var lotteryDraw models.SnowLotteryDraw
	drawResult := db.DB.Where("draw_batch = ?", request.DrawBatch).First(&lotteryDraw)
	if drawResult.Error != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"valid":   false,
			"message": "抽奖波次不存在",
		})
		return
	}

	// 2. 根据订单号从SnowOrderData获取付款时间
	var orderData models.SnowOrderData
	orderResult := db.DB.Where("original_online_order_number = ?", request.OrderNum).First(&orderData)
	if orderResult.Error != nil {
		// 如果original_online_order_number没有找到，尝试在online_order_number字段查找
		orderResult = db.DB.Where("online_order_number = ?", request.OrderNum).First(&orderData)
		if orderResult.Error != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"valid":   false,
				"message": "未找到该订单号",
			})
			return
		}
	}

	// 3. 验证付款时间是否在抽奖时间范围内
	if orderData.PaymentDate == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"valid":   false,
			"message": "订单付款时间为空",
		})
		return
	}

	paymentTime := *orderData.PaymentDate
	isValid := paymentTime.After(lotteryDraw.OrderBeginTime) && paymentTime.Before(lotteryDraw.OrderEndTime)

	if isValid {
		c.JSON(http.StatusOK, gin.H{
			"valid":        true,
			"message":      "订单满足抽奖轮次要求",
			"order_num":    request.OrderNum,
			"draw_batch":   request.DrawBatch,
			"payment_time": paymentTime.Format("2006-01-02 15:04:05"),
			"begin_time":   lotteryDraw.OrderBeginTime.Format("2006-01-02 15:04:05"),
			"end_time":     lotteryDraw.OrderEndTime.Format("2006-01-02 15:04:05"),
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"valid":        false,
			"message":      "订单付款时间不在抽奖轮次时间范围内",
			"order_num":    request.OrderNum,
			"draw_batch":   request.DrawBatch,
			"payment_time": paymentTime.Format("2006-01-02 15:04:05"),
			"begin_time":   lotteryDraw.OrderBeginTime.Format("2006-01-02 15:04:05"),
			"end_time":     lotteryDraw.OrderEndTime.Format("2006-01-02 15:04:05"),
		})
	}
}

// QueryParticipationCount 查询抽奖轮次参与人数
func (sfc *SnowFunctionController) QueryParticipationCount(c *gin.Context) {
	var request QueryParticipationCountRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	// 查询所有用户
	var users []models.SnowSuccessUser
	if err := db.DB.Find(&users).Error; err != nil {
		log.Printf("查询用户列表失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询参与人数失败"})
		return
	}

	// 统计参与人数
	drawBatchStr := strconv.Itoa(request.DrawBatch)
	participationCount := 0

	for _, user := range users {
		// 解析参与状态JSON
		var participationStatus map[string]bool
		if err := json.Unmarshal([]byte(user.ParticipationStatus), &participationStatus); err != nil {
			continue
		}

		// 检查当前轮次是否参与
		if participated, exists := participationStatus[drawBatchStr]; exists && participated {
			participationCount++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"draw_batch":          request.DrawBatch,
		"participation_count": participationCount,
	})
}

// UpgradeUserVip 升级用户会员等级并设置抽奖资格
func (sfc *SnowFunctionController) UpgradeUserVip(c *gin.Context) {
	var request UpgradeUserVipRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	// 步骤1: 将用户升级为4级会员
	isSuccess, err := vip.SetUserVipLevel(4, request.Mobile)
	if err != nil {
		log.Printf("升级会员等级失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "升级会员等级失败: " + err.Error()})
		return
	}

	if !isSuccess {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "升级会员等级操作未成功"})
		return
	}

	// 步骤2: 获取会员信息
	vipInfo, customerInfo, err := vip.GetUserVipLevel(request.Mobile)
	if err != nil {
		log.Printf("获取会员信息失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取会员信息失败: " + err.Error()})
		return
	}

	// 步骤3: 将会员信息传入SnowSuccessUser模型
	// 解析手机号为整数
	mobileInt, err := strconv.Atoi(request.Mobile)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "手机号格式错误"})
		return
	}

	// 尝试查找用户，如果不存在则创建新用户
	var user models.SnowSuccessUser
	result := db.DB.Where("mobile = ?", mobileInt).First(&user)

	// 如果用户不存在，创建新用户
	if result.Error != nil {
		user = models.SnowSuccessUser{
			Mobile: mobileInt,
		}

		// 从会员信息中获取昵称（如果有）
		if customerInfo != nil {
			if nickname, ok := customerInfo["name"].(string); ok {
				user.Nickname = nickname
			}
		}

		// 设置会员来源
		user.MemberSource = "vip_upgrade"

		// 保存新用户
		if err := db.DB.Create(&user).Error; err != nil {
			log.Printf("创建用户记录失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "创建用户记录失败"})
			return
		}
	}

	// 步骤4: 设置抽奖资格默认为{"1": true, "2": true, "3": true}
	defaultDrawEligibility := map[string]bool{
		"1": true,
		"2": true,
		"3": true,
	}

	if err := user.SetDrawEligibility(defaultDrawEligibility); err != nil {
		log.Printf("设置抽奖资格失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "设置抽奖资格失败"})
		return
	}

	// 更新用户信息
	if err := db.DB.Save(&user).Error; err != nil {
		log.Printf("更新用户记录失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新用户记录失败"})
		return
	}

	// 返回成功响应
	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"message":       "用户会员升级成功，抽奖资格已设置",
		"mobile":        request.Mobile,
		"vip_info":      vipInfo,
		"customer_info": customerInfo,
	})
}

// QueryUserVipInfo 查询用户会员信息
func (sfc *SnowFunctionController) QueryUserVipInfo(c *gin.Context) {
	var request QueryUserVipInfoRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	// 调用vip包中的GetUserVipLevel方法获取用户会员信息
	vipInfo, customerInfo, err := vip.GetUserVipLevel(request.Mobile)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "获取会员信息失败: " + err.Error(),
			"mobile": request.Mobile,
		})
		return
	}

	// 返回查询结果
	c.JSON(http.StatusOK, gin.H{
		"mobile":        request.Mobile,
		"vip_info":      vipInfo,
		"customer_info": customerInfo,
	})
}

// WinnerInfo 中奖者信息结构
type WinnerInfo struct {
	DrawCode        string         `json:"draw_code"`
	Mobile          interface{}    `json:"mobile"`
	Nickname        string         `json:"nickname"`
	OrderNum        string         `json:"order_num"`
	Prizes          map[string]int `json:"prizes"`
	ReceiverName    string         `json:"receiver_name"`
	ReceiverPhone   string         `json:"receiver_phone"`
	Province        string         `json:"province"`
	City            string         `json:"city"`
	County          string         `json:"county"`
	DetailedAddress string         `json:"detailed_address"`
	DrawTime        string         `json:"draw_time"`
}

// ExportWinners 导出中奖名单
func (sfc *SnowFunctionController) ExportWinners(c *gin.Context) {
	var request ExportWinnersRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	// 1. 根据抽奖波次获取抽奖信息
	var lotteryDraw models.SnowLotteryDraw
	log.Printf("开始查询抽奖波次 %d 的信息", request.DrawBatch)
	drawResult := db.DB.Where("draw_batch = ?", request.DrawBatch).First(&lotteryDraw)
	if drawResult.Error != nil {
		log.Printf("抽奖波次 %d 不存在", request.DrawBatch)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "抽奖波次不存在",
		})
		return
	}
	log.Printf("抽奖波次信息查询成功，抽奖名称: %s, TotalDrawers: %d", lotteryDraw.DrawName, lotteryDraw.TotalDrawers)

	// 2. 解析WinnersList获取中奖用户信息
	var winners []WinnerInfo
	if lotteryDraw.WinnersList != "" && lotteryDraw.WinnersList != "[]" && lotteryDraw.WinnersList != "{}" {
		log.Printf("开始解析WinnersList，原始数据: %s", lotteryDraw.WinnersList)
		// 尝试直接解析JSON数组
		if err := json.Unmarshal([]byte(lotteryDraw.WinnersList), &winners); err != nil {
			log.Printf("直接解析WinnersList失败: %v，尝试修复格式", err)
			// 尝试修复格式：如果是对象直接用逗号连接的格式，需要转换为标准JSON数组
			if !strings.HasPrefix(lotteryDraw.WinnersList, "[") && strings.Contains(lotteryDraw.WinnersList, "},{") {
				// 尝试修复格式
				formattedWinnersList := "[" + lotteryDraw.WinnersList + "]"
				if err := json.Unmarshal([]byte(formattedWinnersList), &winners); err != nil {
					log.Printf("修复格式后解析WinnersList仍然失败: %v", err)
					c.JSON(http.StatusInternalServerError, gin.H{
						"error": "解析中奖名单失败",
					})
					return
				}
				log.Printf("修复格式后解析WinnersList成功，共%d个中奖者", len(winners))
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "解析中奖名单失败",
				})
				return
			}
		} else {
			log.Printf("直接解析WinnersList成功，共%d个中奖者", len(winners))
		}
	} else {
		log.Printf("WinnersList为空或格式不正确")
		// 如果WinnersList为空，仍然创建空Excel文件
		winners = []WinnerInfo{}
	}

	// 4. 创建Excel文件
	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			log.Printf("关闭Excel文件失败: %v", err)
		}
	}()

	// 设置工作表名称
	f.SetSheetName("Sheet1", "中奖名单")

	// 设置表头
	headers := []string{"序号", "奖品", "手机号", "订单号", "抽奖码", "抽奖轮次", "中奖时间", "收货人昵称", "收货人手机号", "省", "市", "县", "具体地址", "验证状态"}
	for i, header := range headers {
		cell := string(rune('A'+i)) + "1"
		f.SetCellValue("中奖名单", cell, header)
	}

	// 5. 填充数据
	for i, winner := range winners {
		row := i + 2

		// 设置序号
		f.SetCellValue("中奖名单", "A"+strconv.Itoa(row), i+1)

		// 设置奖品信息 - 使用每个用户自己的prizes数据
		prizesInfo := ""
		for prizeName, count := range winner.Prizes {
			if prizesInfo != "" {
				prizesInfo += ", "
			}
			prizesInfo += prizeName + "(" + strconv.Itoa(count) + "份)"
		}
		f.SetCellValue("中奖名单", "B"+strconv.Itoa(row), prizesInfo)

		// 设置手机号 - 处理可能是数字或字符串的情况
		mobileStr := ""
		switch m := winner.Mobile.(type) {
		case float64:
			mobileStr = strconv.FormatFloat(m, 'f', 0, 64)
		case string:
			mobileStr = m
		default:
			mobileStr = ""
		}
		f.SetCellValue("中奖名单", "C"+strconv.Itoa(row), mobileStr)

		// 设置订单号
		f.SetCellValue("中奖名单", "D"+strconv.Itoa(row), winner.OrderNum)

		// 设置抽奖码
		f.SetCellValue("中奖名单", "E"+strconv.Itoa(row), winner.DrawCode)

		// 设置抽奖轮次
		f.SetCellValue("中奖名单", "F"+strconv.Itoa(row), request.DrawBatch)

		// 设置中奖时间
		f.SetCellValue("中奖名单", "G"+strconv.Itoa(row), winner.DrawTime)

		// 设置收货人昵称
		f.SetCellValue("中奖名单", "H"+strconv.Itoa(row), winner.ReceiverName)

		// 设置收货人手机号
		f.SetCellValue("中奖名单", "I"+strconv.Itoa(row), winner.ReceiverPhone)

		// 设置省
		f.SetCellValue("中奖名单", "J"+strconv.Itoa(row), winner.Province)

		// 设置市
		f.SetCellValue("中奖名单", "K"+strconv.Itoa(row), winner.City)

		// 设置县
		f.SetCellValue("中奖名单", "L"+strconv.Itoa(row), winner.County)

		// 设置具体地址
		f.SetCellValue("中奖名单", "M"+strconv.Itoa(row), winner.DetailedAddress)

		// 设置验证状态
		verificationStatus := "未验证"
		if winner.ReceiverName != "" && winner.ReceiverPhone != "" && winner.Province != "" && winner.City != "" {
			verificationStatus = "已验证"
		}
		f.SetCellValue("中奖名单", "N"+strconv.Itoa(row), verificationStatus)
	}

	// 6. 设置列宽
	widths := []float64{8, 30, 15, 20, 15, 10, 15, 15, 15, 10, 10, 10, 30, 10}
	for i, width := range widths {
		col := string(rune('A' + i))
		f.SetColWidth("中奖名单", col, col, width)
	}

	// 7. 生成文件名
	timestamp := time.Now().Format("20060102150405")
	fileName := "中奖名单_第" + strconv.Itoa(request.DrawBatch) + "轮_" + timestamp + ".xlsx"

	// 8. 设置响应头，让浏览器下载文件
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename="+fileName)
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Cache-Control", "no-cache")

	// 9. 写入文件内容到响应体
	if err := f.Write(c.Writer); err != nil {
		log.Printf("写入Excel文件到响应失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "导出Excel失败",
		})
		return
	}

	c.Status(http.StatusOK)
}

// QueryDrawRecordsRequest 查询抽奖记录请求结构体
type QueryDrawRecordsRequest struct {
	DrawBatch int `json:"draw_batch" binding:"required"` // 抽奖轮次
	Page      int `json:"page"`                          // 页码，默认1
	PageSize  int `json:"page_size"`                     // 每页条数，默认20，最大100
}

// QueryDrawRecords 查询抽奖记录，解析模型SnowLotteryDraw的Record字段
func (sfc *SnowFunctionController) QueryDrawRecords(c *gin.Context) {
	var request QueryDrawRecordsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	// 设置默认值
	if request.Page <= 0 {
		request.Page = 1
	}
	if request.PageSize <= 0 {
		request.PageSize = 20
	}

	// 查询抽奖活动
	var lotteryDraw models.SnowLotteryDraw
	if err := db.DB.Where("draw_batch = ?", request.DrawBatch).First(&lotteryDraw).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到指定轮次的抽奖活动"})
		return
	}

	// 解析Record字段
	var records []map[string]interface{}
	if lotteryDraw.Record != "" {
		if err := json.Unmarshal([]byte(lotteryDraw.Record), &records); err != nil {
			log.Printf("解析抽奖记录失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "解析抽奖记录失败"})
			return
		}
	}

	// 计算分页参数
	total := len(records)
	startIndex := (request.Page - 1) * request.PageSize
	endIndex := startIndex + request.PageSize

	// 确保索引不越界
	if startIndex > total {
		startIndex = total
	}
	if endIndex > total {
		endIndex = total
	}

	// 切片获取分页数据
	var paginatedRecords []map[string]interface{}
	if startIndex < total {
		paginatedRecords = records[startIndex:endIndex]
	}

	// 计算最大页数
	maxPage := 0
	if request.PageSize > 0 {
		maxPage = (total + request.PageSize - 1) / request.PageSize
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"records":   paginatedRecords,
		"total":     total,
		"page":      request.Page,
		"page_size": request.PageSize,
		"max_page":  maxPage,
	})
}

// QueryWinnersInfoRequest 查询中奖者信息请求结构体

type QueryWinnersInfoRequest struct {
	DrawBatch int `json:"draw_batch" binding:"required"` // 抽奖轮次
}

// QueryDrawParticipantsRequest 查询抽奖参与者请求结构体
type QueryDrawParticipantsRequest struct {
	DrawBatch int `json:"draw_batch" binding:"required"` // 抽奖轮次
}

// ExportDrawParticipantsExcel 导出抽奖参与者信息为Excel
func (sfc *SnowFunctionController) ExportDrawParticipantsExcel(c *gin.Context) {
	var request QueryDrawParticipantsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	// 1. 查询抽奖活动
	var lotteryDraw models.SnowLotteryDraw
	if err := db.DB.Where("draw_batch = ?", request.DrawBatch).First(&lotteryDraw).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到指定轮次的抽奖活动"})
		return
	}

	// 2. 解析参与者列表
	var userIDs []string
	if lotteryDraw.ParticipantsList != "" {
		if err := json.Unmarshal([]byte(lotteryDraw.ParticipantsList), &userIDs); err != nil {
			log.Printf("解析参与者列表失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "解析参与者列表失败"})
			return
		}
	}

	// 3. 准备参与者数据（复用QueryDrawParticipants的逻辑获取数据）
	var participants []map[string]interface{}

	// 如果有参与者，查询详细信息
	if len(userIDs) > 0 {
		// 遍历userID列表，查询用户信息
		for _, userID := range userIDs {
			// 转换userID为int
			userIDInt, err := strconv.Atoi(userID)
			if err != nil {
				log.Printf("用户ID转换失败: %v", err)
				continue
			}

			// 从SnowUser只查询draw_times和user_id
			var user models.SnowUser
			if err := db.DB.Select("user_id, draw_times, mobile").Where("user_id = ?", userIDInt).First(&user).Error; err != nil {
				log.Printf("查询用户信息失败: %v", err)
				continue
			}

			// 确保DrawTimesMap被正确初始化
			if user.DrawTimesMap == nil {
				user.DrawTimesMap = make(map[string]time.Time)
			}

			// 通过手机号查询SnowSuccessUser
			var successUser models.SnowSuccessUser
			if err := db.DB.Where("mobile = ?", user.Mobile).First(&successUser).Error; err != nil {
				log.Printf("查询成功用户信息失败: %v", err)
				continue
			}

			// 创建参与者信息map
			participant := map[string]interface{}{
				"user_id":  user.UserID,
				"nickname": successUser.Nickname,
				"mobile":   successUser.Mobile,
			}

			// 从SnowUser的DrawTimesMap中获取对应波次的参与时间
			if drawTime, exists := user.DrawTimesMap[strconv.Itoa(request.DrawBatch)]; exists {
				participant["draw_time"] = drawTime.Format("2006-01-02 15:04:05")
			}

			// 从SnowSuccessUser获取对应波次的订单号
			orderNUM, err := successUser.GetOrderNUM()
			if err != nil {
				log.Printf("获取订单号信息失败: %v", err)
			} else {
				batchStr := strconv.Itoa(request.DrawBatch)
				if orderNumForBatch, exists := orderNUM[batchStr]; exists {
					participant["order_num"] = orderNumForBatch
				}
			}

			// 从SnowSuccessUser获取对应波次的成功码
			successCode, err := successUser.GetSuccessCode()
			if err != nil {
				log.Printf("获取成功码信息失败: %v", err)
			} else {
				batchStr := strconv.Itoa(request.DrawBatch)
				if err != nil {
					log.Printf("批次号转换失败: %v", err)
					continue
				}
				if codeForBatch, exists := successCode[batchStr]; exists {
					participant["success_code"] = codeForBatch
					log.Printf("codeForBatch: %v", codeForBatch)
				}
			}
			participants = append(participants, participant)
			log.Printf("participant: %v", participant)
		}
	}

	// 4. 创建Excel文件
	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			log.Printf("关闭Excel文件失败: %v", err)
		}
	}()

	// 设置工作表名称
	f.SetSheetName("Sheet1", "抽奖参与者名单")

	// 设置表头
	headers := []string{"序号", "用户ID", "昵称", "手机号", "抽奖码", "抽奖时间", "订单号"}
	for i, header := range headers {
		cell := string(rune('A'+i)) + "1"
		f.SetCellValue("抽奖参与者名单", cell, header)
	}

	// 5. 填充数据
	for i, participant := range participants {
		row := i + 2 // 从第2行开始填充数据（第1行是表头）

		// 设置序号
		f.SetCellValue("抽奖参与者名单", "A"+strconv.Itoa(row), i+1)

		// 设置用户ID
		if userID, ok := participant["user_id"]; ok {
			f.SetCellValue("抽奖参与者名单", "B"+strconv.Itoa(row), userID)
		}

		// 设置抽奖码
		if drawCode, ok := participant["success_code"]; ok {
			f.SetCellValue("抽奖参与者名单", "E"+strconv.Itoa(row), drawCode)
		}

		// 设置昵称
		if nickname, ok := participant["nickname"]; ok {
			f.SetCellValue("抽奖参与者名单", "C"+strconv.Itoa(row), nickname)
		}

		// 设置手机号
		if mobile, ok := participant["mobile"]; ok {
			f.SetCellValue("抽奖参与者名单", "D"+strconv.Itoa(row), mobile)
		}

		// 设置抽奖时间
		if drawTime, ok := participant["draw_time"]; ok {
			f.SetCellValue("抽奖参与者名单", "F"+strconv.Itoa(row), drawTime)
		}

		// 设置订单号
		if orderNum, ok := participant["order_num"]; ok {
			f.SetCellValue("抽奖参与者名单", "G"+strconv.Itoa(row), orderNum)
		}
	}

	// 6. 设置列宽
	widths := []float64{8, 12, 20, 15, 25, 30}
	for i, width := range widths {
		col := string(rune('A' + i))
		f.SetColWidth("抽奖参与者名单", col, col, width)
	}

	// 7. 生成文件名
	timestamp := time.Now().Format("20060102150405")
	fileName := "抽奖参与者名单_第" + strconv.Itoa(request.DrawBatch) + "轮_" + timestamp + ".xlsx"

	// 8. 设置响应头，让浏览器下载文件
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename="+fileName)
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Cache-Control", "no-cache")

	// 9. 写入文件内容到响应体
	if err := f.Write(c.Writer); err != nil {
		log.Printf("写入Excel文件到响应失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "导出Excel失败",
		})
		return
	}

	c.Status(http.StatusOK)
}

// QueryDrawInfoRequest 查询指定抽奖轮次信息请求结构体

// QueryWinnersInfo 查询指定抽奖轮次的中奖者信息
func (sfc *SnowFunctionController) QueryWinnersInfo(c *gin.Context) {
	var request QueryWinnersInfoRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	// 查询抽奖活动是否存在
	var draw models.SnowLotteryDraw
	if err := db.DB.Where("draw_batch = ?", request.DrawBatch).First(&draw).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "抽奖活动不存在"})
		return
	}

	// 解析中奖名单
	var winners []map[string]interface{}
	if draw.WinnersList != "" {
		// 直接解析JSON数组
		err := json.Unmarshal([]byte(draw.WinnersList), &winners)
		if err != nil {
			log.Printf("解析中奖名单失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "解析中奖名单失败"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"winners":       winners,
		"total":         len(winners),
		"total_drawers": draw.TotalDrawers,
	})
}

// QueryDrawInfo 查询抽奖轮次信息（可查询指定轮次或所有轮次）
// QueryDrawParticipants 查询指定轮次抽奖的参与者信息
func (sfc *SnowFunctionController) QueryDrawParticipants(c *gin.Context) {
	var request QueryDrawParticipantsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	// 查询抽奖活动
	var lotteryDraw models.SnowLotteryDraw
	if err := db.DB.Where("draw_batch = ?", request.DrawBatch).First(&lotteryDraw).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到指定轮次的抽奖活动"})
		return
	}

	// 解析参与者列表
	var userIDs []string
	if lotteryDraw.ParticipantsList != "" {
		if err := json.Unmarshal([]byte(lotteryDraw.ParticipantsList), &userIDs); err != nil {
			log.Printf("解析参与者列表失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "解析参与者列表失败"})
			return
		}
	}

	if len(userIDs) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"success":      true,
			"participants": []interface{}{},
			"total":        0,
		})
		return
	}

	// 构建结果集
	var participants []map[string]interface{}

	// 遍历userID列表，查询用户信息
	for _, userID := range userIDs {
		// 转换userID为int
		userIDInt, err := strconv.Atoi(userID)
		if err != nil {
			log.Printf("用户ID转换失败: %v", err)
			continue
		}

		// 从SnowUser查询必要字段，包括success_code
		var user models.SnowUser
		if err := db.DB.Select("user_id, draw_times, mobile, success_code").Where("user_id = ?", userIDInt).First(&user).Error; err != nil {
			log.Printf("查询用户信息失败: %v", err)
			continue
		}

		// 确保DrawTimesMap被正确初始化
		if user.DrawTimesMap == nil {
			user.DrawTimesMap = make(map[string]time.Time)
		}

		// 通过手机号查询SnowSuccessUser
		var successUser models.SnowSuccessUser
		if err := db.DB.Where("mobile = ?", user.Mobile).First(&successUser).Error; err != nil {
			log.Printf("查询成功用户信息失败: %v", err)
			continue
		}

		// 创建参与者信息map
		participant := map[string]interface{}{
			"user_id":  user.UserID,
			"nickname": successUser.Nickname,
			"mobile":   successUser.Mobile,
		}

		// 从SnowUser的SuccessCodeMap中获取对应波次的抽奖码
		batchStr := strconv.Itoa(request.DrawBatch)
		if drawCode, exists := user.SuccessCodeMap[batchStr]; exists {
			participant["draw_code"] = drawCode
		}

		// 从SnowUser的DrawTimesMap中获取对应波次的参与时间
		if drawTime, exists := user.DrawTimesMap[batchStr]; exists {
			participant["draw_time"] = drawTime.Format(time.RFC3339)
		}

		// 从SnowSuccessUser获取对应波次的订单号
		orderNUM, err := successUser.GetOrderNUM()
		if err != nil {
			log.Printf("获取订单号信息失败: %v", err)
		} else {
			batchStr := strconv.Itoa(request.DrawBatch)
			if orderNumForBatch, exists := orderNUM[batchStr]; exists {
				participant["order_num"] = orderNumForBatch
			} else {
				log.Printf("未找到用户 %s 在波次 %s 的订单号", successUser.Mobile, batchStr)
			}
		}

		participants = append(participants, participant)
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"participants": participants,
		"total":        len(participants),
	})
}

func (sfc *SnowFunctionController) QueryDrawInfo(c *gin.Context) {
	// 解析请求参数，但允许draw_batch为空
	type FlexRequest struct {
		DrawBatch *int `json:"draw_batch"` // 使用指针允许值为nil
	}
	var request FlexRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	// 获取当前时间
	currentTime := time.Now()

	// 如果指定了draw_batch，查询单个抽奖轮次
	if request.DrawBatch != nil {
		var draw models.SnowLotteryDraw
		if err := db.DB.Where("draw_batch = ?", *request.DrawBatch).First(&draw).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "未找到指定的抽奖轮次信息"})
			return
		}

		// 解析Prizes字段（JSON格式）
		var prizes map[string]int
		if err := json.Unmarshal([]byte(draw.Prizes), &prizes); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "解析奖品信息失败"})
			return
		}

		// 判断抽奖状态
		status := "已结束"
		if currentTime.Before(draw.OrderBeginTime) {
			status = "未开始"
		} else if currentTime.Before(draw.OrderEndTime) {
			status = "进行中"
		}

		// 返回指定的字段信息
		c.JSON(http.StatusOK, gin.H{
			"draw_batch":         draw.DrawBatch,
			"draw_name":          draw.DrawName,
			"prizes":             prizes,
			"order_begin_time":   draw.OrderBeginTime.Format("2006-01-02 15:04:05"),
			"order_end_time":     draw.OrderEndTime.Format("2006-01-02 15:04:05"),
			"draw_time":          draw.DrawTime.Format("2006-01-02 15:04:05"),
			"participants_count": draw.ParticipantsCount,
			"total_drawers":      draw.TotalDrawers,
			"status":             status,
		})
		return
	}

	// 如果未指定draw_batch，返回所有抽奖轮次信息
	var draws []models.SnowLotteryDraw
	if err := db.DB.Find(&draws).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询抽奖轮次信息失败"})
		return
	}

	// 构建返回结果
	var result []map[string]interface{}
	for _, draw := range draws {
		// 解析Prizes字段（JSON格式）
		var prizes map[string]int
		if err := json.Unmarshal([]byte(draw.Prizes), &prizes); err != nil {
			continue // 跳过解析失败的记录
		}

		// 判断抽奖状态
		status := "已结束"
		if currentTime.Before(draw.OrderBeginTime) {
			status = "未开始"
		} else if currentTime.Before(draw.OrderEndTime) {
			status = "进行中"
		}

		item := map[string]interface{}{
			"draw_batch":         draw.DrawBatch,
			"draw_name":          draw.DrawName,
			"prizes":             prizes,
			"order_begin_time":   draw.OrderBeginTime.Format("2006-01-02 15:04:05"),
			"order_end_time":     draw.OrderEndTime.Format("2006-01-02 15:04:05"),
			"draw_time":          draw.DrawTime.Format("2006-01-02 15:04:05"),
			"participants_count": draw.ParticipantsCount,
			"total_drawers":      draw.TotalDrawers,
			"status":             status,
		}
		result = append(result, item)
	}

	c.JSON(http.StatusOK, gin.H{
		"total": len(result),
		"data":  result,
	})
}

// LotteryDraw 执行抽奖操作
func (sfc *SnowFunctionController) LotteryDraw(c *gin.Context) {
	log.Printf("===================== 开始执行抽奖操作 =====================")
	var request LotteryDrawRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		log.Printf("请求参数绑定失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}
	log.Printf("抽奖请求参数绑定成功，抽奖轮次: %d", request.DrawBatch)

	// 1. 根据抽奖波次获取抽奖信息
	var lotteryDraw models.SnowLotteryDraw
	drawResult := db.DB.Where("draw_batch = ?", request.DrawBatch).First(&lotteryDraw)
	if drawResult.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "抽奖波次不存在",
		})
		return
	}

	// 2. 解析奖品信息（支持JSON格式和传统格式）
	var prizeName string

	log.Printf("原始奖品数据: %s", lotteryDraw.Prizes)

	if lotteryDraw.Prizes != "" && lotteryDraw.Prizes != "{}" {
		// 首先尝试JSON解析（只取第一个奖品名称）
		var prizesMap map[string]int
		if json.Unmarshal([]byte(lotteryDraw.Prizes), &prizesMap) == nil {
			log.Printf("JSON解析成功，奖品数量: %d", len(prizesMap))
			// 取第一个奖品名称
			for name := range prizesMap {
				prizeName = name
				log.Printf("JSON解析到奖品名称: %s", prizeName)
				break
			}
		} else {
			log.Printf("JSON解析失败，尝试传统格式解析")
			// 分割奖品字符串，处理"奖品:数量"格式
			prizePairs := strings.Split(lotteryDraw.Prizes, ",")
			if len(prizePairs) > 0 {
				pair := strings.TrimSpace(prizePairs[0])
				if pair != "" {
					parts := strings.SplitN(pair, ":", 2)
					if len(parts) > 0 {
						prizeName = strings.TrimSpace(parts[0])
						log.Printf("传统格式解析到奖品名称: %s", prizeName)
					}
				}
			}
		}
	}

	// 特殊处理黄金雪花片奖品，确保它被正确识别
	if prizeName == "" && lotteryDraw.Prizes == `{"黄金雪花片": 15}` {
		log.Printf("检测到黄金雪花片奖品特殊情况，手动设置奖品名称")
		prizeName = "黄金雪花片"
	}

	// 3. 从ParticipantsList字段获取抽奖用户ID列表（格式：["283218", "298149"]）
	drawBatchStr := strconv.Itoa(request.DrawBatch)
	var eligibleUsers []models.SnowSuccessUser

	log.Printf("开始处理抽奖轮次 %d，ParticipantsList内容: %s", request.DrawBatch, lotteryDraw.ParticipantsList)

	if lotteryDraw.ParticipantsList != "" && lotteryDraw.ParticipantsList != "[]" {
		// 解析JSON数组字符串
		var userIDs []string
		if err := json.Unmarshal([]byte(lotteryDraw.ParticipantsList), &userIDs); err == nil {
			log.Printf("从ParticipantsList解析出 %d 个用户ID", len(userIDs))

			if len(userIDs) == 0 {
				log.Printf("ParticipantsList解析为空数组，没有可参与抽奖的用户")
			} else {
				// 直接使用ParticipantsList中的用户ID，不再查询SnowSuccessUser
				userFoundCount := 0
				for _, userIDStr := range userIDs {
					log.Printf("处理用户ID: %s", userIDStr)
					var user models.SnowSuccessUser
					// 将string类型的userIDStr转换为int类型
					userID, err := strconv.Atoi(userIDStr)
					if err != nil {
						log.Printf("用户ID格式错误，跳过: %s, 错误: %v", userIDStr, err)
						continue
					}
					// 设置用户ID，其他字段可能在后续流程中使用
					user.UserID = userID
					userFoundCount++

					// 由于不再查询数据库，直接将用户添加到可选列表
					// 假设所有在ParticipantsList中的用户都有资格参与当前轮次抽奖
					eligibleUsers = append(eligibleUsers, user)
					log.Printf("用户 %d 已添加到抽奖可选列表", userID)
				}
				log.Printf("共处理 %d 个用户，全部添加到抽奖可选列表", userFoundCount)
				log.Printf("共处理 %d 个有效用户，全部符合抽奖条件，添加到可选列表的用户数: %d", userFoundCount, len(eligibleUsers))
			}
		} else {
			log.Printf("ParticipantsList格式解析错误: %v，内容: %s", err, lotteryDraw.ParticipantsList)
		}
	} else {
		log.Printf("ParticipantsList为空或等于[]，无法获取参与用户")
	}

	// 5. 随机抽取获奖者
	var winners []models.SnowSuccessUser
	var winnersInfo []gin.H

	// 根据设置的中奖人数确定实际中奖人数
	winnersCount := lotteryDraw.TotalDrawers
	if winnersCount == 0 {
		// 如果没有设置中奖人数，则默认为10人
		winnersCount = 10
	}
	// 确保不超过参与人数
	if winnersCount > len(eligibleUsers) {
		winnersCount = len(eligibleUsers)
	}

	if winnersCount > 0 {
		log.Printf("开始随机选择获奖者，需要选择 %d 名，参与用户数: %d", winnersCount, len(eligibleUsers))
		// 使用crypto/rand进行安全的随机选择
		selected := make(map[int]bool)
		successIterations := 0
		maxIterations := winnersCount * 10 // 设置最大迭代次数防止无限循环
		iterations := 0

		for len(winners) < winnersCount && len(selected) < len(eligibleUsers) && iterations < maxIterations {
			iterations++
			// 生成随机索引
			n, err := rand.Int(rand.Reader, big.NewInt(int64(len(eligibleUsers))))
			if err != nil {
				log.Printf("生成随机数失败: %v，尝试使用备用随机方法", err)
				// 备用随机方法
				index := time.Now().UnixNano() % int64(len(eligibleUsers))
				if index < 0 {
					index = -index
				}
				// 确保不重复选择
				if !selected[int(index)] {
					selected[int(index)] = true
					winners = append(winners, eligibleUsers[int(index)])
					successIterations++
					log.Printf("备用随机方法成功选择用户，索引: %d, 用户ID: %d", index, eligibleUsers[int(index)].UserID)
				}
			} else {
				index := int(n.Int64())

				// 确保不重复选择
				if !selected[index] {
					selected[index] = true
					winners = append(winners, eligibleUsers[index])
					successIterations++
					log.Printf("成功选择用户，索引: %d, 用户ID: %d", index, eligibleUsers[index].UserID)
				}
			}
		}
		log.Printf("随机选择完成，成功选择 %d 名获奖者，迭代次数: %d", successIterations, iterations)

		// 6. 构建中奖者信息并分配奖品
		log.Printf("开始构建中奖者信息并分配奖品，奖品名称: %s，获奖者总数: %d", prizeName, len(winners))

		// 用于保存需要更新中奖状态的用户
		var usersToUpdate []models.SnowSuccessUser

		for i, winner := range winners {
			// 通过用户ID查询SnowUser获取Mobile
			log.Printf("处理获奖者 #%d: 用户ID=%d", i+1, winner.UserID)
			var snowUser models.SnowUser
			if err := db.DB.Where("user_id = ?", winner.UserID).First(&snowUser).Error; err != nil {
				log.Printf("通过用户ID=%d查询SnowUser失败: %v", winner.UserID, err)
				continue
			}
			log.Printf("成功查询到用户ID=%d的手机号: %d", winner.UserID, snowUser.Mobile)

			// 使用Mobile查询SnowSuccessUser获取详细信息
			var successUser models.SnowSuccessUser
			if err := db.DB.Where("mobile = ?", snowUser.Mobile).First(&successUser).Error; err != nil {
				log.Printf("通过手机号=%d查询SnowSuccessUser失败: %v", snowUser.Mobile, err)
				continue
			}
			log.Printf("成功查询到手机号=%d的详细用户信息: %+v", snowUser.Mobile, successUser)

			// 获取订单号
			orderNums, err := successUser.GetOrderNUM()
			var orderNum string
			if err != nil {
				log.Printf("用户 %d 获取订单号失败: %v", successUser.UserID, err)
			} else if num, exists := orderNums[drawBatchStr]; exists {
				orderNum = num
				log.Printf("用户 %d 获取订单号成功: %s", successUser.UserID, orderNum)
			} else {
				log.Printf("用户 %d 没有轮次 %s 的订单号", successUser.UserID, drawBatchStr)
			}

			// 获取抽奖码
			successCodes, err := successUser.GetSuccessCode()
			var drawCode string
			if err != nil {
				log.Printf("用户 %d 获取抽奖码失败: %v", successUser.UserID, err)
			} else if code, exists := successCodes[drawBatchStr]; exists {
				drawCode = code
				log.Printf("用户 %d 获取抽奖码成功: %s", successUser.UserID, drawCode)
			} else {
				log.Printf("用户 %d 没有轮次 %s 的抽奖码", successUser.UserID, drawBatchStr)
			}

			// 分配奖品（所有获奖者获得相同的奖品）
			var prize string
			if prizeName != "" {
				prize = prizeName
				log.Printf("用户 %d 分配奖品: %s", successUser.UserID, prize)
			} else {
				log.Printf("用户 %d 没有可用奖品分配", successUser.UserID)
			}

			// 更新中奖者的中奖状态
			log.Printf("开始更新用户 %d 的中奖状态", successUser.UserID)
			winningStatus, err := successUser.GetWinningStatus()
			if err != nil {
				log.Printf("用户 %d 获取当前中奖状态失败: %v，使用新map", successUser.UserID, err)
				winningStatus = make(map[string]bool)
			}
			winningStatus[drawBatchStr] = true
			if err := successUser.SetWinningStatus(winningStatus); err != nil {
				log.Printf("设置用户 %d 中奖状态失败: %v", successUser.UserID, err)
				continue
			}

			// 更新中奖时间
			log.Printf("开始更新用户 %d 的中奖时间", successUser.UserID)
			drawSuccessTime, err := successUser.GetDrawSuccessTime()
			if err != nil {
				log.Printf("用户 %d 获取当前中奖时间失败: %v，使用新map", successUser.UserID, err)
				drawSuccessTime = make(map[string]string)
			}
			// 设置当前波次的中奖时间
			drawSuccessTime[drawBatchStr] = time.Now().Format("2006-01-02 15:04:05")
			if err := successUser.SetDrawSuccessTime(drawSuccessTime); err != nil {
				log.Printf("设置用户 %d 中奖时间失败: %v", successUser.UserID, err)
				continue
			}

			// 将需要更新的用户添加到列表
			usersToUpdate = append(usersToUpdate, successUser)

			// 构建符合要求格式的中奖者信息
			winnerInfo := gin.H{
				"draw_code":  drawCode,
				"mobile":     successUser.Mobile,
				"nickname":   successUser.Nickname,
				"order_num":  orderNum,
				"prize_name": prize,
				"draw_time":  time.Now().Format("2006-01-02 15:04:05"), // 添加中奖时间字段
			}
			winnersInfo = append(winnersInfo, winnerInfo)
			log.Printf("获奖者 #%d 信息构建完成: %+v", i+1, winnerInfo)
		}

		// 7. 更新抽奖结果到数据库
		// 更新中奖者的中奖状态
		saveSuccessCount := 0
		saveFailedCount := 0
		for _, user := range usersToUpdate {
			if err := db.DB.Save(&user).Error; err != nil {
				log.Printf("保存用户 %d 中奖状态失败: %v", user.UserID, err)
				saveFailedCount++
			} else {
				log.Printf("用户 %d 中奖状态保存成功", user.UserID)
				saveSuccessCount++
			}
		}
		log.Printf("中奖状态更新完成：成功 %d 个，失败 %d 个", saveSuccessCount, saveFailedCount)

		// 将中奖者信息序列化为JSON并设置到WinnersList
		if len(winnersInfo) > 0 {
			winnersJSON, err := json.Marshal(winnersInfo)
			if err != nil {
				log.Printf("序列化中奖者信息失败: %v", err)
			} else {
				lotteryDraw.WinnersList = string(winnersJSON)
				log.Printf("成功设置WinnersList: %s", lotteryDraw.WinnersList)
			}
		}

		// 8. 更新抽奖活动的参与人数和实际中奖人数
		lotteryDraw.ParticipantsCount = len(eligibleUsers)
		// 记录实际中奖人数但不保存到数据库，因为模型没有ActualDrawers字段
		actualDrawers := len(winners)
		log.Printf("准备更新抽奖活动信息，参与人数: %d, 实际中奖人数: %d",
			lotteryDraw.ParticipantsCount, actualDrawers)

		if err := db.DB.Save(&lotteryDraw).Error; err != nil {
			log.Printf("保存抽奖活动信息失败: %v", err)
		} else {
			log.Printf("抽奖活动信息保存成功，已包含中奖者列表")
		}

		// 10. 额外验证：从保存的 WinnersList 中提取手机号，确保中奖状态已正确设置
		log.Printf("开始额外验证中奖状态，从 WinnersList 中提取手机号")
		if lotteryDraw.WinnersList != "" {
			var winnersFromList []gin.H
			if err := json.Unmarshal([]byte(lotteryDraw.WinnersList), &winnersFromList); err != nil {
				log.Printf("解析 WinnersList 失败: %v", err)
			} else {
				log.Printf("从 WinnersList 中解析出 %d 个获奖者信息", len(winnersFromList))
				for _, winner := range winnersFromList {
					// 从 winner 中获取 mobile
					mobileInterface, exists := winner["mobile"]
					if !exists {
						log.Printf("获奖者信息中缺少 mobile 字段: %+v", winner)
						continue
					}
					// 将 mobileInterface 转换为 int 类型
					mobile, ok := mobileInterface.(float64)
					if !ok {
						log.Printf("mobile 字段类型错误: %v", mobileInterface)
						continue
					}
					mobileInt := int(mobile)
					log.Printf("验证手机号 %d 的中奖状态", mobileInt)

					// 使用手机号查询 SnowSuccessUser
					var successUser models.SnowSuccessUser
					if err := db.DB.Where("mobile = ?", mobileInt).First(&successUser).Error; err != nil {
						log.Printf("通过手机号=%d查询SnowSuccessUser失败: %v", mobileInt, err)
						continue
					}

					// 获取当前中奖状态
					winningStatus, err := successUser.GetWinningStatus()
					if err != nil {
						log.Printf("用户 %d 获取当前中奖状态失败: %v，使用新map", successUser.UserID, err)
						winningStatus = make(map[string]bool)
					}

					// 确保当前波次的中奖状态为 true
					if !winningStatus[drawBatchStr] {
						log.Printf("发现用户 %d 手机号 %d 在波次 %s 的中奖状态未设置为 true，现在设置", successUser.UserID, mobileInt, drawBatchStr)
						winningStatus[drawBatchStr] = true
						if err := successUser.SetWinningStatus(winningStatus); err != nil {
							log.Printf("设置用户 %d 中奖状态失败: %v", successUser.UserID, err)
							continue
						}

						// 同时设置中奖时间
						drawSuccessTime, err := successUser.GetDrawSuccessTime()
						if err != nil {
							log.Printf("用户 %d 获取当前中奖时间失败: %v，使用新map", successUser.UserID, err)
							drawSuccessTime = make(map[string]string)
						}
						drawSuccessTime[drawBatchStr] = time.Now().Format("2006-01-02 15:04:05")
						if err := successUser.SetDrawSuccessTime(drawSuccessTime); err != nil {
							log.Printf("设置用户 %d 中奖时间失败: %v", successUser.UserID, err)
							continue
						}

						// 保存更新后的中奖状态和时间
						if err := db.DB.Save(&successUser).Error; err != nil {
							log.Printf("保存用户 %d 中奖状态和时间失败: %v", successUser.UserID, err)
						} else {
							log.Printf("用户 %d 手机号 %d 在波次 %s 的中奖状态和时间已成功设置", successUser.UserID, mobileInt, drawBatchStr)
						}
					} else {
						log.Printf("用户 %d 手机号 %d 在波次 %s 的中奖状态已正确设置为 true", successUser.UserID, mobileInt, drawBatchStr)
					}
				}
			}
		}
		log.Printf("额外验证中奖状态完成")
	}

	// 9. 返回抽奖结果
	log.Printf("===================== 抽奖操作完成 =====================")
	log.Printf("抽奖结果摘要: 轮次=%d, 抽奖名称=%s, 参与人数=%d, 中奖人数=%d, 奖品名称=%s",
		request.DrawBatch, lotteryDraw.DrawName, len(eligibleUsers), len(winners), prizeName)
	// 确保响应中包含详细的奖品信息
	response := gin.H{
		"success":            true,
		"message":            "抽奖完成",
		"draw_batch":         request.DrawBatch,
		"draw_name":          lotteryDraw.DrawName,
		"prize_name":         prizeName,
		"total_drawers":      len(winners),
		"actual_drawers":     len(winners),
		"participants_count": len(eligibleUsers),
		"winners":            winnersInfo,
		// 添加原始奖品数据用于调试
		"original_prizes_data": lotteryDraw.Prizes,
	}
	log.Printf("返回抽奖结果: %+v", response)
	c.JSON(http.StatusOK, response)
}

// ImportWinners 导入中奖名单
func (sfc *SnowFunctionController) ImportWinners(c *gin.Context) {
	log.Printf("===================== 开始执行导入中奖名单操作 =====================")

	// 1. 获取抽奖轮次参数（从表单中获取）
	drawBatchStr := c.PostForm("draw_batch")
	if drawBatchStr == "" {
		// 尝试从URL参数中获取
		drawBatchStr = c.Query("draw_batch")
	}

	if drawBatchStr == "" {
		log.Printf("抽奖轮次参数缺失")
		c.JSON(http.StatusBadRequest, gin.H{"error": "抽奖轮次参数缺失"})
		return
	}

	// 转换为整数
	drawBatch, err := strconv.Atoi(drawBatchStr)
	if err != nil {
		log.Printf("抽奖轮次参数格式错误: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "抽奖轮次参数格式错误: " + err.Error()})
		return
	}
	log.Printf("导入中奖名单请求参数获取成功，抽奖轮次: %d", drawBatch)

	// 2. 获取上传的Excel文件
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		log.Printf("获取上传文件失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "获取上传文件失败: " + err.Error()})
		return
	}
	defer file.Close()
	log.Printf("获取上传文件成功: %s, 大小: %d", header.Filename, header.Size)

	// 3. 打开Excel文件
	f, err := excelize.OpenReader(file)
	if err != nil {
		log.Printf("打开Excel文件失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "打开Excel文件失败: " + err.Error()})
		return
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Printf("关闭Excel文件失败: %v", err)
		}
	}()

	// 4. 获取第一个工作表
	sheetName := f.GetSheetName(0)
	if sheetName == "" {
		log.Printf("Excel文件中没有工作表")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Excel文件中没有工作表"})
		return
	}
	log.Printf("使用工作表: %s", sheetName)

	// 5. 解析Excel数据
	rows, err := f.GetRows(sheetName)
	if err != nil {
		log.Printf("获取Excel行数据失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取Excel行数据失败: " + err.Error()})
		return
	}

	if len(rows) <= 1 {
		log.Printf("Excel文件中没有数据行")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Excel文件中没有数据行"})
		return
	}

	// 6. 处理每一行数据
	drawBatchStr = strconv.Itoa(drawBatch)
	var winnersInfo []gin.H
	var usersToUpdate []models.SnowSuccessUser

	// 根据抽奖波次获取抽奖信息
	var lotteryDraw models.SnowLotteryDraw
	drawResult := db.DB.Where("draw_batch = ?", drawBatch).First(&lotteryDraw)
	if drawResult.Error != nil {
		log.Printf("抽奖波次不存在: %v", drawResult.Error)
		c.JSON(http.StatusNotFound, gin.H{"error": "抽奖波次不存在"})
		return
	}

	// 解析奖品信息
	var prizeName string
	if lotteryDraw.Prizes != "" && lotteryDraw.Prizes != "{}" {
		var prizesMap map[string]int
		if json.Unmarshal([]byte(lotteryDraw.Prizes), &prizesMap) == nil {
			for name := range prizesMap {
				prizeName = name
				break
			}
		} else {
			prizePairs := strings.Split(lotteryDraw.Prizes, ",")
			if len(prizePairs) > 0 {
				pair := strings.TrimSpace(prizePairs[0])
				if pair != "" {
					parts := strings.SplitN(pair, ":", 2)
					if len(parts) > 0 {
						prizeName = strings.TrimSpace(parts[0])
					}
				}
			}
		}
	}

	// 处理黄金雪花片奖品特殊情况
	if prizeName == "" && lotteryDraw.Prizes == `{"黄金雪花片": 15}` {
		prizeName = "黄金雪花片"
	}

	for i, row := range rows {
		// 跳过表头行
		if i == 0 {
			continue
		}

		// 检查必填字段
		if len(row) < 2 {
			log.Printf("第 %d 行缺少必填字段", i+1)
			continue
		}

		// 获取手机号（B列）
		mobileStr := strings.TrimSpace(row[1])
		if mobileStr == "" {
			log.Printf("第 %d 行手机号为空", i+1)
			continue
		}

		// 转换手机号为整数
		mobile, err := strconv.Atoi(mobileStr)
		if err != nil {
			log.Printf("第 %d 行手机号格式错误: %v", i+1, err)
			continue
		}

		// 7. 根据手机号查询SnowUser
		var snowUser models.SnowUser
		if err := db.DB.Where("mobile = ?", mobile).First(&snowUser).Error; err != nil {
			log.Printf("通过手机号=%d查询SnowUser失败: %v", mobile, err)
			continue
		}
		log.Printf("成功查询到手机号=%d的用户ID: %d", mobile, snowUser.UserID)

		// 8. 查询SnowSuccessUser
		var successUser models.SnowSuccessUser
		if err := db.DB.Where("mobile = ?", mobile).First(&successUser).Error; err != nil {
			log.Printf("通过手机号=%d查询SnowSuccessUser失败: %v", mobile, err)
			continue
		}
		log.Printf("成功查询到手机号=%d的详细用户信息: %+v", mobile, successUser)

		// 9. 从SnowSuccessUser中获取抽奖码
		var drawCode string

		// 从success_code中获取对应波次的抽奖码
		successCodeMap, err := successUser.GetSuccessCode()
		if err != nil {
			log.Printf("解析SnowSuccessUser.SuccessCode失败: %v", err)
		} else {
			if code, exists := successCodeMap[drawBatchStr]; exists {
				drawCode = code
				log.Printf("从SnowSuccessUser.SuccessCode获取到抽奖码: %s", drawCode)
			} else {
				log.Printf("SnowSuccessUser.SuccessCode中没有找到当前波次的抽奖码，波次: %s", drawBatchStr)
			}
		}

		// 10. 获取订单号
		orderNums, err := successUser.GetOrderNUM()
		var orderNum string
		if err != nil {
			log.Printf("用户 %d 获取订单号失败: %v", successUser.UserID, err)
		} else if num, exists := orderNums[drawBatchStr]; exists {
			orderNum = num
			log.Printf("用户 %d 获取订单号成功: %s", successUser.UserID, orderNum)
		}

		// 11. 更新中奖者的中奖状态
		log.Printf("开始更新用户 %d 的中奖状态", successUser.UserID)
		winningStatus, err := successUser.GetWinningStatus()
		if err != nil {
			log.Printf("用户 %d 获取当前中奖状态失败: %v，使用新map", successUser.UserID, err)
			winningStatus = make(map[string]bool)
		}
		winningStatus[drawBatchStr] = true
		if err := successUser.SetWinningStatus(winningStatus); err != nil {
			log.Printf("设置用户 %d 中奖状态失败: %v", successUser.UserID, err)
			continue
		}

		// 12. 更新中奖时间
		log.Printf("开始更新用户 %d 的中奖时间", successUser.UserID)
		drawSuccessTime, err := successUser.GetDrawSuccessTime()
		if err != nil {
			log.Printf("用户 %d 获取当前中奖时间失败: %v，使用新map", successUser.UserID, err)
			drawSuccessTime = make(map[string]string)
		}
		// 设置当前波次的中奖时间
		drawSuccessTime[drawBatchStr] = time.Now().Format("2006-01-02 15:04:05")
		if err := successUser.SetDrawSuccessTime(drawSuccessTime); err != nil {
			log.Printf("设置用户 %d 中奖时间失败: %v", successUser.UserID, err)
			continue
		}

		// 将需要更新的用户添加到列表
		usersToUpdate = append(usersToUpdate, successUser)

		// 构建符合要求格式的中奖者信息
		winnerInfo := gin.H{
			"draw_code":  drawCode,
			"mobile":     successUser.Mobile,
			"nickname":   successUser.Nickname,
			"order_num":  orderNum,
			"prize_name": prizeName,
			"draw_time":  time.Now().Format("2006-01-02 15:04:05"),
		}
		winnersInfo = append(winnersInfo, winnerInfo)
		log.Printf("获奖者 #%d 信息构建完成: %+v", i, winnerInfo)
	}

	// 13. 更新抽奖结果到数据库
	// 更新中奖者的中奖状态
	saveSuccessCount := 0
	saveFailedCount := 0
	for _, user := range usersToUpdate {
		if err := db.DB.Save(&user).Error; err != nil {
			log.Printf("保存用户 %d 中奖状态失败: %v", user.UserID, err)
			saveFailedCount++
		} else {
			log.Printf("用户 %d 中奖状态保存成功", user.UserID)
			saveSuccessCount++
		}
	}
	log.Printf("中奖状态更新完成：成功 %d 个，失败 %d 个", saveSuccessCount, saveFailedCount)

	// 14. 更新抽奖活动的中奖者列表
	if len(winnersInfo) > 0 {
		winnersJSON, err := json.Marshal(winnersInfo)
		if err != nil {
			log.Printf("序列化中奖者信息失败: %v", err)
		} else {
			lotteryDraw.WinnersList = string(winnersJSON)
			log.Printf("成功设置WinnersList: %s", lotteryDraw.WinnersList)

			// 保存抽奖活动信息
			if err := db.DB.Save(&lotteryDraw).Error; err != nil {
				log.Printf("保存抽奖活动信息失败: %v", err)
			} else {
				log.Printf("抽奖活动信息保存成功，已包含中奖者列表")
			}
		}
	}

	// 15. 返回结果
	log.Printf("===================== 导入中奖名单操作完成 =====================")
	response := gin.H{
		"success":        true,
		"message":        "导入完成",
		"draw_batch":     drawBatch,
		"imported_count": len(winnersInfo),
		"save_success":   saveSuccessCount,
		"save_failed":    saveFailedCount,
		"winners":        winnersInfo,
	}
	log.Printf("返回导入结果: %+v", response)
	c.JSON(http.StatusOK, response)
}

// UploadTempImage 上传临时图片到 snow_temp 目录
func (sfc *SnowFunctionController) UploadTempImage(c *gin.Context) {
	log.Printf("===================== 开始执行图片上传操作 =====================")

	// 检查是否上传了文件
	fileHeader, err := c.FormFile("image")
	if err != nil {
		log.Printf("未检测到上传的文件: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择要上传的图片文件"})
		return
	}

	log.Printf("检测到上传文件: %s, 大小: %d bytes, MIME类型: %s",
		fileHeader.Filename, fileHeader.Size, fileHeader.Header.Get("Content-Type"))

	// 验证文件类型
	allowedTypes := map[string]bool{
		"image/jpeg": true,
		"image/jpg":  true,
		"image/png":  true,
		"image/gif":  true,
		"image/webp": true,
	}

	fileType := fileHeader.Header.Get("Content-Type")
	if !allowedTypes[fileType] {
		log.Printf("不支持的文件类型: %s", fileType)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "不支持的文件类型，仅支持 JPG、PNG、GIF、WebP 格式的图片",
		})
		return
	}

	// 验证文件大小 (最大 5MB)
	const maxFileSize = 5 * 1024 * 1024 // 5MB
	if fileHeader.Size > maxFileSize {
		log.Printf("文件大小超限: %d bytes, 最大允许: %d bytes", fileHeader.Size, maxFileSize)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "文件大小超限，请上传小于 5MB 的图片",
		})
		return
	}

	// 获取当前工作目录
	currentDir, err := os.Getwd()
	if err != nil {
		log.Printf("获取工作目录失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取工作目录失败"})
		return
	}

	// 创建 snow_temp 目录
	snowTempDir := filepath.Join(currentDir, "media", "snow_temp")
	if err := os.MkdirAll(snowTempDir, 0755); err != nil {
		log.Printf("创建snow_temp目录失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建上传目录失败"})
		return
	}

	// 生成唯一文件名（统一转换为JPG格式）
	timestamp := time.Now().Unix()
	randomNum := timestamp % 10000
	filename := fmt.Sprintf("temp_%d_%d.jpg", timestamp, randomNum)

	tempSavePath := filepath.Join(snowTempDir, "temp_"+filename)
	jpgSavePath := filepath.Join(snowTempDir, filename)
	log.Printf("准备保存文件到: %s", tempSavePath)

	// 首先保存上传的原始文件
	if err := c.SaveUploadedFile(fileHeader, tempSavePath); err != nil {
		log.Printf("保存临时文件失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "文件保存失败: " + err.Error()})
		return
	}

	// 验证文件是否成功保存
	if _, err := os.Stat(tempSavePath); os.IsNotExist(err) {
		log.Printf("文件保存后验证失败: 文件不存在")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "文件保存验证失败"})
		return
	}

	log.Printf("原始文件保存成功: %s, 大小: %d bytes", tempSavePath, fileHeader.Size)

	// 将图片转换为JPG格式
	if err := sfc.convertImageToJPG(tempSavePath, jpgSavePath, fileType); err != nil {
		log.Printf("图片格式转换失败: %v", err)
		// 删除临时文件
		os.Remove(tempSavePath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "图片格式转换失败: " + err.Error()})
		return
	}

	// 删除临时文件
	if err := os.Remove(tempSavePath); err != nil {
		log.Printf("删除临时文件失败: %v", err)
		// 不影响主流程，继续执行
	}

	log.Printf("文件格式转换成功: %s", filename)

	// 构建返回的URL
	// 获取请求协议和域名（优先检查反向代理头）
	proto := "http"

	// 检查反向代理头信息（X-Forwarded-Proto）
	if forwardedProto := c.Request.Header.Get("X-Forwarded-Proto"); forwardedProto != "" {
		proto = forwardedProto
	} else if forwardedProto := c.Request.Header.Get("X-Forwarded-Ssl"); forwardedProto == "on" {
		// 备选方案：检查X-Forwarded-Ssl头
		proto = "https"
	} else if c.Request.TLS != nil {
		// 直接连接HTTPS
		proto = "https"
	}

	baseURL := fmt.Sprintf("%s://%s", proto, c.Request.Host)
	imageURL := fmt.Sprintf("%s/media/snow_temp/%s", baseURL, filename)

	log.Printf("协议检测: 原始URL=%s, 检测到的协议=%s", c.Request.URL.String(), proto)

	log.Printf("图片上传完成，返回URL: %s", imageURL)

	// 返回成功响应
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "图片上传成功",
		"data": gin.H{
			"filename":      filename,
			"image_url":     imageURL,
			"relative_path": fmt.Sprintf("snow_temp/%s", filename),
			"file_size":     fileHeader.Size,
			"upload_time":   time.Now().Format("2006-01-02 15:04:05"),
		},
	})
}
