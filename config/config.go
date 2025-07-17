package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/pkg/errors"
)

type config struct {
	geminiApiKey  string
	telegramToken string
}

var instance config

func Init() error {
	if err := godotenv.Load(); err != nil {
		return fmt.Errorf("loading .env: %w", errors.WithStack(err))
	}

	geminiApiKey := os.Getenv("GEMINI_API_KEY")
	if geminiApiKey == "" {
		return errors.New("GEMINI_API_KEY is missing")
	}
	telegramToken := os.Getenv("TELEGRAM_TOKEN")
	if telegramToken == "" {
		return errors.New("TELEGRAM_TOKEN is missing")
	}
	instance = config{geminiApiKey: geminiApiKey, telegramToken: telegramToken}
	return nil
}

func GeminiApiKey() string {
	return instance.geminiApiKey
}

func TelegramToken() string {
	return instance.telegramToken
}
