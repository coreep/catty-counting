package telegram

import (
	"context"
	"log/slog"
	"time"

	"github.com/EPecherkin/catty-counting/llm"
	"github.com/EPecherkin/catty-counting/logger"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gocloud.dev/blob"
)

type ChatDeps struct {
	BotDeps
}

func NewChatDeps(lgr *slog.Logger, fileBucket *blob.Bucket, llmClient *llm.Client) *ChatDeps {
	return &ChatDeps{BotDeps: BotDeps{lgr: lgr, fileBucket: fileBucket, llmClient: llmClient}}
}

type Chat struct {
	userID       int64
	close       func()
	deps         *ChatDeps
	updates      chan tgbotapi.Update
	lastActiveAt time.Time
	llmChat      *llm.Chat
	cancelUpdating context.CancelFunc
}

func NewChat(userID int64, closeF func(), deps *ChatDeps) *Chat {
	return &Chat{userID: userID, close: closeF, deps: deps, updates: make(chan tgbotapi.Update)}
}


func (chat *Chat) GoChat(ctx context.Context) {
	chat.deps.lgr.Debug("chat is listening to updates")
	defer func() {
		chat.close()
		if err := recover(); err != nil {
			chat.deps.lgr.With(logger.ERROR, err).Error("panic in GoChat")
		}
	}()
	llmChat, err := chat.deps.llmClient.CreateChat(ctx, chat.deps.lgr)
	if err != nil {
		chat.deps.lgr.With(logger.ERROR, err).Error("Failed to start llm chatting")
		return
	}
	chat.llmChat = llmChat

	// todo. select listeting to done to clean up chat.updates
	for {
		select {
		case update, ok := <-chat.updates:
			if !ok {
				chat.deps.lgr.Debug("chat.updates cancelled")
				return
			}
			chat.deps.lgr.Debug("chat received update")
			if chat.cancelUpdating != nil {
				chat.deps.lgr.Info("chat received update during another excahnge. Interrupting...")
				chat.cancelUpdating()
				chat.cancelUpdating = nil
			}
			updateContext, cancel := context.WithCancel(ctx)
			chat.cancelUpdating = cancel
			go chat.goUpdate(updateContext, update)
		case <-ctx.Done():
			chat.deps.lgr.Debug("exchange interrupted")
			close(chat.updates)
			return
		}
	}
}

func (chat *Chat) goUpdate(ctx context.Context, update tgbotapi.Update) {
	lgr := chat.deps.lgr.
			With(logger.TELEGRAM_UPDATE_ID, update.UpdateID).
			With(logger.TELEGRAM_CHAT_ID, update.Message.Chat.ID).
			With(logger.TELEGRAM_MESSAGE_ID, update.Message.MessageID)

	defer func() {
		if err := recover(); err != nil {
			lgr.With(logger.ERROR, err).Error("panic in goUpdate")
		}
		if chat.cancelUpdating != nil {
			lgr.With(logger.TELEGRAM_MESSAGE_ID, update.Message.MessageID).Debug("exiting during exchange. Interrupting...")
			chat.cancelUpdating()
			chat.cancelUpdating = nil
		}
	}()

	if update.Message == nil || update.Message.IsCommand() {
		lgr.With(logger.TELEGRAM_MESSAGE_ID, update.Message.MessageID).Debug("handling command")
		if err := chat.handleCommand(ctx, update); err != nil {
			lgr.With(logger.ERROR, err).Error("failed to handle command")
		}
	} else {
		lgr.With(logger.TELEGRAM_MESSAGE_ID, update.Message.MessageID).Debug("handling message")
		if err := chat.handleMessage(ctx, update); err != nil {
			lgr.With(logger.ERROR, err).Error("failed to handle message")
		}
	}
	chat.cancelUpdating = nil
}
