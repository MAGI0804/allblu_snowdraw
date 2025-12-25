package method

import (
	"django_to_go/method/pdd"
	"django_to_go/method/taobao"
	"fmt"
	"testing"
	"time"
)

// TestRunDingdan 用于测试dingdan.go的功能
func TestRunDingdan(t *testing.T) {
	// 生成半小时的时间范围
	startTime, endTime := generateHalfHourTimeRange()

	startTime = "2025-11-28 15:30:00"
	endTime = "2025-11-28 16:00:00"
	fmt.Printf("开始测试dingdan.go功能，时间范围：%s 到 %s...\n", startTime, endTime)

	// 调用Main函数并传入时间范围
	Main(startTime, endTime)

	fmt.Println("dingdan.go测试完成")
}

func TestMainYouzanOrder(t *testing.T) {
	// 生成半小时的时间范围
	startTime, endTime := generateHalfHourTimeRange()
	fmt.Printf("开始测试有赞订单接口，时间范围：%s 到 %s...\n", startTime, endTime)

	// 调用MainYouzanOrder函数并传入时间范围
	MainYouzanOrder(startTime, endTime)

	fmt.Println("有赞订单接口测试完成")
}

func TestMainTaobaoOrder(t *testing.T) {
	// 注意：使用命令行参数设置超时时间，如：go test -timeout 5m ./method
	// 生成半小时的时间范围
	startTime, endTime := generateHalfHourTimeRange()
	fmt.Printf("开始测试淘宝订单接口，时间范围：%s 到 %s...\n", startTime, endTime)

	// 调用MainTaobaoOrder函数并传入时间范围
	if err := taobao.MainTaobaoOrder(startTime, endTime); err != nil {
		t.Errorf("淘宝订单接口测试失败: %v", err)
	}

	fmt.Println("淘宝订单接口测试完成")
}

// generateHalfHourTimeRange 生成半小时的时间范围
func generateHalfHourTimeRange() (string, string) {
	// 获取当前时间
	now := time.Now()
	// 计算半小时前的时间作为开始时间
	startTime := now.Add(-30 * time.Minute)
	// 结束时间就是当前时间
	endTime := now

	// 格式化为指定格式
	startTimeStr := startTime.Format("2006-01-02 15:04:05")
	endTimeStr := endTime.Format("2006-01-02 15:04:05")

	return startTimeStr, endTimeStr
}

func TestMainPddOrder(t *testing.T) {
	// 生成半小时的时间范围
	// access_tokens := map[string]string{
	// 	"拼多多官方旗舰店": "3bab67a2267e469b8af8c650f3f01c1f0f68d26b",
	// 	"拼多多童装旗舰店": "94049dfee30044b7ac5632bbe0163ff3480e0199",
	// 	"拼多多户外旗舰店": "4b143a73fc2346eaa226c10672b05377905087df",
	// }
	// const CLIENT_SECRET = "c584c4924f5ed15e393f1f16cb30993c12a655ad"
	fmt.Printf("开始测试拼多多订单接口")
	startTimeStr := "2025-11-21 8:00:00"
	endTimeStr := "2025-11-21 15:00:00"
	pdd.MainPDDOrder(startTimeStr, endTimeStr)
	fmt.Println("拼多多订单接口测试完成")
}
