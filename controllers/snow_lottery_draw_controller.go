package controllers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"django_to_go/db"
	"django_to_go/models"

	"github.com/gin-gonic/gin"
)

// SnowLotteryDrawController 抽奖活动控制器
type SnowLotteryDrawController struct{}

// CreateDrawRequest 创建抽奖请求结构体
type CreateDrawRequest struct {
	DrawBatch      int            `json:"draw_batch" binding:"required"`
	Prizes         map[string]int `json:"prizes" binding:"required"`
	TotalDrawers   int            `json:"total_drawers" binding:"required"`
	OrderBeginTime string         `json:"order_begin_time" binding:"required"`
	OrderEndTime   string         `json:"order_end_time" binding:"required"`
	DrawTime       string         `json:"draw_time" binding:"required"`
	DrawName       string         `json:"draw_name" binding:"required"`
	Remark         string         `json:"remark"`
}

// GetAllDrawInfoRequest 获取所有抽奖信息请求结构体
type GetAllDrawInfoRequest struct {
	UserID int `json:"user_id"`
}

// UpdateDrawRequest 更新抽奖请求结构体
type UpdateDrawRequest struct {
	ID                int            `json:"id" binding:"required"`
	DrawBatch         int            `json:"draw_batch"`
	Prizes            map[string]int `json:"prizes"`
	TotalDrawers      int            `json:"total_drawers"`
	OrderBeginTime    string         `json:"order_begin_time"`
	OrderEndTime      string         `json:"order_end_time"`
	DrawTime          string         `json:"draw_time"`
	DrawName          string         `json:"draw_name"`
	Remark            string         `json:"remark"`
	WinnersList       string         `json:"winners_list"`
	ParticipantsCount int            `json:"participants_count"`
}

// GetWinnersRequest 获取中奖名单请求结构体
type GetWinnersRequest struct {
	DrawBatch int `json:"draw_batch" binding:"required"`
}

// AddWinnersRequest 添加中奖名单请求结构体
type AddWinnersRequest struct {
	DrawBatch int            `json:"draw_batch" binding:"required"`
	OrderNUM  string         `json:"order_num" binding:"required"`
	Prizes    map[string]int `json:"prizes" binding:"required"` // 奖品名称和数量的映射
}

// GetAllWinnersRequest 获取所有中奖名单请求结构体
type GetAllWinnersRequest struct {
	DrawBatch *int `json:"draw_batch"` // 可选参数，不传则返回所有轮次
}

// GetUserWinnersRequest 根据用户ID获取中奖名单请求结构体
type GetUserWinnersRequest struct {
	UserID    *int `json:"user_id"`    // 用户ID，可选参数，为空时返回所有中奖者但只包含基础信息
	DrawBatch *int `json:"draw_batch"` // 可选参数，不传则返回所有轮次
}

// CreateDraw 新增抽奖活动
func (sdc *SnowLotteryDrawController) CreateDraw(c *gin.Context) {
	var request CreateDrawRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	// 解析订单开始时间
	beginTime, err := time.Parse("2006-01-02 15:04:05", request.OrderBeginTime)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "订单开始时间格式错误，应为YYYY-MM-DD HH:MM:SS"})
		return
	}

	// 解析订单结束时间
	endTime, err := time.Parse("2006-01-02 15:04:05", request.OrderEndTime)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "订单结束时间格式错误，应为YYYY-MM-DD HH:MM:SS"})
		return
	}

	// 解析开奖时间
	drawTime, err := time.Parse("2006-01-02 15:04:05", request.DrawTime)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "开奖时间格式错误，应为YYYY-MM-DD HH:MM:SS"})
		return
	}

	// 将prizes map转换为JSON字符串
	prizesJSON, err := json.Marshal(request.Prizes)
	if err != nil {
		log.Printf("序列化prizes失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建抽奖活动失败: 奖品数据格式错误"})
		return
	}

	// 创建抽奖活动记录
	draw := models.SnowLotteryDraw{
		DrawBatch:         request.DrawBatch,
		Prizes:            string(prizesJSON),
		TotalDrawers:      request.TotalDrawers,
		OrderBeginTime:    beginTime,
		OrderEndTime:      endTime,
		DrawTime:          drawTime,
		DrawName:          request.DrawName,
		Remark:            request.Remark,
		ParticipantsCount: 0,    // 初始参与人数为0
		ParticipantsList:  "[]", // 初始参与名单为空JSON数组
		WinnersList:       "",   // 初始中奖名单为空
	}

	// 保存到数据库
	if err := db.DB.Create(&draw).Error; err != nil {
		log.Printf("创建抽奖活动失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建抽奖活动失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "抽奖活动创建成功", "data": draw})
}

// UpdateDraw 修改抽奖活动信息
func (sdc *SnowLotteryDrawController) UpdateDraw(c *gin.Context) {
	var request UpdateDrawRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	// 查询抽奖活动是否存在
	var draw models.SnowLotteryDraw
	if err := db.DB.Where("id = ?", request.ID).First(&draw).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "抽奖活动不存在"})
		return
	}

	// 更新抽奖活动信息
	if request.DrawBatch > 0 {
		draw.DrawBatch = request.DrawBatch
	}
	if len(request.Prizes) > 0 {
		// 将prizes map转换为JSON字符串
		prizesJSON, err := json.Marshal(request.Prizes)
		if err != nil {
			log.Printf("序列化prizes失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "更新抽奖活动失败: 奖品数据格式错误"})
			return
		}
		draw.Prizes = string(prizesJSON)
	}
	if request.TotalDrawers > 0 {
		draw.TotalDrawers = request.TotalDrawers
	}
	if request.OrderBeginTime != "" {
		beginTime, err := time.Parse("2006-01-02 15:04:05", request.OrderBeginTime)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "订单开始时间格式错误，应为YYYY-MM-DD HH:MM:SS"})
			return
		}
		draw.OrderBeginTime = beginTime
	}
	if request.OrderEndTime != "" {
		endTime, err := time.Parse("2006-01-02 15:04:05", request.OrderEndTime)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "订单结束时间格式错误，应为YYYY-MM-DD HH:MM:SS"})
			return
		}
		draw.OrderEndTime = endTime
	}
	if request.DrawTime != "" {
		drawTime, err := time.Parse("2006-01-02 15:04:05", request.DrawTime)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "开奖时间格式错误，应为YYYY-MM-DD HH:MM:SS"})
			return
		}
		draw.DrawTime = drawTime
	}
	if request.DrawName != "" {
		draw.DrawName = request.DrawName
	}
	if request.Remark != "" {
		draw.Remark = request.Remark
	}
	if request.WinnersList != "" {
		draw.WinnersList = request.WinnersList
	}
	if request.ParticipantsCount > 0 {
		draw.ParticipantsCount = request.ParticipantsCount
	}

	// 显式检查并确保participants_list字段始终是有效的JSON格式
	if draw.ParticipantsList == "" {
		draw.ParticipantsList = "[]"
	}
	if draw.Prizes == "" {
		draw.Prizes = "{}"
	}

	// 保存更新
	if err := db.DB.Save(&draw).Error; err != nil {
		log.Printf("更新抽奖活动失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新抽奖活动失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "抽奖活动更新成功", "data": draw})
}

// GetWinners 获取中奖名单
func (sdc *SnowLotteryDrawController) GetWinners(c *gin.Context) {
	var request GetWinnersRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	// 查询抽奖活动
	var draw models.SnowLotteryDraw
	if err := db.DB.Where("draw_batch = ?", request.DrawBatch).First(&draw).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "抽奖活动不存在"})
		return
	}

	// 解析prizes JSON字符串为map
	var prizes map[string]int
	if draw.Prizes != "" && draw.Prizes != "{}" {
		if err := json.Unmarshal([]byte(draw.Prizes), &prizes); err != nil {
			log.Printf("解析prizes失败: %v", err)
			// 解析失败时使用空map
			prizes = make(map[string]int)
		}
	} else {
		prizes = make(map[string]int)
	}

	// 准备返回数据
	responseData := gin.H{
		"draw_batch":         draw.DrawBatch,
		"draw_name":          draw.DrawName,
		"total_drawers":      draw.TotalDrawers,
		"participants_count": draw.ParticipantsCount,
		"winners_list":       draw.WinnersList,
		"prizes":             prizes, // 返回解析后的prizes map
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "获取中奖名单成功", "data": responseData})
}

