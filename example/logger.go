package main

import (
	"fmt"
	"time"

	"utils/logger"
)

func main() {
	// 使用接口
	var log logger.LoggerInterface

	textFormatter := func(level, msg string, t time.Time) string {
		return fmt.Sprintf("[%s] [%s] %s\n",
			t.Format("2006-01-02 15:04:05"), // 自定义时间格式
			level,
			msg,
		)
	}

	log = logger.NewLogger("./app.log", 10, 5, 30, true, textFormatter)

	defer log.(*logger.Logger).Close() // Close 需要具体类型才能调用

	log.Info("启动应用成功")
	log.Warn("内存占用过高")
	log.Error("数据库连接失败")
}
