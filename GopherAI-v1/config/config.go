package config

import (
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/BurntSushi/toml"
)

type MainConfig struct {
	Port    int    `toml:"port"`
	AppName string `toml:"appName"`
	Host    string `toml:"host"`
}

type EmailConfig struct {
	Authcode string `toml:"authcode"`
	Email    string `toml:"email" `
}

type RedisConfig struct {
	RedisPort     int    `toml:"port"`
	RedisDb       int    `toml:"db"`
	RedisHost     string `toml:"host"`
	RedisPassword string `toml:"password"`
}

type MysqlConfig struct {
	MysqlPort         int    `toml:"port"`
	MysqlHost         string `toml:"host"`
	MysqlUser         string `toml:"user"`
	MysqlPassword     string `toml:"password"`
	MysqlDatabaseName string `toml:"databaseName"`
	MysqlCharset      string `toml:"charset"`
}

type JwtConfig struct {
	ExpireDuration int    `toml:"expire_duration"`
	Issuer         string `toml:"issuer"`
	Subject        string `toml:"subject"`
	Key            string `toml:"key"`
}

type Rabbitmq struct {
	RabbitmqPort     int    `toml:"port"`
	RabbitmqHost     string `toml:"host"`
	RabbitmqUsername string `toml:"username"`
	RabbitmqPassword string `toml:"password"`
	RabbitmqVhost    string `toml:"vhost"`
}

type Config struct {
	EmailConfig `toml:"emailConfig"`
	RedisConfig `toml:"redisConfig"`
	MysqlConfig `toml:"mysqlConfig"`
	JwtConfig   `toml:"jwtConfig"`
	MainConfig  `toml:"mainConfig"`
	Rabbitmq    `toml:"rabbitmqConfig"`
}

type RedisKeyConfig struct {
	CaptchaPrefix string
}

var DefaultRedisKeyConfig = RedisKeyConfig{
	CaptchaPrefix: "captcha:%s",
}

var (
	config *Config
	once   sync.Once
)

func configPathCandidates() []string {
	return []string{
		"config/config.local.toml",
		"config/config.toml",
	}
}

// InitConfig 初始化项目配置
func InitConfig() error {
	for _, configPath := range configPathCandidates() {
		if _, err := os.Stat(configPath); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			log.Fatal(err.Error())
			return err
		}
		if _, err := toml.DecodeFile(configPath, config); err != nil {
			log.Fatal(err.Error())
			return err
		}
		return nil
	}
	return fmt.Errorf("no config file found in config/config.local.toml or config/config.toml")
}

func GetConfig() *Config {
	once.Do(func() {
		config = new(Config)
		_ = InitConfig()
	})
	return config
}
