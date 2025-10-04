package config

import (
	"fmt"
	"github.com/spf13/viper"
	"log"
)

type Config struct {
	Telegram_Token      string
	Db_Conn_Str         string
	Rabbit_Url          string
	BugSink_DSN         string
	BugSink_Environment string
	BugSink_Release     string
	BugSink_Enabled     bool
}

var config Config

func C() *Config {
	return &config
}

func Init(file string) {
	log.Printf("[CONFIG] Initializing configuration from file: %s", file)

	viper.SetConfigName(file)
	viper.AddConfigPath(".")

	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Error reading config file: %s", err))
	}

	err = viper.Unmarshal(&config)
	if err != nil {
		panic(fmt.Errorf("Error unmarshalling config: %s", err))
	}

	log.Printf("[CONFIG] Configuration loaded successfully")
	log.Printf("[CONFIG] Database connection string configured")
	log.Printf("[CONFIG] RabbitMQ URL configured")
	log.Printf("[CONFIG] BugSink enabled: %v", config.BugSink_Enabled)
	if config.BugSink_Enabled {
		dsnPreview := config.BugSink_DSN
		if len(dsnPreview) > 20 {
			dsnPreview = dsnPreview[:20] + "..."
		}
		log.Printf("[CONFIG] BugSink DSN configured: %s", dsnPreview)
	}
}
