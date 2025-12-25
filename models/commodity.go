package models

import (
	"time"

	"gorm.io/gorm"
)

// Commodity 商品模型
type Commodity struct {
	CommodityID    string    `gorm:"column:commodity_id;primaryKey;size:100" json:"commodity_id"`
	Name           string    `gorm:"column:name;size:255;not null" json:"name"`
	StyleCode      string    `gorm:"column:style_code;size:50;index" json:"style_code"`
	Category       string    `gorm:"column:category;size:100;not null" json:"category"`
	CategoryDetail string    `gorm:"column:category_detail;size:100;null" json:"category_detail"`
	Price          float64   `gorm:"column:price;not null" json:"price"`
	Image          string    `gorm:"column:image;size:255;not null" json:"image"`
	PromoImage     string    `gorm:"column:promo_image;size:255;null" json:"promo_image"`
	Size           string    `gorm:"column:size;size:50;null" json:"size"`
	Color          string    `gorm:"column:color;size:50;null" json:"color"`
	Height         string    `gorm:"column:height;size:50;null" json:"height"`
	SpecCode       string    `gorm:"column:spec_code;size:100;null" json:"spec_code"`
	ColorImage     string    `gorm:"column:color_image;size:255;null" json:"color_image"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	Notes          string    `gorm:"column:notes;type:text;null" json:"notes"`
}

// TableName 设置表名，与Django模型保持一致
func (Commodity) TableName() string {
	return "Commodity_data"
}

// BeforeSave GORM钩子，确保gorm包被使用
func (c *Commodity) BeforeSave(*gorm.DB) error {
	return nil
}

// CommodityImage 商品图片模型
type CommodityImage struct {
	ID         int       `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	CommodityID string    `gorm:"column:commodity_id;size:100;index" json:"commodity_id"`
	Image      string    `gorm:"column:image;size:255;not null" json:"image"`
	IsMain     bool      `gorm:"column:is_main;default:false" json:"is_main"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

// TableName 设置表名
func (CommodityImage) TableName() string {
	return "Commodity_Images"
}

// CommoditySituation 商品状态模型
type CommoditySituation struct {
	CommodityID  string    `gorm:"column:commodity_id;primaryKey;size:100" json:"commodity_id"`
	Status       string    `gorm:"column:status;size:20;not null" json:"status"`
	OnlineTime   time.Time `gorm:"column:online_time;autoCreateTime" json:"online_time"`
	OfflineTime  time.Time `gorm:"column:offline_time;autoCreateTime" json:"offline_time"`
	SalesVolume  int       `gorm:"column:sales_volume;default:0" json:"sales_volume"`
	Remarks      string    `gorm:"column:remarks;type:text;null" json:"remarks"`
	StyleCode    string    `gorm:"column:style_code;size:50;index" json:"style_code"`
}

// TableName 设置表名
func (CommoditySituation) TableName() string {
	return "Commodity_Situation"
}

// StyleCodeData 款式数据模型
type StyleCodeData struct {
	StyleCode      string                 `gorm:"column:style_code;primaryKey;size:50" json:"style_code"`
	Name           string                 `gorm:"column:name;size:255;not null" json:"name"`
	Image          string                 `gorm:"column:image;size:255;not null" json:"image"`
	Category       string                 `gorm:"column:category;size:100;not null" json:"category"`
	CategoryDetail string                 `gorm:"column:category_detail;size:100;null" json:"category_detail"`
	Price          float64                `gorm:"column:price;not null" json:"price"`
	DisplayPictures string                `gorm:"column:display_pictures;type:json;null" json:"display_pictures"`
	CreatedAt      time.Time              `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time              `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName 设置表名
func (StyleCodeData) TableName() string {
	return "StyleCode_Data"
}

// StyleCodeSituation 款式状态模型
type StyleCodeSituation struct {
	StyleCode      string    `gorm:"column:style_code;primaryKey;size:50" json:"style_code"`
	Status         string    `gorm:"column:status;size:20;not null" json:"status"`
	OnlineTime     time.Time `gorm:"column:online_time;autoCreateTime" json:"online_time"`
	OfflineTime    time.Time `gorm:"column:offline_time;null" json:"offline_time"`
	SyncDataCount  int       `gorm:"column:sync_data_count;default:0" json:"sync_data_count"`
}

// TableName 设置表名
func (StyleCodeSituation) TableName() string {
	return "StyleCode_Situation"
}