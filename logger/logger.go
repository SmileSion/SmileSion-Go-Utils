package logger

import (
	"fmt"
	"os"
	"sync"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

type Logger struct {
	writer    *lumberjack.Logger
	logChan   chan string
	wg        sync.WaitGroup
	closeOnce sync.Once
	formatter Formatter
}

// 默认格式化器
func defaultFormatter(level, msg string, t time.Time) string {
	return fmt.Sprintf("%s [%s] %s\n",
		t.Format("2006-01-02 15:04:05"),
		level,
		msg,
	)
}

// NewLogger 返回 LoggerInterface
func NewLogger(filename string, maxSize, maxBackups, maxAge int, compress bool, formatter Formatter) LoggerInterface {
	if formatter == nil {
		formatter = defaultFormatter
	}
	l := &Logger{
		writer: &lumberjack.Logger{
			Filename:   filename,
			MaxSize:    maxSize,
			MaxBackups: maxBackups,
			MaxAge:     maxAge,
			Compress:   compress,
		},
		logChan:   make(chan string, 1000),
		formatter: formatter,
	}

	l.wg.Add(1)
	go l.run()

	return l
}

func (l *Logger) run() {
	defer l.wg.Done()
	for msg := range l.logChan {
		if _, err := l.writer.Write([]byte(msg)); err != nil {
			fmt.Fprintf(os.Stderr, "logger write error: %v\n", err)
		}
	}
}

func (l *Logger) log(level, msg string) {
	formatted := l.formatter(level, msg, time.Now())
	select {
	case l.logChan <- formatted:
	default:
		// 丢弃日志时，保证至少在 stderr 打出来
		fmt.Fprintf(os.Stderr, "logger channel full, drop log: %s\n", msg)
	}
}

func (l *Logger) Info(msg string)  { l.log("INFO", msg) }
func (l *Logger) Warn(msg string)  { l.log("WARN", msg) }
func (l *Logger) Error(msg string) { l.log("ERROR", msg) }

func (l *Logger) SetFormatter(f Formatter) {
	if f != nil {
		l.formatter = f
	}
}

func (l *Logger) Close() {
	l.closeOnce.Do(func() {
		close(l.logChan)
		l.wg.Wait()
		_ = l.writer.Close()
	})
}
