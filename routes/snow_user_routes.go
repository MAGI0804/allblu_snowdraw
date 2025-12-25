package routes

import (
	"django_to_go/controllers"

	"github.com/gin-gonic/gin"
)

// InitSnowUserRoutes 初始化抽奖用户相关路由
func InitSnowUserRoutes(router *gin.Engine) {
	// 初始化控制器
	snowUserController := &controllers.SnowUserController{}
	SNOWUserGroup := router.Group("/snow_user/")
	{
		// 1. 微信注册 - POST
		SNOWUserGroup.POST("wechat_register", snowUserController.WechatRegister)

		// 2. 订单号与手机号校验 - POST
		SNOWUserGroup.POST("verify_order_mobile", snowUserController.VerifyOrderMobile)

		// 3. 短信验证 - POST
		SNOWUserGroup.POST("send_verification_code", snowUserController.SendVerificationCode)

		// 4. 验证码校验功能已合并到参与抽奖接口中

		// 5. 修改地址 - POST
		SNOWUserGroup.POST("update_address", snowUserController.UpdateAddress)

		// 6. 查询地址 - POST
		SNOWUserGroup.POST("query_address", snowUserController.QueryAddress)

		// 7. 验证用户抽奖资格 - POST
		SNOWUserGroup.POST("verify_draw_eligibility", snowUserController.VerifyDrawEligibility)

		// 8. 参与抽奖 - POST
		SNOWUserGroup.POST("participate_draw", snowUserController.ParticipateDraw)
		// 9. 参与抽奖（仅需手机号校验） - POST
		SNOWUserGroup.POST("participate_draw_by_mobile", snowUserController.ParticipateDrawByMobile)
		// 10. 查找用户信息 - POST
		SNOWUserGroup.POST("find_data", snowUserController.FindData)
		// 11. 查询用户抽奖信息 - POST
		SNOWUserGroup.POST("query_user_draw_info", snowUserController.QueryUserDrawInfo)

		// 12. 删除指定轮次抽奖信息 - POST
		SNOWUserGroup.POST("delete_draw_info", snowUserController.DeleteDrawInfo)

		// 13. 更新用户信息 - POST
		SNOWUserGroup.POST("update_user_info", snowUserController.UpdateUserInfo)

		// 14. 根据手机号和波次修改地址 - POST
		SNOWUserGroup.POST("update_address_by_mobile_and_batch", snowUserController.UpdateAddressByMobileAndBatch)

		// 15. 根据手机号和波次查询地址 - POST
		SNOWUserGroup.POST("query_address_by_mobile_and_batch", snowUserController.QueryAddressByMobileAndBatch)
	}

}
