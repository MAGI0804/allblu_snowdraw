package controllers

import (
	"crypto/rand"
	"django_to_go/config"
	"django_to_go/db"
	vip "django_to_go/method/vip"
	"django_to_go/models"
	"django_to_go/other_method/message"
	"django_to_go/utils"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	// dysmsapi20170525 "github.com/alibabacloud-go/dysmsapi-20170525/v5/client"
	// openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	// util "github.com/alibabacloud-go/tea-utils/v2/service"
	// credential "github.com/aliyun/credentials-go/credentials"
	// "github.com/alibabacloud-go/tea/tea"
)

// SnowUserController 抽奖用户控制器
type SnowUserController struct{}

// DownloadAndStoreAvatar 下载并存储用户头像到本地
func (suc *SnowUserController) DownloadAndStoreAvatar(avatarURL, userID string) (string, error) {
	if avatarURL == "" {
		return "", nil
	}

	// 获取文件扩展名
	fileExt := ".jpg"
	lowerURL := strings.ToLower(avatarURL)
	if strings.Contains(lowerURL, ".png") {
		fileExt = ".png"
	} else if strings.Contains(lowerURL, ".gif") {
		fileExt = ".gif"
	}

	// 确保目录存在
	avatarDir := "./media/user_avatars"
	if err := os.MkdirAll(avatarDir, 0755); err != nil {
		return "", fmt.Errorf("创建头像目录失败: %v", err)
	}

	filename := fmt.Sprintf("avatar_%s%s", userID, fileExt)
	filepath := filepath.Join(avatarDir, filename)

	// 下载头像
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(avatarURL)
	if err != nil {
		return "", fmt.Errorf("下载头像失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("下载头像失败，状态码: %d", resp.StatusCode)
	}

	// 创建文件
	file, err := os.Create(filepath)
	if err != nil {
		return "", fmt.Errorf("创建头像文件失败: %v", err)
	}
	defer file.Close()

	// 复制文件内容
	if _, err := io.Copy(file, resp.Body); err != nil {
		return "", fmt.Errorf("保存头像文件失败: %v", err)
	}

	return fmt.Sprintf("/media/user_avatars/%s", filename), nil
}

// WechatRegisterRequest 微信注册请求
type WechatRegisterRequest struct {
	Code     string                 `json:"code" binding:"required"`
	UserInfo map[string]interface{} `json:"userInfo"`
}

// VerifyOrderMobileRequest 订单号与手机号校验请求
type VerifyOrderMobileRequest struct {
	OrderNum  string `json:"order_num" binding:"required"`
	Mobile    int    `json:"mobile" binding:"required"`
	Platform  string `json:"platform" binding:"required"`
	DrawBatch string `json:"draw_batch" binding:"required"`
}

// SendVerificationCodeRequest 发送验证码请求
type SendVerificationCodeRequest struct {
	UserID int `json:"user_id" binding:"required"`
	Mobile int `json:"mobile" binding:"required"`
}

// VerifyVerificationCodeRequest 验证码校验请求
type VerifyVerificationCodeRequest struct {
	UserID           int `json:"user_id" binding:"required"`
	Mobile           int `json:"mobile" binding:"required"`
	VerificationCode int `json:"verification_code" binding:"required"`
}

// UpdateAddressRequest 修改地址请求
type UpdateAddressRequest struct {
	UserID          int    `json:"user_id" binding:"required"`
	Province        string `json:"province" binding:"required"`
	City            string `json:"city" binding:"required"`
	County          string `json:"county" binding:"required"`
	DetailedAddress string `json:"detailed_address" binding:"required"`
	ReceiverName    string `json:"receiver_name" binding:"required"`
	ReceiverPhone   string `json:"receiver_phone" binding:"required"`
}

// QueryAddressRequest 查询地址请求
type QueryAddressRequest struct {
	UserID int `json:"user_id" binding:"required"`
}

// VerifyDrawEligibilityRequest 验证抽奖资格请求
type VerifyDrawEligibilityRequest struct {
	Mobile    int    `json:"mobile" binding:"required"`
	DrawBatch string `json:"draw_batch"` // 可选参数
}

// ParticipateDrawRequest 参与抽奖请求
type ParticipateDrawRequest struct {
	UserID           int    `json:"user_id" binding:"required"`
	OrderNum         string `json:"order_num" binding:"required"`
	Mobile           int    `json:"mobile" binding:"required"`
	DrawBatch        string `json:"draw_batch" binding:"required"`
	VerificationCode int    `json:"verification_code" binding:"required"` // 新增验证码参数
}

// ParticipateDrawByMobileRequest 仅通过手机号参与抽奖请求
type ParticipateDrawByMobileRequest struct {
	UserID           int    `json:"user_id" binding:"required"`
	Mobile           int    `json:"mobile" binding:"required"`
	DrawBatch        string `json:"draw_batch" binding:"required"`
	VerificationCode int    `json:"verification_code" binding:"required"` // 验证码参数
}

// FindDataRequest 查找用户信息请求
type FindDataRequest struct {
	UserID int `json:"user_id" binding:"required"`
}

// QueryUserDrawInfoRequest 查询用户抽奖信息请求
type QueryUserDrawInfoRequest struct {
	UserID int `json:"user_id" binding:"required"`
}

// DeleteDrawInfoRequest 删除指定轮次抽奖信息请求
type DeleteDrawInfoRequest struct {
	UserID    int    `json:"user_id" binding:"required"`
	DrawBatch string `json:"draw_batch" binding:"required"`
}

// UpdateUserInfoRequest 修改用户信息请求
type UpdateUserInfoRequest struct {
	UserID    int    `json:"user_id" binding:"required"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url"`
}

// WechatRegister 微信注册
func (suc *SnowUserController) WechatRegister(c *gin.Context) {
	var request WechatRegisterRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的JSON格式",
			"error":   err.Error(),
		})
		return
	}

	// 获取微信配置
	cfg := config.LoadConfig()

	// 调用微信API获取openid
	wxURL := fmt.Sprintf("%s?appid=%s&secret=%s&js_code=%s&grant_type=authorization_code",
		cfg.WechatConfig.LoginURL,
		cfg.WechatConfig.AppID,
		cfg.WechatConfig.AppSecret,
		request.Code,
	)

	resp, err := http.Get(wxURL)
	if err != nil {
		log.Printf("微信API请求失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "微信登录失败",
			"error":   "网络请求错误",
		})
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("读取微信API响应失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "微信登录失败",
			"error":   "响应解析错误",
		})
		return
	}

	// 解析微信响应
	var wxResult map[string]interface{}
	if err := json.Unmarshal(body, &wxResult); err != nil {
		log.Printf("解析微信API响应失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "微信登录失败",
			"error":   "响应格式错误",
		})
		return
	}

	// 检查是否有错误
	if _, ok := wxResult["errcode"]; ok {
		errMsg := "未知错误"
		if msg, ok := wxResult["errmsg"].(string); ok {
			errMsg = msg
		}
		log.Printf("微信登录失败: %v", wxResult)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "微信登录失败",
			"error":   errMsg,
		})
		return
	}

	// 获取openid
	openid, ok := wxResult["openid"].(string)
	if !ok || openid == "" {
		log.Printf("微信返回数据中没有openid: %v", wxResult)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "微信登录失败",
			"error":   "未获取到openid",
		})
		return
	}

	// 解析用户信息
	var nickname string
	var avatarURL string

	if request.UserInfo != nil {
		// 尝试解析昵称
		if nicknameVal, ok := request.UserInfo["nickName"].(string); ok {
			nickname = nicknameVal
		} else if nicknameVal, ok := request.UserInfo["nickname"].(string); ok {
			nickname = nicknameVal
		}

		// 尝试解析头像URL - 支持两种字段名格式
		if avatarVal, ok := request.UserInfo["avatarUrl"].(string); ok {
			avatarURL = avatarVal
		} else if avatarVal, ok := request.UserInfo["avatar_url"].(string); ok {
			avatarURL = avatarVal
		}

		log.Printf("接收到的用户信息 - 昵称: %s, 头像URL: %s", nickname, avatarURL)
	}

	// 如果没有昵称，生成默认昵称
	if nickname == "" {
		if len(openid) > 8 {
			nickname = "微信用户_" + openid[:8]
		} else {
			nickname = "微信用户_" + openid
		}
	}

	// 查询或创建用户
	var user models.SnowUser
	if err := db.DB.Where("openid = ?", openid).First(&user).Error; err != nil {
		// 用户不存在，创建新用户
		// 生成唯一的临时手机号 - 使用当前时间戳
		timestamp := time.Now().UnixNano() / int64(time.Millisecond)
		tempMobile := int(timestamp%9000000000) + 1000000000 // 生成10位随机数

		user = models.SnowUser{
			OpenID:           openid,
			Nickname:         nickname,
			AvatarURL:        avatarURL,
			MemberSource:     "wechat",
			RegistrationTime: time.Now(),
			Mobile:           tempMobile,
			// 不直接初始化map字段，让BeforeSave钩子处理
		}

		if err := db.DB.Create(&user).Error; err != nil {
			log.Printf("创建用户失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "注册失败",
				"error":   "用户创建失败",
			})
			return
		}

		// 直接使用传入的头像URL，不进行下载操作

		// 返回注册成功响应
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "注册成功",
			"data": gin.H{
				"user_id":  user.UserID,
				"nickname": user.Nickname,
				"avatar":   user.AvatarURL,
			},
		})
	} else {
		// 用户已存在，检查是否需要更新头像和昵称
		updateNeeded := false

		// 只有当数据库中的头像为空且微信提供了头像时才更新
		if user.AvatarURL == "" && avatarURL != "" {
			user.AvatarURL = avatarURL
			updateNeeded = true
		}

		// 只有当数据库中的昵称是默认值且微信提供了新昵称时才更新
		if strings.HasPrefix(user.Nickname, "微信用户_") && nickname != "" && !strings.HasPrefix(nickname, "微信用户_") {
			user.Nickname = nickname
			updateNeeded = true
		}

		// 如果有更新，保存更改
		if updateNeeded {
			if err := db.DB.Save(&user).Error; err != nil {
				log.Printf("更新用户信息失败: %v", err)
			}
		}

		// 用户已存在，返回当前完整用户信息
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "用户已注册",
			"data": gin.H{
				"user_id":           user.UserID,
				"nickname":          user.Nickname,
				"avatar":            user.AvatarURL,
				"mobile":            user.Mobile,
				"member_source":     user.MemberSource,
				"province":          user.Province,
				"city":              user.City,
				"county":            user.County,
				"detailed_address":  user.DetailedAddress,
				"registration_time": user.RegistrationTime.Format("2006-01-02 15:04:05"),
			},
		})
	}
}

