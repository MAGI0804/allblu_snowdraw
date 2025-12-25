package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Logger 日志工具结构体
type Logger struct {
	filePath string
}

// NewLogger 创建一个新的日志记录器
func NewLogger(logDir, logFileName string) (*Logger, error) {
	// 确保日志目录存在
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("创建日志目录失败: %v", err)
	}

	// 构建完整的日志文件路径
	fullFilePath := filepath.Join(logDir, logFileName)

	// 检查文件是否存在，不存在则创建
	if _, err := os.Stat(fullFilePath); os.IsNotExist(err) {
		file, err := os.Create(fullFilePath)
		if err != nil {
			return nil, fmt.Errorf("创建日志文件失败: %v", err)
		}
		file.Close()
	}

	return &Logger{filePath: fullFilePath}, nil
}

// WriteLog 写入日志到文件
func (l *Logger) WriteLog(level string, format string, args ...interface{}) error {
	// 获取当前时间
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	
	// 格式化日志内容
	logContent := fmt.Sprintf("[%s] [%s] %s\n", timestamp, level, fmt.Sprintf(format, args...))
	
	// 以追加模式打开文件
	file, err := os.OpenFile(l.filePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("打开日志文件失败: %v", err)
	}
	defer file.Close()
	
	// 写入日志
	if _, err := file.WriteString(logContent); err != nil {
		return fmt.Errorf("写入日志失败: %v", err)
	}
	
	return nil
}

// Info 写入信息日志
func (l *Logger) Info(format string, args ...interface{}) error {
	return l.WriteLog("INFO", format, args...)
}

// Error 写入错误日志
func (l *Logger) Error(format string, args ...interface{}) error {
	return l.WriteLog("ERROR", format, args...)
}

// Access 写入访问日志
func (l *Logger) Access(format string, args ...interface{}) error {
	return l.WriteLog("ACCESS", format, args...)
}