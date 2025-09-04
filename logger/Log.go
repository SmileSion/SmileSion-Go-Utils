package logger

import (
	"context"
	"fmt"
	"time"
)

var Log LoggerInterface

func init() {
	textFormatter := func(level, msg string, t time.Time) string {
		return fmt.Sprintf("[%s] [%s] %s\n",
			t.Format("2006-01-02 15:04:05"),
			level,
			msg,
		)
	}

	Log = NewLogger("./logs/app.log", 50, 0, 0, true, textFormatter)
	Log.Info(context.TODO(), "Logger initialized successfully")
}