// AddWinners 添加中奖名单
// AddWinnersRequest 简化后的添加中奖者请求结构

func (sdc *SnowLotteryDrawController) AddWinners(c *gin.Context) {
	var request AddWinnersRequest
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

	// 根据抽奖轮次和订单号查询用户信息

	// 从数据库查询用户信息，这里需要先查询所有用户，然后根据OrderNUM字段过滤
	// 由于OrderNUM是JSON字段，我们需要先获取所有用户，然后在应用层过滤
	var users []models.SnowSuccessUser
	if err := db.DB.Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询用户信息失败"})
		return
	}

	var foundUser *models.SnowSuccessUser
	drawBatchStr := fmt.Sprintf("%d", request.DrawBatch)

	// 在应用层过滤出匹配的用户
	for i := range users {
		userOrderMap, err := users[i].GetOrderNUM()
		if err != nil {
			continue
		}
		if orderNum, exists := userOrderMap[drawBatchStr]; exists && orderNum == request.OrderNUM {
			foundUser = &users[i]
			break
		}
	}

	if foundUser == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到匹配的用户信息"})
		return
	}

	// 检查该手机号是否已经在中奖名单中
	if draw.WinnersList != "" {
		log.Printf("检查手机号重复: WinnersList=%s, 要添加的手机号=%d", draw.WinnersList, foundUser.Mobile)

		// 尝试修复WinnersList格式，使其成为有效的JSON数组
		formattedWinnersList := "[" + draw.WinnersList + "]"
		log.Printf("格式化后的JSON: %s", formattedWinnersList)

		// 尝试解析为JSON数组
		var winners []map[string]interface{}
		err := json.Unmarshal([]byte(formattedWinnersList), &winners)
		if err != nil {
			log.Printf("解析为JSON数组失败: %v，尝试其他方式", err)
			// 如果解析失败，尝试直接查找手机号字符串
			// 构建要查找的手机号字符串形式
			mobileStr := fmt.Sprintf("%d", foundUser.Mobile)
			// 检查字符串中是否包含该手机号
			if strings.Contains(draw.WinnersList, fmt.Sprintf("\"mobile\":%s", mobileStr)) ||
				strings.Contains(draw.WinnersList, fmt.Sprintf("\"mobile\":\"%s\"", mobileStr)) {
				log.Printf("通过字符串匹配发现重复手机号: %s", mobileStr)
				c.JSON(http.StatusConflict, gin.H{"error": "该手机号已经在中奖名单中"})
				return
			}
		} else {
			// 成功解析为数组，遍历检查
			log.Printf("成功解析为JSON数组，共有%d个中奖者", len(winners))
			for i, winner := range winners {
				log.Printf("处理第%d个中奖者", i)
				if mobileVal, ok := winner["mobile"]; ok {
					var mobileInt int
					switch v := mobileVal.(type) {
					case string:
						fmt.Sscanf(v, "%d", &mobileInt)
					case float64:
						mobileInt = int(v)
					case int:
						mobileInt = v
					default:
						log.Printf("未知的手机号类型: %T, 值: %v", v, v)
					}
					log.Printf("解析出的手机号: %d", mobileInt)
					if mobileInt == foundUser.Mobile {
						log.Printf("发现重复手机号: %d", mobileInt)
						c.JSON(http.StatusConflict, gin.H{"error": "该手机号已经在中奖名单中"})
						return
					}
				}
			}
		}
	}

	// 构建中奖者信息，确保包含订单号和对应的奖品以及数量
	winnerData := gin.H{
		"mobile":    foundUser.Mobile,
		"nickname":  foundUser.Nickname,
		"order_num": request.OrderNUM,
		"prizes":    request.Prizes, // 包含奖品名称和数量的映射
	}

	// 将中奖者信息转换为JSON字符串
	winnerJSON, err := json.Marshal(winnerData)
	if err != nil {
		log.Printf("序列化中奖者信息失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "添加中奖名单失败"})
		return
	}

	// 获取用户的抽奖码
	drawBatchStr = fmt.Sprintf("%d", request.DrawBatch)
	var drawCode string
	// 从SnowUser中获取抽奖码
	var snowUser models.SnowUser
	if dbErr := db.DB.Where("mobile = ?", foundUser.Mobile).First(&snowUser).Error; dbErr == nil {
		// 解析抽奖码
		var successCodeMap map[string]string
		if snowUser.SuccessCode != "" {
			if err := json.Unmarshal([]byte(snowUser.SuccessCode), &successCodeMap); err == nil {
				if code, exists := successCodeMap[drawBatchStr]; exists {
					drawCode = code
					log.Printf("获取到用户抽奖码: %s", drawCode)
				}
			}
		}
	}

	// 将抽奖码添加到中奖者信息中
	if drawCode != "" {
		winnerData["draw_code"] = drawCode
		// 重新序列化中奖者信息
		winnerJSON, err = json.Marshal(winnerData)
		if err != nil {
			log.Printf("重新序列化中奖者信息失败: %v", err)
		}
	}

	// 更新中奖名单
	// 如果原中奖名单不为空，则添加逗号分隔
	log.Printf("添加新中奖者前的WinnersList: %s", draw.WinnersList)
	log.Printf("要添加的中奖者JSON: %s", string(winnerJSON))
	if draw.WinnersList != "" {
		draw.WinnersList += "," + string(winnerJSON)
	} else {
		draw.WinnersList = string(winnerJSON)
	}
	log.Printf("添加新中奖者后的WinnersList: %s", draw.WinnersList)

	// 保存更新
	if err := db.DB.Save(&draw).Error; err != nil {
		log.Printf("更新中奖名单失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "添加中奖名单失败"})
		return
	}

	// 更新成功用户状态
	// 1. 更新用户的参与状态为已中奖
	var participationStatus map[string]bool
	if foundUser.ParticipationStatus != "" {
		if err := json.Unmarshal([]byte(foundUser.ParticipationStatus), &participationStatus); err != nil {
			participationStatus = make(map[string]bool)
		}
	} else {
		participationStatus = make(map[string]bool)
	}
	participationStatus[drawBatchStr] = true

	// 2. 更新用户的中奖状态
	winningStatus, err := foundUser.GetWinningStatus()
	if err != nil {
		winningStatus = make(map[string]bool)
	}
	winningStatus[drawBatchStr] = true
	if err := foundUser.SetWinningStatus(winningStatus); err != nil {
		log.Printf("设置中奖状态失败: %v", err)
	}

	// 3. 如果有抽奖码，也更新到SnowSuccessUser
	if drawCode != "" {
		successCodeMap, err := foundUser.GetSuccessCode()
		if err != nil {
			successCodeMap = make(map[string]string)
		}
		successCodeMap[drawBatchStr] = drawCode
		if err := foundUser.SetSuccessCode(successCodeMap); err != nil {
			log.Printf("设置抽奖码失败: %v", err)
		}
	}

	// 保存所有更新
	statusJSON, err := json.Marshal(participationStatus)
	if err == nil {
		foundUser.ParticipationStatus = string(statusJSON)
		db.DB.Save(foundUser)
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "添加中奖名单成功"})
}

