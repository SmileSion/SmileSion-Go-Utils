package main

import (
	"encoding/json"
	"time"

	"utils/logger"
)

func main() {
	// 使用接口
	var log logger.LoggerInterface

	jsonFormatter := func(level, msg string, t time.Time) string {
		m := map[string]interface{}{
			"time":  t.Format(time.RFC3339),
			"level": level,
			"msg":   msg,
		}
		b, _ := json.Marshal(m)
		return string(b) + "\n"
	}

	log = logger.NewLogger("./app.log", 10, 5, 30, true, jsonFormatter)

	defer log.(*logger.Logger).Close() // Close 需要具体类型才能调用

	log.Info("启动应用成功")
	log.Warn("内存占用过高")
	log.Error("数据库连接失败")
}