// VerifyOrderMobile 订单号与手机号校验
func (suc *SnowUserController) VerifyOrderMobile(c *gin.Context) {
	var request VerifyOrderMobileRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	// 查找用户
	var successUser models.SnowSuccessUser
	result := db.DB.Where("mobile = ?", request.Mobile).First(&successUser)

	if result.Error != nil {
		// 用户不存在
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "暂无当前订单信息",
		})
		return
	}

	// 解析order_num JSON字符串
	orderNUMMap, err := successUser.GetOrderNUM()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "订单信息解析失败",
		})
		return
	}

	// 检查请求的波次对应的订单号是否匹配
	if orderNum, exists := orderNUMMap[request.DrawBatch]; exists && orderNum == request.OrderNum {
		// 波次和订单号都匹配
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "验证通过",
		})
	} else {
		// 检查是否存在手机号但订单号不匹配的情况
		var count int64
		db.DB.Model(&models.SnowSuccessUser{}).Where("mobile = ?", request.Mobile).Count(&count)

		if count > 0 {
			// 存在该手机号的记录但订单号不匹配
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "订单号与手机号不匹配",
			})
		} else {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "暂无当前订单信息",
			})
		}
	}
}

// QueryUserDrawInfo 查询用户抽奖信息
func (suc *SnowUserController) QueryUserDrawInfo(c *gin.Context) {
	var request QueryUserDrawInfoRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	// 查询用户
	var user models.SnowUser
	result := db.DB.Where("user_id = ?", request.UserID).First(&user)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	// 获取当前时间
	currentTime := time.Now()

	// 查询当前可参与的抽奖轮次
	var currentDraws []models.SnowLotteryDraw
	db.DB.Where("order_begin_time <= ? AND order_end_time >= ?", currentTime, currentTime).Find(&currentDraws)

	if len(currentDraws) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "当前暂无可参与的抽奖活动",
			"data": gin.H{
				"participation_status": "未参与",
			},
		})
		return
	}

	// 获取当前抽奖轮次
	currentDraw := currentDraws[0]
	drawBatchStr := fmt.Sprintf("%d", currentDraw.DrawBatch)

	// 检查用户是否参与过该抽奖
	var participationStatus map[string]bool
	if user.ParticipationStatusMap != nil {
		participationStatus = user.ParticipationStatusMap
	} else if user.ParticipationStatus != "" {
		if err := json.Unmarshal([]byte(user.ParticipationStatus), &participationStatus); err != nil {
			participationStatus = make(map[string]bool)
		}
	} else {
		participationStatus = make(map[string]bool)
	}

	// 检查是否参与
	if participated, exists := participationStatus[drawBatchStr]; exists && participated {
		// 获取抽奖码
		drawCode := ""
		if user.SuccessCodeMap != nil {
			drawCode = user.SuccessCodeMap[drawBatchStr]
		}

		// 获取订单号
		orderNum := ""
		if user.OrderNumbersMap != nil {
			orderNum = user.OrderNumbersMap[drawBatchStr]
		}

		// 从MobileBatch中获取对应波次的手机号
		mobile := user.Mobile // 默认使用原始手机号
		if user.MobileBatchMap != nil {
			if batchMobile, exists := user.MobileBatchMap[drawBatchStr]; exists {
				mobile = batchMobile
			}
		}

		// 对手机号进行打码处理
		maskedMobile := maskMobile(fmt.Sprintf("%d", mobile))
		// 对订单号进行打码处理
		maskedOrderNum := maskOrderNum(orderNum)

		// 返回参与信息
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "用户已参与当前抽奖活动",
			"data": gin.H{
				"draw_batch":           currentDraw.DrawBatch,
				"draw_code":            drawCode,
				"draw_name":            currentDraw.DrawName,
				"mobile":               maskedMobile,
				"order_num":            maskedOrderNum,
				"draw_end_time":        currentDraw.OrderEndTime.Format("2006-01-02 15:04:05"),
				"participation_status": "已参与",
			},
		})
	} else {
		// 用户未参与当前抽奖
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "用户未参与当前抽奖活动",
			"data": gin.H{
				"draw_batch":           currentDraw.DrawBatch,
				"draw_name":            currentDraw.DrawName,
				"order_begin_time":     currentDraw.OrderBeginTime.Format("2006-01-02 15:04:05"),
				"order_end_time":       currentDraw.OrderEndTime.Format("2006-01-02 15:04:05"),
				"draw_end_time":        currentDraw.OrderEndTime.Format("2006-01-02 15:04:05"),
				"participation_status": "未参与",
			},
		})
	}
}

// FindData 查找用户信息
func (suc *SnowUserController) FindData(c *gin.Context) {
	var request FindDataRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	// 查询用户
	var user models.SnowUser
	if err := db.DB.Where("user_id = ?", request.UserID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	// 构建响应数据
	response := gin.H{
		"user_id":           user.UserID,
		"nickname":          user.Nickname,
		"mobile":            user.Mobile,
		"avatar_url":        user.AvatarURL,
		"member_source":     user.MemberSource,
		"province":          user.Province,
		"city":              user.City,
		"county":            user.County,
		"detailed_address":  user.DetailedAddress,
		"openid":            user.OpenID,
		"registration_time": user.RegistrationTime.Format("2006-01-02 15:04:05"),
	}

	// 添加抽奖码和抽奖波次信息，同时包含开奖时间
	if user.SuccessCodeMap != nil && len(user.SuccessCodeMap) > 0 {
		// 创建带开奖时间的抽奖信息
		enhancedDrawInfo := make(map[string]map[string]interface{})

		// 查询每个抽奖波次对应的抽奖活动信息
		for drawBatch, drawCode := range user.SuccessCodeMap {
			// 查询抽奖活动信息
			var lotteryDraw models.SnowLotteryDraw
			// 将drawBatch转换为int类型进行查询
			var batchInt int
			fmt.Sscanf(drawBatch, "%d", &batchInt)

			drawInfo := map[string]interface{}{
				"draw_code": drawCode,
			}

			// 查询抽奖活动记录
			if err := db.DB.Where("draw_batch = ?", batchInt).First(&lotteryDraw).Error; err == nil {
				// 添加开奖时间信息（使用订单结束时间作为开奖时间）
				drawInfo["draw_time"] = lotteryDraw.DrawTime.Format("2006-01-02 15:04:05")
			}

			enhancedDrawInfo[drawBatch] = drawInfo
		}

		response["draw_info"] = enhancedDrawInfo
	}

	c.JSON(http.StatusOK, response)
}

// SendVerificationCode 短信验证
func (suc *SnowUserController) SendVerificationCode(c *gin.Context) {
	var request SendVerificationCodeRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	// 生成6位随机验证码
	verificationCode, err := generateRandomCode(6)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成验证码失败"})
		return
	}

	// 设置验证码过期时间为当前时间+300秒
	expireTime := time.Now().Add(300 * time.Second)

	// 通过UserID查询用户
	var user models.SnowUser
	result := db.DB.Where("user_id = ?", request.UserID).First(&user)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	// 不再验证手机号是否匹配，因为参与抽奖会校验资格
	// 更新用户手机号为请求中的手机号
	user.Mobile = request.Mobile

	// 更新验证码和过期时间
	user.VerificationCode = verificationCode
	user.VerificationCodeExpire = expireTime

	// 添加日志记录
	fmt.Printf("准备保存验证码: 用户ID=%v, 手机号=%v, 验证码=%v, 过期时间=%v\n",
		user.UserID, user.Mobile, verificationCode, expireTime)

	// 保存到数据库并检查错误
	if err := db.DB.Save(&user).Error; err != nil {
		fmt.Printf("保存验证码失败: 用户ID=%v, 错误=%v\n", user.UserID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "保存验证码失败，请稍后重试",
		})
		return
	}

	fmt.Printf("验证码保存成功: 用户ID=%v, 手机号=%v\n", user.UserID, user.Mobile)

	// 调用短信发送服务
	err = sendSMS(request.Mobile, verificationCode)
	if err != nil {
		// 记录日志但仍然返回成功，避免暴露系统问题给用户
		fmt.Printf("发送短信失败: %v\n", err)
	}

	// 返回响应
	response := gin.H{
		"success": true,
		"message": "验证码发送成功",
	}

	// 在开发模式下，为了方便测试，可以返回验证码（生产环境应移除）
	// 可以通过环境变量或配置来控制是否启用开发模式
	isDevMode := true // 临时设置为true用于测试，生产环境应改为false
	if isDevMode {
		response["verification_code"] = verificationCode
		response["verification_code_expire"] = expireTime
		fmt.Printf("开发模式: 返回验证码 %v 给客户端，过期时间: %v\n", verificationCode, expireTime)
	}

	c.JSON(http.StatusOK, response)
}

