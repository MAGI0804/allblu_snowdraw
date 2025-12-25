package models

import (
	"time"

	"gorm.io/gorm"
)

// User 用户模型

type User struct {
	UserID           int       `gorm:"column:user_id;primaryKey;autoIncrement" json:"user_id"`
	OpenID           string    `gorm:"column:openid;size:100;uniqueIndex;null" json:"openid"`
	UserImg          string    `gorm:"column:user_img;size:255;null" json:"user_img"`
	Mobile           string    `gorm:"column:mobile;size:11;uniqueIndex;null" json:"mobile"`
	Nickname         string    `gorm:"column:nickname;size:100;not null" json:"nickname"`
	Password         string    `gorm:"column:password;size:128;null" json:"password"`
	DefaultReceiver  string    `gorm:"column:default_receiver;size:100;null" json:"default_receiver"`
	Province         string    `gorm:"column:province;size:50;null" json:"province"`
	City             string    `gorm:"column:city;size:50;null" json:"city"`
	County           string    `gorm:"column:county;size:50;null" json:"county"`
	DetailedAddress  string    `gorm:"column:detailed_address;size:255;null" json:"detailed_address"`
	MembershipLevel  int       `gorm:"column:membership_level;default:0" json:"membership_level"`
	RegistrationDate time.Time `gorm:"column:registration_date;autoCreateTime" json:"registration_date"`
	TotalSpending    float64   `gorm:"column:total_spending;type:decimal(10,2);default:0" json:"total_spending"`
	Remarks          string    `gorm:"column:remarks;type:text;null" json:"remarks"`
	LastLogin        time.Time `gorm:"column:last_login;null" json:"last_login"`
	IsActive         bool      `gorm:"column:is_active;default:true" json:"is_active"`
	IsStaff          bool      `gorm:"column:is_staff;default:false" json:"is_staff"`
	IsSuperuser      bool      `gorm:"column:is_superuser;default:false" json:"is_superuser"`
}

// TableName 设置表名
func (User) TableName() string {
	return "users_user"
}

// BeforeCreate 在创建用户前的钩子
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.OpenID != "" && u.Nickname == "" {
		// 如果有OpenID但没有昵称，生成默认昵称
		if len(u.OpenID) > 8 {
			u.Nickname = "微信用户_" + u.OpenID[:8]
		} else {
			u.Nickname = "微信用户_" + u.OpenID
		}
	} else if u.Mobile != "" && u.Nickname == "" {
		// 如果有手机号但没有昵称，生成默认昵称
		if len(u.Mobile) > 4 {
			u.Nickname = "手机用户_" + u.Mobile[len(u.Mobile)-4:]
		} else {
			u.Nickname = "手机用户_" + u.Mobile
		}
	}
	// 设置注册日期为当前时间
	u.RegistrationDate = time.Now()
	return nil
}

// BeforeSave 在保存用户前的钩子
func (u *User) BeforeSave(tx *gorm.DB) error {
	// 确保LastLogin字段不为零值
	if u.LastLogin.IsZero() {
		u.LastLogin = time.Now()
	}
	return nil
}
