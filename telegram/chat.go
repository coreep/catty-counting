package telegram

import (
	"time"

	"github.com/EPecherkin/catty-counting/deps"
	"github.com/EPecherkin/catty-counting/logger"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Chat struct {
	bot             *tgbotapi.BotAPI
	userID          int64
	lastActiveAt    time.Time
	context         [][2]string
	updates         chan tgbotapi.Update
	cancel          func()
	cancelUpdate    *func()
}

func NewChat(bot *tgbotapi.BotAPI, userId int64, cancel func()) *Chat {
	return &Chat{bot: bot, userID: userId, cancel: cancel, updates: make(chan tgbotapi.Update)}
}

func (chat *Chat) Handle(ctx deps.Context) {
	ctx.Deps().Logger().Debug("chat is listening to updates")
	// todo. select listeting to done to clean up chat.apdates
	for update := range chat.updates {
		ctx.Deps().Logger().Debug("chat received update")
		if chat.cancelUpdate != nil {
			ctx.Deps().Logger().Info("chat received update during another interaction. Interrupting...")
			(*chat.cancelUpdate)()
		}
		updateContext, cancel := newUpdateContext(ctx, update)
		chat.cancelUpdate = &cancel
		go chat.goUpdate(updateContext, update)
	}
}

func (chat *Chat) goUpdate(ctx deps.Context, update tgbotapi.Update) {
	defer func() {
		if err := recover(); err != nil {
			ctx.Deps().Logger().With(logger.ERROR, err).Error("panic in goUpdate")
		}
		if chat.cancelUpdate != nil {
			ctx.Deps().Logger().With(logger.MESSAGE, update.Message.MessageID).Debug("exiting during interaction. Interrupting...")
			(*chat.cancelUpdate)()
			chat.cancelUpdate = nil
		}
	}()
	if update.Message == nil || update.Message.IsCommand() {
		ctx.Deps().Logger().With(logger.MESSAGE, update.Message.MessageID).Debug("handling command")
		if err := chat.handleCommand(ctx, update); err != nil {
			ctx.Deps().Logger().With(logger.ERROR, err).Error("failed to handle command")
		}
	} else {
		ctx.Deps().Logger().With(logger.MESSAGE, update.Message.MessageID).Debug("handling message")
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
			With(logger.CHAT, update.Message.Chat.ID).
			With(logger.MESSAGE, update.Message.MessageID),
	))
	newCtx, cancel = newCtx.WithCancel()
	return newCtx, cancel
}