// generateRandomCode 生成指定长度的随机数字验证码
func generateRandomCode(length int) (int, error) {
	// 生成10^(length-1)到10^length-1之间的随机数，确保是固定长度
	min := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(length)-1), nil)
	max := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(length)), nil)
	max.Sub(max, big.NewInt(1))

	diff := new(big.Int).Sub(max, min)
	randomNum, err := rand.Int(rand.Reader, diff.Add(diff, big.NewInt(1)))
	if err != nil {
		return 0, err
	}

	result := new(big.Int).Add(min, randomNum)
	return int(result.Int64()), nil
}

// sendSMS 发送短信验证码，使用阿里云短信服务
func sendSMS(mobile int, code int) error {
	// 导入并使用main包中的SendSms函数
	// 注意：需要在import部分添加相应的导入
	log.Printf("【发送短信】向手机号 %d 发送验证码: %d\n", mobile, code)

	// 将int类型转换为string类型
	mobileStr := fmt.Sprintf("%d", mobile)
	codeStr := fmt.Sprintf("%d", code)

	// 调用main包中的SendSms函数
	result, err := message.SendSms(mobileStr, codeStr)
	if err != nil {
		log.Printf("【发送短信失败】错误信息: %v\n", err)
		return err
	}

	log.Printf("【发送短信成功】响应结果: %s\n", *result)
	return nil
}

// VerifyVerificationCode 验证码校验
func (suc *SnowUserController) VerifyVerificationCode(c *gin.Context) {
	var request VerifyVerificationCodeRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	// 通过UserID查询用户
	var user models.SnowUser
	result := db.DB.Where("user_id = ?", request.UserID).First(&user)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "用户不存在",
		})
		return
	}

	// 验证手机号是否匹配
	if user.Mobile != request.Mobile {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "手机号与用户不匹配",
		})
		return
	}

	// 检查验证码是否过期
	if time.Now().After(user.VerificationCodeExpire) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "验证码已过期",
		})
		return
	}

	if user.VerificationCode == request.VerificationCode {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "验证码正确",
		})
	} else {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "验证码错误",
		})
	}
}

// UpdateAddress 修改地址
func (suc *SnowUserController) UpdateAddress(c *gin.Context) {
	var request UpdateAddressRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	var user models.SnowUser
	result := db.DB.Where("user_id = ?", request.UserID).First(&user)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "用户不存在",
		})
		return
	}

	// 更新地址
	user.Province = request.Province
	user.City = request.City
	user.County = request.County
	user.DetailedAddress = request.DetailedAddress
	user.ReceiverName = request.ReceiverName
	user.ReceiverPhone = request.ReceiverPhone
	db.DB.Save(&user)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "地址更新成功",
	})
}

// QueryAddress 查询地址
func (suc *SnowUserController) QueryAddress(c *gin.Context) {
	var request QueryAddressRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	var user models.SnowUser
	result := db.DB.Where("user_id = ?", request.UserID).First(&user)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "用户不存在",
		})
		return
	}

	// 用户存在，返回地址信息
	c.JSON(http.StatusOK, gin.H{
		"success":             true,
		"user_id":             user.UserID,
		"province":            user.Province,
		"city":                user.City,
		"county":              user.County,
		"detailed_address":    user.DetailedAddress,
		"receiver_name":       user.ReceiverName,
		"receiver_phone":      user.ReceiverPhone,
		"verification_status": user.VerificationStatus,
	})
}

// UpdateAddressByMobileAndBatchRequest 根据手机号和波次修改地址请求结构体
type UpdateAddressByMobileAndBatchRequest struct {
	Mobile          string `json:"mobile" binding:"required"`
	DrawBatch       int    `json:"draw_batch" binding:"required"`
	Province        string `json:"province" binding:"required"`
	City            string `json:"city" binding:"required"`
	County          string `json:"county" binding:"required"`
	DetailedAddress string `json:"detailed_address" binding:"required"`
	ReceiverName    string `json:"receiver_name" binding:"required"`
	ReceiverPhone   string `json:"receiver_phone" binding:"required"`
}

// UpdateAddressByMobileAndBatch 根据手机号和波次修改地址
func (suc *SnowUserController) UpdateAddressByMobileAndBatch(c *gin.Context) {
	var request UpdateAddressByMobileAndBatchRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	// 转换抽奖波次为字符串
	drawBatchStr := strconv.Itoa(request.DrawBatch)

	// 查询所有SnowUser
	var snowUsers []models.SnowUser
	if err := db.DB.Find(&snowUsers).Error; err != nil {
		log.Printf("查询SnowUser列表失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询用户信息失败"})
		return
	}

	// 查找匹配的用户
	var matchedUser *models.SnowUser
	for i := range snowUsers {
		user := &snowUsers[i]
		// 检查MobileBatchMap或MobileBatch JSON字段
		var mobileBatchMap map[string]int
		if user.MobileBatchMap != nil {
			mobileBatchMap = user.MobileBatchMap
		} else if user.MobileBatch != "" {
			if err := json.Unmarshal([]byte(user.MobileBatch), &mobileBatchMap); err != nil {
				continue
			}
		} else {
			continue
		}

		// 检查当前波次的手机号是否匹配
		if mobileStr, exists := mobileBatchMap[drawBatchStr]; exists {
			if strconv.Itoa(mobileStr) == request.Mobile {
				matchedUser = user
				break
			}
		}
	}

	if matchedUser == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "未找到匹配的用户信息",
		})
		return
	}

	// 更新地址
	matchedUser.Province = request.Province
	matchedUser.City = request.City
	matchedUser.County = request.County
	matchedUser.DetailedAddress = request.DetailedAddress
	matchedUser.ReceiverName = request.ReceiverName
	matchedUser.ReceiverPhone = request.ReceiverPhone
	db.DB.Save(matchedUser)

	// 同时更新 SnowLotteryDraw 的 WinnersList 中的地址信息
	// 查询对应的抽奖活动
	var lotteryDraw models.SnowLotteryDraw
	if err := db.DB.Where("draw_batch = ?", request.DrawBatch).First(&lotteryDraw).Error; err != nil {
		log.Printf("查询抽奖活动失败: %v", err)
		// 继续执行，不影响地址更新结果
	} else {
		// 解析 WinnersList
		var winnersList []map[string]interface{}
		if lotteryDraw.WinnersList != "" {
			if err := json.Unmarshal([]byte(lotteryDraw.WinnersList), &winnersList); err != nil {
				log.Printf("解析 WinnersList 失败: %v", err)
				// 继续执行，不影响地址更新结果
			} else {
				// 查找对应的中奖记录并更新地址信息
				for i, winner := range winnersList {
					if winnerMobile, ok := winner["mobile"].(float64); ok {
						// 将手机号转换为字符串进行比较
						if strconv.Itoa(int(winnerMobile)) == request.Mobile {
							// 更新地址信息
							winnersList[i]["province"] = request.Province
							winnersList[i]["city"] = request.City
							winnersList[i]["county"] = request.County
							winnersList[i]["detailed_address"] = request.DetailedAddress
							winnersList[i]["receiver_name"] = request.ReceiverName
							winnersList[i]["receiver_phone"] = request.ReceiverPhone
							break
						}
					}
				}

				// 更新 WinnersList
				updatedWinnersList, err := json.Marshal(winnersList)
				if err != nil {
					log.Printf("序列化 WinnersList 失败: %v", err)
					// 继续执行，不影响地址更新结果
				} else {
					lotteryDraw.WinnersList = string(updatedWinnersList)
					if err := db.DB.Save(&lotteryDraw).Error; err != nil {
						log.Printf("保存 WinnersList 失败: %v", err)
						// 继续执行，不影响地址更新结果
					}
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "地址更新成功",
	})
}

// QueryAddressByMobileAndBatchRequest 根据手机号和波次查询地址请求结构体
type QueryAddressByMobileAndBatchRequest struct {
	Mobile    string `json:"mobile" binding:"required"`
	DrawBatch int    `json:"draw_batch" binding:"required"`
}

// QueryAddressByMobileAndBatch 根据手机号和波次查询地址
func (suc *SnowUserController) QueryAddressByMobileAndBatch(c *gin.Context) {
	var request QueryAddressByMobileAndBatchRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	// 转换抽奖波次为字符串
	drawBatchStr := strconv.Itoa(request.DrawBatch)

	// 查询所有SnowUser
	var snowUsers []models.SnowUser
	if err := db.DB.Find(&snowUsers).Error; err != nil {
		log.Printf("查询SnowUser列表失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询用户信息失败"})
		return
	}

	// 查找匹配的用户
	var matchedUser *models.SnowUser
	for i := range snowUsers {
		user := &snowUsers[i]
		// 检查MobileBatchMap或MobileBatch JSON字段
		var mobileBatchMap map[string]int
		if user.MobileBatchMap != nil {
			mobileBatchMap = user.MobileBatchMap
		} else if user.MobileBatch != "" {
			if err := json.Unmarshal([]byte(user.MobileBatch), &mobileBatchMap); err != nil {
				continue
			}
		} else {
			continue
		}

		// 检查当前波次的手机号是否匹配
		if mobileStr, exists := mobileBatchMap[drawBatchStr]; exists {
			if strconv.Itoa(mobileStr) == request.Mobile {
				matchedUser = user
				break
			}
		}
	}

	if matchedUser == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "未找到匹配的用户信息",
		})
		return
	}

	// 用户存在，返回地址信息
	c.JSON(http.StatusOK, gin.H{
		"success":             true,
		"user_id":             matchedUser.UserID,
		"province":            matchedUser.Province,
		"city":                matchedUser.City,
		"county":              matchedUser.County,
		"detailed_address":    matchedUser.DetailedAddress,
		"receiver_name":       matchedUser.ReceiverName,
		"receiver_phone":      matchedUser.ReceiverPhone,
		"verification_status": matchedUser.VerificationStatus,
	})
}

