package telegram

import (
	"context"
	"fmt"
	"time"

	"github.com/EPecherkin/catty-counting/deps"
	"github.com/EPecherkin/catty-counting/llm"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

package telegram

import (
	"context"
	"fmt"
	"time"

	"github.com/EPecherkin/catty-counting/deps"
	"github.com/EPecherkin/catty-counting/llm"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (chat *Chat) handleMessage(ctx deps.Context, update tgbotapi.Update) error {
	chat.lastActiveAt = time.Now()
	message := update.Message
	// .toai todo[Save user message in DB]

	thinkingCtx, cancelThinking := context.WithCancel(context.Background())
	defer cancelThinking()
	thinkingMessage, err := chat.sendThinkingMessage(thinkingCtx, message.Chat.ID)
	if err != nil {
		return fmt.Errorf("sending thinking message: %w", err)
	}

	llm.HandleUserMessage(ctx, chat.bot, update, thinkingMessage)

	// .toai done[
	//   Call a separate function to respond user with "Thinking" plus 3 dots after that. Every 2 seconds this message should be edited and the amount of dots should change on that message, indicating that "thinking" is in progress
	//   Call llm.HandleUserMessage. It should stream llm's answer, updating the previous sent messasge. BTW, while stream is active - the animated dots should dissapear.
	// ]

	return nil
}

func (chat *Chat) sendThinkingMessage(ctx context.Context, chatID int64) (*tgbotapi.Message, error) {
	msg := tgbotapi.NewMessage(chatID, "Thinking")
	message, err := chat.bot.Send(msg)
	if err != nil {
		return nil, fmt.Errorf("sending initial thinking message: %w", err)
	}

	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		dots := 1
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				text := "Thinking"
				for i := 0; i < dots; i++ {
					text += "."
				}
				dots = (dots % 3) + 1
				editMsg := tgbotapi.NewEditMessageText(chatID, message.MessageID, text)
				_, err := chat.bot.Send(editMsg)
				if err != nil {
					// Log error, but don't stop the ticker
				}
			}
		}
	}()

	return &message, nil
}
