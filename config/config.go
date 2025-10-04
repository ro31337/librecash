package config

import (
	"fmt"
	"github.com/spf13/viper"
	"log"
)

type Config struct {
	Telegram_Token string
	Db_Conn_Str    string
	Rabbit_Url     string
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
}
