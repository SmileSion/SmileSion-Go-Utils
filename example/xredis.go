package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"utils/db//xredis"
)

func main() {
	ctx := context.Background()

	// 配置 Redis
	cfg := xredis.Config{
		Addr:     "127.0.0.1:6379",
		DB:       0,
		Password: "yourpassword", // 设置密码
		DB:       2,              // 选择 Redis DB 2
	}

	rdb, err := xredis.Open(ctx, cfg)
	if err != nil {
		log.Fatal("open redis error:", err)
	}
	defer rdb.Close()

	// --------------------------------
	// 异步写 (进入队列，由工作池消费)
	// --------------------------------
	rdb.Enqueue(func(c redis.Cmdable) error {
		return c.Set(ctx, "async:key", "hello async", time.Minute).Err()
	})

	// --------------------------------
	// 同步写 (立即执行)
	// --------------------------------
	if err := rdb.ExecSync(ctx, func(c redis.Cmdable) error {
		return c.Set(ctx, "sync:key", "hello sync", time.Minute).Err()
	}); err != nil {
		log.Println("sync set error:", err)
	}

	// --------------------------------
	// 同步读
	// --------------------------------
	if err := rdb.ExecSync(ctx, func(c redis.Cmdable) error {
		val, err := c.Get(ctx, "sync:key").Result()
		if err != nil {
			return err
		}
		fmt.Println("sync:key =", val)
		return nil
	}); err != nil {
		log.Println("sync get error:", err)
	}

	// 稍微等一下，让异步任务被消费掉
	time.Sleep(500 * time.Millisecond)

	if err := rdb.ExecSync(ctx, func(c redis.Cmdable) error {
		val, err := c.Get(ctx, "async:key").Result()
		if err != nil {
			return err
		}
		fmt.Println("async:key =", val)
		return nil
	}); err != nil {
		log.Println("async get error:", err)
	}
}
