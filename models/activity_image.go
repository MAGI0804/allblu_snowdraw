package models

import (
	"time"
)

// ActivityImage 活动图片模型
// 与Django项目中的ActivityImage模型完全同步
type ActivityImage struct {
	ID          int        `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Image       string     `gorm:"column:image;size:255;not null" json:"image"`
	Status      string     `gorm:"column:status;size:20;default:'pending'" json:"status"`
	OnlineTime  *time.Time  `gorm:"column:online_time;null" json:"online_time"`
	OfflineTime *time.Time  `gorm:"column:offline_time;null" json:"offline_time"`
	Commodities string     `gorm:"column:commodities;type:text;null" json:"commodities"`
	Category    string     `gorm:"column:category;size:100;null" json:"category"`
	Notes       string     `gorm:"column:notes;type:text;null" json:"notes"`
	Order       *int       `gorm:"column:order;null" json:"order"` // 图片顺序字段，改为指针类型支持null
	CreatedAt   time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName 设置表名，与Django模型保持一致
func (ActivityImage) TableName() string {
	return "Activity_Image"
}