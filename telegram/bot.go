package telegram

import (
	"context"
	"fmt"
	"os"

	"github.com/EPecherkin/catty-counting/deps"
	"github.com/EPecherkin/catty-counting/logger"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	"github.com/pkg/errors"
)

// RunBot Spinning up Telegram Bot
func RunBot(cctx context.Context) {
	ctx := newBotContext(cctx)
	bot, err := setup(ctx)
	if err != nil {
		ctx.Deps().Logger().With(logger.ERROR, err).Error("Failed to Bot")
		os.Exit(1)
	}

	HandleChatting(ctx, bot)
}

func newBotContext(cctx context.Context) deps.Context {
	return deps.NewContext(
		cctx,
		deps.NewDeps(
			logger.NewLogger(),
			// .toai [add db.NewConnection()](todo)
		),
	)
}

func setup(ctx deps.Context) (*tgbotapi.BotAPI, error) {
	if err := godotenv.Load(); err != nil {
		return nil, fmt.Errorf("loading .env: %w", errors.WithStack(err))
	}
	bot, err := setupBot(ctx)
	if err != nil {
		return nil, fmt.Errorf("setting bot: %w", err)
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
		return nil, fmt.Errorf("creating bot client: %w", err)
	}
	ctx.Deps().Logger().With("user", bot.Self.UserName).Info("Authorized on account")

	bot.Debug = true
	return bot, nil
}
