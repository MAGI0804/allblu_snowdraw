// 显示文件头部内容
package controllers

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"django_to_go/config"
	"django_to_go/db"
	"django_to_go/models"
	"django_to_go/utils"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// UserController 用户控制器

type UserController struct{}
type UserQueryRequest struct {
	UserId int `json:"user_id"`
}

// UserRegistration 用户注册
func (uc *UserController) UserRegistration(c *gin.Context) {
	phone := c.GetHeader("X-Phone")
	nickname := c.GetHeader("X-Nickname")
	password := c.GetHeader("X-Password")

	// 验证必要的请求头
	if phone == "" || nickname == "" || password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少必要请求头"})
		return
	}

	// 验证手机号格式
	phoneRegex := regexp.MustCompile(`^1[3-9]\d{9}$`)
	if !phoneRegex.MatchString(phone) {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "手机号格式错误"})
		return
	}

	// 检查用户是否已存在
	var existingUser models.User
	if err := db.DB.Where("mobile = ?", phone).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "手机号已被注册"})
		return
	}

	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("密码加密失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
		return
	}

	// 创建新用户
	now := time.Now()
	user := models.User{
		Mobile:           phone,
		Nickname:         nickname,
		Password:         string(hashedPassword),
		RegistrationDate: now,
		IsActive:         true,
		IsStaff:          false,
	}

	if err := db.DB.Create(&user).Error; err != nil {
		log.Printf("创建用户失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"user_id":           user.UserID,
		"registration_time": user.RegistrationDate.Format("2006-01-02 15:04:05"),
	})
}

// UserQuery 查询用户信息
func (uc *UserController) UserQuery(c *gin.Context) {
	// 解析请求体
	var req UserQueryRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求体格式"})
		return
	}

	// 查询用户
	var user models.User
	if err := db.DB.Where("user_id = ?", int(req.UserId)).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	// 构建响应数据
	responseData := make(map[string]interface{})
	responseData["user_id"] = user.UserID
	responseData["openid"] = user.OpenID
	responseData["mobile"] = user.Mobile
	responseData["nickname"] = user.Nickname
	responseData["default_receiver"] = user.DefaultReceiver
	responseData["province"] = user.Province
	responseData["city"] = user.City
	responseData["county"] = user.County
	responseData["detailed_address"] = user.DetailedAddress
	responseData["membership_level"] = user.MembershipLevel
	responseData["registration_date"] = user.RegistrationDate.Format("2006-01-02 15:04:05")
	responseData["total_spending"] = user.TotalSpending
	responseData["remarks"] = user.Remarks
	responseData["is_active"] = user.IsActive
	responseData["is_staff"] = user.IsStaff

	// 处理头像URL
	if user.UserImg != "" {
		// 获取请求的协议，考虑反向代理环境
		proto := utils.GetRequestProto(c)
		baseURL := fmt.Sprintf("%s://%s", proto, c.Request.Host)
		// 构建完整的URL，确保只包含一个media前缀
		var fullImagePath string
		if strings.HasPrefix(user.UserImg, "media/") {
			// 如果user.UserImg已经以media/开头，只添加一个/前缀
			fullImagePath = "/" + user.UserImg
		} else if !strings.HasPrefix(user.UserImg, "/") {
			// 如果user.UserImg不以/开头，添加/media/前缀
			fullImagePath = "/media/" + user.UserImg
		} else {
			// 如果user.UserImg已经以/开头，直接使用
			fullImagePath = user.UserImg
		}
		responseData["user_img"] = utils.BuildFullImageURL(baseURL, fullImagePath)
	}

	c.JSON(http.StatusOK, responseData)
}

