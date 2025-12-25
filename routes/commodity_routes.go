package routes

import (
	"django_to_go/controllers"

	"github.com/gin-gonic/gin"
)

// InitCommodityRoutes 初始化商品相关路由 - 与Django版本commodity.urls完全匹配
func InitCommodityRoutes(router *gin.Engine) {
	// 初始化商品控制器
	commodityController := &controllers.CommodityController{}

	// 添加前置URL前缀
	commodityGroup := router.Group("/commodity/")
	{
		// 商品相关路由 - 与Django版本commodity.urls完全匹配
		commodityGroup.POST("get_all_categories", commodityController.GetAllCategories)
		commodityGroup.POST("search_style_codes", commodityController.SearchStyleCodes)
		commodityGroup.POST("add_goods", commodityController.AddGoods)
		commodityGroup.POST("delete_goods", commodityController.DeleteGoods)
		commodityGroup.POST("search_commodity_data", commodityController.SearchCommodityData)
		commodityGroup.POST("goods_query", commodityController.GoodsQuery)
		commodityGroup.POST("change_commodity_data", commodityController.ChangeCommodityData)
		commodityGroup.POST("change_commodity_status_online", commodityController.ChangeCommodityStatusOnline)
		commodityGroup.POST("change_commodity_status_offline", commodityController.ChangeCommodityStatusOffline)
		commodityGroup.POST("get_commodity_status", commodityController.GetCommodityStatus)
		commodityGroup.POST("search_products_by_name", commodityController.SearchProductsByName)
		commodityGroup.POST("batch_get_products_by_ids", commodityController.BatchGetProductsByIDs)
		commodityGroup.POST("style-code/status/online", commodityController.ChangeStyleCodeStatusOnline)
		commodityGroup.POST("style-code/status/offline", commodityController.ChangeStyleCodeStatusOffline)
		commodityGroup.POST("style-code/commodities", commodityController.GetCommoditiesByStyleCode)
		commodityGroup.POST("update_style_code_info", commodityController.UpdateStyleCodeInfo)
	}
}
