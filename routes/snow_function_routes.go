package routes

import (
	"django_to_go/controllers"

	"github.com/gin-gonic/gin"
)

// InitSnowFunctionRoutes 初始化抽奖功能相关路由
func InitSnowFunctionRoutes(router *gin.Engine) {
	// 创建控制器实例
	functionController := controllers.NewSnowFunctionController()

	// 定义路由组，前缀为snow_function
	functionGroup := router.Group("/snow_function/")
	{
		// 根据抽奖码和轮次查询用户信息
		functionGroup.POST("query_user_by_code", functionController.QueryUserByCode)

		// 校验订单号是否满足抽奖轮次要求
		functionGroup.POST("validate_order", functionController.ValidateOrder)

		// 查询抽奖轮次参与人数
		functionGroup.POST("query_participation_count", functionController.QueryParticipationCount)

		// 升级用户会员
		functionGroup.POST("upgrade_user_vip", functionController.UpgradeUserVip)

		// 查询用户会员信息
		functionGroup.POST("query_user_vip_info", functionController.QueryUserVipInfo)

		//抽奖
		functionGroup.POST("draw_to_win", functionController.LotteryDraw)

		//导出中奖名单
		functionGroup.POST("export_winners", functionController.ExportWinners)

		// 查询抽奖记录
		functionGroup.POST("query_draw_records", functionController.QueryDrawRecords)

		// 查询中奖者信息
		functionGroup.POST("query_winners_info", functionController.QueryWinnersInfo)

		// 查询指定抽奖轮次的信息
		functionGroup.POST("query_draw_info", functionController.QueryDrawInfo)

		// 查询指定轮次抽奖的参与者信息
		functionGroup.POST("query_draw_participants", functionController.QueryDrawParticipants)

		// 导出指定轮次抽奖的参与者信息为Excel
		functionGroup.POST("export_draw_participants_excel", functionController.ExportDrawParticipantsExcel)

		// 单次抽取一个中奖者
		functionGroup.POST("draw_single_winner", functionController.DrawSingleWinner)

		// 查询符合抽奖条件的用户信息
		functionGroup.POST("query_eligible_users", functionController.QueryEligibleUsers)

		// 上传临时图片
		functionGroup.POST("upload_temp_image", functionController.UploadTempImage)

		// 导入中奖名单
		functionGroup.POST("import_winners", functionController.ImportWinners)
	}
}
