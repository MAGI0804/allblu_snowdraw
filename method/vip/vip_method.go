package method

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

// GetYouzanAccessToken 获取有赞open平台access_token
func GetYouzanAccessToken() (string, error) {
	url := "https://open.youzanyun.com/auth/token"
	data := map[string]interface{}{
		"authorize_type": "silent",
		"client_id":      "379981eff640bbb278",
		"client_secret":  "1ef6d04d42b03784bd75fc1b74493c06",
		"grant_id":       "15707004",
		"refresh":        false,
	}

	dataBytes, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("序列化请求数据失败: %v", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(dataBytes))
	if err != nil {
		return "", fmt.Errorf("请求access_token失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("解析响应失败: %v", err)
	}

	if success, ok := result["success"].(bool); ok && success {
		if code, ok := result["code"].(float64); ok && code == 200 {
			if dataMap, ok := result["data"].(map[string]interface{}); ok {
				if accessToken, ok := dataMap["access_token"].(string); ok {
					return accessToken, nil
				}
			}
		}
	}

	message := "未知错误"
	if msg, ok := result["message"].(string); ok {
		message = msg
	}
	return "", fmt.Errorf("获取access_token失败: %s", message)
}

// GetUserVipLevel 获取用户会员等级信息和会员详细信息
// account_id: 用户账号ID
// 返回：会员等级信息map、会员详细信息map和可能的错误
func GetUserVipLevel(accountID string) (map[string]interface{}, map[string]interface{}, error) {
	// 首先获取access_token
	accessToken, err := GetYouzanAccessToken()
	if err != nil {
		return nil, nil, fmt.Errorf("获取access_token失败: %v", err)
	}

	// 构建请求URL - 会员等级接口
	url := fmt.Sprintf("https://open.youzanyun.com/api/youzan.scrm.level.get.userlevel/1.0.0?access_token=%s", accessToken)

	// 构建请求体 - 会员等级接口
	requestBody := map[string]interface{}{
		"user": map[string]interface{}{
			"account_type": 2,
			"account_id":   accountID,
		},
	}

	dataBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, nil, fmt.Errorf("序列化请求数据失败: %v", err)
	}

	// 发送POST请求 - 会员等级接口
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(dataBytes))
	if err != nil {
		return nil, nil, fmt.Errorf("请求会员等级信息失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("读取响应失败: %v", err)
	}

	// 解析响应
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, nil, fmt.Errorf("解析响应失败: %v", err)
	}

	// 检查请求是否成功
	success, ok := result["success"].(bool)
	if !ok || !success {
		message := "请求失败"
		if msg, ok := result["message"].(string); ok {
			message = msg
		}
		return nil, nil, fmt.Errorf("获取会员等级信息失败: %s", message)
	}

	// 提取数据部分
	vipData, ok := result["data"].(map[string]interface{})
	if !ok {
		return nil, nil, fmt.Errorf("解析data字段失败")
	}

	// 调用会员详细信息接口
	customerInfo, err := getUserCustomerInfo(accessToken, accountID)
	if err != nil {
		// 如果获取会员详细信息失败，仍然返回等级信息，但customerInfo为nil
		return vipData, nil, nil
	}

	// 返回会员等级信息和会员详细信息
	return vipData, customerInfo, nil
}