// VerifyDrawEligibility 验证用户抽奖资格
func (suc *SnowUserController) VerifyDrawEligibility(c *gin.Context) {
	var request VerifyDrawEligibilityRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	var user models.SnowUser
	result := db.DB.Where("mobile = ?", request.Mobile).First(&user)
	if result.Error != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "用户不存在",
		})
		return
	}

	// 根据是否指定波次返回不同结果
	if request.DrawBatch != "" {
		// 由于 EligibilityStatus 现在是 JSON 字符串，我们需要检查 AfterFind 钩子是否已将其解析到 EligibilityStatusMap
		if user.EligibilityStatusMap != nil {
			eligible, exists := user.EligibilityStatusMap[request.DrawBatch]
			if !exists {
				c.JSON(http.StatusOK, gin.H{
					"success": false,
					"message": "暂无该波次抽奖资格",
				})
			} else {
				c.JSON(http.StatusOK, gin.H{
					"success": eligible,
					"message": map[bool]string{true: "具备抽奖资格", false: "不具备抽奖资格"}[eligible],
				})
			}
		} else {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "暂无该波次抽奖资格",
			})
		}
	} else {
		// 返回所有波次的资格
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"eligibility_status": user.EligibilityStatusMap,
			},
		})
	}
}

// ParticipateDraw 参与抽奖
// generateUniqueCode 生成四位不重复的字母数字码
func generateUniqueCode() (string, error) {
	// 字符集：大小写字母 + 数字，排除容易混淆的字符
	charset := "ABCDEFGHJKMNPQRSTUVWXYZabcdefghjkmnpqrstuvwxyz23456789"
	charsetLength := big.NewInt(int64(len(charset)))
	maxAttempts := 1000 // 最大尝试次数，避免无限循环

	for attempt := 0; attempt < maxAttempts; attempt++ {
		result := make([]byte, 4)
		for i := 0; i < 4; i++ {
			randomIndex, err := rand.Int(rand.Reader, charsetLength)
			if err != nil {
				return "", err
			}
			result[i] = charset[randomIndex.Int64()]
		}

		// 生成的抽奖码
		drawCode := string(result)

		// 检查抽奖码是否已存在于SnowSuccessUser表
		var successUserCount int64
		// 使用精确的JSON查询匹配完整的抽奖码值
		db.DB.Model(&models.SnowSuccessUser{}).Where("success_code LIKE ? OR success_code LIKE ? OR success_code LIKE ?",
			"%\""+drawCode+"\"%", // 匹配 "code" 格式
			"%:'"+drawCode+"'%",  // 匹配 'code' 格式
			"%:"+drawCode+",%",   // 匹配 :code, 格式
		).Count(&successUserCount)
		if successUserCount > 0 {
			// 抽奖码已存在，继续尝试
			continue
		}

		// 检查抽奖码是否已存在于SnowUser表
		var snowUserCount int64
		db.DB.Model(&models.SnowUser{}).Where("success_code LIKE ? OR success_code LIKE ? OR success_code LIKE ?",
			"%\""+drawCode+"\"%", // 匹配 "code" 格式
			"%:'"+drawCode+"'%",  // 匹配 'code' 格式
			"%:"+drawCode+",%",   // 匹配 :code, 格式
		).Count(&snowUserCount)
		if snowUserCount > 0 {
			// 抽奖码已存在，继续尝试
			continue
		}

		// 抽奖码不存在，返回
		return drawCode, nil
	}

	// 超过最大尝试次数，返回错误
	return "", fmt.Errorf("无法生成唯一的抽奖码，已尝试%d次", maxAttempts)
}