// GetUserWinners 根据用户ID获取中奖名单（根据用户ID决定返回信息详细程度）
func (sdc *SnowLotteryDrawController) GetUserWinners(c *gin.Context) {
	log.Printf("开始查询中奖名单")
	var request GetUserWinnersRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		log.Printf("请求参数错误: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	// 检查是否提供了用户ID
	hasUserID := request.UserID != nil && *request.UserID > 0
	if hasUserID {
		log.Printf("根据用户ID查询: %d", *request.UserID)
	} else {
		log.Printf("未提供用户ID，将返回所有中奖者的基础信息")
	}

	// 构建查询条件
	query := db.DB.Model(&models.SnowLotteryDraw{})
	if request.DrawBatch != nil {
		log.Printf("按抽奖轮次查询: %d", *request.DrawBatch)
		query = query.Where("draw_batch = ?", *request.DrawBatch)
	} else {
		log.Printf("查询所有抽奖轮次")
	}

	// 执行查询
	var draws []models.SnowLotteryDraw
	if err := query.Find(&draws).Error; err != nil {
		log.Printf("查询抽奖活动失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询中奖名单失败"})
		return
	}
	log.Printf("查询到的抽奖轮次数量: %d", len(draws))

	// 检查是否所有抽奖轮次都已到了可查询时间
	currentTime := time.Now()
	log.Printf("当前时间: %v", currentTime)

	var allWinners []gin.H

	for _, draw := range draws {
		log.Printf("抽奖轮次: draw_batch=%d, OrderEndTime=%v, WinnersList长度=%d",
			draw.DrawBatch, draw.OrderEndTime, len(draw.WinnersList))

		// 检查order_end_time时间
		if draw.OrderEndTime.After(currentTime) {
			log.Printf("抽奖轮次%d未到查询时间", draw.DrawBatch)
			continue
		}

		// 解析winners_list
		var winners []gin.H
		if draw.WinnersList != "" {
			log.Printf("开始解析winners_list，抽奖轮次: %d", draw.DrawBatch)

			// 预处理winners_list，处理非标准JSON格式
			processedWinnersList := draw.WinnersList
			// 检查是否是对象直接用逗号连接的格式，需要转换为标准JSON数组
			if !strings.HasPrefix(processedWinnersList, "[") && strings.Contains(processedWinnersList, "},{") {
				// 转换为标准JSON数组格式
				processedWinnersList = "[" + processedWinnersList + "]"
			}

			// 尝试解析为单个对象
			var winnerObj map[string]interface{}
			if err := json.Unmarshal([]byte(processedWinnersList), &winnerObj); err == nil {
				log.Printf("解析为单个对象成功，抽奖轮次: %d", draw.DrawBatch)
				winnerData := make(gin.H)
				for k, v := range winnerObj {
					winnerData[k] = v
				}
				// 根据是否提供用户ID决定返回信息的处理方式
				if hasUserID {
					// 提供了用户ID，根据用户ID决定返回信息的详细程度
					winners = append(winners, applyMaskingByUser(winnerData, draw.DrawBatch, *request.UserID))
				} else {
					// 未提供用户ID，只返回基础信息
					minimalData := make(gin.H)
					// 保留抽奖码
					if drawCode, ok := winnerData["draw_code"]; ok {
						minimalData["draw_code"] = drawCode
					}
					// 对手机号进行打码并保留
					if mobile, ok := winnerData["mobile"]; ok {
						var mobileStr string
						switch v := mobile.(type) {
						case string:
							mobileStr = v
						case int:
							mobileStr = fmt.Sprintf("%d", v)
						case float64:
							mobileStr = fmt.Sprintf("%.0f", v)
						default:
							mobileStr = fmt.Sprintf("%v", v)
						}
						if len(mobileStr) >= 7 {
							minimalData["mobile"] = maskMobile(mobileStr)
						}
					}
					// 保留奖品信息
					if prize_name, ok := winnerData["prize_name"]; ok {
						minimalData["prize_name"] = prize_name
					}
					if prizes, ok := winnerData["prizes"]; ok {
						minimalData["prizes"] = prizes
					}
					// 添加抽奖轮次
					minimalData["draw_batch"] = draw.DrawBatch
					winners = append(winners, minimalData)
				}
			} else {
				// 尝试解析为数组
				var parsedWinners []map[string]interface{}
				if err := json.Unmarshal([]byte(processedWinnersList), &parsedWinners); err == nil {
					// 成功解析为数组
					log.Printf("原始格式解析成功，共有%d个中奖者", len(parsedWinners))
					for _, w := range parsedWinners {
						winnerData := make(gin.H)
						for k, v := range w {
							winnerData[k] = v
						}
						// 根据是否提供用户ID决定返回信息的处理方式
						if hasUserID {
							// 提供了用户ID，根据用户ID决定返回信息的详细程度
							winners = append(winners, applyMaskingByUser(winnerData, draw.DrawBatch, *request.UserID))
						} else {
							// 未提供用户ID，只返回基础信息
							minimalData := make(gin.H)
							// 保留抽奖码
							if drawCode, ok := winnerData["draw_code"]; ok {
								minimalData["draw_code"] = drawCode
							}
							// 对手机号进行打码并保留
							if mobile, ok := winnerData["mobile"]; ok {
								var mobileStr string
								switch v := mobile.(type) {
								case string:
									mobileStr = v
								case int:
									mobileStr = fmt.Sprintf("%d", v)
								case float64:
									mobileStr = fmt.Sprintf("%.0f", v)
								default:
									mobileStr = fmt.Sprintf("%v", v)
								}
								if len(mobileStr) >= 7 {
									minimalData["mobile"] = maskMobile(mobileStr)
								}
							}
							// 保留奖品信息
							if prize_name, ok := winnerData["prize_name"]; ok {
								minimalData["prize_name"] = prize_name
							}
							if prizes, ok := winnerData["prizes"]; ok {
								minimalData["prizes"] = prizes
							}
							// 添加抽奖轮次
							minimalData["draw_batch"] = draw.DrawBatch
							winners = append(winners, minimalData)
						}
					}
				} else {
					// 尝试修复格式
					log.Printf("解析失败，尝试修复格式: %v", err)
					// 尝试添加数组括号
					fixedWinnersList := "[" + draw.WinnersList + "]"
					if err := json.Unmarshal([]byte(fixedWinnersList), &parsedWinners); err == nil {
						// 成功解析修复后的JSON
						log.Printf("修复后解析成功，共有%d个中奖者", len(parsedWinners))
						for _, w := range parsedWinners {
							winnerData := make(gin.H)
							for k, v := range w {
								winnerData[k] = v
							}
							// 根据是否提供用户ID决定返回信息的处理方式
							if hasUserID {
								// 提供了用户ID，根据用户ID决定返回信息的详细程度
								winners = append(winners, applyMaskingByUser(winnerData, draw.DrawBatch, *request.UserID))
							} else {
								// 未提供用户ID，只返回基础信息
								minimalData := make(gin.H)
								// 保留抽奖码
								if drawCode, ok := winnerData["draw_code"]; ok {
									minimalData["draw_code"] = drawCode
								}
								// 对手机号进行打码并保留
								if mobile, ok := winnerData["mobile"]; ok {
									var mobileStr string
									switch v := mobile.(type) {
									case string:
										mobileStr = v
									case int:
										mobileStr = fmt.Sprintf("%d", v)
									case float64:
										mobileStr = fmt.Sprintf("%.0f", v)
									default:
										mobileStr = fmt.Sprintf("%v", v)
									}
									if len(mobileStr) >= 7 {
										minimalData["mobile"] = maskMobile(mobileStr)
									}
								}
								// 保留奖品信息
								if prize, ok := winnerData["prize"]; ok {
									minimalData["prize"] = prize
								}
								if prizes, ok := winnerData["prizes"]; ok {
									minimalData["prizes"] = prizes
								}
								// 保留prize_name字段（优先显示）
								if prizeName, ok := winnerData["prize_name"]; ok {
									minimalData["prize_name"] = prizeName
								}
								// 添加抽奖轮次
								minimalData["draw_batch"] = draw.DrawBatch
								winners = append(winners, minimalData)
							}
						}
					} else {
						log.Printf("修复后解析仍失败: %v", err)
					}
				}
			}
			log.Printf("该轮次共处理了%d个中奖者", len(winners))
			allWinners = append(allWinners, winners...)
		} else {
			log.Printf("该轮次没有中奖名单")
		}
	}
	log.Printf("总共查询到%d个中奖者", len(allWinners))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "获取中奖名单成功",
		"data":    allWinners,
		"count":   len(allWinners),
	})
}

