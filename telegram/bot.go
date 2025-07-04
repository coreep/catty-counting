package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	// "regexp"
	// "strings"

	"github.com/EPecherkin/catty-counting/deps"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	"github.com/pkg/errors"
)

// RunBot Spinning up Telegram Bot
func RunBot(cctx context.Context) {
	ctx := *initCtx(cctx)
	bot, err := setup(ctx)

	if err != nil {
		ctx.Deps().Logger().With("err", err).Error("Failed to Bot")
		os.Exit(1)
	}

	runBot(ctx, bot)
}

func initCtx(cctx context.Context) (*deps.Context) {
	return deps.NewContext(cctx, deps.NewDeps(slog.New(slog.NewJSONHandler(os.Stdout, nil))))
}

func setup(ctx deps.Context) (*tgbotapi.BotAPI, error) {
	if err := godotenv.Load(); err != nil {
		return nil, fmt.Errorf("loading .env: %v", err)
	}
	bot, err := setupBot(ctx)
	if err != nil {
		return nil, fmt.Errorf("setting bot: %v", err)
	}

	return bot, nil
}

func setupBot(ctx deps.Context) (*tgbotapi.BotAPI, error) {
	token := os.Getenv("TELEGRAM_TOKEN")
	if token == "" {
		return nil, errors.New("TELEGRAM_TOKEN is missing")
	}
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("creating bot client: %v", err)
	}
	ctx.Deps().Logger().With("user", bot.Self.UserName).Info("Authorized on account")

	bot.Debug = true
	return bot, nil
}

func runBot(ctx deps.Context, bot *tgbotapi.BotAPI) error {
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60
	updates := bot.GetUpdatesChan(updateConfig)

	for update := range updates {
		if update.Message == nil || update.Message.IsCommand() {
			continue
		}

		userMessage := update.Message.Text
		slog.Info("Received message from user: %s", userMessage)

		// response, err := llm.QueryLLM(userMessage)
		// if err != nil {
		// 	log.Printf("Error querying LLM backend: %v", err)
		// 	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Sorry, something went wrong.")
		// 	bot.Send(msg)
		// 	continue
		// }
		// log.Printf("Raw response from LLM %s\n", response)
		//
		// // Every answer from DeepSeek has <think/> tags with details. We don't need to show them to the user
		// re := regexp.MustCompile(`<think>[\s\S]*?</think>\n?`)
		// response = strings.Trim(re.ReplaceAllString(response, ""), " \n")
		//
		// msg := tgbotapi.NewMessage(update.Message.Chat.ID, response)
		// if _, err := bot.Send(msg); err != nil {
		// 	log.Printf("Error sending message to user: %v", err)
		// }
	}
	return nil
}
