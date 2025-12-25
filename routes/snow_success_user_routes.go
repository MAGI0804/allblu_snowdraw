package routes

import (
	"django_to_go/controllers"

	"github.com/gin-gonic/gin"
)

// InitSnowSuccessUserRoutes 初始化抽奖成功用户相关路由
func InitSnowSuccessUserRoutes(router *gin.Engine) {
	// 创建控制器实例
	successUserController := controllers.NewSnowSuccessUserController()

	// 定义路由组
	successUserGroup := router.Group("/snow_success/")
	{
		// 新增资格用户
		successUserGroup.POST("add_success_user", successUserController.AddEligibilityUser)
		// 修改资格用户信息
		successUserGroup.POST("update_success_user", successUserController.UpdateEligibilityUser)
		// 验证抽奖
		successUserGroup.POST("verify_lottery", successUserController.VerifyLottery)
	}
}