// GetAllWinners 获取中奖名单（支持按抽奖轮次筛选）
func (sdc *SnowLotteryDrawController) GetAllWinners(c *gin.Context) {
	log.Printf("开始查询中奖名单")
	var request GetAllWinnersRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		log.Printf("请求参数错误，使用默认请求: %v", err)
		// 如果请求参数错误，尝试将其视为没有参数的请求
		request = GetAllWinnersRequest{}
	}

	// 构建查询条件
	query := db.DB.Model(&models.SnowLotteryDraw{})
	if request.DrawBatch != nil {
		log.Printf("按抽奖轮次查询: %d", *request.DrawBatch)
		query = query.Where("draw_batch = ?", *request.DrawBatch)
	} else {
		log.Printf("查询所有抽奖轮次")
	}

	// 执行查询
	var draws []models.SnowLotteryDraw
	if err := query.Find(&draws).Error; err != nil {
		log.Printf("查询抽奖活动失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询中奖名单失败"})
		return
	}
	log.Printf("查询到的抽奖轮次数量: %d", len(draws))

	// 检查是否所有抽奖轮次都已到了可查询时间
	currentTime := time.Now()
	log.Printf("当前时间: %v", currentTime)
	for i, draw := range draws {
		log.Printf("抽奖轮次%d: draw_batch=%d, OrderEndTime=%v, WinnersList长度=%d",
			i+1, draw.DrawBatch, draw.OrderEndTime, len(draw.WinnersList))
		// 检查order_end_time时间
		if draw.OrderEndTime.After(currentTime) {
			log.Printf("抽奖轮次%d未到查询时间", draw.DrawBatch)
			// 如果是按特定轮次查询，直接返回提示
			if request.DrawBatch != nil && *request.DrawBatch == draw.DrawBatch {
				c.JSON(http.StatusForbidden, gin.H{"error": fmt.Sprintf("请在%v后查询该轮次中奖名单", draw.OrderEndTime.Format("2006-01-02 15:04:05"))})
				return
			}
			// 如果是查询所有轮次，跳过该轮次
			draws = append(draws[:i], draws[i+1:]...)
			i-- // 调整索引，因为删除了一个元素
		}
	}

	// 如果没有符合条件的抽奖轮次，返回空列表
	if len(draws) == 0 {
		log.Printf("没有符合条件的抽奖轮次")
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "暂无可查询的中奖名单",
			"data":    []gin.H{},
			"count":   0,
		})
		return
	}
	log.Printf("符合条件的抽奖轮次数量: %d", len(draws))

	// 准备返回的中奖名单数据
	var allWinners []gin.H

	// 处理每个抽奖活动的中奖名单
	for _, draw := range draws {
		log.Printf("处理抽奖轮次%d的中奖名单", draw.DrawBatch)
		if draw.WinnersList != "" {
			log.Printf("WinnersList内容: %s", draw.WinnersList)
			var winners []gin.H

			// 日志显示格式类似: {obj1},{obj2}，需要修复为有效的JSON数组
			// 修复格式：添加中括号使其成为有效的JSON数组
			formattedWinnersList := "[" + draw.WinnersList + "]"
			log.Printf("修复后的JSON: %s", formattedWinnersList)

			// 尝试解析修复后的JSON
			var parsedWinners []map[string]interface{}
			err := json.Unmarshal([]byte(formattedWinnersList), &parsedWinners)
			if err != nil {
				log.Printf("修复后解析失败: %v，尝试原始格式", err)
				// 如果修复后解析失败，尝试原始格式
				err = json.Unmarshal([]byte(draw.WinnersList), &parsedWinners)
				if err != nil {
					log.Printf("原始格式解析失败: %v，尝试分割处理", err)
					// 尝试更智能的分割处理
					// 查找所有JSON对象的开始和结束位置
					var objStart []int
					var objEnd []int
					braceCount := 0
					for i, char := range draw.WinnersList {
						if char == '{' {
							if braceCount == 0 {
								objStart = append(objStart, i)
							}
							braceCount++
						} else if char == '}' {
							braceCount--
							if braceCount == 0 {
								objEnd = append(objEnd, i)
							}
						}
					}

					// 提取每个JSON对象
					for i := 0; i < len(objStart) && i < len(objEnd); i++ {
						objStr := draw.WinnersList[objStart[i] : objEnd[i]+1]
						log.Printf("提取的JSON对象: %s", objStr)
						var tempWinner map[string]interface{}
						if err := json.Unmarshal([]byte(objStr), &tempWinner); err == nil {
							// 复制到gin.H
							winnerData := make(gin.H)
							for k, v := range tempWinner {
								winnerData[k] = v
							}
							// 添加打码处理
							maskedData := applyMasking(winnerData, draw.DrawBatch)
							winners = append(winners, maskedData)
							log.Printf("添加了一个中奖者")
						} else {
							log.Printf("解析单个对象失败: %v", err)
						}
					}
				} else {
					// 成功解析为数组
					log.Printf("原始格式解析成功，共有%d个中奖者", len(parsedWinners))
					for _, w := range parsedWinners {
						winnerData := make(gin.H)
						for k, v := range w {
							winnerData[k] = v
						}
						winners = append(winners, applyMasking(winnerData, draw.DrawBatch))
					}
				}
			} else {
				// 成功解析修复后的JSON
				log.Printf("修复后解析成功，共有%d个中奖者", len(parsedWinners))
				for _, w := range parsedWinners {
					winnerData := make(gin.H)
					for k, v := range w {
						winnerData[k] = v
					}
					winners = append(winners, applyMasking(winnerData, draw.DrawBatch))
				}
			}
			log.Printf("该轮次共处理了%d个中奖者", len(winners))
			allWinners = append(allWinners, winners...)
		} else {
			log.Printf("该轮次没有中奖名单")
		}
	}
	log.Printf("总共查询到%d个中奖者", len(allWinners))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "获取中奖名单成功",
		"data":    allWinners,
		"count":   len(allWinners),
	})
}

