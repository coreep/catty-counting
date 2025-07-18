package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/EPecherkin/catty-counting/llm"
	"github.com/EPecherkin/catty-counting/logger"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
	"gocloud.dev/blob"
)

const (
	EDIT_INTERVAL      = 2 * time.Second
	THINKING_THRESHOLD = 4 * time.Second
)

type ResponseDeps struct {
	lgr        *slog.Logger
	tgbot      *tgbotapi.BotAPI
	fileBucket *blob.Bucket
	llmChat    *llm.Chat
}

func NewResponseDeps(lgr *slog.Logger, tgbot *tgbotapi.BotAPI, fileBucket *blob.Bucket, llmChat *llm.Chat) *ResponseDeps {
	return &ResponseDeps{lgr: lgr, tgbot: tgbot, fileBucket: fileBucket, llmChat: llmChat}
}

type Response struct {
	update   tgbotapi.Update
	response *tgbotapi.Message
	close    func()
	deps     *ResponseDeps
}

func NewResponse(update tgbotapi.Update, close func(), deps *ResponseDeps) *Response {
	deps.lgr = deps.lgr.
		With(logger.TELEGRAM_UPDATE_ID, update.UpdateID).
		With(logger.TELEGRAM_CHAT_ID, update.Message.Chat.ID).
		With(logger.TELEGRAM_MESSAGE_ID, update.Message.MessageID)

	return &Response{update: update, close: close, deps: deps}
}

func (resp *Response) GoRespond(ctx context.Context) {
	defer func() {
		if err := recover(); err != nil {
			resp.deps.lgr.With(logger.ERROR, err).Error("panic in goUpdate")
		}
		resp.close()
	}()

	if resp.update.Message == nil || resp.update.Message.IsCommand() {
		resp.deps.lgr.Warn("Can't handle commands yet")
	} else {
		resp.deps.lgr.Debug("handling message")
		if err := resp.handleMessage(ctx); err != nil {
			resp.deps.lgr.With(logger.ERROR, err).Error("failed to handle message")
		}
	}
}

func (resp *Response) handleMessage(ctx context.Context) error {
	message := resp.update.Message
	// .toai todo[Save user message in DB]

	responseMessage, err := resp.sendMessage(message.Chat.ID, "Thinking...")
	if err != nil {
		return fmt.Errorf("sending initial response message: %w", err)
	}

	responseChan := make(chan string) // closed by GoTalk
	errorChan := make(chan error)
	defer close(errorChan)
	go resp.deps.llmChat.GoTalk(ctx, message.Text, responseChan, errorChan)
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
				resp.deps.lgr.With(logger.ERROR, errors.WithStack(err)).Debug("llm talk context closed")
			}
			return nil
		case chunk, ok := <-responseChan:
			if !ok {
				if err = resp.editMessage(responseMessage, responseText); err != nil {
					return fmt.Errorf("updating response last time: %w", err)
				}
				return nil
			}
			responseText += chunk
		case errFromChan, ok := <-errorChan:
			if !ok {
				if err = resp.editMessage(responseMessage, responseText+"\nSomething went wrong. Try again, please."); err != nil {
					return fmt.Errorf("updating response on error: %w", err)
				}
				return nil
			}
			return fmt.Errorf("errChan speaking: %w", errFromChan)
		case <-updater.C:
			if responseText != sentText {
				lastUpdate = time.Now()
				sentText = responseText
				if err = resp.editMessage(responseMessage, responseText); err != nil {
					return fmt.Errorf("updating response: %w", err)
				}
			} else if since := time.Since(lastUpdate); since > THINKING_THRESHOLD {
				dots := strings.Repeat(".", 1+int(since.Seconds())%3)
				thinking := "\n(Thinking" + dots + ")"
				if err = resp.editMessage(responseMessage, responseText+thinking); err != nil {
					return fmt.Errorf("updating thinking: %w", err)
				}
			}
		}
	}
}

func (resp *Response) sendMessage(chatID int64, text string) (*tgbotapi.Message, error) {
	attrs := tgbotapi.NewMessage(chatID, text)
	message, err := resp.deps.tgbot.Send(attrs)
	if err != nil {
		return nil, fmt.Errorf("sending message: %w", errors.WithStack(err))
	}
	return &message, nil
}

func (resp *Response) editMessage(message *tgbotapi.Message, text string) error {
	editMsg := tgbotapi.NewEditMessageText(message.Chat.ID, message.MessageID, text)
	_, err := resp.deps.tgbot.Send(editMsg)
	if err != nil {
		return fmt.Errorf("updating message: %w", errors.WithStack(err))
	}
	return nil
}
