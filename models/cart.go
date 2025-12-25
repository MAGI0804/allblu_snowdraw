package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"
)

// CartItemJSON 购物车商品项JSON结构
type CartItemJSON struct {
	Quantity  int    `json:"quantity"`
	AddedTime string `json:"added_time"`
}

// CartItemsMap 自定义类型用于JSON序列化和反序列化
type CartItemsMap map[string]CartItemJSON

// Scan 实现sql.Scanner接口，用于从数据库读取JSON数据
func (c *CartItemsMap) Scan(value interface{}) error {
	if value == nil {
		*c = make(CartItemsMap)
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("类型断言失败：无法将数据库值转换为[]byte")
	}

	var result CartItemsMap
	if err := json.Unmarshal(bytes, &result); err != nil {
		return err
	}

	*c = result
	return nil
}

// Value 实现driver.Valuer接口，用于将数据序列化为JSON存储到数据库
func (c CartItemsMap) Value() (driver.Value, error) {
	if len(c) == 0 {
		// 空map存储为空JSON对象
		return "{}", nil
	}

	bytes, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}

	return string(bytes), nil
}

// Cart 购物车模型 - 与Django版本完全匹配
type Cart struct {
	CartID    int             `gorm:"column:cart_id;primaryKey;autoIncrement" json:"cart_id"`
	UserID    int             `gorm:"column:user_id;index" json:"user_id"`
	CartItems CartItemsMap    `gorm:"column:cart_items;type:json" json:"cart_items"`
	Remarks   string          `gorm:"column:remarks;null" json:"remarks"`
	CreatedAt time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time       `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName 设置表名
func (Cart) TableName() string {
	return "cart_cart"
}

// BeforeSave GORM钩子，确保gorm包被使用
func (c *Cart) BeforeSave(*gorm.DB) error {
	return nil
}
