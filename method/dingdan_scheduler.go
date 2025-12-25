package method

import (
	"django_to_go/method/pdd"
	"django_to_go/method/taobao"
	"fmt"
	"log"
	"time"
)

// 全局变量，用于存储上一次执行的时间范围
var (
	lastStartTime time.Time
	lastEndTime   time.Time
	schedulerInitialized bool
)

func init() {
	// 初始化时设置为未初始化状态，在StartDingdanScheduler中进行实际初始化
	schedulerInitialized = false
}

// StartDingdanScheduler 启动订单定时调度器
func StartDingdanScheduler() {
	log.Println("订单定时调度器启动，按照配置的时间范围执行")

	// 初始化调度器
	if !schedulerInitialized {
		initializeScheduler()
		schedulerInitialized = true
	}

	// 执行初始化后的批量任务（从0点到当前最近的半小时）
	executeInitialBulkJobs()

	// 立即执行一次最新的任务（确保从当前半小时开始）
	executeAllJobs()

	// 计算下一次应该执行的时间（整点或半点）
	now := time.Now()
	var nextExecuteTime time.Time
	
	if now.Minute() < 30 {
		// 当前时间不到半点，下一次执行时间为当前小时的半点
		nextExecuteTime = time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 30, 0, 0, now.Location())
	} else {
		// 当前时间已过半点，下一次执行时间为下一小时的整点
		nextExecuteTime = time.Date(now.Year(), now.Month(), now.Day(), now.Hour()+1, 0, 0, 0, now.Location())
	}
	
	// 计算等待时间
	waitDuration := nextExecuteTime.Sub(now)
	log.Printf("等待到下次执行时间：%s，等待时长：%v", nextExecuteTime.Format("2006-01-02 15:04:05"), waitDuration)
	
	// 等待到下一个半点或整点
	time.Sleep(waitDuration)
	
	// 立即执行一次
	executeAllJobs()
	
	// 创建定时器，每30分钟执行一次
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	// 监听定时器
	for {
		select {
		case <-ticker.C:
			log.Println("定时器触发，开始执行定时任务")
			executeAllJobs()
		}
	}
}

// initializeScheduler 初始化调度器，设置起始时间为当天0点
func initializeScheduler() {
	now := time.Now()
	// 设置起始时间为当天0点
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	
	lastStartTime = today
	lastEndTime = today
	
	log.Printf("调度器初始化完成，起始时间设置为：%s", lastStartTime.Format("2006-01-02 15:04:05"))
}

// executeInitialBulkJobs 执行初始化后的批量任务（从0点到上一个已完成的半小时区间）
func executeInitialBulkJobs() {
	now := time.Now()
	// 计算上一个已完成的半小时区间的结束时间
	currentHour := now.Hour()
	currentMinute := now.Minute()
	
	var targetTime time.Time
	if currentMinute < 30 {
		// 如果当前分钟小于30，上一个已完成的区间是 [hour-1:30, hour:00]
		if currentHour == 0 {
			// 处理跨天情况
			targetTime = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		} else {
			targetTime = time.Date(now.Year(), now.Month(), now.Day(), currentHour, 0, 0, 0, now.Location())
		}
	} else {
		// 如果当前分钟大于等于30，上一个已完成的区间是 [hour:00, hour:30]
		targetTime = time.Date(now.Year(), now.Month(), now.Day(), currentHour, 30, 0, 0, now.Location())
	}
	
	log.Printf("开始执行批量初始化任务，从0点到 %s", targetTime.Format("2006-01-02 15:04:05"))
	
	// 循环执行从0点到目标时间的所有半小时区间
	for lastEndTime.Before(targetTime) {
		// 计算下一个半小时区间
		newStartTime := lastEndTime
		newEndTime := newStartTime.Add(30 * time.Minute)
		
		// 确保不超过目标时间
		if newEndTime.After(targetTime) {
			break
		}
		
		// 更新全局变量
		lastStartTime = newStartTime
		lastEndTime = newEndTime
		
		// 执行这个时间区间的任务
		executeJobsForTimeRange(newStartTime, newEndTime)
	}
	
	log.Printf("批量初始化任务执行完成，当前时间范围：%s 到 %s", 
		lastStartTime.Format("2006-01-02 15:04:05"), 
		lastEndTime.Format("2006-01-02 15:04:05"))
}

// executeAllJobs 执行所有定时任务
func executeAllJobs() {
	// 计算上一个已完成的半小时区间
	now := time.Now()
	hour := now.Hour()
	minute := now.Minute()
	
	// 确定上一个半小时区间的结束时间
	var endTime time.Time
	if minute < 30 {
		// 如果当前分钟小于30，上一个区间是 [hour-1:30, hour:00]
		if hour == 0 {
			// 处理跨天情况
			endTime = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		} else {
			endTime = time.Date(now.Year(), now.Month(), now.Day(), hour, 0, 0, 0, now.Location())
		}
	} else {
		// 如果当前分钟大于等于30，上一个区间是 [hour:00, hour:30]
		endTime = time.Date(now.Year(), now.Month(), now.Day(), hour, 30, 0, 0, now.Location())
	}
	
	// 计算开始时间（结束时间减去30分钟）
	startTime := endTime.Add(-30 * time.Minute)
	
	// 更新全局变量
	lastStartTime = startTime
	lastEndTime = endTime
	
	// 执行指定时间范围的任务
	executeJobsForTimeRange(startTime, endTime)
}

// executeJobsForTimeRange 执行指定时间范围的所有任务
func executeJobsForTimeRange(startTime, endTime time.Time) {
	fmt.Printf("[%s] 开始执行定时任务，时间范围：%s 到 %s\n", 
		time.Now().Format("2006-01-02 15:04:05"), 
		startTime.Format("2006-01-02 15:04:05"), 
		endTime.Format("2006-01-02 15:04:05"))

	// 格式化时间字符串
	startTimeStr := startTime.Format("2006-01-02 15:04:05")
	endTimeStr := endTime.Format("2006-01-02 15:04:05")

	// 执行聚水潭订单任务
	log.Println("开始执行聚水潭订单任务...")
	Main(startTimeStr, endTimeStr)
	log.Println("聚水潭订单任务执行完成")

	// 执行有赞订单任务
	log.Println("开始执行有赞订单任务...")
	MainYouzanOrder(startTimeStr, endTimeStr)
	log.Println("有赞订单任务执行完成")

	// 执行淘宝订单任务
	log.Println("开始执行淘宝订单任务...")
	err := taobao.MainTaobaoOrder(startTimeStr, endTimeStr)
	if err != nil {
		log.Printf("淘宝订单任务执行失败: %v\n", err)
	} else {
		log.Println("淘宝订单任务执行完成")
	}

	// 执行拼多多订单任务
	log.Println("开始执行拼多多订单任务...")
	pdd.MainPDDOrder(startTimeStr, endTimeStr)
	log.Println("拼多多订单任务执行完成")

	fmt.Printf("[%s] 所有定时任务执行完成\n", time.Now().Format("2006-01-02 15:04:05"))
}

// generateNextTimeRange 生成下一个时间范围（保留此函数以兼容其他可能的调用）
func generateNextTimeRange() (time.Time, time.Time) {
	// 由于executeAllJobs已经更新了时间逻辑，这里直接返回当前的全局时间范围
	return lastStartTime, lastEndTime
}

// 主函数入口，用于直接运行调度器
func main() {
	StartDingdanScheduler()
}