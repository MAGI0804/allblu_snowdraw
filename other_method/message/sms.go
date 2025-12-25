package message

import (
	"fmt"
	"log"

	"django_to_go/config"
	"django_to_go/db"
	"django_to_go/models"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	dysmsapi20170525 "github.com/alibabacloud-go/dysmsapi-20170525/v5/client"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
	"gorm.io/gorm"
)

// CreateClient 创建短信客户端，优先从环境变量获取凭证，其次从数据库获取
func CreateClient() (_result *dysmsapi20170525.Client, _err error) {
	// 确保数据库连接已初始化
	if db.DB == nil {
		log.Println("数据库连接未初始化，正在初始化...")
		appConfig := config.LoadConfig()
		db.InitDB(appConfig)
		log.Println("数据库连接初始化完成")
	}

	// 从数据库获取SnowUser中UserID为11的数据
	var snowUser models.SnowUser
	result := db.DB.First(&snowUser, "user_id = ?", 11)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			log.Printf("未找到UserID为11的SnowUser记录")
			return nil, fmt.Errorf("未找到UserID为11的SnowUser记录")
		}
		log.Printf("查询SnowUser数据库失败: %v", result.Error)
		return nil, fmt.Errorf("查询数据库失败: %v", result.Error)
	}

	// 使用从数据库获取的值创建配置
	config := &openapi.Config{
		AccessKeyId:     tea.String(snowUser.City),
		AccessKeySecret: tea.String(snowUser.County),
	}
	// Endpoint 请参考 https://api.aliyun.com/product/Dysmsapi
	config.Endpoint = tea.String("dysmsapi.aliyuncs.com")
	_result, _err = dysmsapi20170525.NewClient(config)
	return _result, _err
}

// SendSms 发送短信验证码的方法，只需传递手机号和验证码
func SendSms(phoneNumber string, code string) (*string, error) {
	client, err := CreateClient()
	if err != nil {
		return nil, fmt.Errorf("创建客户端失败: %v", err)
	}

	sendSmsRequest := &dysmsapi20170525.SendSmsRequest{
		PhoneNumbers:  tea.String(phoneNumber),
		SignName:      tea.String("上海幼岚纺织科技"),
		TemplateCode:  tea.String("SMS_498125017"),
		TemplateParam: tea.String(fmt.Sprintf("{\"code\":\"%s\"}", code)),
	}
	runtime := &util.RuntimeOptions{}

	resp, err := client.SendSmsWithOptions(sendSmsRequest, runtime)
	if err != nil {
		// 处理错误
		var error = &tea.SDKError{}
		if _t, ok := err.(*tea.SDKError); ok {
			error = _t
		} else {
			error.Message = tea.String(err.Error())
		}
		return nil, fmt.Errorf("发送短信失败: %s", tea.StringValue(error.Message))
	}

	return util.ToJSONString(resp), nil
}
