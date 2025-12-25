package models

import (
	"time"
)

// Product 商品模型
type Product struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:100;not null" json:"name"`
	Price     float64   `gorm:"not null" json:"price"`
	ImageURL  string    `gorm:"size:255" json:"image_url"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName 设置表名
func (p *Product) TableName() string {
	return "product_product"
}