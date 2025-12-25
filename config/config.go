package config

import (
	"os"
	"time"
)

// Config 应用配置结构体
type Config struct {
	DBConfig
	JWTConfig
	WechatConfig
}

// DBConfig 数据库配置
type DBConfig struct {
	Driver   string
	Host     string
	Port     string
	Username string
	Password string
	DBName   string
	Charset  string
}

// JWTConfig JWT配置
type JWTConfig struct {
	SecretKey       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

// WechatConfig 微信配置
type WechatConfig struct {
	AppID     string
	AppSecret string
	LoginURL  string
}

// LoadConfig 加载配置
func LoadConfig() Config {
	// 从环境变量获取配置，如果没有则使用默认值
	return Config{
		DBConfig: DBConfig{
			Driver:   getEnv("DB_ENGINE", "mysql"),
			Host:     getEnv("DB_HOST", "rm-uf623nc5tc2icc2u9uo.mysql.rds.aliyuncs.com"),
			Port:     getEnv("DB_PORT", "3306"),
			Username: getEnv("DB_USER", "youlansy"),
			Password: getEnv("DB_PASSWORD", "Allblu2022#"),
			DBName:   getEnv("DB_NAME", "wechat_member"),
			Charset:  "utf8mb4",
		},
		JWTConfig: JWTConfig{
			SecretKey:       getEnv("SECRET_KEY", "django-insecure-m89*e)$=h*0a#6!$=e+r@w5%985!j!f8y2mt%@#qw%o=i#!f*6"),
			AccessTokenTTL:  time.Minute * 60,
			RefreshTokenTTL: time.Hour * 24,
		},
		WechatConfig: WechatConfig{
			AppID:     "wx54051aca55f3cfcd",
			AppSecret: "be64553f64f8c90cb43f51c8e95959c7",
			LoginURL:  "https://api.weixin.qq.com/sns/jscode2session",
		},
	}
}

// getEnv 从环境变量获取值，如果不存在则使用默认值
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
