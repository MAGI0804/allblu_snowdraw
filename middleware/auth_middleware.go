package middleware

import (
	"django_to_go/config"
	"django_to_go/db"
	"django_to_go/models"
	"django_to_go/utils"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

// JWTAuthMiddleware JWT认证中间件
func JWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string

		// 尝试从Authorization头获取token
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			// 检查token格式
			authParts := strings.SplitN(authHeader, " ", 2)
			if len(authParts) == 2 && authParts[0] == "Bearer" {
				tokenString = authParts[1]
			}
		}

		// 如果Authorization头中没有有效的token，尝试从URL参数access_token获取
		if tokenString == "" {
			tokenString = c.Query("access_token")
		}

		// 如果两种方式都没有获取到token，返回未授权
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization token is required, either in header or as access_token query parameter"})
			c.Abort()
			return
		}
		// 解析token
		cfg := config.LoadConfig()
		token, err := utils.ParseToken(tokenString, cfg)
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		// 提取用户信息
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			c.Abort()
			return
		}

		// 获取用户ID
		userIDStr, ok := claims["sub"].(string)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in token"})
			c.Abort()
			return
		}

		// 将用户ID存储到上下文中
		c.Set("userID", userIDStr)
		c.Next()
	}
}

// AccessTokenValidationMiddleware access_token验证中间件
// 除了特定豁免路径外，其他所有路径都需要验证access_token
func AccessTokenValidationMiddleware() gin.HandlerFunc {
	// 定义豁免路径列表
	exemptPaths := []string{
		"/admin/",
		"/static/",
		"/media/",
		"/access_token/get_token",
		"/access_token/get_ips",
	}

	return func(c *gin.Context) {
		// 检查当前路径是否在豁免列表中
		path := c.Request.URL.Path
		for _, exemptPath := range exemptPaths {
			if strings.HasPrefix(path, exemptPath) {
				// 豁免路径，直接通过
				c.Next()
				return
			}
		}

		// 从GET或POST参数中获取access_token
		tokenString := c.Query("access_token")
		if tokenString == "" {
			// 尝试从POST表单中获取
			tokenString = c.PostForm("access_token")
		}

		// 如果没有获取到token，返回未授权
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "Missing access token",
			})
			c.Abort()
			return
		}

		// 验证token是否存在且有效
		var token models.AccessToken
		if err := db.DB.Where("access_token = ?", tokenString).First(&token).Error; err != nil {
			// token不存在或无效
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "Invalid access token",
			})
			c.Abort()
			return
		}

		// 注释掉IP和token强绑定检查
		/*
		// 获取请求IP地址
		clientIP := c.ClientIP()
		// 如果X-Forwarded-For头存在，优先使用它
		xForwardedFor := c.GetHeader("X-Forwarded-For")
		if xForwardedFor != "" {
			// 提取第一个IP地址
			ips := strings.Split(xForwardedFor, ",")
			if len(ips) > 0 {
				clientIP = strings.TrimSpace(ips[0])
			}
		}

		// 验证IP是否与token绑定
		if token.IPAddress != clientIP {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "IP address does not match token",
			})
			c.Abort()
			return
		}
		*/

		// 验证通过，继续处理请求
		c.Next()
	}
}

// CORSMiddleware 跨域中间件
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

var (
	// 全局日志器实例
	accessLogger *utils.Logger
	loggerOnce   sync.Once
)

// 初始化日志器
func initLogger() {
	// 创建一个全局的访问日志记录器
	var err error
	accessLogger, err = utils.NewLogger("/app/logs", "access.log")
	if err != nil {
		fmt.Printf("初始化访问日志记录器失败: %v\n", err)
	}
}

// RequestLogMiddleware 请求日志中间件
func RequestLogMiddleware() gin.HandlerFunc {
	// 确保日志器只被初始化一次
	loggerOnce.Do(initLogger)

	return func(c *gin.Context) {
		// 获取客户端IP地址
		clientIP := c.ClientIP()

		// 记录请求信息和IP地址到文件
		if accessLogger != nil {
			if err := accessLogger.Access("IP: %s, 方法: %s, 路径: %s", clientIP, c.Request.Method, c.Request.URL.Path); err != nil {
				// 如果写入文件失败，继续打印到控制台
				fmt.Printf("[访问日志] IP: %s, 方法: %s, 路径: %s\n", clientIP, c.Request.Method, c.Request.URL.Path)
				fmt.Printf("写入日志文件失败: %v\n", err)
			}
		} else {
			// 如果日志器未初始化，继续打印到控制台
			fmt.Printf("[访问日志] IP: %s, 方法: %s, 路径: %s\n", clientIP, c.Request.Method, c.Request.URL.Path)
		}

		// 继续处理请求
		c.Next()
	}
}

// ErrorHandlerMiddleware 错误处理中间件
func ErrorHandlerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// 处理错误
		if len(c.Errors) > 0 {
			for _, _ = range c.Errors {
				// 这里可以根据需要添加错误处理逻辑
				// 例如记录错误日志、返回统一的错误格式等
			}
		}
	}
}
