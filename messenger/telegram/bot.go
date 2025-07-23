package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/EPecherkin/catty-counting/config"
	"github.com/EPecherkin/catty-counting/llm"
	"github.com/EPecherkin/catty-counting/logger"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
	"gocloud.dev/blob"
)

const (
	TIMEOUT = 60
	OFFSET  = 0
)

type BotDeps struct {
	lgr        *slog.Logger
	files *blob.Bucket
	llmc  llm.Client
}

func NewBotDeps(lgr *slog.Logger, files *blob.Bucket, llmc llm.Client) *BotDeps {
	return &BotDeps{lgr: lgr, files: files, llmc: llmc}
}

type Bot struct {
	deps  *BotDeps
	tgbot *tgbotapi.BotAPI
	chats map[int64]*Chat
}

func NewBot(deps *BotDeps) *Bot {
	return &Bot{
		deps:  deps,
		chats: make(map[int64]*Chat), // int64 tgbotapi.Update.Message.From.ID
	}
}

func (bot *Bot) Run(ctx context.Context) {
	bot.deps.lgr.Debug("Running Bot")
	err := bot.setup()
	if err != nil {
		bot.deps.lgr.With(logger.ERROR, err).Error("Failed to Bot")
		os.Exit(1)
	}

	bot.handleUpdates(ctx)
}

func (bot *Bot) setup() error {
	token := config.TelegramToken()
	tgbot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return fmt.Errorf("creating telegram bot client: %w", errors.WithStack(err))
	}
	bot.deps.lgr.With("user", tgbot.Self.UserName).Info("Authorized on account")

	tgbot.Debug = true
	bot.tgbot = tgbot
	return nil
}

func (bot *Bot) handleUpdates(ctx context.Context) {
	bot.deps.lgr.With("timeout", TIMEOUT).With("offset", OFFSET).Info("listening for updates")
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = TIMEOUT
	updates := bot.tgbot.GetUpdatesChan(updateConfig)

	for update := range updates {
		bot.deps.lgr.With("update", update).Debug("bot received update")
		chat := bot.chatFor(ctx, update.Message.From.ID)
		if chat != nil {
			chat.updates <- update
		}
	}
}

func (bot *Bot) chatFor(ctx context.Context, userID int64) *Chat {
	chat, ok := bot.chats[userID]
	if !ok || chat == nil {
		chatCtx, cancel := context.WithCancel(ctx)
		closeF := func() {
			bot.chats[userID] = nil
			cancel()
		}
		chat = NewChat(userID, closeF, NewChatDeps(bot.deps.lgr, bot.tgbot, bot.deps.files, bot.deps.llmc))
		bot.chats[userID] = chat
		go chat.GoChat(chatCtx)
	}
	return chat
}
