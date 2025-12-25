package models

import (
	"time"
)

// OperationUser 操作用户模型
type OperationUser struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Username  string    `gorm:"size:50;not null;unique" json:"username"`
	Password  string    `gorm:"size:100;not null" json:"password"`
	UserType  string    `gorm:"size:20;not null" json:"user_type"`
	Status    string    `gorm:"size:20;not null" json:"status"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName 设置表名
func (ou *OperationUser) TableName() string {
	return "operation_operationuser"
}