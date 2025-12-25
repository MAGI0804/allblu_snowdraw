package controllers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"django_to_go/db"
	"django_to_go/models"

	"github.com/gin-gonic/gin"
)

// SnowSuccessUserController 抽奖成功用户控制器
type SnowSuccessUserController struct{}

// NewSnowSuccessUserController 创建成功用户控制器实例
func NewSnowSuccessUserController() *SnowSuccessUserController {
	return &SnowSuccessUserController{}
}

// AddEligibilityUserRequest 新增资格用户请求结构体
type AddEligibilityUserRequest struct {
	UserID    int    `json:"user_id" binding:"required"`
	Nickname  string `json:"nickname" binding:"required"`
	Mobile    int    `json:"mobile" binding:"required"`
	OrderNum  string `json:"order_num" binding:"required"`
	DrawBatch int    `json:"draw_batch" binding:"required"`
	Remarks   string `json:"remarks"`
}

// UpdateEligibilityUserRequest 修改资格用户信息请求结构体
type UpdateEligibilityUserRequest struct {
	UserID    int    `json:"user_id" binding:"required"`
	Nickname  string `json:"nickname"`
	Mobile    int    `json:"mobile"`
	OrderNum  string `json:"order_num"`
	DrawBatch int    `json:"draw_batch"`
	Remarks   string `json:"remarks"`
}

