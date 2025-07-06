package telegram

import (
	"time"

	"github.com/EPecherkin/catty-counting/deps"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	// "github.com/EPecherkin/catty-counting/llm"
)

func (chat *Chat) handleMessage(ctx deps.Context, update tgbotapi.Update) error {
	chat.lastActiveAt = time.Now()
	message := update.Message
	// .toai [Save user message in DB](todo)
	// .toai [
	//   Call a separate function to respond user with "Thinking" plus 3 dots after that. Every 2 seconds this message should be edited and the amount of dots should change on that message, indicating that "thinking" is in progress
	//   Call llm.HandleUserMessage. It should stream llm's answer, updating the previous sent messasge. BTW, while stream is active - the animated dots should dissapear.
	// ]
	
	return nil
}
