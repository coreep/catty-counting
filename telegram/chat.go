package telegram

import (
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/EPecherkin/catty-counting/deps"
	"github.com/EPecherkin/catty-counting/logger"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Chat struct {
	UserID       int64
	Cancel       func()
	cancelUpdate *func()
	Updates      chan tgbotapi.Update
	LastActiveAt time.Time
}

func NewChat(userId int64, cancel func()) *Chat {
	return &Chat{UserID: userId, Cancel: cancel}
}

func (chat *Chat) Handle(ctx deps.Context) {
	for update := range chat.Updates {
		if chat.cancelUpdate != nil {
			(*chat.cancelUpdate)()
		}
		updateContext, cancel := newUpdateContext(ctx, update)
		chat.cancelUpdate = &cancel
		go chat.goUpdate(updateContext, update)
	}
}

func (chat *Chat) goUpdate(ctx deps.Context, update tgbotapi.Update) {
	defer func() {
		if chat.cancelUpdate != nil {
			(*chat.cancelUpdate)()
			chat.cancelUpdate = nil
		}
	}()
	if update.Message == nil || update.Message.IsCommand() {
		if err := chat.handleCommand(ctx, update); err != nil {
			ctx.Deps().Logger().With(logger.ERROR, err).Error("failed to handle command")
		}
	} else {
		if err := chat.handleMessage(ctx, update); err != nil {
			ctx.Deps().Logger().With(logger.ERROR, err).Error("failed to handle message")
		}
	}
	chat.cancelUpdate = nil
}

func (chat *Chat) handleCommand(ctx deps.Context, update tgbotapi.Update) error {
}

func (chat *Chat) handleMessage(ctx deps.Context, update tgbotapi.Update) error {
	if err != nil {
		log.Printf("Error querying LLM backend: %v", err)
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Sorry, something went wrong.")
		bot.Send(msg)
		continue
	}
	log.Printf("Raw response from LLM %s\n", response)

	// Every answer from DeepSeek has <think/> tags with details. We don't need to show them to the user
	re := regexp.MustCompile(`<think>[\s\S]*?</think>\n?`)
	response = strings.Trim(re.ReplaceAllString(response, ""), " \n")

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, response)
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending message to user: %v", err)
	}
}

func newChatContext(ctx deps.Context, userId int64) (newCtx deps.Context, cancel func()) {
	newCtx = ctx.WithDeps(deps.NewDeps(
		ctx.Deps().Logger().
			With(logger.USER, userId),
	))
	newCtx, cancel = newCtx.WithCancel()
	return newCtx, cancel
}

func newUpdateContext(ctx deps.Context, update tgbotapi.Update) (newCtx deps.Context, cancel func()) {
	newCtx = ctx.WithDeps(deps.NewDeps(
		ctx.Deps().Logger().
			With(logger.UPDATE, update.UpdateID).
			With(logger.CHAT, update.Message.Chat.ID),
	))
	newCtx, cancel = newCtx.WithCancel()
	return newCtx, cancel
}
