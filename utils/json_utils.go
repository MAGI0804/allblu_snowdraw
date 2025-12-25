package utils

import (
	"encoding/json"
	"fmt"
)

// MapToJSONString 将map转换为JSON字符串
// 参数:
// - m: 要转换的map
// 返回值:
// - JSON字符串
// - 错误信息
func MapToJSONString(m map[string]interface{}) (string, error) {
	jsonData, err := json.Marshal(m)
	if err != nil {
		return "", fmt.Errorf("map转换为JSON失败: %v", err)
	}
	return string(jsonData), nil
}

// StringMapToJSONString 将map[string]string转换为JSON字符串
// 参数:
// - m: 要转换的map[string]string
// 返回值:
// - JSON字符串
// - 错误信息
func StringMapToJSONString(m map[string]string) (string, error) {
	jsonData, err := json.Marshal(m)
	if err != nil {
		return "", fmt.Errorf("map转换为JSON失败: %v", err)
	}
	return string(jsonData), nil
}

// JSONStringToMap 将JSON字符串转换为map
// 参数:
// - jsonStr: JSON字符串
// 返回值:
// - 解析后的map
// - 错误信息
func JSONStringToMap(jsonStr string) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := json.Unmarshal([]byte(jsonStr), &result)
	if err != nil {
		return nil, fmt.Errorf("JSON转换为map失败: %v", err)
	}
	return result, nil
}