// UserModify 修改用户信息
func (uc *UserController) UserModify(c *gin.Context) {
	// 生成请求ID用于日志跟踪
	requestID := fmt.Sprintf("%d", time.Now().UnixNano())
	log.Printf("[UserModify] 请求开始，requestID: %s, Content-Type: %s", requestID, c.ContentType())

	// 检查是否为multipart/form-data请求
	if strings.Contains(c.ContentType(), "multipart/form-data") {
		// 处理文件上传请求
		log.Printf("[UserModify] 检测到multipart/form-data请求，requestID: %s", requestID)
		userIDStr := c.PostForm("user_id")
		if userIDStr == "" {
			log.Printf("[UserModify] 缺少user_id参数，requestID: %s", requestID)
			c.JSON(http.StatusBadRequest, gin.H{"error": "缺少user_id参数"})
			return
		}

		var userID int
		fmt.Sscanf(userIDStr, "%d", &userID)

		// 查询用户
		var user models.User
		if err := db.DB.Where("user_id = ?", userID).First(&user).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
			return
		}

		// 处理普通字段更新
		nickname := c.PostForm("nickname")
		if nickname != "" {
			user.Nickname = nickname
		}

		defaultReceiver := c.PostForm("default_receiver")
		if defaultReceiver != "" {
			user.DefaultReceiver = defaultReceiver
		}

		province := c.PostForm("province")
		if province != "" {
			user.Province = province
		}

		city := c.PostForm("city")
		if city != "" {
			user.City = city
		}

		county := c.PostForm("county")
		if county != "" {
			user.County = county
		}

		detailedAddress := c.PostForm("detailed_address")
		if detailedAddress != "" {
			user.DetailedAddress = detailedAddress
		}

		membershipLevel := c.PostForm("membership_level")
		if membershipLevel != "" {
			var level int
			fmt.Sscanf(membershipLevel, "%d", &level)
			user.MembershipLevel = level
		}

		remarks := c.PostForm("remarks")
		if remarks != "" {
			user.Remarks = remarks
		}

		// 处理头像上传
		log.Printf("[UserModify] 尝试获取上传文件，requestID: %s", requestID)
		file, header, err := c.Request.FormFile("user_img")
		if err != nil {
			log.Printf("[UserModify] 获取文件失败: %v, requestID: %s", err, requestID)
		} else if header != nil && file != nil {
			log.Printf("[UserModify] 成功获取文件: %s, 大小: %d字节, requestID: %s", header.Filename, header.Size, requestID)
			// 生成唯一文件名
			uniqueFilename := utils.GenerateUniqueFilename(header.Filename)

			// 定义保存路径 - 只保存相对路径，不包含media前缀
			savePath := fmt.Sprintf("user_avatars/%s", uniqueFilename)

			// 确保目录存在 - 使用与静态文件服务一致的相对路径
			log.Printf("[UserModify] 尝试创建目录: ./media/user_avatars, requestID: %s", requestID)
			if err := os.MkdirAll("./media/user_avatars", 0755); err != nil {
				log.Printf("[UserModify] 创建目录失败: %v, requestID: %s", err, requestID)
			}

			// 创建文件 - 使用相对路径
			fullPath := "./media/" + savePath
			log.Printf("[UserModify] 尝试创建文件: %s, requestID: %s", fullPath, requestID)
			dst, err := os.Create(fullPath)
			if err != nil {
				log.Printf("[UserModify] 创建文件失败: %v, requestID: %s", err, requestID)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "文件保存失败: " + err.Error()})
				return
			}
			defer dst.Close()

			// 复制文件内容
			log.Printf("[UserModify] 开始复制文件内容, requestID: %s", requestID)
			bytesWritten, err := io.Copy(dst, file)
			if err != nil {
				log.Printf("[UserModify] 文件复制失败: %v, requestID: %s", err, requestID)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "文件保存失败: " + err.Error()})
				return
			}
			log.Printf("[UserModify] 文件复制成功, 写入字节数: %d, requestID: %s", bytesWritten, requestID)

			// 更新用户头像路径
			user.UserImg = savePath
			log.Printf("[UserModify] 更新用户头像路径: %s, requestID: %s", savePath, requestID)
		}

		// 保存更新
		if err := db.DB.Save(&user).Error; err != nil {
			log.Printf("更新用户信息失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器处理失败"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "用户信息更新成功"})
		return
	}

	// 处理普通JSON请求
	var requestData map[string]interface{}
	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的JSON格式"})
		return
	}

	userIDFloat, ok := requestData["user_id"].(float64)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少user_id参数"})
		return
	}

	userID := int(userIDFloat)

	// 查询用户
	var user models.User
	if err := db.DB.Where("user_id = ?", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	// 检查是否尝试修改手机号
	if _, ok := requestData["mobile"]; ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "禁止修改手机号"})
		return
	}

	// 更新字段
	if nickname, ok := requestData["nickname"].(string); ok {
		user.Nickname = nickname
	}

	if defaultReceiver, ok := requestData["default_receiver"].(string); ok {
		user.DefaultReceiver = defaultReceiver
	}

	if province, ok := requestData["province"].(string); ok {
		user.Province = province
	}

	if city, ok := requestData["city"].(string); ok {
		user.City = city
	}

	if county, ok := requestData["county"].(string); ok {
		user.County = county
	}

	if detailedAddress, ok := requestData["detailed_address"].(string); ok {
		user.DetailedAddress = detailedAddress
	}

	if membershipLevel, ok := requestData["membership_level"].(float64); ok {
		user.MembershipLevel = int(membershipLevel)
	}

	if remarks, ok := requestData["remarks"].(string); ok {
		user.Remarks = remarks
	}

	// 保存更新
	if err := db.DB.Save(&user).Error; err != nil {
		log.Printf("更新用户信息失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器处理失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "用户信息更新成功"})
}

// UserGetID 根据手机号获取用户ID
func (uc *UserController) UserGetID(c *gin.Context) {
	mobile := c.Query("mobile")
	if mobile == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少mobile参数"})
		return
	}

	var user models.User
	if err := db.DB.Where("mobile = ?", mobile).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user_id": user.UserID})
}

