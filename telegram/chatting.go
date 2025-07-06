package telegram

import (
	"github.com/EPecherkin/catty-counting/deps"
	"github.com/EPecherkin/catty-counting/logger"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const TIMEOUT = 60

func HandleChatting(ctx deps.Context, bot *tgbotapi.BotAPI) {
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = TIMEOUT
	updates := bot.GetUpdatesChan(updateConfig)

	for update := range updates {
		chat := chatFor(ctx, update)
		chat.updates <- update
	}
}

var chats map[int64]*Chat // int64 tgbotapi.Update.Message.From.ID

func chatFor(ctx deps.Context, update tgbotapi.Update) *Chat {
	userId := update.Message.From.ID
	chat, ok := chats[userId]
	if !ok || chat == nil {
		chatCtx, cancel := newChatContext(ctx, userId)
		chat = NewChat(userId, cancel)
		chats[userId] = chat
		go goChat(chatCtx, chat)
	}
	return chat
}

func goChat(ctx deps.Context, chat *Chat) {
	defer func() {
		if err := recover(); err != nil {
			ctx.Deps().Logger().With(logger.ERROR, err).Error("panic in goChat")
			chats[chat.userID] = nil
			chat.cancel()
		}
	}()
	chat.Handle(ctx)
}
