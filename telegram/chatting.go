package telegram

import (
	"github.com/EPecherkin/catty-counting/deps"
	"github.com/EPecherkin/catty-counting/logger"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const TIMEOUT = 60
const OFFSET = 0

func HandleChatting(ctx deps.Context, bot *tgbotapi.BotAPI) {
	ctx.Deps().Logger().With("timeout", TIMEOUT).With("offset", OFFSET).Info("listening for updates")
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = TIMEOUT
	updates := bot.GetUpdatesChan(updateConfig)

	for update := range updates {
		ctx.Deps().Logger().With("update", update).Debug("bot received update")
		chat := chatFor(ctx, bot, update)
		chat.updates <- update
	}
}

var chats = make(map[int64]*Chat) // int64 tgbotapi.Update.Message.From.ID

func chatFor(ctx deps.Context, bot *tgbotapi.BotAPI, update tgbotapi.Update) *Chat {
	userID := update.Message.From.ID
	chat, ok := chats[userID]
	if !ok || chat == nil {
		chatCtx, cancel := newChatContext(ctx, userID)
		chatCtx.Deps().Logger().Info("creating new chat for a user")
		chat = NewChat(bot, userID, cancel)
		chats[userID] = chat
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
