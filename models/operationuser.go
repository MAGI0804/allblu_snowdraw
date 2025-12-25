package models

import (
	"database/sql"
	"fmt"
	"math/rand"
	"strings"
	"time"
)

// DjangoCustomerServiceUser 客服用户模型 - 从Django项目移植
// 对应Django模型CustomerServiceUser
type DjangoCustomerServiceUser struct {
	UserID   string `json:"user_id" gorm:"column:user_id;primaryKey;size:6;unique;not null"`
	Nickname string `json:"nickname" gorm:"column:nickname;size:100;not null"`
	Mobile   string `json:"mobile" gorm:"column:mobile;size:20;unique;not null"`
	Password string `json:"password" gorm:"column:password;size:255;not null"`
}

// TableName 设置DjangoCustomerServiceUser的表名为Customer_service_user
func (DjangoCustomerServiceUser) TableName() string {
	return "Customer_service_user"
}

// BeforeSave 钩子函数，用于生成唯一的user_id
func (u *DjangoCustomerServiceUser) BeforeSave(tx *sql.Tx) (err error) {
	if u.UserID == "" {
		// 初始化随机数生成器
		rand.Seed(time.Now().UnixNano())

		// 生成6位数字作为user_id，确保在DjangoCustomerServiceUser和DjangoOperationUser中唯一
		for {
			userID := generateRandomUserID()

			// 检查user_id是否已存在
			var count int

			// 检查DjangoCustomerServiceUser中是否存在
			query1 := "SELECT COUNT(*) FROM Customer_service_user WHERE user_id = ?"
			err = tx.QueryRow(query1, userID).Scan(&count)
			if err != nil {
				return fmt.Errorf("check customer service user exists failed: %v", err)
			}
			if count > 0 {
				continue
			}

			// 检查DjangoOperationUser中是否存在
			query2 := "SELECT COUNT(*) FROM Operation_user WHERE user_id = ?"
			err = tx.QueryRow(query2, userID).Scan(&count)
			if err != nil {
				return fmt.Errorf("check operation user exists failed: %v", err)
			}
			if count > 0 {
				continue
			}

			u.UserID = userID
			break
		}
	}
	return nil
}

// DjangoOperationUser 运营用户模型 - 从Django项目移植
// 对应Django模型OperationUser
type DjangoOperationUser struct {
	UserID   string `json:"user_id" gorm:"column:user_id;primaryKey;size:6;unique;not null"`
	Nickname string `json:"nickname" gorm:"column:nickname;size:100;not null"`
	Mobile   string `json:"mobile" gorm:"column:mobile;size:20;unique;not null"`
	Password string `json:"password" gorm:"column:password;size:255;not null"`
	Level    int    `json:"level" gorm:"column:level;not null"`
}

// TableName 设置DjangoOperationUser的表名为Operation_user
func (DjangoOperationUser) TableName() string {
	return "Operation_user"
}

// BeforeSave 钩子函数，用于生成唯一的user_id
func (u *DjangoOperationUser) BeforeSave(tx *sql.Tx) (err error) {
	if u.UserID == "" {
		// 初始化随机数生成器
		rand.Seed(time.Now().UnixNano())

		// 生成6位数字作为user_id，确保在DjangoCustomerServiceUser和DjangoOperationUser中唯一
		for {
			userID := generateRandomUserID()

			// 检查user_id是否已存在
			var count int

			// 检查DjangoOperationUser中是否存在
			query1 := "SELECT COUNT(*) FROM Operation_user WHERE user_id = ?"
			err = tx.QueryRow(query1, userID).Scan(&count)
			if err != nil {
				return fmt.Errorf("check operation user exists failed: %v", err)
			}
			if count > 0 {
				continue
			}

			// 检查DjangoCustomerServiceUser中是否存在
			query2 := "SELECT COUNT(*) FROM Customer_service_user WHERE user_id = ?"
			err = tx.QueryRow(query2, userID).Scan(&count)
			if err != nil {
				return fmt.Errorf("check customer service user exists failed: %v", err)
			}
			if count > 0 {
				continue
			}

			u.UserID = userID
			break
		}
	}
	return nil
}

// generateRandomUserID 生成6位随机数字作为用户ID
func generateRandomUserID() string {
	digits := "0123456789"
	var result strings.Builder
	for i := 0; i < 6; i++ {
		result.WriteByte(digits[rand.Intn(len(digits))])
	}
	return result.String()
}