// applyMasking 对敏感信息进行打码处理
func applyMasking(winnerData gin.H, drawBatch int) gin.H {
	log.Printf("开始打码处理，原始数据: %+v", winnerData)
	maskedData := make(gin.H)

	// 复制所有字段
	for k, v := range winnerData {
		maskedData[k] = v
	}

	// 添加抽奖轮次
	maskedData["draw_batch"] = drawBatch
	log.Printf("添加抽奖批次: %d", drawBatch)

	// 对手机号进行打码
	if mobile, ok := winnerData["mobile"]; ok {
		log.Printf("处理手机号，原始类型: %T, 值: %v", mobile, mobile)
		var mobileStr string
		switch v := mobile.(type) {
		case string:
			log.Printf("手机号是字符串类型: %s", v)
			mobileStr = v
		case int:
			log.Printf("手机号是整数类型: %d", v)
			mobileStr = fmt.Sprintf("%d", v)
		case float64:
			log.Printf("手机号是浮点数类型: %.0f", v)
			mobileStr = fmt.Sprintf("%.0f", v)
		default:
			log.Printf("未知的手机号类型: %T, 值: %v", v, v)
			mobileStr = fmt.Sprintf("%v", v)
		}
		if len(mobileStr) >= 7 {
			maskedMobile := maskMobile(mobileStr)
			maskedData["mobile"] = maskedMobile
			log.Printf("手机号打码后: %s", maskedMobile)
		}
	}

	// 对昵称进行打码
	if nickname, ok := winnerData["nickname"]; ok {
		log.Printf("处理昵称，类型: %T, 值: %v", nickname, nickname)
		if nicknameStr, ok := nickname.(string); ok {
			maskedNickname := maskNickname(nicknameStr)
			maskedData["nickname"] = maskedNickname
			log.Printf("昵称打码后: %s", maskedNickname)
		}
	}

	// 对订单号进行打码
	if orderNum, ok := winnerData["order_num"]; ok {
		log.Printf("处理订单号，类型: %T, 值: %v", orderNum, orderNum)
		if orderNumStr, ok := orderNum.(string); ok {
			maskedOrderNum := maskOrderNum(orderNumStr)
			maskedData["order_num"] = maskedOrderNum
			log.Printf("订单号打码后: %s", maskedOrderNum)
		}
	}

	log.Printf("打码处理完成，结果: %+v", maskedData)
	return maskedData
}

