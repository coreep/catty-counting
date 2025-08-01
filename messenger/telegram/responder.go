package telegram

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/EPecherkin/catty-counting/deps"
	"github.com/EPecherkin/catty-counting/logger"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
)

const (
	EDIT_INTERVAL      = 2 * time.Second
	THINKING_THRESHOLD = 4 * time.Second
)

type Responder struct {
	message *tgbotapi.Message
	close   func()

	response chan string

	receiver *Receiver

	deps deps.Deps
}

func NewResponder(close func(), receiver *Receiver, deps deps.Deps) *Responder {
	deps.Logger = deps.Logger.With(logger.CALLER, "messenger.telegram.Responder")

	return &Responder{close: close, receiver: receiver, deps: deps, response: make(chan string)}
}

func (resp *Responder) GoRespond(ctx context.Context) {
	defer func() {
		resp.deps.Logger.Debug("stopping responder")
		if err := recover(); err != nil {
			resp.deps.Logger.With(logger.ERROR, err).Error("panic in GoRespond")
		}
		resp.close()
	}()

	responseMessage, err := resp.sendMessage("(Thinking...)")
	if err != nil {
		resp.deps.Logger.With(logger.ERROR, err).Error("Failed to send initial message")
		return
	}

	var responseText string
	var sentText string

	updater := time.NewTicker(EDIT_INTERVAL)
	defer updater.Stop()
	lastUpdate := time.Now()

	for {
		select {
		case <-ctx.Done():
			lgr := resp.deps.Logger
			err = ctx.Err()
			if err != nil {
				lgr = lgr.With(logger.ERROR, errors.WithStack(err))
			}
			lgr.Debug("telegram responder context closed")
			return
		case chunk, ok := <-resp.response:
			if !ok {
				resp.deps.Logger.Debug("response channel closed")
				if err = resp.editMessage(responseMessage, responseText); err != nil {
					resp.deps.Logger.With(logger.ERROR, err).Error("Failed to update message last time")
				}
				return
			}
			resp.deps.Logger.Debug("attaching chunk to response message")
			responseText += chunk
		case <-updater.C:
			if responseText != sentText {
				lastUpdate = time.Now()
				sentText = responseText
				if err = resp.editMessage(responseMessage, responseText); err != nil {
					resp.deps.Logger.With(logger.ERROR, err).Error("Failed to update message")
					return
				}
			} else if since := time.Since(lastUpdate); since > THINKING_THRESHOLD {
				dots := strings.Repeat(".", 1+int(since.Seconds())%int(3*THINKING_THRESHOLD.Seconds()))
				thinking := "\n(Thinking" + dots + ")"
				if err = resp.editMessage(responseMessage, responseText+thinking); err != nil {
					resp.deps.Logger.With(logger.ERROR, err).Error("Failed to update thinking")
				}
			}
		}
	}
}

func (resp *Responder) sendMessage(text string) (*tgbotapi.Message, error) {
	chatID := resp.receiver.telegramUserID
	attrs := tgbotapi.NewMessage(chatID, text)
	message, err := resp.receiver.client.tgbot.Send(attrs)
	if err != nil {
		return nil, fmt.Errorf("sending message: %w", errors.WithStack(err))
	}
	resp.deps.Logger.Debug("sent message to user")
	return &message, nil
}

func (resp *Responder) editMessage(message *tgbotapi.Message, text string) error {
	editMsg := tgbotapi.NewEditMessageText(message.Chat.ID, message.MessageID, text)
	_, err := resp.receiver.client.tgbot.Send(editMsg)
	if err != nil {
		return fmt.Errorf("updating message: %w", errors.WithStack(err))
	}
	resp.deps.Logger.Debug("updated message to user")
	return nil
}
