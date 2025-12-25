package controllers

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"django_to_go/config"
	"django_to_go/db"
	"django_to_go/models"
	"django_to_go/utils"
	"golang.org/x/crypto/bcrypt"
)

// AccessTokenController 访问令牌控制器
type AccessTokenController struct{}

// TokenObtainPair 获取JWT令牌 - 对应Django的TokenObtainPairView
func (ac *AccessTokenController) TokenObtainPair(c *gin.Context) {
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
func (ac *AccessTokenController) TokenRefresh(c *gin.Context) {
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

// GetToken 获取访问令牌
func (ac *AccessTokenController) GetToken(c *gin.Context) {
	// 获取客户端IP地址，优先使用X-Forwarded-For
	xForwardedFor := c.GetHeader("X-Forwarded-For")
	var ipAddress string
	if xForwardedFor != "" {
		// 提取第一个IP地址
		ips := strings.Split(xForwardedFor, ",")
		if len(ips) > 0 {
			ipAddress = strings.TrimSpace(ips[0])
		} else {
			ipAddress = c.ClientIP()
		}
	} else {
		ipAddress = c.ClientIP()
	}

	// 确保IP不为空
	if ipAddress == "" {
		ipAddress = "unknown"
	}

	// 检查该IP是否已存在token
	var tokenObj models.AccessToken
	if err := db.DB.Where("ip_address = ?", ipAddress).First(&tokenObj).Error; err == nil {
		c.JSON(http.StatusOK, gin.H{
			"code":        200,
			"message":     "Token already exists for this IP",
			"access_token": tokenObj.AccessToken,
		})
		return
	}

	// 生成32位随机不重复token
	var accessToken string
	for {
		// 生成16字节随机数据，转为32位十六进制字符串
		randomBytes := make([]byte, 16)
		rand.Read(randomBytes)
		accessToken = hex.EncodeToString(randomBytes)
	
		// 检查token是否已存在
		if err := db.DB.Where("access_token = ?", accessToken).First(&tokenObj).Error; err != nil {
			// 不存在则使用此token
			break
		}
	}

	// 创建新token记录
	newToken := models.AccessToken{
		IPAddress:   ipAddress,
		AccessToken: accessToken,
		RegisterTime: time.Now(),
	}

	if err := db.DB.Create(&newToken).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "Database error: " + err.Error(),
		})
		return
	}
	// 注意：成功响应需要放在if代码块外面
	c.JSON(http.StatusCreated, gin.H{
		"code":        201,
		"message":     "Token generated successfully",
		"access_token": newToken.AccessToken,
	})
}

// GetIPs 获取所有唯一IP地址
func (ac *AccessTokenController) GetIPs(c *gin.Context) {
	// 检查请求方法是否为POST
	if c.Request.Method != "POST" {
		c.JSON(http.StatusMethodNotAllowed, gin.H{
			"code":    405,
			"message": "Method not allowed",
		})
		return
	}
	// 解析请求体数据
	var requestData struct {
		ShopName string `json:"shopname"`
	}

	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "Invalid JSON format",
		})
		return
	}

	// 验证shopname参数
	if requestData.ShopName != "youlan_kids" {
		c.JSON(http.StatusForbidden, gin.H{
			"code":    403,
			"message": "Invalid shopname",
		})
		return
	}

	// 获取所有唯一IP地址
	var ipAddresses []string
	if err := db.DB.Model(&models.AccessToken{}).Distinct().Pluck("ip_address", &ipAddresses).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "Server error: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":        200,
		"message":     "IP addresses retrieved successfully",
		"ip_addresses": ipAddresses,
	})
}