func (suc *SnowUserController) ParticipateDrawByMobile(c *gin.Context) {
	var request ParticipateDrawByMobileRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	// 查询用户
	var user models.SnowUser
	result := db.DB.Where("user_id = ?", request.UserID).First(&user)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	// 注：根据需求，参与抽奖时不再验证手机号是否匹配

	// 校验验证码是否正确
	if user.VerificationCode != request.VerificationCode {
		c.JSON(http.StatusBadRequest, gin.H{"error": "验证码错误"})
		return
	}

	// 检查验证码是否过期
	if time.Now().After(user.VerificationCodeExpire) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "验证码已过期"})
		return
	}

	// 新增验证1：根据抽奖波次从SnowLotteryDraw中找到指定DrawBatch的抽奖信息
	var lotteryDraw models.SnowLotteryDraw
	lotteryResult := db.DB.Where("draw_batch = ?", request.DrawBatch).First(&lotteryDraw)
	if lotteryResult.Error != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "抽奖波次不存在"})
		return
	}

	// 新增验证：检查用户是否在所有轮次的中奖名单中
	var allDraws []models.SnowLotteryDraw
	if err := db.DB.Find(&allDraws).Error; err != nil {
		fmt.Printf("查询所有抽奖轮次失败: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询抽奖记录失败"})
		return
	}

	// 定义中奖者结构，用于解析WinnersList
	type Winner struct {
		DrawCode  string `json:"draw_code"`
		Mobile    int    `json:"mobile"`
		Nickname  string `json:"nickname"`
		OrderNum  string `json:"order_num"`
		PrizeName string `json:"prize_name"`
	}

	// 遍历所有抽奖轮次
	for _, draw := range allDraws {
		if draw.WinnersList == "" {
			continue
		}

		var winners []Winner
		if err := json.Unmarshal([]byte(draw.WinnersList), &winners); err != nil {
			fmt.Printf("解析中奖名单失败(轮次: %d): %v\n", draw.DrawBatch, err)
			continue
		}

		// 检查当前用户手机号是否在中奖名单中
		for _, winner := range winners {
			if winner.Mobile == request.Mobile {
				c.JSON(http.StatusBadRequest, gin.H{"error": "该用户已中奖过，无法参与该轮抽奖"})
				return
			}
		}
	}

	// 1. 验证手机号是否在snow_success_user中存在
	var successUser models.SnowSuccessUser
	// 直接通过手机号查询（修复表名引用问题）
	existResult := db.DB.Where("mobile = ?", request.Mobile).First(&successUser)
	if existResult.Error != nil {
		// 用户不存在于SnowSuccessUser，需要查询会员信息
		fmt.Printf("用户不存在于SnowSuccessUser，查询会员信息: %d\n", request.Mobile)

		// 调用vip包直接获取用户会员信息 - 将int类型转换为string
		mobileStr := strconv.Itoa(request.Mobile)
		vipInfo, customerInfo, err := vip.GetUserVipLevel(mobileStr)
		if err != nil {
			fmt.Printf("获取会员信息失败: %v\n", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "该用户没有资格，请联系下单店铺客服进行参与"})
			return
		}

		// 检查是否为森林会员（level_value为4）
		if levelValue, ok := vipInfo["level_value"].(float64); ok && levelValue == 4 {
			fmt.Printf("用户是森林会员，保存会员信息: %s\n", mobileStr)

			// 创建新用户
			successUser = models.SnowSuccessUser{
				Mobile: request.Mobile,
			}

			// 从会员信息中获取昵称（如果有）
			if customerInfo != nil {
				if nickname, ok := customerInfo["name"].(string); ok {
					successUser.Nickname = nickname
				}
			}

			// 设置会员来源
			successUser.MemberSource = "vip_upgrade"

			// 保存新用户
			if err := db.DB.Create(&successUser).Error; err != nil {
				fmt.Printf("创建用户记录失败: %v\n", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "创建用户记录失败"})
				return
			}

			// 设置抽奖资格默认为{"1": true, "2": true, "3": true}
			defaultDrawEligibility := map[string]bool{
				"1": true,
				"2": true,
				"3": true,
			}

			if err := successUser.SetDrawEligibility(defaultDrawEligibility); err != nil {
				fmt.Printf("设置抽奖资格失败: %v\n", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "设置抽奖资格失败"})
				return
			}

			// 更新用户信息
			if err := db.DB.Save(&successUser).Error; err != nil {
				fmt.Printf("更新用户记录失败: %v\n", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "更新用户记录失败"})
				return
			}

			fmt.Printf("森林会员信息保存成功: %d\n", request.Mobile)
		} else if !ok {
			// vipInfo格式错误
			fmt.Printf("会员信息格式错误: %v\n", vipInfo)
			c.JSON(http.StatusBadRequest, gin.H{"error": "该用户没有资格，请联系下单店铺客服进行参与"})
			return
		} else {
			// 不是森林会员
			fmt.Printf("用户不是森林会员，不符合资格: %s\n", mobileStr)
			c.JSON(http.StatusBadRequest, gin.H{"error": "该用户没有资格，请联系下单店铺客服进行参与"})
			return
		}
	}

	// 2. 验证用户是否有当前波次的抽奖资格
	drawEligibility, err := successUser.GetDrawEligibility()
	if err != nil {
		// 如果解析失败，记录错误并继续验证
		fmt.Printf("解析DrawEligibility失败: %v\n", err)
		// 使用空map继续验证，确保即使解析失败也能正常检查
		drawEligibility = make(map[string]bool)
	}

	// 直接使用request.DrawBatch（已经是字符串类型）检查抽奖资格
	if eligible, exists := drawEligibility[request.DrawBatch]; !exists || !eligible {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该用户没有当前波次的抽奖资格"})
		return
	}

	// 不再验证订单号匹配，符合资格要求即可参与抽奖

	// 2. 验证该波次用户是否已参与
	var participationStatus map[string]bool
	if successUser.ParticipationStatus != "" {
		if err := json.Unmarshal([]byte(successUser.ParticipationStatus), &participationStatus); err != nil {
			// 如果解析失败，创建新的map
			participationStatus = make(map[string]bool)
		}
	} else {
		participationStatus = make(map[string]bool)
	}

	// 检查是否已经参与过该波次
	if participated, exists := participationStatus[request.DrawBatch]; exists && participated {
		fmt.Printf("用户ID=%v 尝试重复参与波次=%v 的抽奖，已阻止\n", successUser.UserID, request.DrawBatch)
		c.JSON(http.StatusBadRequest, gin.H{"error": "该波次已参与过抽奖"})
		return
	}

	// 不再设置MemberSource，因为不再需要平台信息

	// 生成四位不重复的字母数字抽奖码
	drawCode, err := generateUniqueCode()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成抽奖码失败"})
		return
	}

	// 更新参与状态 - 使用临时 map 字段
	if user.ParticipationStatusMap == nil {
		user.ParticipationStatusMap = make(map[string]bool)
	}
	if user.OrderNumbersMap == nil {
		user.OrderNumbersMap = make(map[string]string)
	}
	if user.DrawTimesMap == nil {
		user.DrawTimesMap = make(map[string]time.Time)
	}
	if user.SuccessCodeMap == nil {
		user.SuccessCodeMap = make(map[string]string)
	}
	user.ParticipationStatusMap[request.DrawBatch] = true
	// 不记录订单号，仅使用手机号参与
	user.OrderNumbersMap[request.DrawBatch] = ""
	user.DrawTimesMap[request.DrawBatch] = time.Now()
	// 保存抽奖码，格式为波次:抽奖码
	user.SuccessCodeMap[request.DrawBatch] = drawCode

	// 添加当前波次的手机号到MobileBatch
	user.MobileBatchMap[request.DrawBatch] = request.Mobile

	if err := db.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "参与抽奖失败"})
		return
	}

	// 更新SnowSuccessUser中的参与状态
	// 已在前面的验证步骤中查询到successUser，直接更新参与状态
	participationStatus[request.DrawBatch] = true

	// 记录参与状态更新
	fmt.Printf("更新用户参与状态: 用户ID=%v, 波次=%v, 状态=true\n", successUser.UserID, request.DrawBatch)

	updatedStatus, err := json.Marshal(participationStatus)
	if err != nil {
		// 记录错误但继续保存
		fmt.Printf("JSON序列化失败: %v\n", err)
	} else {
		successUser.ParticipationStatus = string(updatedStatus)
	}

	// 不更新OrderNUM字段，因为没有订单号

	// 同时给SnowSuccessUser添加抽奖码
	successCodeMap, err := successUser.GetSuccessCode()
	if err != nil {
		// 如果解析失败，创建新的map
		successCodeMap = make(map[string]string)
	}

	// 添加当前波次的抽奖码
	successCodeMap[request.DrawBatch] = drawCode

	// 将map序列化为JSON并保存
	if err := successUser.SetSuccessCode(successCodeMap); err != nil {
		fmt.Printf("设置抽奖码失败: %v\n", err)
	}

	// 保存更新
	db.DB.Save(&successUser)

	// 更新抽奖活动的参与者名单和参与人数
	// 复用前面已声明的lotteryDraw变量
	lotteryResult = db.DB.Where("draw_batch = ?", request.DrawBatch).First(&lotteryDraw)
	if lotteryResult.Error == nil {
		// 解析参与名单
		var participantsList []string
		if lotteryDraw.ParticipantsList != "" {
			if err := json.Unmarshal([]byte(lotteryDraw.ParticipantsList), &participantsList); err != nil {
				// 解析失败，初始化空列表
				participantsList = []string{}
			}
		} else {
			participantsList = []string{}
		}

		// 检查用户是否已在参与名单中
		userIDStr := fmt.Sprintf("%d", user.UserID)
		userExists := false
		for _, id := range participantsList {
			if id == userIDStr {
				userExists = true
				break
			}
		}

		// 如果用户不在列表中，添加用户ID并更新参与人数
		if !userExists {
			participantsList = append(participantsList, userIDStr)
			// 处理JSON序列化可能的错误
			participantsJSON, err := json.Marshal(participantsList)
			if err != nil {
				// 记录错误但继续处理，确保功能不中断
				fmt.Printf("JSON序列化参与者列表失败: %v\n", err)
				// 使用默认值避免程序崩溃
				participantsJSON = []byte("[]")
			}
			lotteryDraw.ParticipantsList = string(participantsJSON)
			lotteryDraw.ParticipantsCount = len(participantsList)

			// 详细日志记录当前状态
			fmt.Printf("[抽奖记录调试] 开始处理抽奖记录，用户ID: %d, DrawBatch: %s\n", request.UserID, request.DrawBatch)
			fmt.Printf("[抽奖记录调试] 更新前Record字段: '%s'\n", lotteryDraw.Record)

			// 1. 创建新的抽奖记录
			newRecord := map[string]interface{}{
				"participate": time.Now().Format("2006-01-02 15:04:05"),
				"nickname":    successUser.Nickname,
				"mobile":      request.Mobile,
				"draw_code":   drawCode,
			}
			fmt.Printf("[抽奖记录调试] 新创建的抽奖记录: %+v\n", newRecord)

			// 2. 解析现有的Record字段
			var records []map[string]interface{}
			records = []map[string]interface{}{} // 确保始终初始化

			if lotteryDraw.Record != "" {
				fmt.Printf("[抽奖记录调试] 尝试解析现有记录\n")
				if err := json.Unmarshal([]byte(lotteryDraw.Record), &records); err != nil {
					// 解析失败，记录错误并初始化空列表
					fmt.Printf("[抽奖记录调试] 解析现有抽奖记录失败: %v\n", err)
					records = []map[string]interface{}{}
				} else {
					fmt.Printf("[抽奖记录调试] 解析现有记录成功，记录数量: %d\n", len(records))
				}
			} else {
				fmt.Printf("[抽奖记录调试] Record字段为空，使用空列表\n")
			}

			// 3. 添加新记录到列表
			records = append(records, newRecord)
			fmt.Printf("[抽奖记录调试] 添加新记录后，总记录数: %d\n", len(records))

			// 4. 序列化为JSON并更新Record字段
			recordsJSON, err := json.Marshal(records)
			if err != nil {
				fmt.Printf("[抽奖记录调试] JSON序列化抽奖记录失败: %v\n", err)
			} else {
				lotteryDraw.Record = string(recordsJSON)
				fmt.Printf("[抽奖记录调试] 序列化后抽奖记录: '%s'\n", lotteryDraw.Record)
			}

			// 保存更新 - 使用事务确保数据一致性
			fmt.Printf("[抽奖记录调试] 开始保存更新到数据库\n")
			// 先验证lotteryDraw是否有有效的ID
			if lotteryDraw.ID == 0 {
				fmt.Printf("[抽奖记录调试] 错误: lotteryDraw.ID为0，无法保存\n")
			} else {
				// 使用事务更新
				tx := db.DB.Begin()
				if tx.Error != nil {
					fmt.Printf("[抽奖记录调试] 开启事务失败: %v\n", tx.Error)
				} else {
					// 直接更新Record字段
					if err := tx.Model(&models.SnowLotteryDraw{}).Where("id = ?", lotteryDraw.ID).Update("record", lotteryDraw.Record).Error; err != nil {
						tx.Rollback()
						fmt.Printf("[抽奖记录调试] 更新Record字段失败: %v\n", err)
					} else {
						// 同时更新ParticipantsList和ParticipantsCount
						if err := tx.Model(&models.SnowLotteryDraw{}).Where("id = ?", lotteryDraw.ID).Updates(map[string]interface{}{
							"participants_list":  lotteryDraw.ParticipantsList,
							"participants_count": lotteryDraw.ParticipantsCount,
						}).Error; err != nil {
							tx.Rollback()
							fmt.Printf("[抽奖记录调试] 更新参与者信息失败: %v\n", err)
						} else {
							if err := tx.Commit().Error; err != nil {
								fmt.Printf("[抽奖记录调试] 提交事务失败: %v\n", err)
							} else {
								fmt.Printf("[抽奖记录调试] 事务提交成功，抽奖记录已保存，DrawBatch: %s\n", request.DrawBatch)
							}
						}
					}
				}
			}
		}
	}

	// 返回成功响应，包含抽奖码和开奖时间
	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"message":       "参与抽奖成功",
		"draw_batch":    request.DrawBatch,
		"draw_code":     drawCode,
		"order_num":     "", // 新路由不需要订单号，返回空字符串
		"code_format":   fmt.Sprintf("%s:%s", request.DrawBatch, drawCode),
		"draw_end_time": lotteryDraw.DrawTime.Format("2006-01-02 15:04:05"),
	})
}

