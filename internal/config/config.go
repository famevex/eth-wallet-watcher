package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	TelegramToken string
	ProxyURL      string
}

func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
		return nil, err
	}

	conf := Config{
		TelegramToken: os.Getenv("TELEGRAM_TOKEN"),
		ProxyURL: os.Getenv("PROXY_URL"),
	}
	return &conf, nil
}