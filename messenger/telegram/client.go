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
	chats    map[int64]*Chat
	messages chan *base.MessageRequest

	lgr   *slog.Logger
	dbc   *gorm.DB
	files *blob.Bucket
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

	if config.LogLevel() == "debug" {
		tgbot.Debug = true
	}

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
		chat.updates <- update
	}
}

func (client *Client) chatFor(ctx context.Context, userID int64) *Chat {
	chat, ok := client.chats[userID]
	if !ok || chat == nil {
		chatCtx, cancel := context.WithCancel(ctx)
		closeF := func() {
			client.chats[userID] = nil
			cancel()
		}
		chat = NewChat(userID, closeF, client)
		client.chats[userID] = chat
		go chat.GoReceiveMessages(chatCtx)
	}
	return chat
}
