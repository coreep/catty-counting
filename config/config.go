package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/pkg/errors"
)

var (
	host    string
	apiPort string

	logLevel string

	fileBucket string

	geminiApiKey  string
	openAiApiKey  string
	telegramToken string
)

func Init() error {
	if err := godotenv.Load(); err != nil {
		return fmt.Errorf("loading .env: %w", errors.WithStack(err))
	}

	logLevel = os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	apiPort = os.Getenv("API_PORT")
	if apiPort == "" {
		apiPort = "8080"
	}

	checkAndSet := map[string]*string{
		"HOST":        &host,
		"FILE_BUCKET": &fileBucket,
		// "GEMINI_API_KEY": &geminiApiKey,
		"OPENAI_API_KEY": &openAiApiKey,
		"TELEGRAM_TOKEN": &telegramToken,
	}
	for varname, varvar := range checkAndSet {
		if err := ensurePresent(varname, varvar); err != nil {
			return err
		}
	}

	return nil
}

func ensurePresent(varname string, varvar *string) error {
	varval := os.Getenv(varname)
	if varval == "" {
		return errors.New(varname + " is missing")
	}
	*varvar = varval
	return nil
}

func LogDebug() bool {
	return logLevel == "debug"
}

func Host() string {
	return host
}

func FileBucket() string {
	return fileBucket
}

func ApiPort() string {
	return apiPort
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
