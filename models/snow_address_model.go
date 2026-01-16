package models

import "time"

type SnowAddress struct {
	UserId          int        `gorm:"column:user_id;primaryKey" json:"user_id"`
	ReceiverName    string     `gorm:"column:receiver_name;type:varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;null" json:"receiver_name"`       // 收货人姓名
	ReceiverPhone   string     `gorm:"column:receiver_phone;type:varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;null" json:"receiver_phone"`      // 收货人联系电话
	Province        string     `gorm:"column:province;type:varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;null" json:"province"`                  //省
	City            string     `gorm:"column:city;type:varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;null" json:"city"`                          //市
	County          string     `gorm:"column:county;type:varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;null" json:"county"`                      //县
	DetailedAddress string     `gorm:"column:detailed_address;type:varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;null" json:"detailed_address"` //具体地址
	ExpressCompany  string     `gorm:"column:express_company;type:varchar(255)" json:"express_company"`                                                         //快递公司
	ExpressNumer    string     `gorm:"column:express_numer;type:varchar(255)" json:"express_numer"`                                                             //快递单号
	FillTime        *time.Time `gorm:"column:fill_time;null" json:"fill_time"`                                                                                  //更新时间
	Remark          string     `gorm:"column:remark;type:varchar(255)" json:"remark"`                                                                           //备注
}

func (SnowAddress) TableName() string {
	return "snow_address"
}
