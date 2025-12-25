package main

import (
	"django_to_go/config"
	"django_to_go/db"

	// "django_to_go/method"
	"django_to_go/middleware"
	"django_to_go/routes"
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	// 加载配置
	appConfig := config.LoadConfig()

	// 初始化数据库
	db.InitDB(appConfig)
	// 运行数据库迁移，同步表结构变更
	db.RunMigrations()

	// 在goroutine中启动订单定时调度器
	log.Println("正在启动订单定时调度器...")
	// go method.StartDingdanScheduler()

	// 创建Gin引擎
	router := gin.Default()

	// 设置中间件
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(middleware.CORSMiddleware())
	router.Use(middleware.RequestLogMiddleware())
	router.Use(middleware.ErrorHandlerMiddleware())
	router.Use(middleware.AccessTokenValidationMiddleware())

	// 设置静态文件服务
	router.Static("/static", "./staticfiles")
	router.Static("/media", "./media")

	// 初始化路由
	routes.InitRoutes(router)

	// 启动服务器
	port := "8088"
	log.Printf("Server starting on port %s\n", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
