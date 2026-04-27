package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	TelegramToken string
	ProxyURL      string
	DatabaseURL string
	AlchemyWssURL string
	UsdcContract string
}

func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		return nil, fmt.Errorf("loading .env: %w", err)
	}

	conf := Config{
		TelegramToken: os.Getenv("TELEGRAM_TOKEN"),
		ProxyURL: os.Getenv("PROXY_URL"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		AlchemyWssURL: os.Getenv("ALCHEMY_WSS_URL"),
		UsdcContract: os.Getenv("USDC_CONTRACT"),
	}
	if conf.TelegramToken == "" {
		return nil, fmt.Errorf("TELEGRAM_TOKEN is required")
	}
	if conf.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if conf.AlchemyWssURL == ""{
		return nil, fmt.Errorf("ALCHEMY_WSS_URL is required")
	}
	return &conf, nil
}