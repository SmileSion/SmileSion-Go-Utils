package logger

import "time"

// LoggerInterface 定义日志接口
type LoggerInterface interface {
	Info(msg string)
	Warn(msg string)
	Error(msg string)
	SetFormatter(f Formatter)
}

// Formatter 定义日志格式化方法
type Formatter func(level, msg string, t time.Time) string