func (suc *SnowUserController) ParticipateDraw(c *gin.Context) {
	var request ParticipateDrawRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	// 查询用户
	var user models.SnowUser
	result := db.DB.Where("user_id = ?", request.UserID).First(&user)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	// 注：根据需求，参与抽奖时不再验证手机号是否匹配

	// 校验验证码是否正确
	if user.VerificationCode != request.VerificationCode {
		c.JSON(http.StatusBadRequest, gin.H{"error": "验证码错误"})
		return
	}

	// 检查验证码是否过期
	if time.Now().After(user.VerificationCodeExpire) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "验证码已过期"})
		return
	}

	// 新增验证1：根据抽奖波次从SnowLotteryDraw中找到指定DrawBatch的抽奖信息
	var lotteryDraw models.SnowLotteryDraw
	lotteryResult := db.DB.Where("draw_batch = ?", request.DrawBatch).First(&lotteryDraw)
	if lotteryResult.Error != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "抽奖波次不存在"})
		return
	}

	// 新增验证2：根据订单号在SnowOrderData中查询信息，与OriginalOnlineOrderNumber精确匹配
	var orderData models.SnowOrderData
	orderResult := db.DB.Where("original_online_order_number = ?", request.OrderNum).First(&orderData)
	if orderResult.Error != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "订单不存在"})
		return
	}

	// 验证订单是否符合抽奖轮次活动要求
	if orderData.PaymentDate == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该订单不符合该抽奖轮次活动要求"})
		return
	}

	// 为了确保时区一致性，将所有时间转换为UTC进行比较
	receiptTimeUTC := orderData.PaymentDate.UTC()
	beginTimeUTC := lotteryDraw.OrderBeginTime.UTC()
	endTimeUTC := lotteryDraw.OrderEndTime.UTC()

	// 添加详细日志，记录所有时间信息
	fmt.Printf("验证时间: 订单支付时间=%v, 开始时间=%v, 结束时间=%v\n",
		orderData.PaymentDate, lotteryDraw.OrderBeginTime, lotteryDraw.OrderEndTime)
	fmt.Printf("UTC时间比较: 订单支付时间=%v, 开始时间=%v, 结束时间=%v\n",
		receiptTimeUTC, beginTimeUTC, endTimeUTC)

	// 使用UTC时间进行比较
	if receiptTimeUTC.Before(beginTimeUTC) || receiptTimeUTC.After(endTimeUTC) {
		fmt.Printf("时间验证失败: %v 不在 %v 和 %v 之间\n", receiptTimeUTC, beginTimeUTC, endTimeUTC)
		c.JSON(http.StatusBadRequest, gin.H{"error": "该订单不符合该抽奖轮次活动要求"})
		return
	}

	// 1. 验证手机号是否在snow_success_user中存在
	var successUser models.SnowSuccessUser
	// 直接通过手机号查询（修复表名引用问题）
	existResult := db.DB.Where("mobile = ?", request.Mobile).First(&successUser)
	if existResult.Error != nil {
		// 用户不存在于SnowSuccessUser，需要查询会员信息
		fmt.Printf("用户不存在于SnowSuccessUser，查询会员信息: %d\n", request.Mobile)

		// 调用vip包直接获取用户会员信息 - 将int类型转换为string
		mobileStr := strconv.Itoa(request.Mobile)
		vipInfo, customerInfo, err := vip.GetUserVipLevel(mobileStr)
		if err != nil {
			fmt.Printf("获取会员信息失败: %v\n", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "该用户没有资格，请联系下单店铺客服进行参与"})
			return
		}

		// 检查是否为森林会员（level_value为4）
		if levelValue, ok := vipInfo["level_value"].(float64); ok && levelValue == 4 {
			fmt.Printf("用户是森林会员，保存会员信息: %s\n", mobileStr)

			// 创建新用户
			successUser = models.SnowSuccessUser{
				Mobile: request.Mobile,
			}

			// 从会员信息中获取昵称（如果有）
			if customerInfo != nil {
				if nickname, ok := customerInfo["name"].(string); ok {
					successUser.Nickname = nickname
				}
			}

			// 设置会员来源
			successUser.MemberSource = "vip_upgrade"

			// 保存新用户
			if err := db.DB.Create(&successUser).Error; err != nil {
				fmt.Printf("创建用户记录失败: %v\n", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "创建用户记录失败"})
				return
			}

			// 设置抽奖资格默认为{"1": true, "2": true, "3": true}
			defaultDrawEligibility := map[string]bool{
				"1": true,
				"2": true,
				"3": true,
			}

			if err := successUser.SetDrawEligibility(defaultDrawEligibility); err != nil {
				fmt.Printf("设置抽奖资格失败: %v\n", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "设置抽奖资格失败"})
				return
			}

			// 更新用户信息
			if err := db.DB.Save(&successUser).Error; err != nil {
				fmt.Printf("更新用户记录失败: %v\n", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "更新用户记录失败"})
				return
			}

			fmt.Printf("森林会员信息保存成功: %d\n", request.Mobile)
		} else if !ok {
			// vipInfo格式错误
			fmt.Printf("会员信息格式错误: %v\n", vipInfo)
			c.JSON(http.StatusBadRequest, gin.H{"error": "该用户没有资格，请联系下单店铺客服进行参与"})
			return
		} else {
			// 不是森林会员
			fmt.Printf("用户不是森林会员，不符合资格: %s\n", mobileStr)
			c.JSON(http.StatusBadRequest, gin.H{"error": "该用户没有资格，请联系下单店铺客服进行参与"})
			return
		}
	}

	// 2. 验证用户是否有当前波次的抽奖资格
	drawEligibility, err := successUser.GetDrawEligibility()
	if err != nil {
		// 如果解析失败，记录错误并继续验证
		fmt.Printf("解析DrawEligibility失败: %v\n", err)
		// 使用空map继续验证，确保即使解析失败也能正常检查
		drawEligibility = make(map[string]bool)
	}

	// 直接使用request.DrawBatch（已经是字符串类型）检查抽奖资格
	if eligible, exists := drawEligibility[request.DrawBatch]; !exists || !eligible {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该用户没有当前波次的抽奖资格"})
		return
	}

	// 不再验证订单号匹配，符合资格要求即可参与抽奖

	// 新增验证：在一个轮次中订单号不能重复参与抽奖
	// 查询是否有其他用户在当前波次使用了相同的订单号
	var existingUser models.SnowUser
	existOrderResult := db.DB.Where("order_numbers LIKE ? AND user_id != ?", "%"+request.OrderNum+"%", request.UserID).First(&existingUser)
	if existOrderResult.Error == nil {
		// 如果找到了其他用户使用了相同订单号，检查是否在当前波次使用
		if existingUser.OrderNumbersMap == nil && existingUser.OrderNumbers != "" {
			// 尝试解析OrderNumbers
			if err := json.Unmarshal([]byte(existingUser.OrderNumbers), &existingUser.OrderNumbersMap); err != nil {
				existingUser.OrderNumbersMap = make(map[string]string)
			}
		}
		if existingUser.OrderNumbersMap != nil {
			if orderNum, exists := existingUser.OrderNumbersMap[request.DrawBatch]; exists && orderNum == request.OrderNum {
				fmt.Printf("订单号重复：波次=%v, 订单号=%v, 已被用户ID=%v使用\n", request.DrawBatch, request.OrderNum, existingUser.UserID)
				c.JSON(http.StatusBadRequest, gin.H{"error": "该订单号已在当前波次参与过抽奖"})
				return
			}
		}
	}

	// 2. 验证该波次用户是否已参与
	var participationStatus map[string]bool
	if successUser.ParticipationStatus != "" {
		if err := json.Unmarshal([]byte(successUser.ParticipationStatus), &participationStatus); err != nil {
			// 如果解析失败，创建新的map
			participationStatus = make(map[string]bool)
		}
	} else {
		participationStatus = make(map[string]bool)
	}
	// 检查是否已经参与过该波次
	if participated, exists := participationStatus[request.DrawBatch]; exists && participated {
		fmt.Printf("用户ID=%v 尝试重复参与波次=%v 的抽奖，已阻止\n", successUser.UserID, request.DrawBatch)
		c.JSON(http.StatusBadRequest, gin.H{"error": "该波次已参与过抽奖"})
		return
	}

	// 不再设置MemberSource，因为不再需要平台信息

	// 生成四位不重复的字母数字抽奖码
	drawCode, err := generateUniqueCode()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成抽奖码失败"})
		return
	}

	// 更新参与状态 - 使用临时 map 字段
	if user.ParticipationStatusMap == nil {
		user.ParticipationStatusMap = make(map[string]bool)
	}
	if user.OrderNumbersMap == nil {
		user.OrderNumbersMap = make(map[string]string)
	}
	if user.DrawTimesMap == nil {
		user.DrawTimesMap = make(map[string]time.Time)
	}
	if user.SuccessCodeMap == nil {
		user.SuccessCodeMap = make(map[string]string)
	}
	user.ParticipationStatusMap[request.DrawBatch] = true
	// 直接使用请求中的订单号
	user.OrderNumbersMap[request.DrawBatch] = request.OrderNum
	user.DrawTimesMap[request.DrawBatch] = time.Now()
	// 保存抽奖码，格式为波次:抽奖码
	user.SuccessCodeMap[request.DrawBatch] = drawCode

	// 添加当前波次的手机号到MobileBatch
	user.MobileBatchMap[request.DrawBatch] = request.Mobile

	if err := db.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "参与抽奖失败"})
		return
	}

	// 更新SnowSuccessUser中的参与状态
	// 已在前面的验证步骤中查询到successUser，直接更新参与状态
	participationStatus[request.DrawBatch] = true

	// 记录参与状态更新
	fmt.Printf("更新用户参与状态: 用户ID=%v, 波次=%v, 状态=true\n", successUser.UserID, request.DrawBatch)

	updatedStatus, err := json.Marshal(participationStatus)
	if err != nil {
		// 记录错误但继续保存
		fmt.Printf("JSON序列化失败: %v\n", err)
	} else {
		successUser.ParticipationStatus = string(updatedStatus)
	}

	// 更新OrderNUM字段（JSON格式），维护所有波次的订单号
	// 获取现有订单号map
	orderNUMMap, err := successUser.GetOrderNUM()
	if err != nil {
		// 如果解析失败，创建新的map
		orderNUMMap = make(map[string]string)
	}

	// 添加或更新当前波次的订单号
	orderNUMMap[request.DrawBatch] = request.OrderNum

	// 将map序列化为JSON并保存
	if err := successUser.SetOrderNUM(orderNUMMap); err != nil {
		fmt.Printf("设置OrderNUM失败: %v\n", err)
	}

	// 同时给SnowSuccessUser添加抽奖码
	successCodeMap, err := successUser.GetSuccessCode()
	if err != nil {
		// 如果解析失败，创建新的map
		successCodeMap = make(map[string]string)
	}

	// 添加当前波次的抽奖码
	successCodeMap[request.DrawBatch] = drawCode

	// 将map序列化为JSON并保存
	if err := successUser.SetSuccessCode(successCodeMap); err != nil {
		fmt.Printf("设置抽奖码失败: %v\n", err)
	}

	// 保存更新
	db.DB.Save(&successUser)

	// 更新抽奖活动的参与者名单和参与人数
	// 复用前面已声明的lotteryDraw变量
	lotteryResult = db.DB.Where("draw_batch = ?", request.DrawBatch).First(&lotteryDraw)
	if lotteryResult.Error == nil {
		// 解析参与名单
		var participantsList []string
		if lotteryDraw.ParticipantsList != "" {
			if err := json.Unmarshal([]byte(lotteryDraw.ParticipantsList), &participantsList); err != nil {
				// 解析失败，初始化空列表
				participantsList = []string{}
			}
		} else {
			participantsList = []string{}
		}

		// 检查用户是否已在参与名单中
		userIDStr := fmt.Sprintf("%d", user.UserID)
		userExists := false
		for _, id := range participantsList {
			if id == userIDStr {
				userExists = true
				break
			}
		}

		// 如果用户不在列表中，添加用户ID并更新参与人数
		if !userExists {
			participantsList = append(participantsList, userIDStr)
			// 处理JSON序列化可能的错误
			participantsJSON, err := json.Marshal(participantsList)
			if err != nil {
				// 记录错误但继续处理，确保功能不中断
				fmt.Printf("JSON序列化参与者列表失败: %v\n", err)
				// 使用默认值避免程序崩溃
				participantsJSON = []byte("[]")
			}
			lotteryDraw.ParticipantsList = string(participantsJSON)
			lotteryDraw.ParticipantsCount = len(participantsList)

			// 详细日志记录当前状态
			fmt.Printf("[抽奖记录调试] 开始处理抽奖记录，用户ID: %d, DrawBatch: %s\n", request.UserID, request.DrawBatch)
			fmt.Printf("[抽奖记录调试] 更新前Record字段: '%s'\n", lotteryDraw.Record)

			// 1. 创建新的抽奖记录
			newRecord := map[string]interface{}{
				"participate": time.Now().Format("2006-01-02 15:04:05"),
				"nickname":    successUser.Nickname,
				"mobile":      request.Mobile,
				"draw_code":   drawCode,
			}
			fmt.Printf("[抽奖记录调试] 新创建的抽奖记录: %+v\n", newRecord)

			// 2. 解析现有的Record字段
			var records []map[string]interface{}
			records = []map[string]interface{}{} // 确保始终初始化

			if lotteryDraw.Record != "" {
				fmt.Printf("[抽奖记录调试] 尝试解析现有记录\n")
				if err := json.Unmarshal([]byte(lotteryDraw.Record), &records); err != nil {
					// 解析失败，记录错误并初始化空列表
					fmt.Printf("[抽奖记录调试] 解析现有抽奖记录失败: %v\n", err)
					records = []map[string]interface{}{}
				} else {
					fmt.Printf("[抽奖记录调试] 解析现有记录成功，记录数量: %d\n", len(records))
				}
			} else {
				fmt.Printf("[抽奖记录调试] Record字段为空，使用空列表\n")
			}

			// 3. 添加新记录到列表
			records = append(records, newRecord)
			fmt.Printf("[抽奖记录调试] 添加新记录后，总记录数: %d\n", len(records))

			// 4. 序列化为JSON并更新Record字段
			recordsJSON, err := json.Marshal(records)
			if err != nil {
				fmt.Printf("[抽奖记录调试] JSON序列化抽奖记录失败: %v\n", err)
			} else {
				lotteryDraw.Record = string(recordsJSON)
				fmt.Printf("[抽奖记录调试] 序列化后抽奖记录: '%s'\n", lotteryDraw.Record)
			}

			// 保存更新 - 使用事务确保数据一致性
			fmt.Printf("[抽奖记录调试] 开始保存更新到数据库\n")
			// 先验证lotteryDraw是否有有效的ID
			if lotteryDraw.ID == 0 {
				fmt.Printf("[抽奖记录调试] 错误: lotteryDraw.ID为0，无法保存\n")
			} else {
				// 使用事务更新
				tx := db.DB.Begin()
				if tx.Error != nil {
					fmt.Printf("[抽奖记录调试] 开启事务失败: %v\n", tx.Error)
				} else {
					// 直接更新Record字段
					if err := tx.Model(&models.SnowLotteryDraw{}).Where("id = ?", lotteryDraw.ID).Update("record", lotteryDraw.Record).Error; err != nil {
						tx.Rollback()
						fmt.Printf("[抽奖记录调试] 更新Record字段失败: %v\n", err)
					} else {
						// 同时更新ParticipantsList和ParticipantsCount
						if err := tx.Model(&models.SnowLotteryDraw{}).Where("id = ?", lotteryDraw.ID).Updates(map[string]interface{}{
							"participants_list":  lotteryDraw.ParticipantsList,
							"participants_count": lotteryDraw.ParticipantsCount,
						}).Error; err != nil {
							tx.Rollback()
							fmt.Printf("[抽奖记录调试] 更新参与者信息失败: %v\n", err)
						} else {
							if err := tx.Commit().Error; err != nil {
								fmt.Printf("[抽奖记录调试] 提交事务失败: %v\n", err)
							} else {
								fmt.Printf("[抽奖记录调试] 事务提交成功，抽奖记录已保存，DrawBatch: %s\n", request.DrawBatch)
							}
						}
					}
				}
			}

			// 重新查询验证是否保存成功
			var verifyDraw models.SnowLotteryDraw
			if err := db.DB.Where("id = ?", lotteryDraw.ID).First(&verifyDraw).Error; err != nil {
				fmt.Printf("[抽奖记录调试] 验证查询失败: %v\n", err)
			} else {
				fmt.Printf("[抽奖记录调试] 验证保存结果，更新后Record: '%s'\n", verifyDraw.Record)
			}
		}
	}

	// 返回成功响应，包含抽奖码、订单号和开奖时间
	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"message":       "参与抽奖成功",
		"draw_batch":    request.DrawBatch,
		"draw_code":     drawCode,
		"order_num":     request.OrderNum,
		"code_format":   fmt.Sprintf("%s:%s", request.DrawBatch, drawCode),
		"draw_end_time": lotteryDraw.DrawTime.Format("2006-01-02 15:04:05"),
	})
}