// VerificationStatus 验证登录状态
func (uc *UserController) VerificationStatus(c *gin.Context) {
	var requestData map[string]interface{}
	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的JSON格式"})
		return
	}

	mobile, ok := requestData["mobile"].(string)
	password, ok2 := requestData["password"].(string)

	if !ok || !ok2 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":    "缺少必要参数",
			"required": []string{"mobile", "password"},
		})
		return
	}

	// 验证手机号格式
	phoneRegex := regexp.MustCompile(`^1[3-9]\d{9}$`)
	if !phoneRegex.MatchString(mobile) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "手机号格式错误"})
		return
	}

	// 查询用户
	var user models.User
	if err := db.DB.Where("mobile = ?", mobile).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "手机号未注册"})
		return
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "账号存在但密码错误"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "登录状态验证成功",
		"user_id": user.UserID,
	})
}

// WechatLogin 微信小程序登录
func (uc *UserController) WechatLogin(c *gin.Context) {
	var requestData map[string]interface{}
	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的JSON格式"})
		return
	}

	code, ok := requestData["code"].(string)
	if !ok || code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少code参数"})
		return
	}

	// 获取微信配置
	cfg := config.LoadConfig()

	// 调用微信API获取openid
	wxURL := fmt.Sprintf("%s?appid=%s&secret=%s&js_code=%s&grant_type=authorization_code",
		cfg.WechatConfig.LoginURL,
		cfg.WechatConfig.AppID,
		cfg.WechatConfig.AppSecret,
		code,
	)

	resp, err := http.Get(wxURL)
	if err != nil {
		log.Printf("微信API请求失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "微信登录失败"})
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("读取微信API响应失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "微信登录失败"})
		return
	}

	var wxResult map[string]interface{}
	if err := json.Unmarshal(body, &wxResult); err != nil {
		log.Printf("解析微信API响应失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "微信登录失败"})
		return
	}

	// 检查是否有错误
	if _, ok := wxResult["errcode"]; ok {
		errMsg := wxResult["errmsg"].(string)
		log.Printf("微信登录失败: %v", wxResult)
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("微信登录失败: %s", errMsg)})
		return
	}

	openid, ok := wxResult["openid"].(string)
	if !ok || openid == "" {
		log.Printf("微信返回数据中没有openid: %v", wxResult)
		c.JSON(http.StatusBadRequest, gin.H{"error": "微信登录失败，未获取到openid"})
		return
	}

	// 获取用户信息
	userInfo := make(map[string]interface{})
	if userInfoMap, ok := requestData["userInfo"].(map[string]interface{}); ok {
		userInfo = userInfoMap
	}

	// 解析昵称
	nickname := ""
	if nicknameVal, ok := userInfo["nickName"].(string); ok {
		nickname = nicknameVal
	} else if nicknameVal, ok := userInfo["nickname"].(string); ok {
		nickname = nicknameVal
	}

	if nickname == "" {
		// 如果没有昵称，生成默认昵称
		if len(openid) > 8 {
			nickname = "微信用户_" + openid[:8]
		} else {
			nickname = "微信用户_" + openid
		}
	}

	// 解析头像URL
	avatarURL := ""
	if avatarVal, ok := userInfo["avatarUrl"].(string); ok {
		avatarURL = avatarVal
	} else if avatarVal, ok := userInfo["avatar_url"].(string); ok {
		avatarURL = avatarVal
	}

	// 查询或创建用户
	var user models.User
	if err := db.DB.Where("openid = ?", openid).First(&user).Error; err != nil {
		// 用户不存在，创建新用户
		user = models.User{
			OpenID:           openid,
			Nickname:         nickname,
			UserImg:          avatarURL,
			RegistrationDate: time.Now(),
			IsActive:         true,
			IsStaff:          false,
		}

		if err := db.DB.Create(&user).Error; err != nil {
			log.Printf("创建用户失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
			return
		}
	} else {
		// 用户已存在，检查是否需要更新头像和昵称
		// 只有当数据库中的头像为空且微信提供了头像时才更新
		if user.UserImg == "" && avatarURL != "" {
			user.UserImg = avatarURL
		}
		// 只有当数据库中的昵称是默认值且微信提供了新昵称时才更新
		if strings.HasPrefix(user.Nickname, "微信用户_") && nickname != "" && !strings.HasPrefix(nickname, "微信用户_") {
			user.Nickname = nickname
		}
		// 更新最后登录时间
		user.LastLogin = time.Now()
		// 保存更新
		if err := db.DB.Save(&user).Error; err != nil {
			log.Printf("更新用户信息失败: %v", err)
		}
	}

	// 生成令牌
	accessToken, refreshToken, err := utils.GenerateTokens(user.UserID, cfg)
	if err != nil {
		log.Printf("生成令牌失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
		return
	}

	// 准备响应数据
	responseData := gin.H{
		"code":    200,
		"message": "登录成功",
		"data": gin.H{
			"token": gin.H{
				"access":  accessToken,
				"refresh": refreshToken,
			},
			"user_id":  user.UserID,
			"nickname": user.Nickname,
		},
	}

	// 如果有头像，返回完整的头像URL
	if user.UserImg != "" {
		// 获取请求的协议，考虑反向代理环境
		proto := utils.GetRequestProto(c)
		baseURL := fmt.Sprintf("%s://%s", proto, c.Request.Host)
		// 检查头像URL是否已经是完整URL，如果不是则构建完整URL
		if !strings.HasPrefix(user.UserImg, "http://") && !strings.HasPrefix(user.UserImg, "https://") {
			responseData["data"].(gin.H)["avatar_url"] = utils.BuildFullImageURL(baseURL, user.UserImg, "media")
		} else {
			responseData["data"].(gin.H)["avatar_url"] = user.UserImg
		}
	}

	c.JSON(http.StatusOK, responseData)
}

// AddData 添加用户数据
func (uc *UserController) AddData(c *gin.Context) {
	var requestData map[string]interface{}
	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的JSON格式"})
		return
	}

	// 检查必要参数
	userIdFloat, ok := requestData["user_id"].(float64)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少user_id参数"})
		return
	}

	dataType, ok := requestData["data_type"].(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少data_type参数"})
		return
	}

	dataValue, ok := requestData["data_value"].(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少data_value参数"})
		return
	}

	userID := int(userIdFloat)

	// 检查用户是否存在
	var user models.User
	if err := db.DB.Where("user_id = ?", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	// 创建用户数据
	userData := models.UserData{
		UserID:     userID,
		DataType:   dataType,
		DataValue:  dataValue,
		CreateTime: time.Now(),
	}

	if err := db.DB.Create(&userData).Error; err != nil {
		log.Printf("添加用户数据失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "用户数据添加成功",
		"data_id": userData.ID,
	})
}

// FindData 查询用户数据
func (uc *UserController) FindData(c *gin.Context) {
	var requestData map[string]interface{}
	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的JSON格式"})
		return
	}

	// 检查必要参数
	userIdFloat, ok := requestData["user_id"].(float64)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少user_id参数"})
		return
	}

	userID := int(userIdFloat)

	// 检查用户是否存在
	var user models.User
	if err := db.DB.Where("user_id = ?", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	// 查询条件
	query := db.DB.Where("user_id = ?", userID)

	// 如果指定了数据类型
	if dataType, ok := requestData["data_type"].(string); ok && dataType != "" {
		query = query.Where("data_type = ?", dataType)
	}

	// 分页参数
	page := 1
	pageSize := 10

	if pageFloat, ok := requestData["page"].(float64); ok {
		page = int(pageFloat)
	}

	if pageSizeFloat, ok := requestData["page_size"].(float64); ok {
		pageSize = int(pageSizeFloat)
	}

	// 查询数据
	var userDatas []models.UserData
	var total int64

	query.Model(&models.UserData{}).Count(&total)
	query.Order("create_time DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&userDatas)

	// 构建响应数据
	result := make([]map[string]interface{}, 0, len(userDatas))
	for _, data := range userDatas {
		item := map[string]interface{}{
			"id":          data.ID,
			"data_type":   data.DataType,
			"data_value":  data.DataValue,
			"create_time": data.CreateTime.Format("2006-01-02 15:04:05"),
		}
		result = append(result, item)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"list":      result,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// TokenObtainPair 获取JWT令牌 - 对应Django的TokenObtainPairView
func (uc *UserController) TokenObtainPair(c *gin.Context) {
	var requestData map[string]interface{}
	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的JSON格式"})
		return
	}

	mobile, ok := requestData["mobile"].(string)
	password, ok2 := requestData["password"].(string)

	if !ok || !ok2 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":    "缺少必要参数",
			"required": []string{"mobile", "password"},
		})
		return
	}

	// 验证手机号格式
	phoneRegex := regexp.MustCompile(`^1[3-9]\d{9}$`)
	if !phoneRegex.MatchString(mobile) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "手机号格式错误"})
		return
	}

	// 查询用户
	var user models.User
	if err := db.DB.Where("mobile = ?", mobile).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "手机号未注册"})
		return
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "账号存在但密码错误"})
		return
	}

	// 获取配置
	cfg := config.LoadConfig()

	// 生成令牌
	accessToken, refreshToken, err := utils.GenerateTokens(user.UserID, cfg)
	if err != nil {
		log.Printf("生成令牌失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
		return
	}

	// 返回与Django相同格式的响应
	c.JSON(http.StatusOK, gin.H{
		"access":  accessToken,
		"refresh": refreshToken,
	})
}

// TokenRefresh 刷新JWT令牌 - 对应Django的TokenRefreshView
func (uc *UserController) TokenRefresh(c *gin.Context) {
	var requestData map[string]interface{}
	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的JSON格式"})
		return
	}

	refreshToken, ok := requestData["refresh"].(string)
	if !ok || refreshToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少refresh参数"})
		return
	}

	// 获取配置
	cfg := config.LoadConfig()

	// 解析并验证刷新令牌
	newAccessToken, err := utils.RefreshAccessToken(refreshToken, cfg)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的刷新令牌"})
		return
	}

	// 返回与Django相同格式的响应
	c.JSON(http.StatusOK, gin.H{
		"access": newAccessToken,
	})
}
