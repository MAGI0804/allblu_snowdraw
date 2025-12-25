package routes

import (
	"django_to_go/controllers"

	"github.com/gin-gonic/gin"
)

// InitSnowLotteryDrawRoutes 初始化抽奖活动相关路由
func InitSnowLotteryDrawRoutes(router *gin.Engine) {
	// 初始化抽奖控制器
	drawController := &controllers.SnowLotteryDrawController{}

	// 添加前置URL前缀
	drawGroup := router.Group("/snow_lottery/")
	{
		// 新增抽奖
		drawGroup.POST("create_draw", drawController.CreateDraw)
		// 修改抽奖信息
		drawGroup.POST("update_draw", drawController.UpdateDraw)
		// 获取中奖名单
		drawGroup.POST("get_winners", drawController.GetWinners)
		// 查询所有抽奖信息或单个抽奖信息
		drawGroup.POST("get_all_draws", drawController.GetAllDraws)
		// 添加中奖名单
		drawGroup.POST("add_winners", drawController.AddWinners)
		// 获取所有中奖名单（支持按抽奖轮次筛选）
		drawGroup.POST("get_all_winners", drawController.GetAllWinners)
		// 根据用户ID获取中奖名单（支持按抽奖轮次筛选，根据用户ID决定返回信息详细程度）
		drawGroup.POST("get_user_winners", drawController.GetUserWinners)
		// 查询所有抽奖信息，包含抽奖波次、开奖时间、开奖情况、前5个抽奖码
		drawGroup.POST("get_all_draw_info", drawController.GetAllDrawInfo)
	}
}