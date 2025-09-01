package config

import (
	"fmt"
	"os"
	"sync"

	"github.com/BurntSushi/toml"
)

// 全局配置实例
var (
	cfg  *Config
	once sync.Once
)

// Config 结构体（可以根据需要扩展）
type Config struct {
	App      AppConfig      `toml:"app"`
	Database DatabaseConfig `toml:"database"`
	Redis    RedisConfig    `toml:"redis"`
	// 可以继续扩展其他子配置...
}

type AppConfig struct {
	Name string `toml:"name"`
	Port int    `toml:"port"`
	Mode string `toml:"mode"`
}

type DatabaseConfig struct {
	Host     string `toml:"host"`
	Port     int    `toml:"port"`
	User     string `toml:"user"`
	Password string `toml:"password"`
	DBName   string `toml:"dbname"`
}

type RedisConfig struct {
	Addr     string `toml:"addr"`
	Password string `toml:"password"`
	DB       int    `toml:"db"`
}

// LoadConfig 支持加载多个 toml 文件（后面的会覆盖前面的同名字段）
func LoadConfig(files ...string) (*Config, error) {
	var err error
	once.Do(func() {
		cfg = &Config{}
		for _, file := range files {
			if _, err = os.Stat(file); os.IsNotExist(err) {
				err = fmt.Errorf("配置文件不存在: %s", file)
				return
			}
			if _, err = toml.DecodeFile(file, cfg); err != nil {
				err = fmt.Errorf("解析配置文件失败 %s: %v", file, err)
				return
			}
		}
	})
	return cfg, err
}

// GetConfig 获取全局配置
func GetConfig() *Config {
	if cfg == nil {
		panic("配置尚未初始化，请先调用 LoadConfig()")
	}
	return cfg
}