// getUserCustomerInfo 获取用户会员详细信息
// accessToken: 访问令牌
// accountID: 用户账号ID
// 返回：会员详细信息map和可能的错误
func getUserCustomerInfo(accessToken, accountID string) (map[string]interface{}, error) {
	// 构建请求URL - 会员详细信息接口
	url := fmt.Sprintf("https://open.youzanyun.com/api/youzan.scrm.customer.get/3.1.0?access_token=%s", accessToken)

	// 构建请求体 - 会员详细信息接口
	requestBody := map[string]interface{}{
		"account": map[string]interface{}{
			"account_type": "Mobile",
			"account_id":   accountID,
		},
	}

	dataBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求数据失败: %v", err)
	}

	// 发送POST请求
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(dataBytes))
	if err != nil {
		return nil, fmt.Errorf("请求会员详细信息失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	// 解析响应
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	// 检查请求是否成功
	success, ok := result["success"].(bool)
	if !ok || !success {
		message := "请求失败"
		if msg, ok := result["message"].(string); ok {
			message = msg
		}
		return nil, fmt.Errorf("获取会员详细信息失败: %s", message)
	}

	// 提取数据部分
	data, ok := result["data"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("解析data字段失败")
	}

	// 返回会员详细信息
	return data, nil
}

// IsUserForestMember 判断用户是否为森林会员
// account_id: 用户账号ID
// 返回：是否为森林会员、会员等级信息、会员详细信息和可能的错误
func IsUserForestMember(accountID string) (bool, map[string]interface{}, map[string]interface{}, error) {
	// 获取会员等级信息和会员详细信息
	vipInfo, customerInfo, err := GetUserVipLevel(accountID)
	if err != nil {
		return false, nil, nil, err
	}

	// 判断data是否为空（用户不是会员）
	if len(vipInfo) == 0 {
		return false, vipInfo, customerInfo, nil
	}

	// 判断是否为森林会员
	isForestMember := false
	if levelName, ok := vipInfo["level_name"].(string); ok {
		// 检查level_name是否包含"森林会员"
		isForestMember = strings.Contains(levelName, "森林会员")
	}

	// 返回是否为森林会员、会员等级信息和会员详细信息
	return isForestMember, vipInfo, customerInfo, nil
}

// getLevelAliasByLevel 根据会员等级获取对应的level_alias
// level: 会员等级 (1-4)
// 返回：对应的level_alias和可能的错误
func getLevelAliasByLevel(level int) (string, error) {
	// 定义等级与level_alias的映射关系
	levelAliases := map[int]string{
		1: "Y26u1zizjudxy4",
		2: "Y2xgdgw8ft0ffg",
		3: "Y2ocqrc6usv3fg",
		4: "Y2g2obkqqmrcv0",
	}

	// 获取对应等级的level_alias
	alias, exists := levelAliases[level]
	if !exists {
		return "", fmt.Errorf("无效的会员等级: %d，等级必须在1-4之间", level)
	}

	return alias, nil
}

// SetUserVipLevel 设置用户会员等级
// level: 会员等级 (1-4)
// accountID: 用户账号ID（手机号）
// 返回：操作是否成功和可能的错误
func SetUserVipLevel(level int, accountID string) (bool, error) {
	// 首先获取access_token
	accessToken, err := GetYouzanAccessToken()
	if err != nil {
		return false, fmt.Errorf("获取access_token失败: %v", err)
	}

	// 获取对应等级的level_alias
	levelAlias, err := getLevelAliasByLevel(level)
	if err != nil {
		return false, err
	}

	// 构建请求URL
	url := fmt.Sprintf("https://open.youzanyun.com/api/youzan.scrm.customer.level.set/4.0.0?access_token=%s", accessToken)

	// 构建请求体
	requestBody := map[string]interface{}{
		"params": map[string]interface{}{
			"level_alias": levelAlias,
			"user": map[string]interface{}{
				"account_type": 2,
				"account_id":   accountID,
			},
		},
	}

	dataBytes, err := json.Marshal(requestBody)
	if err != nil {
		return false, fmt.Errorf("序列化请求数据失败: %v", err)
	}

	// 发送POST请求
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(dataBytes))
	if err != nil {
		return false, fmt.Errorf("请求设置会员等级失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("读取响应失败: %v", err)
	}

	// 解析响应
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return false, fmt.Errorf("解析响应失败: %v", err)
	}

	// 检查请求是否成功
	success, ok := result["success"].(bool)
	if !ok || !success {
		message := "请求失败"
		if msg, ok := result["message"].(string); ok {
			message = msg
		}
		return false, fmt.Errorf("设置会员等级失败: %s", message)
	}

	// 提取data部分，判断是否设置成功
	data, ok := result["data"].(map[string]interface{})
	if !ok {
		return false, fmt.Errorf("解析data字段失败")
	}

	isSuccess, ok := data["is_success"].(bool)
	if !ok {
		return false, fmt.Errorf("解析is_success字段失败")
	}

	return isSuccess, nil
}
