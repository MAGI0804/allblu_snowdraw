package models

import (
	"time"

	"gorm.io/gorm"
)

// Address 用户地址模型
type Address struct {
	AddressID       int       `gorm:"column:address_id;primaryKey;autoIncrement" json:"address_id"`
	UserID          int       `gorm:"column:user_id;index" json:"user_id"`
	ReceiverName    string    `gorm:"column:receiver_name;size:100;not null" json:"receiver_name"`
	PhoneNumber     string    `gorm:"column:phone_number;size:20;not null" json:"phone_number"`
	Province        string    `gorm:"column:province;size:50;not null" json:"province"`
	City            string    `gorm:"column:city;size:50;not null" json:"city"`
	County          string    `gorm:"column:county;size:50;not null" json:"county"`
	DetailedAddress string    `gorm:"column:detailed_address;size:255;not null" json:"detailed_address"`
	IsDefault       bool      `gorm:"column:is_default;default:false" json:"is_default"`
	CreatedAt       time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName 设置表名
func (Address) TableName() string {
	return "addresses"
}

// BeforeSave GORM钩子，确保gorm包被使用
func (a *Address) BeforeSave(*gorm.DB) error {
	return nil
}
