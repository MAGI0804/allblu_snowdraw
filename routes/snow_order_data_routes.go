package routes

import (
	"django_to_go/controllers"

	"github.com/gin-gonic/gin"
)

// InitSnowOrderDataRoutes 初始化订单数据相关路由
func InitSnowOrderDataRoutes(router *gin.Engine) {
	// 初始化控制器
	snowOrderDataController := &controllers.SnowOrderDataController{}
	snowOrderDataGroup := router.Group("/snow_order_data/")
	{
		// 导入订单数据 - POST
		snowOrderDataGroup.POST("import", snowOrderDataController.ImportOrderData)
	}
}
