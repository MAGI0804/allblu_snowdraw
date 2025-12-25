package models

import (
	"crypto/rand"
	"encoding/json"
	"math/big"
	"time"

	"gorm.io/gorm"
)

// SnowUser 抽奖活动用户信息模型
// 对应数据库表: snow_uesr
type SnowUser struct {
	UserID                 int                  `gorm:"column:user_id;primaryKey" json:"user_id"` // 手动生成的随机六位数字
	Nickname               string               `gorm:"column:nickname;type:varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;not null" json:"nickname"`
	Mobile                 int                  `gorm:"column:mobile;" json:"mobile"`
	OrderNumbers           string               `gorm:"column:order_numbers;type:json CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci" json:"-"` // JSON字符串，不直接序列化
	DrawTimes              string               `gorm:"column:draw_times;type:json CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci" json:"-"`
	EligibilityStatus      string               `gorm:"column:eligibility_status;type:json CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci" json:"-"`
	RegistrationTime       time.Time            `gorm:"column:registration_time;autoCreateTime" json:"registration_time"`
	ParticipationStatus    string               `gorm:"column:participation_status;type:json CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci" json:"-"`
	OrderSignTime          string               `gorm:"column:order_sign_time;type:json CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci" json:"-"`
	MemberSource           string               `gorm:"column:member_source;type:varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;null" json:"member_source"`
	Province               string               `gorm:"column:province;type:varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;null" json:"province"`
	City                   string               `gorm:"column:city;type:varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;null" json:"city"`
	County                 string               `gorm:"column:county;type:varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;null" json:"county"`
	DetailedAddress        string               `gorm:"column:detailed_address;type:varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;null" json:"detailed_address"`
	OpenID                 string               `gorm:"column:openid;type:varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;uniqueIndex;null" json:"openid"`
	AvatarURL              string               `gorm:"column:avatar_url;type:varchar(500) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;null" json:"avatar_url"` // 用户头像URL
	VerificationCode       int                  `gorm:"column:verification_code;null" json:"verification_code"`
	VerificationCodeExpire time.Time            `gorm:"column:verification_code_expire;null" json:"verification_code_expire"` // 验证码过期时间
	Remarks                string               `gorm:"column:remarks;type:text CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;null" json:"remarks"`
	ReceiverName           string               `gorm:"column:receiver_name;type:varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;null" json:"receiver_name"`  // 收货人姓名
	ReceiverPhone          string               `gorm:"column:receiver_phone;type:varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;null" json:"receiver_phone"` // 收货人联系电话
	SuccessCode            string               `gorm:"column:success_code;type:json CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci" json:"-"`                            // JSON字符串，抽奖波次:抽奖码
	OrderNumbersMap        map[string]string    `gorm:"-" json:"order_numbers"`
	DrawTimesMap           map[string]time.Time `gorm:"-" json:"draw_times"`
	EligibilityStatusMap   map[string]bool      `gorm:"-" json:"eligibility_status"`
	ParticipationStatusMap map[string]bool      `gorm:"-" json:"participation_status"`
	OrderSignTimeMap       map[string]time.Time `gorm:"-" json:"order_sign_time"`
	SuccessCodeMap         map[string]string    `gorm:"-" json:"success_code"`                                           // 抽奖波次:抽奖码映射
	MobileBatch            string               `gorm:"column:mobile_batch;type:json CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci" json:"-"`                          // JSON字符串，手机号批次
	MobileBatchMap         map[string]int       `gorm:"-" json:"mobile_batch"`                                           // 手机号批次映射
	VerificationStatus     string               `gorm:"column:verification_status;type:json CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci" json:"verification_status"` // 验证状态:true(false)
}

// TableName 设置表名
func (SnowUser) TableName() string {
	return "snow_uesr"
}

