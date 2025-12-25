package routes

import (
	"django_to_go/controllers"

	"github.com/gin-gonic/gin"
)

// InitCartRoutes 初始化购物车相关路由 - 与Django版本cart.urls完全匹配
func InitCartRoutes(router *gin.Engine) {
	// 初始化购物车控制器
	cartController := &controllers.CartController{}

	// 添加前置URL前缀
	cartGroup := router.Group("/cart/")
	{
		// 添加商品到购物车 - POST方法
		cartGroup.POST("add_to_cart", cartController.AddToCart)

		// 批量删除购物车商品 - 同时支持DELETE和POST方法
		cartGroup.DELETE("batch_delete_from_cart", cartController.BatchDeleteFromCart)
		cartGroup.POST("batch_delete_from_cart", cartController.BatchDeleteFromCart)

		// 查询购物车所有商品 - 同时支持GET和POST方法
		cartGroup.GET("query_cart_items", cartController.QueryCartItems)
		cartGroup.POST("query_cart_items", cartController.QueryCartItems)

		// 更新购物车商品数量 - 同时支持PUT和POST方法
		cartGroup.PUT("update_cart_item_quantity", cartController.UpdateCartItemQuantity)
		cartGroup.POST("update_cart_item_quantity", cartController.UpdateCartItemQuantity)

		// 购物车商品数量加1 - POST方法
		cartGroup.POST("increase_cart_item_quantity", cartController.IncreaseCartItemQuantity)

		// 购物车商品数量减1（不能减到0）- POST方法
		cartGroup.POST("decrease_cart_item_quantity", cartController.DecreaseCartItemQuantity)

		// 清空购物车 - 同时支持DELETE和POST方法
		cartGroup.DELETE("clear_cart", cartController.ClearCart)
		cartGroup.POST("clear_cart", cartController.ClearCart)
	}
}