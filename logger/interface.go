package logger

import (
	"context"
	"time"
)

// LoggerInterface 定义日志接口
type LoggerInterface interface {
	Info(ctx context.Context, format string, args ...interface{})
	Warn(ctx context.Context, format string, args ...interface{})
	Error(ctx context.Context, format string, args ...interface{})
	SetFormatter(f Formatter)
	Close()
}


// Formatter 定义日志格式化方法
type Formatter func(level, msg string, t time.Time) string
