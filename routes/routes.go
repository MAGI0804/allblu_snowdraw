package routes

import (
	"django_to_go/controllers"

	"github.com/gin-gonic/gin"
)

// InitRoutes 初始化路由配置 - 完全匹配Django版本
func InitRoutes(router *gin.Engine) {
	// 初始化控制器
	accessTokenController := &controllers.AccessTokenController{}

	// JWT 令牌相关路由 - 与Django版本完全匹配
	router.POST("api/token/obtain/", accessTokenController.TokenObtainPair)
	router.POST("api/token/refresh/", accessTokenController.TokenRefresh)

	// 初始化商品相关路由 - 与Django版本commodity.urls完全匹配
	InitCommodityRoutes(router)

	// 初始化用户相关路由 - 与Django版本users.urls完全匹配
	InitUserRoutes(router)

	// 初始化订单相关路由 - 与Django版本order.urls完全匹配
	InitOrderRoutes(router)

	// Access Token 相关路由 - 完全匹配Django access_token.urls
	router.POST("access_token/get_token", accessTokenController.GetToken)
	router.POST("access_token/get_ips", accessTokenController.GetIPs)

	// 初始化运营用户相关路由 - 与Django版本OperationUser.urls完全匹配
	InitOperationUserRoutes(router)

	// 初始化活动相关路由 - 与Django版本activity.urls完全匹配
	InitActivityRoutes(router)

	// 初始化购物车相关路由 - 与Django版本cart.urls完全匹配
	InitCartRoutes(router)

	// 初始化地址相关路由 - 与Django版本address.urls完全匹配
	InitAddressRoutes(router)

	// 初始化退货订单相关路由
	InitReturnOrderRoutes(router)

	// 初始化抽奖活动相关路由
	InitSnowLotteryDrawRoutes(router)

	// 初始化抽奖成功用户相关路由
	InitSnowSuccessUserRoutes(router)

	// 初始化抽奖用户相关路由
	InitSnowUserRoutes(router)

	// 初始化订单数据相关路由
	InitSnowOrderDataRoutes(router)

	// 初始化抽奖功能相关路由
	InitSnowFunctionRoutes(router)

	//初始化填写地址路由
	InitSnowAddressRoutes(router)

	// 测试路由
	router.GET("api/test/", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "Server is running"})
	})

	// 健康检查路由
	router.GET("api/health/", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// 404 路由 - 完全匹配Django的自定义404视图
	router.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{"error": "页面不存在"})
	})

	// 405 路由
	router.NoMethod(func(c *gin.Context) {
		c.JSON(405, gin.H{"error": "请求方法不允许"})
	})
}
