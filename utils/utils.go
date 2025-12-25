package utils

import (
	"crypto/rand"
	"django_to_go/config"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

// GenerateTokens 生成访问令牌和刷新令牌
func GenerateTokens(userID int, cfg config.Config) (string, string, error) {
	// 生成访问令牌
	expirationTime := time.Now().Add(time.Duration(cfg.JWTConfig.AccessTokenTTL) * time.Hour)
	claims := &jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(expirationTime),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		NotBefore: jwt.NewNumericDate(time.Now()),
		Subject:   fmt.Sprintf("%d", userID),
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedAccessToken, err := accessToken.SignedString([]byte(cfg.JWTConfig.SecretKey))
	if err != nil {
		return "", "", err
	}

	// 生成刷新令牌
	refreshExpirationTime := time.Now().Add(time.Duration(cfg.JWTConfig.RefreshTokenTTL) * time.Hour)
	refreshClaims := &jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(refreshExpirationTime),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		NotBefore: jwt.NewNumericDate(time.Now()),
		Subject:   fmt.Sprintf("%d", userID),
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	signedRefreshToken, err := refreshToken.SignedString([]byte(cfg.JWTConfig.SecretKey))
	if err != nil {
		return "", "", err
	}

	return signedAccessToken, signedRefreshToken, nil
}

// ParseToken 解析JWT令牌
func ParseToken(tokenString string, cfg config.Config) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// 验证签名算法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(cfg.JWTConfig.SecretKey), nil
	})

	return token, err
}

// RefreshAccessToken 只刷新访问令牌 - 用于TokenRefreshView
func RefreshAccessToken(refreshTokenString string, cfg config.Config) (string, error) {
	token, err := ParseToken(refreshTokenString, cfg)
	if err != nil {
		return "", err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return "", fmt.Errorf("invalid refresh token")
	}

	// 获取用户ID
	userIDStr, ok := claims["sub"].(string)
	if !ok {
		return "", fmt.Errorf("invalid user ID in token")
	}

	var userID int
	fmt.Sscanf(userIDStr, "%d", &userID)

	// 只生成新的访问令牌
	accessToken, _, err := GenerateTokens(userID, cfg)
	if err != nil {
		return "", err
	}

	return accessToken, nil
}

// RefreshToken 刷新访问令牌
func RefreshToken(refreshTokenString string, cfg config.Config) (string, string, error) {
	token, err := ParseToken(refreshTokenString, cfg)
	if err != nil {
		return "", "", err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return "", "", fmt.Errorf("invalid refresh token")
	}

	// 获取用户ID
	userIDStr, ok := claims["sub"].(string)
	if !ok {
		return "", "", fmt.Errorf("invalid user ID in token")
	}

	var userID int
	fmt.Sscanf(userIDStr, "%d", &userID)

	// 生成新的访问令牌和刷新令牌
	return GenerateTokens(userID, cfg)
}

// GenerateUniqueFilename 生成唯一的文件名
func GenerateUniqueFilename(originalFilename string) string {
	timestamp := time.Now().UnixNano()
	randomBytes := make([]byte, 8)
	_, err := rand.Read(randomBytes)
	if err != nil {
		// 如果生成随机数失败，只使用时间戳
		return fmt.Sprintf("%d_%s", timestamp, originalFilename)
	}

	randomStr := base64.URLEncoding.EncodeToString(randomBytes)
	// 移除base64中的特殊字符
	randomStr = removeSpecialChars(randomStr)

	return fmt.Sprintf("%d_%s_%s", timestamp, randomStr, originalFilename)
}

// removeSpecialChars 移除字符串中的特殊字符
func removeSpecialChars(s string) string {
	result := ""
	for _, char := range s {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') {
			result += string(char)
		}
	}
	return result
}

// IsValidPhone 验证手机号格式是否正确
func IsValidPhone(phone string) bool {
	// 简单验证：11位数字，以1开头
	if len(phone) != 11 {
		return false
	}

	for i, char := range phone {
		if i == 0 && char != '1' {
			return false
		}
		if char < '0' || char > '9' {
			return false
		}
	}

	return true
}

// FormatDateTime 格式化时间
func FormatDateTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

// ParseDateTime 解析时间字符串
func ParseDateTime(datetimeStr string) (time.Time, error) {
	return time.Parse("2006-01-02 15:04:05", datetimeStr)
}

// Pagination 分页辅助函数
func Pagination(pageNum, pageSize int) (int, int) {
	if pageNum <= 0 {
		pageNum = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	offset := (pageNum - 1) * pageSize
	return offset, pageSize
}

// GetRequestProto 获取请求的协议，考虑反向代理环境
// 参数:
// - c: Gin上下文
// 返回值:
// - 协议字符串 ("http" 或 "https")
func GetRequestProto(c *gin.Context) string {
	// 首先尝试从X-Forwarded-Proto头获取真实协议（用于反向代理环境）
	proto := c.Request.Header.Get("X-Forwarded-Proto")
	if proto != "" {
		return proto
	}
	// 如果没有X-Forwarded-Proto头，使用请求的协议
	if c.Request.URL.Scheme != "" {
		return c.Request.URL.Scheme
	}
	// 默认返回http
	return "http"
}

// BuildFullImageURL 构建完整的图片URL
// 参数:
// - baseURL: 基础URL，包含正确的协议前缀，如 "http://example.com" 或 "https://example.com"
// - imagePath: 图片路径
// - prefix: 可选的路径前缀，如 "media"
// 返回值:
// - 完整的图片URL字符串
// 注意：调用者需要确保baseURL包含正确的协议前缀（http://或https://）
func BuildFullImageURL(baseURL, imagePath string, prefix ...string) string {
	// 如果图片路径为空，直接返回空字符串
	if imagePath == "" {
		return ""
	}

	// 检查图片路径是否已经是完整URL
	if len(imagePath) >= 8 && (imagePath[:8] == "https://" || imagePath[:7] == "http://") {
		return imagePath
	}

	// 完全尊重传入的baseURL的协议，不做任何修改
	// 调用者应该已经确保baseURL包含正确的协议前缀
	// 这样可以避免在反向代理环境中的协议检测问题

	// 确定要添加的前缀
	pathPrefix := ""
	if len(prefix) > 0 && prefix[0] != "" {
		pathPrefix = prefix[0]
	}

	// 构建完整的URL
	var fullURL string
	if len(imagePath) > 0 && imagePath[0] == '/' {
		// 如果图片路径以斜杠开头，直接拼接baseURL
		fullURL = baseURL + imagePath
	} else {
		// 如果图片路径不以斜杠开头，根据是否有前缀来决定是否添加斜杠
		if pathPrefix != "" {
			fullURL = baseURL + "/" + pathPrefix + "/" + imagePath
		} else {
			fullURL = baseURL + "/" + imagePath
		}
	}

	return fullURL
}
