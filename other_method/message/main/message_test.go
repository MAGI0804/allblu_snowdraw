package main

import (
	"fmt"
	"testing"
)

func TestSendSms(t *testing.T) {
	fmt.Println("开始测试发送短信功能...")

	// 测试发送短信（使用测试手机号和验证码）
	result, err := SendSms("18107290804", "123344")
	if err != nil {
		t.Errorf("发送短信失败: %v", err)
		return
	}

	fmt.Println("发送短信结果:", result)
	fmt.Println("短信发送测试完成")
}
