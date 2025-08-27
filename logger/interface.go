package logger

import "time"

// LoggerInterface 定义日志接口
type LoggerInterface interface {
	Info(format string, args ...interface{})
	Warn(format string, args ...interface{})
	Error(format string, args ...interface{})
	SetFormatter(f Formatter)
}


// Formatter 定义日志格式化方法
type Formatter func(level, msg string, t time.Time) string
