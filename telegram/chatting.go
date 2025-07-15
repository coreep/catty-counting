package telegram

import (
	"github.com/EPecherkin/catty-counting/deps"
	"github.com/EPecherkin/catty-counting/logger"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"google.golang.org/genai"
)

const (
	TIMEOUT = 60
	OFFSET  = 0
)

var chats = make(map[int64]*Chat) // int64 tgbotapi.Update.Message.From.ID

func HandleChatting(ctx deps.Context, bot *tgbotapi.BotAPI, llm *genai.Client) {
	ctx.Deps().Logger().With("timeout", TIMEOUT).With("offset", OFFSET).Info("listening for updates")
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = TIMEOUT
	updates := bot.GetUpdatesChan(updateConfig)

	for update := range updates {
		ctx.Deps().Logger().With("update", update).Debug("bot received update")
		chat := chatFor(ctx, update, bot, llm)
		if chat != nil {
			chat.updates <- update
		}
	}
}

func chatFor(ctx deps.Context, update tgbotapi.Update, bot *tgbotapi.BotAPI, llm *genai.Client) *Chat {
	userID := update.Message.From.ID
	chat, ok := chats[userID]
	if !ok || chat == nil {
		chatCtx, cancel := newChatContext(ctx, userID)
		chatCtx.Deps().Logger().Info("creating new chat")
		chat = NewChat(userID, cancel, bot, llm)
		chats[userID] = chat
		go goChat(chatCtx, chat)
	}
	return chat
}

func closeChat(ctx deps.Context, chat *Chat) {
	chats[chat.userID] = nil
	chat.cancel()
}

func goChat(ctx deps.Context, chat *Chat) {
	defer func() {
		if err := recover(); err != nil {
			ctx.Deps().Logger().With(logger.ERROR, err).Error("panic in goChat")
			closeChat(ctx, chat)
		}
	}()
	chat.Handle(ctx)
}
