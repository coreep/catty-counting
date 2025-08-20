package telegram

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/EPecherkin/catty-counting/db"
	"github.com/EPecherkin/catty-counting/deps"
	"github.com/EPecherkin/catty-counting/log"
	"github.com/EPecherkin/catty-counting/messenger/base"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
)

const (
	EDIT_INTERVAL      = 2 * time.Second
	THINKING_THRESHOLD = 1 * time.Second
)

type Responder struct {
	message   db.Message
	onMessage base.OnMessageCallback

	close func()

	response chan string

	receiver *Receiver

	deps deps.Deps
}

func NewResponder(close func(), message db.Message, onMessage base.OnMessageCallback, receiver *Receiver, deps deps.Deps) *Responder {
	deps.Logger = deps.Logger.With(log.CALLER, "messenger.telegram.Responder")

	return &Responder{close: close, message: message, onMessage: onMessage, receiver: receiver, deps: deps, response: make(chan string)}
}

func (resp *Responder) GoRespond(ctx context.Context) {
	defer func() {
		resp.deps.Logger.Debug("stopping responder")
		if err := recover(); err != nil {
			resp.deps.Logger.With(log.ERROR, err).Error("panic in GoRespond")
		}
		resp.close()
	}()

	responseMessage, err := resp.sendMessage("Thinking...")
	if err != nil {
		resp.deps.Logger.With(log.ERROR, err).Error("Failed to send initial message")
		return
	}

	var responseText string
	var sentText string

	updater := time.NewTicker(EDIT_INTERVAL)
	defer updater.Stop()
	lastUpdate := time.Now()

	go resp.goOnMessage(ctx)

	for {
		select {
		case <-ctx.Done():
			logger := resp.deps.Logger
			err = ctx.Err()
			if err != nil {
				logger = logger.With(log.ERROR, errors.WithStack(err))
			}
			logger.Debug("telegram responder context closed")
			return
		case chunk, ok := <-resp.response:
			if !ok {
				resp.deps.Logger.Debug("response channel closed")
				if responseText == "" {
					resp.deps.Logger.Error("Response from LLM is empty.")
					// TODO: move to constant
					responseText = "Sorry, I couldn't process that. Could you try to re-phrase that?"
				}
				if err = resp.editMessage(responseMessage, responseText); err != nil {
					resp.deps.Logger.With(log.ERROR, err).Error("Failed to update message last time")
				}
				return
			}
			resp.deps.Logger.With("chunk", chunk).Debug("attaching chunk to response message")
			responseText += chunk
		case <-updater.C:
			if responseText != sentText {
				lastUpdate = time.Now()
				sentText = responseText
				if err = resp.editMessage(responseMessage, responseText); err != nil {
					resp.deps.Logger.With(log.ERROR, err).Error("Failed to update message")
					return
				}
			} else if since := time.Since(lastUpdate); since > THINKING_THRESHOLD {
				dots := strings.Repeat(".", 1+int(since.Seconds())%3)
				thinking := "\n(Thinking" + dots + ")"
				if err = resp.editMessage(responseMessage, responseText+thinking); err != nil {
					resp.deps.Logger.With(log.ERROR, err).Error("Failed to update thinking")
				}
			}
		}
	}
}

func (resp *Responder) goOnMessage(ctx context.Context) {
	defer func() {
		close(resp.response)
		if err := recover(); err != nil {
			resp.deps.Logger.With(log.ERROR, err).Error("failed to process onMessage")
		} else {
			resp.deps.Logger.Debug("goOnMessage finished")
		}
	}()
	resp.onMessage(ctx, resp.message, resp.response)
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
	if message.Text == text {
		return nil
	}
	editMsg := tgbotapi.NewEditMessageText(message.Chat.ID, message.MessageID, text)
	_, err := resp.receiver.client.tgbot.Send(editMsg)
	if err != nil {
		return fmt.Errorf("updating message: %w", errors.WithStack(err))
	}
	resp.deps.Logger.Debug("updated message to user")
	return nil
}
