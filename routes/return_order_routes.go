package routes

import (
	"django_to_go/controllers"

	"github.com/gin-gonic/gin"
)

// InitReturnOrderRoutes 初始化退货订单相关路由
func InitReturnOrderRoutes(router *gin.Engine) {
	// 初始化订单控制器
	orderController := &controllers.OrderController{}

	// 添加前置URL前缀
	returnOrderGroup := router.Group("/return_order/")
	{
		// 退货订单发货路由
		returnOrderGroup.POST("deliver", orderController.ReturnOrderDeliver)
		// 退货订单签收路由
		returnOrderGroup.POST("receive", orderController.ReturnOrderReceive)
		// 退货订单取消路由
		returnOrderGroup.POST("cancel", orderController.ReturnOrderCancel)
		// 退货订单修改买家信息路由
		returnOrderGroup.POST("update_buyer_info", orderController.ReturnOrderUpdateBuyerInfo)
	}
}