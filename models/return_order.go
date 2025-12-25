package models

import (
	"time"
)

// ReturnOrder 退换货订单模型
type ReturnOrder struct {
	ID             uint      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ReturnID       string    `gorm:"column:return_id;size:30;not null;uniqueIndex" json:"return_id"` // 退换货订单号
	OrderID        string    `gorm:"column:order_id;size:20;not null" json:"order_id"`                // 关联订单号
	ProductList    string    `gorm:"column:product_list;type:text;not null" json:"product_list"`       // 商品列表
	Type           string    `gorm:"column:type;size:20;not null" json:"type"`                        // 类型：return(退货), exchange(换货)
	Status         string    `gorm:"column:status;size:20;not null;default:'pending'" json:"status"`  // 状态：pending(待处理), processing(处理中), shipped(已发货), completed(已完成), canceled(已取消)
	RequestTime    time.Time `gorm:"column:request_time;autoCreateTime" json:"request_time"`          // 申请时间
	ShippedTime    time.Time `gorm:"column:shipped_time;null" json:"shipped_time"`                    // 发货时间
	CanceledTime   time.Time `gorm:"column:canceled_time;null" json:"canceled_time"`                  // 取消时间
	CompletedTime  time.Time `gorm:"column:completed_time;null" json:"completed_time"`                // 完成时间
	ExpressCompany string    `gorm:"column:express_company;size:50;null" json:"express_company"`      // 退货物流公司
	ExpressNumber  string    `gorm:"column:express_number;size:50;null" json:"express_number"`        // 退货物流单号
	Reason         string    `gorm:"column:reason;type:text;not null" json:"reason"`                  // 退换货原因
	BuyerProvince  string    `gorm:"column:buyer_province;size:50;null" json:"buyer_province"`        // 买方省
	BuyerCity      string    `gorm:"column:buyer_city;size:50;null" json:"buyer_city"`                // 买方市
	BuyerCounty    string    `gorm:"column:buyer_county;size:50;null" json:"buyer_county"`            // 买方县
	BuyerAddress   string    `gorm:"column:buyer_address;size:255;null" json:"buyer_address"`         // 买方具体地址
	BuyerPhone     string    `gorm:"column:buyer_phone;size:15;null" json:"buyer_phone"`              // 买方联系电话
	Remarks        string    `gorm:"column:remarks;type:text;null" json:"remarks"`                    // 备注
}

// TableName 设置表名
func (ReturnOrder) TableName() string {
	return "return_order_data"
}