// AddEligibilityUser 新增资格用户（支持一个用户参加多个波次）
func (ssuc *SnowSuccessUserController) AddEligibilityUser(c *gin.Context) {
	var request AddEligibilityUserRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	// 检查用户是否已存在
	var user models.SnowSuccessUser
	var isExistingUser bool
	drawBatchStr := strconv.Itoa(request.DrawBatch)

	if err := db.DB.Where("user_id = ?", request.UserID).First(&user).Error; err == nil {
		// 用户已存在，更新波次信息
		isExistingUser = true

		// 获取现有抽奖资格并更新
		drawEligibility, err := user.GetDrawEligibility()
		if err != nil {
			drawEligibility = make(map[string]bool)
		}
		drawEligibility[drawBatchStr] = true
		if err := user.SetDrawEligibility(drawEligibility); err != nil {
			log.Printf("设置抽奖资格失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "新增资格用户失败"})
			return
		}

		// 获取现有订单号映射并更新当前波次的订单号
		orderNUMMap, err := user.GetOrderNUM()
		if err != nil {
			orderNUMMap = make(map[string]string)
		}
		orderNUMMap[drawBatchStr] = request.OrderNum
		if err := user.SetOrderNUM(orderNUMMap); err != nil {
			log.Printf("设置订单号失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "新增资格用户失败"})
			return
		}

		// 更新用户基本信息（如果有新值）
		if request.Nickname != "" {
			user.Nickname = request.Nickname
		}
		if request.Mobile > 0 {
			user.Mobile = request.Mobile
		}
		if request.Remarks != "" {
			user.Remarks = request.Remarks
		}
	}

	// 如果用户不存在，创建新用户
	if !isExistingUser {
		// 初始化抽奖资格映射，设置当前波次为有资格
		drawEligibilityMap := map[string]bool{
			drawBatchStr: true,
		}

		// 创建订单号映射，使用波次作为键，订单号作为值，支持多个波次
		orderNUMMap := map[string]string{
			drawBatchStr: request.OrderNum,
		}

		// 创建用户记录
		user = models.SnowSuccessUser{
			UserID:              request.UserID,
			Nickname:            request.Nickname,
			Mobile:              request.Mobile,
			DrawEligibility:     "{}",
			ParticipationStatus: "{}",
			WinningStatus:       "{}",
			OrderNUM:            "{}",
			Remarks:             request.Remarks,
		}

		// 设置JSON字段
		if err := user.SetDrawEligibility(drawEligibilityMap); err != nil {
			log.Printf("设置抽奖资格失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "新增资格用户失败"})
			return
		}

		if err := user.SetOrderNUM(orderNUMMap); err != nil {
			log.Printf("设置订单号失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "新增资格用户失败"})
			return
		}

		// 保存到数据库
		if err := db.DB.Create(&user).Error; err != nil {
			log.Printf("新增资格用户失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "新增资格用户失败"})
			return
		}
	} else {
		// 用户已存在，更新到数据库
		if err := db.DB.Save(&user).Error; err != nil {
			log.Printf("更新资格用户失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "更新资格用户失败"})
			return
		}
	}

	// 创建响应数据结构，将JSON字符串字段解析为实际的map对象
	drawEligibility, _ := user.GetDrawEligibility()
	orderNUM, _ := user.GetOrderNUM()

	responseData := gin.H{
		"user_id":              user.UserID,
		"nickname":             user.Nickname,
		"mobile":               user.Mobile,
		"member_source":        user.MemberSource,
		"draw_eligibility":     drawEligibility,
		"participation_status": map[string]interface{}{},
		"winning_status":       map[string]interface{}{},
		"order_num":            orderNUM,
		"remarks":              user.Remarks,
	}

	// 根据操作类型显示不同的成功消息
	message := "新增资格用户成功"
	if isExistingUser {
		message = "更新资格用户成功"
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": message,
		"data":    responseData,
	})
}

// VerifyLotteryRequest 验证抽奖请求结构体
type VerifyLotteryRequest struct {
	UserID int `json:"user_id" binding:"required"`
	Batch  int `json:"batch" binding:"required"`
}

// VerifyLottery 验证抽奖
func (ssuc *SnowSuccessUserController) VerifyLottery(c *gin.Context) {
	var request VerifyLotteryRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	// 1. 根据 user_id 和 batch 在 SnowUser 中查询对应的手机号
	var snowUser models.SnowUser
	if err := db.DB.Where("user_id = ?", request.UserID).First(&snowUser).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	// 2. 从 MobileBatch 中获取对应 batch 的手机号
	batchStr := strconv.Itoa(request.Batch)
	mobileBatchMap, err := snowUser.GetMobileBatch()
	if err != nil {
		mobileBatchMap = make(map[string]int)
	}

	var mobile int
	if batchMobile, exists := mobileBatchMap[batchStr]; exists {
		mobile = batchMobile
	} else {
		// 如果 MobileBatch 中没有对应 batch 的手机号，使用默认手机号
		mobile = snowUser.Mobile
	}

	// 3. 在 SnowSuccessUser 中通过手机号定位用户
	var successUser models.SnowSuccessUser
	if err := db.DB.Where("mobile = ?", mobile).First(&successUser).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "抽奖用户不存在"})
		return
	}

	// 4. 在 SnowSuccessUser 的 VerificationStatus 字段中添加 "batch:true"
	verificationStatus, err := successUser.GetVerificationStatus()
	if err != nil {
		verificationStatus = make(map[string]bool)
	}
	verificationStatus[batchStr] = true

	if err := successUser.SetVerificationStatus(verificationStatus); err != nil {
		log.Printf("设置抽奖用户验证状态失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "验证抽奖失败"})
		return
	}

	// 5. 在 SnowUser 的 VerificationStatus 字段中添加 "batch:true"
	snowUserVerificationStatus, err := snowUser.GetVerificationStatus()
	if err != nil {
		snowUserVerificationStatus = make(map[string]bool)
	}
	snowUserVerificationStatus[batchStr] = true

	if err := snowUser.SetVerificationStatus(snowUserVerificationStatus); err != nil {
		log.Printf("设置用户验证状态失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "验证抽奖失败"})
		return
	}

	// 6. 保存更新
	if err := db.DB.Save(&successUser).Error; err != nil {
		log.Printf("保存抽奖用户验证状态失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "验证抽奖失败"})
		return
	}

	if err := db.DB.Save(&snowUser).Error; err != nil {
		log.Printf("保存用户验证状态失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "验证抽奖失败"})
		return
	}

	// 7. 将 SnowUser 的地址信息写入到 SnowLotteryDraw 的 WinnersList 中
	// 7.1 查询对应的抽奖活动
	var lotteryDraw models.SnowLotteryDraw
	if err := db.DB.Where("draw_batch = ?", request.Batch).First(&lotteryDraw).Error; err != nil {
		log.Printf("查询抽奖活动失败: %v", err)
		// 继续执行，不影响验证结果
	} else {
		// 7.2 解析 WinnersList
		var winnersList []map[string]interface{}
		if lotteryDraw.WinnersList != "" {
			if err := json.Unmarshal([]byte(lotteryDraw.WinnersList), &winnersList); err != nil {
				log.Printf("解析 WinnersList 失败: %v", err)
				// 继续执行，不影响验证结果
			} else {
				// 7.3 查找对应的中奖记录并添加地址信息
				for i, winner := range winnersList {
					if winnerMobile, ok := winner["mobile"].(float64); ok {
						if int(winnerMobile) == mobile {
							// 添加地址信息
							winnersList[i]["receiver_name"] = snowUser.ReceiverName
							winnersList[i]["receiver_phone"] = snowUser.ReceiverPhone
							winnersList[i]["province"] = snowUser.Province
							winnersList[i]["city"] = snowUser.City
							winnersList[i]["county"] = snowUser.County
							winnersList[i]["detailed_address"] = snowUser.DetailedAddress
							// 覆盖现有的昵称
							winnersList[i]["nickname"] = snowUser.Nickname
							break
						}
					}
				}

				// 7.4 更新 WinnersList
				updatedWinnersList, err := json.Marshal(winnersList)
				if err != nil {
					log.Printf("序列化 WinnersList 失败: %v", err)
					// 继续执行，不影响验证结果
				} else {
					lotteryDraw.WinnersList = string(updatedWinnersList)
					if err := db.DB.Save(&lotteryDraw).Error; err != nil {
						log.Printf("保存 WinnersList 失败: %v", err)
						// 继续执行，不影响验证结果
					}
				}
			}
		}
	}

	// 8. 返回结果
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "抽奖验证成功",
		"data": gin.H{
			"user_id":             successUser.UserID,
			"mobile":              successUser.Mobile,
			"verification_status": verificationStatus,
		},
	})
}

// UpdateEligibilityUser 修改资格用户信息
func (ssuc *SnowSuccessUserController) UpdateEligibilityUser(c *gin.Context) {
	var request UpdateEligibilityUserRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	// 查询用户是否存在
	var user models.SnowSuccessUser
	if err := db.DB.Where("user_id = ?", request.UserID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	// 更新用户信息
	if request.Nickname != "" {
		user.Nickname = request.Nickname
	}
	if request.Mobile > 0 {
		user.Mobile = request.Mobile
	}
	if request.Remarks != "" {
		user.Remarks = request.Remarks
	}

	// 如果提供了订单号和抽奖波次，则更新订单信息
	if request.OrderNum != "" && request.DrawBatch > 0 {
		drawBatchStr := strconv.Itoa(request.DrawBatch)

		// 获取现有抽奖资格并更新
		drawEligibility, err := user.GetDrawEligibility()
		if err != nil {
			drawEligibility = make(map[string]bool)
		}
		drawEligibility[drawBatchStr] = true
		if err := user.SetDrawEligibility(drawEligibility); err != nil {
			log.Printf("更新抽奖资格失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "修改资格用户信息失败"})
			return
		}

		// 获取现有订单号映射并更新当前波次的订单号，支持多个波次
		orderNUMMap, err := user.GetOrderNUM()
		if err != nil {
			orderNUMMap = make(map[string]string)
		}
		orderNUMMap[drawBatchStr] = request.OrderNum
		if err := user.SetOrderNUM(orderNUMMap); err != nil {
			log.Printf("更新订单号失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "修改资格用户信息失败"})
			return
		}
	}

	// 保存更新
	if err := db.DB.Save(&user).Error; err != nil {
		log.Printf("修改资格用户信息失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "修改资格用户信息失败"})
		return
	}

	// 创建响应数据结构，将JSON字符串字段解析为实际的map对象
	drawEligibility, _ := user.GetDrawEligibility()
	orderNUM, _ := user.GetOrderNUM()

	responseData := gin.H{
		"user_id":              user.UserID,
		"nickname":             user.Nickname,
		"mobile":               user.Mobile,
		"member_source":        user.MemberSource,
		"draw_eligibility":     drawEligibility,
		"participation_status": map[string]interface{}{},
		"winning_status":       map[string]interface{}{},
		"order_num":            orderNUM,
		"remarks":              user.Remarks,
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "修改资格用户信息成功",
		"data":    responseData,
	})
}
