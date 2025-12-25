package routes

import (
	"django_to_go/controllers"

	"github.com/gin-gonic/gin"
)

// InitAddressRoutes 初始化地址相关路由 - 与Django版本address.urls完全匹配
func InitAddressRoutes(router *gin.Engine) {
	// 初始化地址控制器
	addressController := &controllers.AddressController{}

	// 添加前置URL前缀
	addressGroup := router.Group("/address/")
	{
		// 地址相关路由 - 与Django版本address.urls完全匹配
		addressGroup.POST("add_address", addressController.AddAddress)
		addressGroup.POST("delete_address", addressController.DeleteAddress)
		addressGroup.POST("update_address", addressController.UpdateAddress)
		addressGroup.POST("set_default_address", addressController.SetDefaultAddress)
		addressGroup.POST("get_addresses", addressController.GetAddresses)
		addressGroup.POST("get_address_by_id", addressController.GetAddressByID)
	}
}