package models

import (
	"time"

	_ "gorm.io/gorm"
)

// UserData 用户数据模型
// 用于存储用户的各类数据

type UserData struct {
	ID         int       `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserID     int       `gorm:"column:user_id;not null" json:"user_id"`
	DataType   string    `gorm:"column:data_type;type:varchar(100);not null" json:"data_type"`
	DataValue  string    `gorm:"column:data_value;type:text" json:"data_value"`
	CreateTime time.Time `gorm:"column:create_time;autoCreateTime" json:"create_time"`
}

// TableName 设置表名
func (UserData) TableName() string {
	return "user_data"
}