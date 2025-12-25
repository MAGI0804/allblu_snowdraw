package routes

import (
	"django_to_go/controllers"

	"github.com/gin-gonic/gin"
)

// InitUserRoutes 初始化用户相关路由 - 与Django版本users.urls完全匹配
func InitUserRoutes(router *gin.Engine) {
	// 初始化用户控制器
	userController := &controllers.UserController{}

	// 添加前置URL前缀
	userGroup := router.Group("/ordinary_user/")
	{
		// 用户相关路由 - 与Django版本users.urls完全匹配
		userGroup.POST("add_data", userController.UserRegistration)
		userGroup.POST("find_data", userController.UserQuery)
		userGroup.POST("Modify_data", userController.UserModify)
		userGroup.GET("get_user_id", userController.UserGetID)
		userGroup.POST("verification_status", userController.VerificationStatus)
		userGroup.POST("wechat_login", userController.WechatLogin)
	}
}
