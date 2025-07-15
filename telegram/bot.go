package telegram

import (
	"fmt"
	"os"

	"github.com/EPecherkin/catty-counting/config"
	"github.com/EPecherkin/catty-counting/deps"
	"github.com/EPecherkin/catty-counting/logger"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
)

// RunBot Spinning up Telegram Bot
func RunBot(ctx deps.Context) {
	ctx.Deps().Logger().Debug("Running Bot")
	bot, err := setup(ctx)
	if err != nil {
		ctx.Deps().Logger().With(logger.ERROR, err).Error("Failed to Bot")
		os.Exit(1)
	}

	HandleChatting(ctx, bot)
}

func setup(ctx deps.Context) (*tgbotapi.BotAPI, error) {
	bot, err := setupBot(ctx)
	if err != nil {
		return nil, fmt.Errorf("setting bot: %w", err)
	}

	return bot, nil
}

func setupBot(ctx deps.Context) (*tgbotapi.BotAPI, error) {
	token := config.TelegramToken()
	if token == "" {
		return nil, errors.New("TELEGRAM_TOKEN is missing")
	}
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("creating bot client: %w", err)
	}
	ctx.Deps().Logger().With("user", bot.Self.UserName).Info("Authorized on account")

	bot.Debug = true
	return bot, nil
}
