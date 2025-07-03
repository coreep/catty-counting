package telegram

import (
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

// Dependency provider
type Deps struct {
	logger slog.Logger
}

func (this *Deps) Logger() *slog.Logger {
	return &this.logger
}

func RunBot() {
	deps, bot, err := setup()
	if err != nil {
		deps.logger.Error("Failed to Bot", slog.Any("error", err))
		os.Exit(1)
	}

	runBot(deps, bot)
}

func setup() (*Deps, *tgbotapi.BotAPI, error) {
	deps := Deps{
		logger: *slog.New(slog.NewTextHandler(os.Stdout, nil)),
	}

	if err := godotenv.Load(); err != nil {
		return nil, nil, fmt.Errorf("loading .env: %w", err)
	}
	bot, err := setupBot(&deps)
	if err != nil {
		return nil, nil, fmt.Errorf("setting bot: %w", err)
	}

	return &deps, bot, nil
}

func setupBot(deps *Deps) (*tgbotapi.BotAPI, error) {
	bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_TOKEN"))
	if err != nil {
		return nil, fmt.Errorf("creating bot client: %w", err)
	}
	deps.logger.Info("Authorized on account: " + bot.Self.UserName)

	bot.Debug = true
	return bot, nil
}

func runBot(deps *Deps, bot *tgbotapi.BotAPI) error {
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60
	updates := bot.GetUpdatesChan(updateConfig)

	for update := range updates {
		if update.Message == nil || update.Message.IsCommand() {
			continue
		}

		userMessage := update.Message.Text
		log.Printf("Received message from user: %s", userMessage)

		response, err := llm.QueryLLM(userMessage)
		if err != nil {
			log.Printf("Error querying LLM backend: %v", err)
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Sorry, something went wrong.")
			bot.Send(msg)
			continue
		}
		log.Printf("Raw response from LLM %s\n", response)

		// Every answer from DeepSeek has <think/> tags with details. We don't need to show them to the user
		re := regexp.MustCompile(`<think>[\s\S]*?</think>\n?`)
		response = strings.Trim(re.ReplaceAllString(response, ""), " \n")

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, response)
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending message to user: %v", err)
		}
	}
}
