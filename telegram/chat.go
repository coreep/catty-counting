package telegram

import (
	"time"

	"github.com/EPecherkin/catty-counting/deps"
	"github.com/EPecherkin/catty-counting/logger"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Chat struct {
	userID       int64
	lastActiveAt time.Time
	context      [][2]string
	updates      chan tgbotapi.Update
	cancel       func()
	cancelUpdate *func()
}

func NewChat(userId int64, cancel func()) *Chat {
	return &Chat{userID: userId, cancel: cancel}
}

func (chat *Chat) Handle(ctx deps.Context) {
	for update := range chat.updates {
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
