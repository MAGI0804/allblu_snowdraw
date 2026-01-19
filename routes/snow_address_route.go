package routes

import (
	"django_to_go/controllers"

	"github.com/gin-gonic/gin"
)

func InitSnowAddressRoutes(router *gin.Engine) {
	// 初始化抽奖控制器
	addressController := &controllers.SnowAddressController{}

	// 添加前置URL前缀
	addressGroup := router.Group("/snow_address/")
	{
		addressGroup.POST("qualification_verification", addressController.QualificationAddressVerification) //校验资格
		addressGroup.POST("query_address", addressController.QueryUserAddress)                              //查询地址
		addressGroup.POST("update_address", addressController.UpdateAddress)                                //填写地址
		addressGroup.POST("addrssses_query", addressController.BatchQueryAddress)                           //批量查询地址
		addressGroup.POST("export_excel_addrssses", addressController.ExportAllAddress)                     //导出地址到Excel
		addressGroup.POST("query_address_by_mobile", addressController.QueryAddressByMobile)                //根据手机号查询地址
		addressGroup.POST("update_address_by_mobile", addressController.UpdateAddressByMobile)              //根据手机号更新地址
	}
}