// maskMobile 对手机号进行打码
func maskMobile(mobile string) string {
	if len(mobile) < 7 {
		return mobile
	}
	return mobile[:3] + "****" + mobile[len(mobile)-4:]
}

// maskNickname 对昵称进行打码
func maskNickname(nickname string) string {
	if len(nickname) <= 1 {
		return "*"
	}
	if len(nickname) == 2 {
		return string(nickname[0]) + "*"
	}
	// 对3个字符以上的昵称，保留首尾字符，中间用*替代
	return string(nickname[0]) + strings.Repeat("*", len(nickname)-2) + string(nickname[len(nickname)-1])
}

// maskOrderNum 对订单号进行打码
func maskOrderNum(orderNum string) string {
	if len(orderNum) <= 4 {
		return strings.Repeat("*", len(orderNum))
	}
	// 保留首尾各2个字符，中间用*替代
	return orderNum[:2] + strings.Repeat("*", len(orderNum)-4) + orderNum[len(orderNum)-2:]
}

// applyMaskingByUser 根据用户ID决定返回信息的详细程度
func applyMaskingByUser(winnerData gin.H, drawBatch int, currentUserID int) gin.H {
	log.Printf("开始根据用户ID进行打码处理，当前用户ID: %d", currentUserID)
	maskedData := make(gin.H)

	// 复制所有字段
	for k, v := range winnerData {
		maskedData[k] = v
	}

	// 添加抽奖轮次
	maskedData["draw_batch"] = drawBatch
	log.Printf("添加抽奖批次: %d", drawBatch)

	// 获取当前用户的手机号，用于判断是否是当前用户
	var currentUser models.SnowUser
	db.DB.Where("user_id = ?", currentUserID).First(&currentUser)
	currentUserMobile := currentUser.Mobile
	log.Printf("当前用户手机号: %d", currentUserMobile)

	// 获取中奖者的手机号
	var winnerMobile int
	if mobile, ok := winnerData["mobile"]; ok {
		log.Printf("处理手机号，原始类型: %T, 值: %v", mobile, mobile)
		switch v := mobile.(type) {
		case string:
			fmt.Sscanf(v, "%d", &winnerMobile)
		case int:
			winnerMobile = v
		case float64:
			winnerMobile = int(v)
		default:
			fmt.Sscanf(fmt.Sprintf("%v", v), "%d", &winnerMobile)
		}
		log.Printf("中奖者手机号: %d", winnerMobile)
	}

	// 判断是否是当前用户
	isCurrentUser := currentUserMobile != 0 && winnerMobile == currentUserMobile
	log.Printf("是否是当前用户: %v", isCurrentUser)

	if isCurrentUser {
		// 是当前用户，返回详细信息，但仍对敏感信息进行基本打码
		// 对手机号进行打码
		if mobile, ok := winnerData["mobile"]; ok {
			var mobileStr string
			switch v := mobile.(type) {
			case string:
				mobileStr = v
			case int:
				mobileStr = fmt.Sprintf("%d", v)
			case float64:
				mobileStr = fmt.Sprintf("%.0f", v)
			default:
				mobileStr = fmt.Sprintf("%v", v)
			}
			if len(mobileStr) >= 7 {
				maskedMobile := maskMobile(mobileStr)
				maskedData["mobile"] = maskedMobile
			}
		}

		// 对昵称进行打码（当前用户的昵称也稍微打码）
		if nickname, ok := winnerData["nickname"]; ok {
			if nicknameStr, ok := nickname.(string); ok {
				maskedNickname := maskNickname(nicknameStr)
				maskedData["nickname"] = maskedNickname
			}
		}

		// 对订单号进行打码
		if orderNum, ok := winnerData["order_num"]; ok {
			if orderNumStr, ok := orderNum.(string); ok {
				maskedOrderNum := maskOrderNum(orderNumStr)
				maskedData["order_num"] = maskedOrderNum
			}
		}
	} else {
		// 不是当前用户，只返回抽奖码、打码的手机号和奖品信息
		minimalData := make(gin.H)

		// 保留抽奖码
		if drawCode, ok := winnerData["draw_code"]; ok {
			minimalData["draw_code"] = drawCode
		}

		// 保留打码后的手机号
		if mobile, ok := winnerData["mobile"]; ok {
			var mobileStr string
			switch v := mobile.(type) {
			case string:
				mobileStr = v
			case int:
				mobileStr = fmt.Sprintf("%d", v)
			case float64:
				mobileStr = fmt.Sprintf("%.0f", v)
			default:
				mobileStr = fmt.Sprintf("%v", v)
			}
			if len(mobileStr) >= 7 {
				maskedMobile := maskMobile(mobileStr)
				minimalData["mobile"] = maskedMobile
			}
		}

		// 保留奖品信息（prizes和prize_name）
		if prizes, ok := winnerData["prizes"]; ok {
			minimalData["prizes"] = prizes
		}

		// 保留prize_name字段（优先显示）
		if prizeName, ok := winnerData["prize_name"]; ok {
			minimalData["prize_name"] = prizeName
		}

		// 添加抽奖轮次
		minimalData["draw_batch"] = drawBatch

		// 替换为最小数据集
		maskedData = minimalData
	}

	log.Printf("根据用户ID打码处理完成，结果: %+v", maskedData)
	return maskedData
}