// DeleteDrawInfo 删除指定轮次抽奖信息
func (suc *SnowUserController) DeleteDrawInfo(c *gin.Context) {
	var request DeleteDrawInfoRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的JSON格式",
			"error":   err.Error(),
		})
		return
	}

	// 1. 查询SnowUser
	var user models.SnowUser
	if err := db.DB.Where("user_id = ?", request.UserID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "未找到用户信息",
		})
		return
	}

	// 2. 查询SnowSuccessUser（可选）- 调整为更宽松的查询策略
	var successUser models.SnowSuccessUser
	log.Printf("尝试查询SnowSuccessUser记录，用户ID: %v, 手机号: %v", request.UserID, user.Mobile)

	// 首先尝试通过UserID查询
	result := db.DB.Where("user_id = ?", request.UserID).First(&successUser)
	successUserFound := result.Error == nil

	// 如果通过UserID未找到，再尝试通过手机号查询
	if !successUserFound {
		log.Printf("通过UserID未找到SnowSuccessUser记录，尝试通过手机号查询")
		result = db.DB.Where("mobile = ?", user.Mobile).First(&successUser)
		successUserFound = result.Error == nil
	}

	if successUserFound {
		log.Printf("成功找到SnowSuccessUser记录: UserID=%v, Mobile=%v", successUser.UserID, successUser.Mobile)
	} else {
		log.Printf("未找到SnowSuccessUser记录，继续执行SnowUser操作")
	}

	// 3. 删除SnowUser中的抽奖码、参与状态和订单号
	// 3.1 删除抽奖码
	if user.SuccessCodeMap != nil {
		delete(user.SuccessCodeMap, request.DrawBatch)
	}

	// 3.2 删除参与状态
	if user.ParticipationStatusMap != nil {
		delete(user.ParticipationStatusMap, request.DrawBatch)
	}

	// 3.3 删除订单号
	log.Printf("删除SnowUser中的订单号，抽奖轮次: %v", request.DrawBatch)
	if user.OrderNumbersMap != nil {
		log.Printf("当前订单号: %+v", user.OrderNumbersMap)
		delete(user.OrderNumbersMap, request.DrawBatch)
		log.Printf("删除后的订单号: %+v", user.OrderNumbersMap)
	}

	// 3.3 保存SnowUser的更新
	if err := db.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "保存用户信息失败",
			"error":   err.Error(),
		})
		return
	}

	// 4. 如果找到SnowSuccessUser记录，则删除其中的抽奖码和订单号
	if successUserFound {
		log.Printf("开始删除SnowSuccessUser中的抽奖码，抽奖轮次: %v", request.DrawBatch)
		successCodeMap, err := successUser.GetSuccessCode()
		if err != nil {
			// 如果解析失败，初始化新的map
			log.Printf("解析抽奖码失败: %v，重新初始化map", err)
			successCodeMap = make(map[string]string)
		} else {
			log.Printf("当前抽奖码: %+v", successCodeMap)
		}
		delete(successCodeMap, request.DrawBatch)
		log.Printf("删除后的抽奖码: %+v", successCodeMap)
		if err := successUser.SetSuccessCode(successCodeMap); err != nil {
			log.Printf("更新成功用户抽奖码失败: %v", err)
		} else {
			log.Printf("成功更新SnowSuccessUser中的抽奖码")
		}

		// 4.1 删除订单号
		log.Printf("开始删除SnowSuccessUser中的订单号，抽奖轮次: %v", request.DrawBatch)
		orderNUMMap, err := successUser.GetOrderNUM()
		if err != nil {
			// 如果解析失败，初始化新的map
			log.Printf("解析订单号失败: %v，重新初始化map", err)
			orderNUMMap = make(map[string]string)
		} else {
			log.Printf("当前订单号: %+v", orderNUMMap)
		}
		delete(orderNUMMap, request.DrawBatch)
		log.Printf("删除后的订单号: %+v", orderNUMMap)
		if err := successUser.SetOrderNUM(orderNUMMap); err != nil {
			log.Printf("更新成功用户订单号失败: %v", err)
		} else {
			log.Printf("成功更新SnowSuccessUser中的订单号")
		}
	}

	// 如果找到SnowSuccessUser记录，则继续处理参与状态和中奖状态
	if successUserFound {
		// 5. 删除SnowSuccessUser中的参与状态
		log.Printf("开始处理SnowSuccessUser中的参与状态，抽奖轮次: %v", request.DrawBatch)
		participationStatusMap := make(map[string]bool)

		// 尝试解析现有的参与状态
		if successUser.ParticipationStatus != "" && successUser.ParticipationStatus != "{}" {
			log.Printf("当前参与状态: %v", successUser.ParticipationStatus)
			if err := json.Unmarshal([]byte(successUser.ParticipationStatus), &participationStatusMap); err != nil {
				// 解析失败，记录错误并使用空map
				log.Printf("解析参与状态失败: %v", err)
				participationStatusMap = make(map[string]bool)
			} else {
				log.Printf("解析后的参与状态: %+v", participationStatusMap)
			}
		}

		// 删除指定轮次的参与状态
		delete(participationStatusMap, request.DrawBatch)
		log.Printf("删除后的参与状态: %+v", participationStatusMap)

		// 序列化为JSON并更新
		participationStatusJSON, err := json.Marshal(participationStatusMap)
		if err != nil {
			log.Printf("序列化参与状态失败: %v", err)
			// 不返回错误，继续执行
		} else {
			successUser.ParticipationStatus = string(participationStatusJSON)
			log.Printf("更新后的参与状态JSON: %v", successUser.ParticipationStatus)
		}

		// 6. 更新中奖状态为false
		log.Printf("开始更新中奖状态，抽奖轮次: %v", request.DrawBatch)
		winningStatus, err := successUser.GetWinningStatus()
		if err != nil {
			// 如果解析失败，初始化新的map
			log.Printf("解析中奖状态失败: %v", err)
			winningStatus = make(map[string]bool)
		} else {
			log.Printf("当前中奖状态: %+v", winningStatus)
		}
		winningStatus[request.DrawBatch] = false
		log.Printf("更新后的中奖状态: %+v", winningStatus)
		if err := successUser.SetWinningStatus(winningStatus); err != nil {
			log.Printf("更新中奖状态失败: %v", err)
		} else {
			log.Printf("成功更新中奖状态")
		}

		// 保存SnowSuccessUser的更新
		log.Printf("开始保存SnowSuccessUser的更新")
		if err := db.DB.Save(&successUser).Error; err != nil {
			log.Printf("保存成功用户信息失败: %v", err)
		} else {
			log.Printf("成功保存SnowSuccessUser的更新")
		}
	}

	// 返回最终成功响应
	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"message":    "删除指定轮次抽奖信息成功",
		"user_id":    request.UserID,
		"draw_batch": request.DrawBatch,
	})
}

