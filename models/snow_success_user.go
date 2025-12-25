package models

import (
	"bytes"
	"encoding/json"
	"log"
	"strings"

	"gorm.io/gorm"
)

// SnowSuccessUser 抽奖成功用户信息模型
// 对应数据库表: snow_success__uesr
type SnowSuccessUser struct {
	UserID              int    `gorm:"column:user_id;primaryKey;autoIncrement" json:"user_id"`
	Nickname            string `gorm:"column:nickname;type:varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;not null" json:"nickname"`
	Mobile              int    `gorm:"column:mobile;uniqueIndex" json:"mobile"`
	MemberSource        string `gorm:"column:member_source;type:varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;null" json:"member_source"`          //会员来源
	DrawEligibility     string `gorm:"column:draw_eligibility;type:json CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci" json:"draw_eligibility"`                // 抽奖资格:true(false)
	ParticipationStatus string `gorm:"column:participation_status;type:json CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci" json:"participation_status"`        // 参与情况:true(false)
	WinningStatus       string `gorm:"column:winning_status;type:json CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci" json:"winning_status"`                    // 抽奖轮次:true(false)
	OrderNUM            string `gorm:"column:order_num;type:json CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci" json:"order_num"`                              // 订单
	SuccessCode         string `gorm:"column:success_code;type:json CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci" json:"success_code"`                        // 抽奖码，JSON格式：波次:抽奖码
	MobileBatch         string `gorm:"column:mobile_batch;type:json CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci" json:"mobile_batch"`                        // 手机号批次，JSON格式：波次:手机号
	DrawSuccessTime     string `gorm:"column:draw_success_time;type:varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;null" json:"draw_success_time"` // 中奖时间
	VerificationStatus  string `gorm:"column:verification_status;type:json CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci" json:"verification_status"`          // 验证状态:true(false)
	VerificationTime    string `gorm:"column:verification_time;type:varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;null" json:"verification_time"` // 验证时间
	Remarks             string `gorm:"column:remarks;type:text CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;null" json:"remarks"`                             // 备注
}

// TableName 设置表名
func (SnowSuccessUser) TableName() string {
	return "snow_success__uesr"
}

// BeforeCreate 在创建记录前的钩子
func (s *SnowSuccessUser) BeforeCreate(tx *gorm.DB) error {
	// 初始化JSON字段为空字符串，避免存储null
	if s.DrawEligibility == "" {
		s.DrawEligibility = "{}"
	}
	if s.ParticipationStatus == "" {
		s.ParticipationStatus = "{}"
	}
	if s.WinningStatus == "" {
		s.WinningStatus = "{}"
	}
	if s.OrderNUM == "" {
		s.OrderNUM = "{}"
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
	return nil
}

// AfterFind 在查询后自动解析JSON字段
func (s *SnowSuccessUser) AfterFind(tx *gorm.DB) error {
	// 确保所有JSON字段都被正确初始化
	if s.DrawEligibility == "" {
		s.DrawEligibility = "{}"
	}
	if s.ParticipationStatus == "" {
		s.ParticipationStatus = "{}"
	}
	if s.WinningStatus == "" {
		s.WinningStatus = "{}"
	}
	if s.OrderNUM == "" {
		s.OrderNUM = "{}"
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
	return nil
}

// SetDrawEligibility 设置抽奖资格
func (s *SnowSuccessUser) SetDrawEligibility(data map[string]bool) error {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false) // 不进行HTML转义
	if err := encoder.Encode(data); err != nil {
		return err
	}
	// 去掉encoder.Encode添加的换行符
	s.DrawEligibility = strings.TrimSpace(buf.String())
	return nil
}

// GetDrawEligibility 获取抽奖资格
func (s *SnowSuccessUser) GetDrawEligibility() (map[string]bool, error) {
	var data map[string]bool
	if err := json.Unmarshal([]byte(s.DrawEligibility), &data); err != nil {
		return nil, err
	}
	return data, nil
}

// SetOrderNUM 设置订单号
func (s *SnowSuccessUser) SetOrderNUM(data map[string]string) error {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false) // 不进行HTML转义
	if err := encoder.Encode(data); err != nil {
		return err
	}
	// 去掉encoder.Encode添加的换行符
	s.OrderNUM = strings.TrimSpace(buf.String())
	return nil
}

