package telegram

import (
	"fmt"
	"strings"
	"time"

	"github.com/EPecherkin/catty-counting/deps"
	"github.com/EPecherkin/catty-counting/llm"
	"github.com/EPecherkin/catty-counting/logger"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
)

const (
	EDIT_INTERVAL      = 2 * time.Second
	THINKING_THRESHOLD = 4 * time.Second
)

func (chat *Chat) handleMessage(ctx deps.Context, update tgbotapi.Update) error {
	chat.lastActiveAt = time.Now()
	message := update.Message
	// .toai todo[Save user message in DB]

	responseMessage, err := chat.sendMessage(message.Chat.ID, "Thinking...")
	if err != nil {
		return fmt.Errorf("sending initial response message: %w", err)
	}

	responseChan := make(chan string) // closed by GoTalk
	errorChan := make(chan error)
	defer close(errorChan)
	go llm.GoTalk(ctx, message.Text, responseChan, errorChan)
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
				ctx.Deps().Logger().With(logger.ERROR, errors.WithStack(err)).Debug("llm talk context closed")
			}
			return nil
		case chunk, ok := <-responseChan:
			if !ok {
				if err = chat.editMessage(responseMessage, responseText); err != nil {
					return fmt.Errorf("updating response last time: %w", err)
				}
				return nil
			}
			responseText += chunk
		case errFromChan, ok := <-errorChan:
			if !ok {
				if err = chat.editMessage(responseMessage, responseText + "\nSomething went wrong. Try again, please."); err != nil {
					return fmt.Errorf("updating response on error: %w", err)
				}
				return nil
			}
			return fmt.Errorf("errChan speaking: %w", &errFromChan)
		case <-updater.C:
			if responseText != sentText {
				lastUpdate = time.Now()
				sentText = responseText
				if err = chat.editMessage(responseMessage, responseText); err != nil {
					return fmt.Errorf("updating response: %w", err)
				}
			} else if since := time.Since(lastUpdate); since > THINKING_THRESHOLD {
				dots := strings.Repeat(".", 1+int(since.Seconds())%3)
				thinking := "\n(Thinking" + dots + ")"
				if err = chat.editMessage(responseMessage, responseText+thinking); err != nil {
					return fmt.Errorf("updating thinking: %w", err)
				}
			}
		}
	}
}

func (chat *Chat) sendMessage(chatID int64, text string) (*tgbotapi.Message, error) {
	attrs := tgbotapi.NewMessage(chatID, text)
	message, err := chat.bot.Send(attrs)
	if err != nil {
		return nil, fmt.Errorf("sending message: %w", errors.WithStack(err))
	}
	return &message, nil
}

func (chat *Chat) editMessage(message *tgbotapi.Message, text string) error {
	editMsg := tgbotapi.NewEditMessageText(message.Chat.ID, message.MessageID, text)
	_, err := chat.bot.Send(editMsg)
	if err != nil {
		return fmt.Errorf("updating message: %w", errors.WithStack(err))
	}
	return nil
}
