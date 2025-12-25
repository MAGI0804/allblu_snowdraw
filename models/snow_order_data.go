package models

import (
	"time"
)

// SnowOrderData 订单数据模型
type SnowOrderData struct {
	ID                        int        `gorm:"primaryKey;autoIncrement"`
	SerialNumber              int        `gorm:"column:serial_number;type:int;not null"`                   // 序号
	OnlineOrderNumber         string     `gorm:"column:online_order_number;type:text;not null"`            // 线上订单号
	OrderStatus               string     `gorm:"column:order_status;type:varchar(50);not null"`            // 订单状态
	Store                     string     `gorm:"column:store;type:varchar(255);not null"`                  // 店铺
	OrderDate                 *time.Time `gorm:"column:order_date;type:datetime"`                          // 订单日期（使用指针类型支持NULL）
	ShipDate                  *time.Time `gorm:"column:ship_date;type:datetime"`                           // 发货日期（使用指针类型支持NULL）
	PaymentDate               *time.Time `gorm:"column:payment_date;type:datetime"`                        // 付款日期（使用指针类型支持NULL）
	SellerID                  string     `gorm:"column:seller_id;type:varchar(100);not null"`              // 买家id
	ConfirmReceiptTime        *time.Time `gorm:"column:confirm_receipt_time;type:datetime"`                // 确认收货时间（使用指针类型支持NULL）
	ConsigneeName             string     `gorm:"column:consignee_name;type:varchar(100);not null"`         // 收货人姓名
	Province                  string     `gorm:"column:province;type:varchar(100);not null"`               // 省
	City                      string     `gorm:"column:city;type:varchar(100);not null"`                   // 市
	County                    string     `gorm:"column:county;type:varchar(100);not null"`                 // 县
	TrackingNumber            string     `gorm:"column:tracking_number;type:varchar(100)"`                 // 快递单号
	OriginalOnlineOrderNumber string     `gorm:"column:original_online_order_number;type:text"`            // 原始线上订单号
	ActualPaymentAmount       float64    `gorm:"column:actual_payment_amount;type:decimal(10,2);not null"` // 实付金额
	ReturnQuantity            int        `gorm:"column:return_quantity;type:int;default:0"`                // 退货数量
	ReturnAmount              float64    `gorm:"column:return_amount;type:decimal(10,2);default:0"`        // 退货金额
	OnlineSubOrderNumber      string     `gorm:"column:online_sub_order_number;type:text"`                 // 线上子订单编号
	Remark                    string     `gorm:"column:remark;type:text"`                                  // 备注
	CreatedAt                 time.Time  `gorm:"autoCreateTime"`
	UpdatedAt                 time.Time  `gorm:"autoUpdateTime"`
}

// TableName 设置表名
func (SnowOrderData) TableName() string {
	return "snow_order_data"
}
