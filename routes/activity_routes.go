package routes

import (
	"django_to_go/controllers"

	"github.com/gin-gonic/gin"
)

// InitActivityRoutes 初始化活动相关路由 - 与Django版本activity.urls完全匹配
func InitActivityRoutes(router *gin.Engine) {
	// 初始化活动控制器
	activityController := &controllers.ActivityController{}

	// 添加前置URL前缀
	activityGroup := router.Group("/activity/")
	{
		// 活动相关路由 - 与Django版本完全匹配
		activityGroup.POST("add_activity_img", activityController.AddActivityImg)
		activityGroup.POST("update_activity_image_relations", activityController.UpdateActivityImageRelations)
		activityGroup.POST("activity_image_online", activityController.ActivityImageOnline)
		activityGroup.POST("activity_image_offline", activityController.ActivityImageOffline)
		activityGroup.POST("batch_query_activity_images", activityController.BatchQueryActivityImages)
		activityGroup.POST("batch_update_activity_image_order", activityController.BatchUpdateActivityImageOrder)
	}
}