// BeforeSave 在保存记录前的钩子 - 序列化map为JSON字符串
func (s *SnowUser) BeforeSave(tx *gorm.DB) error {
	// 如果是新记录且UserID为0，则生成随机六位数字ID
	if tx.Statement.Changed("UserID") || s.UserID == 0 {
		// 生成100000-999999之间的随机数
		for {
			// 生成随机数
			max := big.NewInt(900000) // 999999 - 100000 + 1
			randomNum, err := rand.Int(rand.Reader, max)
			if err != nil {
				return err
			}
			s.UserID = int(randomNum.Int64()) + 100000

			// 检查是否已存在
			var count int64
			tx.Model(&SnowUser{}).Where("user_id = ?", s.UserID).Count(&count)
			if count == 0 {
				break
			}
		}
	}

	// 初始化空字符串
	if s.OrderNumbers == "" {
		s.OrderNumbers = "{}"
	}
	if s.DrawTimes == "" {
		s.DrawTimes = "{}"
	}
	if s.EligibilityStatus == "" {
		s.EligibilityStatus = "{}"
	}
	if s.ParticipationStatus == "" {
		s.ParticipationStatus = "{}"
	}
	if s.OrderSignTime == "" {
		s.OrderSignTime = "{}"
	}
	if s.SuccessCode == "" {
		s.SuccessCode = "{}"
	}
	if s.MobileBatch == "" {
		s.MobileBatch = "{}"
	}
	if s.VerificationStatus == "" {
		s.VerificationStatus = "{}"
	}

	// 设置VerificationCodeExpire的默认值，避免0000-00-00无效日期时间值
	if s.VerificationCodeExpire.IsZero() {
		// 设置为当前时间加1年，作为默认值
		s.VerificationCodeExpire = time.Now().AddDate(1, 0, 0)
	}

	// 如果map不为nil，序列化到JSON字符串
	if s.OrderNumbersMap != nil {
		if data, err := json.Marshal(s.OrderNumbersMap); err == nil {
			s.OrderNumbers = string(data)
		}
	}
	if s.EligibilityStatusMap != nil {
		if data, err := json.Marshal(s.EligibilityStatusMap); err == nil {
			s.EligibilityStatus = string(data)
		}
	}
	if s.ParticipationStatusMap != nil {
		if data, err := json.Marshal(s.ParticipationStatusMap); err == nil {
			s.ParticipationStatus = string(data)
		}
	}
	if s.SuccessCodeMap != nil {
		if data, err := json.Marshal(s.SuccessCodeMap); err == nil {
			s.SuccessCode = string(data)
		}
	}

	// 序列化MobileBatchMap为JSON字符串
	if s.MobileBatchMap != nil {
		if data, err := json.Marshal(s.MobileBatchMap); err == nil {
			s.MobileBatch = string(data)
		}
	}

	// 时间类型字段需要特殊处理（简化版，实际使用时可能需要自定义时间格式）
	if s.DrawTimesMap != nil {
		if data, err := json.Marshal(s.DrawTimesMap); err == nil {
			s.DrawTimes = string(data)
		}
	}
	if s.OrderSignTimeMap != nil {
		if data, err := json.Marshal(s.OrderSignTimeMap); err == nil {
			s.OrderSignTime = string(data)
		}
	}

	// 设置注册时间
	if s.RegistrationTime.IsZero() {
		s.RegistrationTime = time.Now()
	}
	return nil
}

// GetMobileBatch 获取手机号批次
func (s *SnowUser) GetMobileBatch() (map[string]int, error) {
	var data map[string]int
	if s.MobileBatch != "" && s.MobileBatch != "{}" {
		if err := json.Unmarshal([]byte(s.MobileBatch), &data); err != nil {
			return nil, err
		}
	} else {
		data = make(map[string]int)
	}
	return data, nil
}