// GetAllDraws 查询所有抽奖信息或单个抽奖信息
func (sdc *SnowLotteryDrawController) GetAllDraws(c *gin.Context) {
	// 初始化drawID变量
	drawID := ""

	// 首先尝试从JSON请求体中获取draw_id
	var requestBody map[string]interface{}
	if err := c.ShouldBindJSON(&requestBody); err == nil {
		// JSON解析成功，从请求体中获取draw_id
		if id, ok := requestBody["draw_id"]; ok {
			drawID = fmt.Sprintf("%v", id)
		}
	} else {
		// JSON解析失败，尝试从表单中获取draw_id
		drawID = c.PostForm("draw_id")
	}

	// 处理返回数据
	var responseData []gin.H

	// 如果提供了draw_id，则查询单个抽奖活动
	if drawID != "" {
		var draw models.SnowLotteryDraw
		if err := db.DB.Where("id = ?", drawID).First(&draw).Error; err != nil {
			log.Printf("查询抽奖活动失败: %v", err)
			c.JSON(http.StatusNotFound, gin.H{"error": "抽奖活动不存在"})
			return
		}

		// 解析prizes JSON字符串为map
		var prizes map[string]int
		if draw.Prizes != "" && draw.Prizes != "{}" {
			if err := json.Unmarshal([]byte(draw.Prizes), &prizes); err != nil {
				log.Printf("解析prizes失败: %v", err)
				// 解析失败时使用空map
				prizes = make(map[string]int)
			}
		} else {
			prizes = make(map[string]int)
		}

		// 格式化时间
		orderBeginTime := draw.OrderBeginTime.Format("2006-01-02 15:04:05")
		orderEndTime := draw.OrderEndTime.Format("2006-01-02 15:04:05")

		// 构建返回数据
		drawData := gin.H{
			"id":               draw.ID,
			"draw_batch":       draw.DrawBatch,
			"prizes":           prizes,
			"total_drawers":    draw.TotalDrawers,
			"order_begin_time": orderBeginTime,
			"order_end_time":   orderEndTime,
			"draw_name":        draw.DrawName,
		}
		responseData = append(responseData, drawData)
	} else {
		// 查询所有抽奖活动
		var draws []models.SnowLotteryDraw
		if err := db.DB.Find(&draws).Error; err != nil {
			log.Printf("查询所有抽奖活动失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "查询抽奖活动失败"})
			return
		}

		for _, draw := range draws {
			// 解析prizes JSON字符串为map
			var prizes map[string]int
			if draw.Prizes != "" && draw.Prizes != "{}" {
				if err := json.Unmarshal([]byte(draw.Prizes), &prizes); err != nil {
					log.Printf("解析prizes失败: %v", err)
					// 解析失败时使用空map
					prizes = make(map[string]int)
				}
			} else {
				prizes = make(map[string]int)
			}

			// 格式化时间
			orderBeginTime := draw.OrderBeginTime.Format("2006-01-02 15:04:05")
			orderEndTime := draw.OrderEndTime.Format("2006-01-02 15:04:05")

			// 构建返回数据
			drawData := gin.H{
				"id":               draw.ID,
				"draw_batch":       draw.DrawBatch,
				"prizes":           prizes,
				"total_drawers":    draw.TotalDrawers,
				"order_begin_time": orderBeginTime,
				"order_end_time":   orderEndTime,
				"draw_name":        draw.DrawName,
			}
			responseData = append(responseData, drawData)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "查询所有抽奖活动成功",
		"data":    responseData,
		"count":   len(responseData),
	})
}

// GetAllDrawInfo 查询所有抽奖信息，包含抽奖波次、开奖时间、开奖情况、前5个抽奖码
func (sdc *SnowLotteryDrawController) GetAllDrawInfo(c *gin.Context) {
	// 查询所有抽奖活动
	var draws []models.SnowLotteryDraw
	if err := db.DB.Find(&draws).Error; err != nil {
		log.Printf("查询所有抽奖活动失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询抽奖活动失败"})
		return
	}

	var responseData []gin.H
	currentTime := time.Now()
	type WinningInfo struct {
		Message         string `json:"message"`
		Batch           string `json:"batch"`
		DrawSuccessTime string `json:"draw_success_time"`
	}
	var userUnverifiedWinnings []WinningInfo
	var userVerifiedWinnings []WinningInfo

	// 检查请求体中是否包含user_id
	var request GetAllDrawInfoRequest
	if err := c.ShouldBindJSON(&request); err == nil {
		// 如果user_id不为0
		if request.UserID != 0 {
			// 查询对应的SnowUser记录
			var user models.SnowUser
			if err := db.DB.First(&user, "user_id = ?", request.UserID).Error; err == nil {
				// 解析MobileBatchMap
				if err := user.AfterFind(nil); err == nil {
					// 遍历所有波次和手机号
					var hasAnyWinning bool
					for batch, mobile := range user.MobileBatchMap {
						// 查询对应的SnowSuccessUser记录
						var successUser models.SnowSuccessUser
						if err := db.DB.First(&successUser, "mobile = ?", mobile).Error; err == nil {
							// 获取WinningStatus
							winningStatus, err := successUser.GetWinningStatus()
							if err == nil {
								// 获取VerificationStatus
								verificationStatus, err := successUser.GetVerificationStatus()
								if err == nil {
									// 获取中奖时间
									drawSuccessTimeMap, err := successUser.GetDrawSuccessTime()
									drawSuccessTime := ""
									var isExpired bool

									// 检查该波次是否中奖
									if winningStatus[batch] {
										hasAnyWinning = true

										// 获取该波次的中奖时间
										if err == nil {
											drawSuccessTime = drawSuccessTimeMap[batch]

											// 检查验证时间是否过期（7天）
											if drawSuccessTime != "" {
												// 解析中奖时间
												winTime, err := time.Parse("2006-01-02 15:04:05", drawSuccessTime)
												if err == nil {
													// 检查是否超过7天
													if time.Since(winTime) > 7*24*time.Hour {
														isExpired = true
													}
												}
											}
										}

										// 检查是否未验证且验证时间已过
										if !verificationStatus[batch] {
											if isExpired {
												// 验证时间已过
												userUnverifiedWinnings = append(userUnverifiedWinnings, WinningInfo{
													Message:         fmt.Sprintf("第%s波的验证时间已过", batch),
													Batch:           batch,
													DrawSuccessTime: drawSuccessTime,
												})
											} else {
												// 未验证且验证时间未过
												userUnverifiedWinnings = append(userUnverifiedWinnings, WinningInfo{
													Message:         fmt.Sprintf("该用户暂未确认第%s波次的中奖验证，请验证", batch),
													Batch:           batch,
													DrawSuccessTime: drawSuccessTime,
												})
											}
										} else {
											// 已验证的情况
											userVerifiedWinnings = append(userVerifiedWinnings, WinningInfo{
												Message:         fmt.Sprintf("该用户已确认第%s波次的中奖验证", batch),
												Batch:           batch,
												DrawSuccessTime: drawSuccessTime,
											})
										}
									}
								}
							}
						}
					}
					// 如果没有未验证的中奖记录，检查是否有任何中奖记录
					if len(userUnverifiedWinnings) == 0 && len(userVerifiedWinnings) == 0 {
						if !hasAnyWinning {
							// 没有任何中奖记录，添加提示信息
							userUnverifiedWinnings = append(userUnverifiedWinnings, WinningInfo{
								Message: "该用户未中奖",
								Batch:   "",
							})
						}
					}
				}
			}
		}
	}

	for _, draw := range draws {
		// 判断是否已开奖（根据开奖时间判断）
		isDrawn := currentTime.After(draw.DrawTime)

		// 获取前5个抽奖码（只从WinnersList解析）
		var topDrawCodes []string
		if isDrawn && draw.WinnersList != "" && draw.WinnersList != "{}" && draw.WinnersList != "[]" {
			var winners []map[string]interface{}

			// 尝试直接解析为JSON数组
			if err := json.Unmarshal([]byte(draw.WinnersList), &winners); err == nil {
				// 成功解析为数组，提取前5个抽奖码
				for _, winner := range winners {
					if drawCode, ok := winner["draw_code"].(string); ok && drawCode != "" {
						topDrawCodes = append(topDrawCodes, drawCode)
						if len(topDrawCodes) >= 5 {
							break // 达到5个即可
						}
					}
				}
			} else {
				log.Printf("解析WinnersList失败，抽奖波次: %d, 错误: %v", draw.DrawBatch, err)
			}
		}

		// 构建返回数据
		drawData := gin.H{
			"draw_batch":       draw.DrawBatch,
			"draw_end_time":    draw.DrawTime.Format("2006-01-02 15:04:05"),
			"is_drawn":         isDrawn,
			"top_5_draw_codes": topDrawCodes,
		}
		responseData = append(responseData, drawData)
	}

	// 构建响应数据
	response := gin.H{
		"success": true,
		"message": "查询所有抽奖信息成功",
		"data":    responseData,
		"count":   len(responseData),
	}

	// 如果有未验证的中奖信息，添加到响应中
	if len(userUnverifiedWinnings) > 0 {
		response["unverified_winnings"] = userUnverifiedWinnings
	}
	// 如果有已验证的中奖信息，添加到响应中
	if len(userVerifiedWinnings) > 0 {
		response["verified_winnings"] = userVerifiedWinnings
	}

	c.JSON(http.StatusOK, response)
}
