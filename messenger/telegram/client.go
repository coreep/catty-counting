package telegram

import (
	"context"
	"fmt"

	"github.com/EPecherkin/catty-counting/config"
	"github.com/EPecherkin/catty-counting/deps"
	"github.com/EPecherkin/catty-counting/logger"
	"github.com/EPecherkin/catty-counting/messenger/base"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
)

const (
	TIMEOUT = 60
	OFFSET  = 0
)

type Client struct {
	tgbot *tgbotapi.BotAPI
	// db.User.TelegramID to Receiver. TODO:change to db.User.ID
	receiverPerUser map[int64]*Receiver
	messages        chan *base.MessageRequest

	deps deps.Deps
}

func CreateClient(deps deps.Deps) (base.Client, error) {
	deps.Logger = deps.Logger.With(logger.CALLER, "messenger.telegram.client")
	deps.Logger.Debug("Creating telegram client")
	token := config.TelegramToken()
	tgbot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("creating telegram bot client: %w", errors.WithStack(err))
	}
	deps.Logger.Info("Authorized on account " + tgbot.Self.UserName)

	if logger.IsDebug() {
		tgbot.Debug = true
	}

	return &Client{tgbot: tgbot, messages: make(chan *base.MessageRequest), deps: deps, receiverPerUser: make(map[int64]*Receiver)}, nil
}

func (client *Client) GoListen(ctx context.Context) {
	client.deps.Logger.Debug("Running client")

	client.handleUpdates(ctx)
}

func (client *Client) Messages() <-chan *base.MessageRequest {
	return client.messages
}

func (client *Client) handleUpdates(ctx context.Context) {
	client.deps.Logger.With("timeout", TIMEOUT).With("offset", OFFSET).Info("listening for updates")
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = TIMEOUT
	updates := client.tgbot.GetUpdatesChan(updateConfig)

	for update := range updates {
		client.deps.Logger.With("update", update).Debug("bot received update")

		receiver := client.receiverFor(ctx, update.Message.From.ID)
		receiver.updates <- update
	}
}

func (client *Client) receiverFor(ctx context.Context, userID int64) *Receiver {
	handler, ok := client.receiverPerUser[userID]
	if !ok || handler == nil {
		client.deps.Logger.Debug("building new handler")
		handlerCtx, cancel := context.WithCancel(ctx)
		closeF := func() {
			if client.receiverPerUser[userID] != nil {
				client.receiverPerUser[userID].deps.Logger.Debug("Closing handler")
			}
			client.receiverPerUser[userID] = nil
			cancel()
		}
		handler = NewReceiver(userID, closeF, client, client.deps)
		client.receiverPerUser[userID] = handler
		go handler.GoReceiveMessages(handlerCtx)
	} else {
		client.deps.Logger.Debug("utilizing existing handler")
	}
	return handler
}
