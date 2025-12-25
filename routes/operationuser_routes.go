package routes

import (
	"django_to_go/controllers"

	"github.com/gin-gonic/gin"
)

// InitOperationUserRoutes 初始化运营用户相关路由
// 与Django版本的OperationUser/urls.py完全匹配
func InitOperationUserRoutes(router *gin.Engine) {
	// 创建运营用户控制器实例
	operationUserController := &controllers.OperationUserController{}

	// 为所有运营用户相关路由添加"OperationUser"前缀
	operationUserGroup := router.Group("/OperationUser")
	{
		// 添加客服用户 - 对应Django的add_service_user
		operationUserGroup.POST("/add_service_user", operationUserController.AddServiceUser)

		// 添加运营用户 - 对应Django的add_operation_user
		operationUserGroup.POST("/add_operation_user", operationUserController.AddOperationUser)

		// 验证登录状态 - 对应Django的verification_status
		operationUserGroup.POST("/verification_status", operationUserController.VerificationStatus)

		// 修改密码 - 对应Django的change_password
		operationUserGroup.POST("/change_password", operationUserController.ChangePassword)
	}
}
