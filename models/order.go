package models

import (
	"time"

	"gorm.io/gorm"
)

// Order 订单模型 - 与Django版本保持一致
type Order struct {
	OrderID          string    `gorm:"column:order_id;primaryKey;size:20" json:"order_id"`
	UserID           int       `gorm:"column:user_id;not null;default:0" json:"user_id"`
	ReceiverName     string    `gorm:"column:receiver_name;size:100;not null" json:"receiver_name"`
	ReceiverPhone    string    `gorm:"column:receiver_phone;size:15;null" json:"receiver_phone"`
	ExpressCompany   string    `gorm:"column:express_company;size:50;null" json:"express_company"`
	ExpressNumber    string    `gorm:"column:express_number;size:50;null" json:"express_number"`
	LogisticsProcess string    `gorm:"column:logistics_process;type:text;null" json:"logistics_process"`
	Province         string    `gorm:"column:province;size:50;not null" json:"province"`
	City             string    `gorm:"column:city;size:50;not null" json:"city"`
	County           string    `gorm:"column:county;size:50;not null" json:"county"`
	DetailedAddress  string    `gorm:"column:detailed_address;size:255;not null" json:"detailed_address"`
	OrderAmount      float64   `gorm:"column:order_amount;type:decimal(10,2);not null" json:"order_amount"`
	ProductList      string    `gorm:"column:product_list;type:text;not null" json:"product_list"`
	Status           string    `gorm:"column:status;size:20;not null;default:'pending'" json:"status"`
	OrderTime        time.Time `gorm:"column:order_time;autoCreateTime" json:"order_time"`
	Remarks          string    `gorm:"column:remarks;type:text;null" json:"remarks"`
	PaymentMethod    string    `gorm:"column:payment_method;size:20;null" json:"payment_method"`
	DeliveryMethod   string    `gorm:"column:delivery_method;size:20;null" json:"delivery_method"`
	PaymentTime      time.Time `gorm:"column:payment_time;null" json:"payment_time"`
	ShippedTime      time.Time `gorm:"column:shipped_time;null" json:"shipped_time"`
	DeliveredTime    time.Time `gorm:"column:delivered_time;null" json:"delivered_time"`
	CanceledTime     time.Time `gorm:"column:canceled_time;null" json:"canceled_time"`
	ProcessingTime   time.Time `gorm:"column:processing_time;null" json:"processing_time"`
}

// TableName 设置表名 - 与Django版本保持一致
func (Order) TableName() string {
	return "order_data"
}

// BeforeSave GORM钩子，确保gorm包被使用
func (o *Order) BeforeSave(*gorm.DB) error {
	return nil
}