// SetMobileBatch 设置手机号批次
func (s *SnowUser) SetMobileBatch(data map[string]int) error {
	if data == nil {
		data = make(map[string]int)
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	s.MobileBatch = string(jsonData)
	return nil
}

// GetVerificationStatus 获取验证状态
func (s *SnowUser) GetVerificationStatus() (map[string]bool, error) {
	var data map[string]bool
	if s.VerificationStatus != "" && s.VerificationStatus != "{}" {
		if err := json.Unmarshal([]byte(s.VerificationStatus), &data); err != nil {
			return nil, err
		}
	} else {
		data = make(map[string]bool)
	}
	return data, nil
}

// SetVerificationStatus 设置验证状态
func (s *SnowUser) SetVerificationStatus(data map[string]bool) error {
	if data == nil {
		data = make(map[string]bool)
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	s.VerificationStatus = string(jsonData)
	return nil
}

// GetSuccessCode 获取抽奖码
func (s *SnowUser) GetSuccessCode() (map[string]string, error) {
	var data map[string]string
	if s.SuccessCode != "" && s.SuccessCode != "{}" {
		if err := json.Unmarshal([]byte(s.SuccessCode), &data); err != nil {
			return nil, err
		}
	} else {
		data = make(map[string]string)
	}
	return data, nil
}

// AfterFind 从数据库读取后反序列化JSON字符串为map
func (s *SnowUser) AfterFind(tx *gorm.DB) error {
	// 反序列化JSON字符串到map
	if s.OrderNumbers != "" && s.OrderNumbers != "{}" {
		var orderNumbersMap map[string]string
		if err := json.Unmarshal([]byte(s.OrderNumbers), &orderNumbersMap); err == nil {
			s.OrderNumbersMap = orderNumbersMap
		}
	}
	if s.EligibilityStatus != "" && s.EligibilityStatus != "{}" {
		var eligibilityStatusMap map[string]bool
		if err := json.Unmarshal([]byte(s.EligibilityStatus), &eligibilityStatusMap); err == nil {
			s.EligibilityStatusMap = eligibilityStatusMap
		}
	}
	if s.ParticipationStatus != "" && s.ParticipationStatus != "{}" {
		var participationStatusMap map[string]bool
		if err := json.Unmarshal([]byte(s.ParticipationStatus), &participationStatusMap); err == nil {
			s.ParticipationStatusMap = participationStatusMap
		}
	}
	if s.SuccessCode != "" && s.SuccessCode != "{}" {
		var successCodeMap map[string]string
		if err := json.Unmarshal([]byte(s.SuccessCode), &successCodeMap); err == nil {
			s.SuccessCodeMap = successCodeMap
		}
	}
	if s.DrawTimes != "" && s.DrawTimes != "{}" {
		var drawTimesMap map[string]time.Time
		if err := json.Unmarshal([]byte(s.DrawTimes), &drawTimesMap); err == nil {
			s.DrawTimesMap = drawTimesMap
		}
	}
	if s.OrderSignTime != "" && s.OrderSignTime != "{}" {
		var orderSignTimeMap map[string]time.Time
		if err := json.Unmarshal([]byte(s.OrderSignTime), &orderSignTimeMap); err == nil {
			s.OrderSignTimeMap = orderSignTimeMap
		}
	}

	// 反序列化MobileBatch JSON字符串到map
	if s.MobileBatch != "" && s.MobileBatch != "{}" {
		var mobileBatchMap map[string]int
		if err := json.Unmarshal([]byte(s.MobileBatch), &mobileBatchMap); err == nil {
			s.MobileBatchMap = mobileBatchMap
		}
	}

	// 初始化nil的map
	if s.OrderNumbersMap == nil {
		s.OrderNumbersMap = make(map[string]string)
	}
	if s.EligibilityStatusMap == nil {
		s.EligibilityStatusMap = make(map[string]bool)
	}
	if s.ParticipationStatusMap == nil {
		s.ParticipationStatusMap = make(map[string]bool)
	}
	if s.DrawTimesMap == nil {
		s.DrawTimesMap = make(map[string]time.Time)
	}
	if s.OrderSignTimeMap == nil {
		s.OrderSignTimeMap = make(map[string]time.Time)
	}
	if s.SuccessCodeMap == nil {
		s.SuccessCodeMap = make(map[string]string)
	}
	if s.MobileBatchMap == nil {
		s.MobileBatchMap = make(map[string]int)
	}
	return nil
}

