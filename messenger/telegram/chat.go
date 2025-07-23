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
	tgbot *tgbotapi.BotAPI
	BotDeps
}

func NewChatDeps(lgr *slog.Logger, tgbot *tgbotapi.BotAPI, files *blob.Bucket, llmc llm.Client) *ChatDeps {
	return &ChatDeps{tgbot: tgbot, BotDeps: BotDeps{lgr: lgr, files: files, llmc: llmc}}
}

type Chat struct {
	userID       int64
	close        func()
	deps         *ChatDeps
	updates      chan tgbotapi.Update
	lastActiveAt time.Time
	llmChat      llm.Chat
	response     *Response
}

func NewChat(userID int64, closeF func(), deps *ChatDeps) *Chat {
	deps.lgr = deps.lgr.With(logger.TELEGRAM_USER_ID, userID)
	deps.lgr.Info("creating new chat")
	return &Chat{userID: userID, close: closeF, deps: deps, updates: make(chan tgbotapi.Update)}
}

func (chat *Chat) GoChat(ctx context.Context) {
	chat.deps.lgr.Debug("chat is listening to updates")
	defer func() {
		if err := recover(); err != nil {
			chat.deps.lgr.With(logger.ERROR, err).Error("panic in GoChat")
		}
		chat.close()
	}()

	llmChat, err := chat.deps.llmc.CreateChat(ctx, chat.deps.lgr)
	if err != nil {
		chat.deps.lgr.With(logger.ERROR, err).Error("Failed to start llm chatting")
		return
	}
	chat.llmChat = llmChat

	// todo. select listeting to done to clean up chat.updates
	for {
		select {
		case update := <-chat.updates:
			chat.lastActiveAt = time.Now()
			chat.deps.lgr.Debug("chat received update")
			if chat.response != nil {
				chat.deps.lgr.Info("chat received update during another exchange. Interrupting...")
				chat.response.close()
			}
			responseCtx, cancel := context.WithCancel(ctx)
			closeF := func() {
				chat.response = nil
				cancel()
			}
			response := NewResponse(update, closeF, NewResponseDeps(chat.deps.lgr, chat.deps.tgbot, chat.deps.files, chat.llmChat))
			go response.GoRespond(responseCtx)
		case <-ctx.Done():
			chat.deps.lgr.Debug("exchange interrupted")
			close(chat.updates)
			return
		}
	}
}