// GetOrderNUM 获取订单号
func (s *SnowSuccessUser) GetOrderNUM() (map[string]string, error) {
	var data map[string]string
	if err := json.Unmarshal([]byte(s.OrderNUM), &data); err != nil {
		return nil, err
	}
	return data, nil
}

// SetWinningStatus 设置中奖状态
func (s *SnowSuccessUser) SetWinningStatus(data map[string]bool) error {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false) // 不进行HTML转义
	if err := encoder.Encode(data); err != nil {
		return err
	}
	// 去掉encoder.Encode添加的换行符
	s.WinningStatus = strings.TrimSpace(buf.String())
	return nil
}

// GetWinningStatus 获取中奖状态
func (s *SnowSuccessUser) GetWinningStatus() (map[string]bool, error) {
	var data map[string]bool
	if err := json.Unmarshal([]byte(s.WinningStatus), &data); err != nil {
		return nil, err
	}
	return data, nil
}

// SetSuccessCode 设置抽奖码
func (s *SnowSuccessUser) SetSuccessCode(data map[string]string) error {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false) // 不进行HTML转义
	if err := encoder.Encode(data); err != nil {
		return err
	}
	// 去掉encoder.Encode添加的换行符
	s.SuccessCode = strings.TrimSpace(buf.String())
	return nil
}

// GetSuccessCode 获取抽奖码
func (s *SnowSuccessUser) GetSuccessCode() (map[string]string, error) {
	// 确保SuccessCode不为空
	if s.SuccessCode == "" {
		s.SuccessCode = "{}"
	}

	log.Printf("Debug - SnowSuccessUser.GetSuccessCode: SuccessCode内容: %s", s.SuccessCode)

	var data map[string]string
	if err := json.Unmarshal([]byte(s.SuccessCode), &data); err != nil {
		log.Printf("Error - SnowSuccessUser.GetSuccessCode: 解析失败: %v", err)
		return make(map[string]string), nil // 返回空map而不是错误，避免影响流程
	}

	// 如果解析后data为nil，初始化一个空map
	if data == nil {
		data = make(map[string]string)
	}

	log.Printf("Debug - SnowSuccessUser.GetSuccessCode: 解析结果: %+v", data)
	return data, nil
}

// SetMobileBatch 设置手机号批次
func (s *SnowSuccessUser) SetMobileBatch(data map[string]int) error {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false) // 不进行HTML转义
	if err := encoder.Encode(data); err != nil {
		return err
	}
	// 去掉encoder.Encode添加的换行符
	s.MobileBatch = strings.TrimSpace(buf.String())
	return nil
}

// GetMobileBatch 获取手机号批次
func (s *SnowSuccessUser) GetMobileBatch() (map[string]int, error) {
	var data map[string]int
	if err := json.Unmarshal([]byte(s.MobileBatch), &data); err != nil {
		return nil, err
	}
	return data, nil
}

// SetVerificationStatus 设置验证状态
func (s *SnowSuccessUser) SetVerificationStatus(data map[string]bool) error {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false) // 不进行HTML转义
	if err := encoder.Encode(data); err != nil {
		return err
	}
	// 去掉encoder.Encode添加的换行符
	s.VerificationStatus = strings.TrimSpace(buf.String())
	return nil
}

// GetVerificationStatus 获取验证状态
func (s *SnowSuccessUser) GetVerificationStatus() (map[string]bool, error) {
	var data map[string]bool
	if err := json.Unmarshal([]byte(s.VerificationStatus), &data); err != nil {
		return nil, err
	}
	return data, nil
}

// SetDrawSuccessTime 设置中奖时间
func (s *SnowSuccessUser) SetDrawSuccessTime(data map[string]string) error {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false) // 不进行HTML转义
	if err := encoder.Encode(data); err != nil {
		return err
	}
	// 去掉encoder.Encode添加的换行符
	s.DrawSuccessTime = strings.TrimSpace(buf.String())
	return nil
}

// GetDrawSuccessTime 获取中奖时间
func (s *SnowSuccessUser) GetDrawSuccessTime() (map[string]string, error) {
	// 如果字段为空，返回空map
	if s.DrawSuccessTime == "" {
		return make(map[string]string), nil
	}
	var data map[string]string
	if err := json.Unmarshal([]byte(s.DrawSuccessTime), &data); err != nil {
		return nil, err
	}
	return data, nil
}
