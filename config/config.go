package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/pkg/errors"
)

var (
	fileBucket    string
	geminiApiKey  string
	openAiApiKey  string
	telegramToken string
)

func Init() error {
	if err := godotenv.Load(); err != nil {
		return fmt.Errorf("loading .env: %w", errors.WithStack(err))
	}

	fileBucket = os.Getenv("FILE_BUCKET")
	if geminiApiKey == "" {
		return errors.New("FILE_BUCKET is missing")
	}
	// geminiApiKey = os.Getenv("GEMINI_API_KEY")
	// if geminiApiKey == "" {
	// 	return errors.New("GEMINI_API_KEY is missing")
	// }
	openAiApiKey = os.Getenv("OPENAI_API_KEY")
	if openAiApiKey == "" {
		return errors.New("OPENAI_API_KEY is missing")
	}
	telegramToken = os.Getenv("TELEGRAM_TOKEN")
	if telegramToken == "" {
		return errors.New("TELEGRAM_TOKEN is missing")
	}
	return nil
}

func FileBucket() string {
	return fileBucket
}

func GeminiApiKey() string {
	return geminiApiKey
}

func OpenAiApiKey() string {
	return openAiApiKey
}

func TelegramToken() string {
	return telegramToken
}
