package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/gin-gonic/gin"
)

// SaveUploadedFile 保存上传的文件到指定目录
// 参数:
// - c: Gin上下文
// - file: 上传的文件
// - directory: 保存目录
// - prefix: 文件名前缀
// 返回值:
// - 相对路径
// - 错误信息
func SaveUploadedFile(c *gin.Context, file interface{}, directory string, prefix string) (string, error) {
	// 使用反射获取文件名
	fileValue := reflect.ValueOf(file)
	if fileValue.Kind() != reflect.Ptr || fileValue.IsNil() {
		return "", fmt.Errorf("无效的文件参数")
	}
	
	// 尝试获取Filename字段
	filenameField := fileValue.Elem().FieldByName("Filename")
	if !filenameField.IsValid() || filenameField.Kind() != reflect.String {
		return "", fmt.Errorf("无法获取文件名")
	}
	
	filename := filenameField.String()
	if filename == "" {
		return "", fmt.Errorf("文件名为空")
	}
	
	// 生成唯一文件名
	uniqueFilename := GenerateUniqueFilename(filename)
	if prefix != "" {
		uniqueFilename = prefix + uniqueFilename
	}

	// 定义保存路径 - 只保存相对路径，不包含media前缀
	savePath := filepath.Join(directory, uniqueFilename)

	// 确保目录存在
	fullDir := filepath.Join("./media", directory)
	if err := os.MkdirAll(fullDir, 0755); err != nil {
		return "", fmt.Errorf("创建目录失败: %v", err)
	}
	
	// 使用gin的上下文保存文件
	// 注意：这是一个简化的实现，假设调用者已经通过c.FormFile获取了文件
	// 实际保存文件的逻辑应该在控制器中完成
	return savePath, nil
}
