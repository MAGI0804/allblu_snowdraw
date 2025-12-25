package db

import (
	"django_to_go/config"
	"fmt"
	"log"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// InitDB 初始化数据库连接
func InitDB(appConfig config.Config) {
	// 构建DSN（数据源名称）
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=%s&parseTime=True&loc=Local",
		appConfig.DBConfig.Username,
		appConfig.DBConfig.Password,
		appConfig.DBConfig.Host,
		appConfig.DBConfig.Port,
		appConfig.DBConfig.DBName,
		appConfig.DBConfig.Charset,
	)
	fmt.Println(string(dsn))
	// 配置GORM日志
	newLogger := logger.New(
		log.New(log.Writer(), "[GORM] ", log.LstdFlags), // 日志输出到标准输出
		logger.Config{
			SlowThreshold:             time.Second, // 慢SQL阈值
			LogLevel:                  logger.Info, // 日志级别
			IgnoreRecordNotFoundError: true,        // 忽略ErrRecordNotFound错误
			ParameterizedQueries:      true,        // 参数化查询
			Colorful:                  false,       // 禁用彩色输出
		},
	)

	// 连接数据库
	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}

	// 获取底层的sql.DB对象
	sqlDB, err := DB.DB()
	if err != nil {
		log.Fatalf("获取数据库连接池失败: %v", err)
	}

	// 配置连接池
	sqlDB.SetMaxIdleConns(10)                  // 设置空闲连接数
	sqlDB.SetMaxOpenConns(100)                 // 设置最大连接数
	sqlDB.SetConnMaxLifetime(time.Hour)        // 设置连接的最大生命周期
	sqlDB.SetConnMaxIdleTime(time.Minute * 30) // 设置连接的最大空闲时间

	log.Println("数据库连接成功")
}
