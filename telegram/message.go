package telegram

import (
	"fmt"
	"strings"
	"time"

	"github.com/EPecherkin/catty-counting/deps"
	"github.com/EPecherkin/catty-counting/llm"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
)

func (chat *Chat) handleMessage(ctx deps.Context, update tgbotapi.Update) error {
	chat.lastActiveAt = time.Now()
	message := update.Message
	// .toai todo[Save user message in DB]

	responseMessage, err := chat.sendMessage(message.Chat.ID, message.Text)
	if err != nil {
		return fmt.Errorf("sending initial response message: %w", err)
	}

	responseChan := make(chan<- string)
	errorChan := make(chan<- error)
	go llm.HandleUserMessage(ctx, responseMessage.Text, responseChan, errorChan)
	var responseText string

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	lastUpdate := time.Now()

	for {
		select {
		case chunk, ok := <-responseChan:
			if !ok {
				return
			}
			responseText += chunk
			lastUpdate = time.Now()
			editMsg := tgbotapi.NewEditMessageText(responseMessage.Chat.ID, responseMessage.MessageID, responseText)
			_, err := chat.bot.Send(editMsg)
			if err != nil {
				// Log error
			}
		case err := <-errChan:
		case <-ticker.C:
			if time.Since(lastUpdate) > 4*time.Second {
				chat.showThinkingAnimation(responseMessage, responseText, &lastUpdate)
			}
		}
	}

	return nil
}

func (chat *Chat) showThinkingAnimation(message *tgbotapi.Message, currentText string, lastUpdate *time.Time) {
	dots := 1
	for {
		select {
		case <-time.After(2 * time.Second):
			if time.Since(*lastUpdate) < 4*time.Second {
				return
			}
			var thinkingDots strings.Builder
			for i := 0; i < dots; i++ {
				thinkingDots.WriteString(".")
			}
			text := fmt.Sprintf("%s (Thinking%s)", currentText, thinkingDots.String())
			dots = (dots % 3) + 1
			editMsg := tgbotapi.NewEditMessageText(message.Chat.ID, message.MessageID, text)
			_, err := chat.bot.Send(editMsg)
			if err != nil {
				// Log error
			}
		default:
			time.Sleep(100 * time.Millisecond)
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
