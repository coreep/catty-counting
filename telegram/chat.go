package telegram

import (
	"time"

	"github.com/EPecherkin/catty-counting/deps"
	"github.com/EPecherkin/catty-counting/llm"
	"github.com/EPecherkin/catty-counting/logger"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"google.golang.org/genai"
)

type Chat struct {
	bot          *tgbotapi.BotAPI
	userID       int64
	lastActiveAt time.Time
	context      [][2]string
	llm          *genai.Client
	llmChat      *genai.Chat
	updates      chan tgbotapi.Update
	cancel       func()
	cancelUpdate *func()
}

func NewChat(userID int64, cancel func(), bot *tgbotapi.BotAPI, llm *genai.Client) *Chat {
	return &Chat{bot: bot, llm: llm, userID: userID, cancel: cancel, updates: make(chan tgbotapi.Update)}
}

func (chat *Chat) Handle(ctx deps.Context) {
	llmChat, err := llm.NewChat(ctx)
	if err != nil {
		ctx.Deps().Logger().With(logger.ERROR, err).Error("Failed to start llm chatting")
		closeChat(ctx, chat)
		return
	}
	chat.llmChat = llmChat

	ctx.Deps().Logger().Debug("chat is listening to updates")
	// todo. select listeting to done to clean up chat.updates
	for {
		select {
		case update, ok := <-chat.updates:
			if !ok {
				ctx.Deps().Logger().Debug("chat.updates cancelled")
				return
			}
			ctx.Deps().Logger().Debug("chat received update")
			if chat.cancelUpdate != nil {
				ctx.Deps().Logger().Info("chat received update during another excahnge. Interrupting...")
				(*chat.cancelUpdate)()
			}
			updateContext, cancel := newUpdateContext(ctx, update)
			chat.cancelUpdate = &cancel
			go chat.goUpdate(updateContext, update)
		case <-ctx.Done():
			ctx.Deps().Logger().Debug("exchange interrupted")
			close(chat.updates)
			return
		}
	}
}

func (chat *Chat) goUpdate(ctx deps.Context, update tgbotapi.Update) {
	defer func() {
		if err := recover(); err != nil {
			ctx.Deps().Logger().With(logger.ERROR, err).Error("panic in goUpdate")
		}
		if chat.cancelUpdate != nil {
			ctx.Deps().Logger().With(logger.MESSAGE, update.Message.MessageID).Debug("exiting during exchange. Interrupting...")
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
		nil,
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
		nil,
	))
	newCtx, cancel = newCtx.WithCancel()
	return newCtx, cancel
}
