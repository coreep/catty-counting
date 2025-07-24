package telegram

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/EPecherkin/catty-counting/config"
	"github.com/EPecherkin/catty-counting/logger"
	"github.com/EPecherkin/catty-counting/messenger/base"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
	"gocloud.dev/blob"
	"gorm.io/gorm"
)

const (
	TIMEOUT = 60
	OFFSET  = 0
)

type Client struct {
	tgbot    *tgbotapi.BotAPI
	messages chan *base.MessageRequest
	lgr      *slog.Logger
	dbc      *gorm.DB
	files    *blob.Bucket
}

func CreateClient(lgr *slog.Logger, dbc *gorm.DB, files *blob.Bucket) (base.Client, error) {
	lgr = lgr.With(logger.CALLER, "telegram client")
	lgr.Debug("Creating telegram client")
	token := config.TelegramToken()
	tgbot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("creating telegram bot client: %w", errors.WithStack(err))
	}
	lgr.Info("Authorized on account " + tgbot.Self.UserName)

	// TODO: configurable debug mode
	tgbot.Debug = true

	return &Client{tgbot: tgbot, messages: make(chan *base.MessageRequest), lgr: lgr, dbc: dbc, files: files}, nil
}

func (client *Client) GoTalk(ctx context.Context) {
	client.lgr.Debug("Running client")

	client.handleUpdates(ctx)
}

func (client *Client) Messages() <-chan *base.MessageRequest {
	return client.messages
}

func (client *Client) handleUpdates(ctx context.Context) {
	client.lgr.With("timeout", TIMEOUT).With("offset", OFFSET).Info("listening for updates")
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = TIMEOUT
	updates := client.tgbot.GetUpdatesChan(updateConfig)

	for update := range updates {
		client.lgr.With("update", update).Debug("bot received update")
		chat := client.chatFor(ctx, update.Message.From.ID)
		if chat != nil {
			chat.updates <- update
		}
	}
}

func (client *Client) chatFor(ctx context.Context, userID int64) *Chat {
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
