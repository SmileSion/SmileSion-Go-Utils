package main

import (
	"fmt"
	"log"

	"utils/config"
)

func main() {
	// 加载多个配置文件
	_, err := config.LoadConfig("config1.toml", "config2.toml")
	if err != nil {
		log.Fatal(err)
	}

	cfg := config.GetConfig()

	fmt.Println("AppName:", cfg.App.Name)
	fmt.Println("AppPort:", cfg.App.Port)
	fmt.Println("Database Host:", cfg.Database.Host)
	fmt.Println("Redis Addr:", cfg.Redis.Addr)
}
