package models

import (
	"time"
)

// SnowLotteryDraw 抽奖活动模型
type SnowLotteryDraw struct {
	ID                int       `gorm:"primaryKey;autoIncrement"`
	DrawBatch         int       `gorm:"column:draw_batch;type:int;not null"`            // 抽奖波次
	Prizes            string    `gorm:"column:prizes;type:json"`                        // 抽奖奖品，JSON格式字符串
	TotalDrawers      int       `gorm:"column:total_drawers;type:int;not null"`         // 中奖人数
	OrderBeginTime    time.Time `gorm:"column:order_begin_time;type:datetime;not null"` // 订单开始时间
	OrderEndTime      time.Time `gorm:"column:order_end_time;type:datetime;not null"`   // 订单结束时间
	DrawTime          time.Time `gorm:"column:draw_time;type:datetime;"`                // 开奖时间
	ParticipantsCount int       `gorm:"column:participants_count;type:int;not null"`    // 参与人数
	ParticipantsList  string    `gorm:"column:participants_list;type:json"`             // 参与抽奖名单，JSON数组格式存储user_id
	WinnersList       string    `gorm:"column:winners_list;type:text"`                  // 中奖名单
	DrawName          string    `gorm:"column:draw_name;type:varchar(255);not null"`    // 抽奖名称
	Remark            string    `gorm:"column:remark;type:text"`                        // 备注
	CreatedAt         time.Time `gorm:"autoCreateTime"`
	Record            string    `gorm:"column:record;type:text"` // 抽奖记录
	UpdatedAt         time.Time `gorm:"autoUpdateTime"`
}

// TableName 设置表名
func (SnowLotteryDraw) TableName() string {
	return "snow_lottery_draw"
}

// BeforeCreate 钩子函数，初始化JSON字段
func (s *SnowLotteryDraw) BeforeCreate() error {
	if s.Prizes == "" {
		s.Prizes = "{}"
	}
	if s.ParticipantsList == "" {
		s.ParticipantsList = "[]"
	}
	return nil
}

// BeforeSave 钩子函数，确保JSON字段始终有效
func (s *SnowLotteryDraw) BeforeSave() error {
	if s.Prizes == "" {
		s.Prizes = "{}"
	}
	if s.ParticipantsList == "" {
		s.ParticipantsList = "[]"
	}
	return nil
}