// UpdateUserInfo 更新用户信息
func (suc *SnowUserController) UpdateUserInfo(c *gin.Context) {
	// 获取表单数据
	userIDStr := c.PostForm("user_id")
	nickname := c.PostForm("nickname")
	println(nickname)
	println(userIDStr)

	// 验证用户ID
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的用户ID",
		})
		return
	}

	// 查询用户
	var user models.SnowUser
	if err := db.DB.Where("user_id = ?", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "用户不存在",
		})
		return
	}

	// 处理头像文件上传
	var avatarURL string
	header, err := c.FormFile("avatar")
	// 添加调试日志
	if err != nil {
		fmt.Printf("获取头像文件失败: %v\n", err)
	} else if header != nil {
		fmt.Printf("获取到头像文件: %s, 大小: %d bytes\n", header.Filename, header.Size)
		// 获取文件
		file, err := header.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "打开头像文件失败: " + err.Error(),
			})
			return
		}
		defer file.Close()
		// 保存文件到用户头像目录
		directory := "user_avatars"
		filename := utils.GenerateUniqueFilename(header.Filename)
		fmt.Printf("生成唯一文件名: %s\n", filename)

		// 确保目录存在
		fullDir := filepath.Join("./media", directory)
		fmt.Printf("保存目录: %s\n", fullDir)
		if err := os.MkdirAll(fullDir, 0755); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "创建目录失败: " + err.Error(),
			})
			return
		}

		// 构建保存路径
		savePath := filepath.Join(fullDir, filename)
		fmt.Printf("保存路径: %s\n", savePath)

		// 使用gin的SaveUploadedFile保存文件
		if err := c.SaveUploadedFile(header, savePath); err != nil {
			fmt.Printf("保存文件失败: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "保存头像失败: " + err.Error(),
			})
			return
		}

		// 验证文件是否成功保存
		if _, err := os.Stat(savePath); os.IsNotExist(err) {
			fmt.Printf("文件保存后不存在: %s\n", savePath)
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "文件保存后验证失败: 文件不存在",
			})
			return
		}
		fmt.Printf("文件保存成功: %s\n", savePath)

		// 构建完整的图片URL
		// 使用正斜杠构建路径，避免Windows系统上的反斜杠问题
		imagePath := directory + "/" + filename
		proto := utils.GetRequestProto(c)
		baseURL := fmt.Sprintf("%s://%s", proto, c.Request.Host)
		avatarURL = utils.BuildFullImageURL(baseURL, imagePath, "media")
		fmt.Printf("构建的头像URL: %s\n", avatarURL)
	}

	// 更新用户信息
	if nickname != "" {
		user.Nickname = nickname
	}
	if avatarURL != "" {
		user.AvatarURL = avatarURL
	}

	// 保存更新
	if err := db.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "更新用户信息失败",
			"error":   err.Error(),
		})
		return
	}

	// 返回更新后的用户信息
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "用户信息更新成功",
		"data": gin.H{
			"user_id":    user.UserID,
			"nickname":   user.Nickname,
			"avatar_url": user.AvatarURL,
		},
	})
}
