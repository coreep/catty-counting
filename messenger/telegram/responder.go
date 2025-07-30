package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

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

	chat *Chat

	lgr *slog.Logger
}

func NewResponder(close func(), chat *Chat) *Responder {
	lgr := chat.lgr.With(logger.CALLER, "messenger.telegram.Responder")

	return &Responder{close: close, chat: chat, lgr: lgr}
}

func (resp *Responder) GoRespond(ctx context.Context) {
	defer func() {
		if err := recover(); err != nil {
			resp.lgr.With(logger.ERROR, err).Error("panic in GoRespond")
		}
		resp.close()
	}()

	responseMessage, err := resp.sendMessage("Thinking...")
	if err != nil {
		resp.lgr.With(logger.ERROR, err).Error("Failed to send initial message")
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
			err = ctx.Err()
			if err != nil {
				resp.lgr.With(logger.ERROR, errors.WithStack(err)).Debug("llm talk context closed")
			}
			return
		case chunk, ok := <-resp.response:
			if !ok {
				if err = resp.editMessage(responseMessage, responseText); err != nil {
					resp.lgr.With(logger.ERROR, err).Error("Failed to update message last time")
				}
				return
			}
			responseText += chunk
		case <-updater.C:
			if responseText != sentText {
				lastUpdate = time.Now()
				sentText = responseText
				if err = resp.editMessage(responseMessage, responseText); err != nil {
					resp.lgr.With(logger.ERROR, err).Error("Failed to update message")
					return
				}
			} else if since := time.Since(lastUpdate); since > THINKING_THRESHOLD {
				dots := strings.Repeat(".", 1+int(since.Seconds())%3)
				thinking := "\n(Thinking" + dots + ")"
				if err = resp.editMessage(responseMessage, responseText+thinking); err != nil {
					resp.lgr.With(logger.ERROR, err).Error("Failed to update thinking")
				}
			}
		}
	}
}

func (resp *Responder) sendMessage(text string) (*tgbotapi.Message, error) {
	chatID := resp.chat.telegramUserID
	attrs := tgbotapi.NewMessage(chatID, text)
	message, err := resp.chat.client.tgbot.Send(attrs)
	if err != nil {
		return nil, fmt.Errorf("sending message: %w", errors.WithStack(err))
	}
	return &message, nil
}

func (resp *Responder) editMessage(message *tgbotapi.Message, text string) error {
	editMsg := tgbotapi.NewEditMessageText(message.Chat.ID, message.MessageID, text)
	_, err := resp.chat.client.tgbot.Send(editMsg)
	if err != nil {
		return fmt.Errorf("updating message: %w", errors.WithStack(err))
	}
	return nil
}
