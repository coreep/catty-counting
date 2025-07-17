package telegram

import (
	"fmt"
	"os"

	"github.com/EPecherkin/catty-counting/config"
	"github.com/EPecherkin/catty-counting/deps"
	"github.com/EPecherkin/catty-counting/llm"
	"github.com/EPecherkin/catty-counting/logger"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
)

const (
	TIMEOUT = 60
	OFFSET  = 0
)

type Bot struct {
	llmClient *llm.Client
	tgbot     *tgbotapi.BotAPI
	chats     map[int64]*Chat
}

func NewBot(llmClient *llm.Client) *Bot {
	return &Bot{
		llmClient: llmClient,
		chats:     make(map[int64]*Chat), // int64 tgbotapi.Update.Message.From.ID
	}
}

func (bot *Bot) Run(ctx deps.Context) {
	ctx.Deps().Logger().Debug("Running Bot")
	err := bot.setup(ctx)
	if err != nil {
		ctx.Deps().Logger().With(logger.ERROR, err).Error("Failed to Bot")
		os.Exit(1)
	}

	bot.handleChatting(ctx)
}

func (bot *Bot) setup(ctx deps.Context) error {
	token := config.TelegramToken()
	tgbot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return fmt.Errorf("creating telegram bot client: %w", errors.WithStack(err))
	}
	ctx.Deps().Logger().With("user", tgbot.Self.UserName).Info("Authorized on account")

	tgbot.Debug = true
	bot.tgbot = tgbot
	return nil
}

func (bot *Bot) handleChatting(ctx deps.Context) {
	ctx.Deps().Logger().With("timeout", TIMEOUT).With("offset", OFFSET).Info("listening for updates")
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = TIMEOUT
	updates := bot.tgbot.GetUpdatesChan(updateConfig)

	for update := range updates {
		ctx.Deps().Logger().With("update", update).Debug("bot received update")
		chat := bot.chatFor(ctx, update)
		if chat != nil {
			chat.updates <- update
		}
	}
}

func (bot *Bot) chatFor(ctx deps.Context, update tgbotapi.Update) *Chat {
	userID := update.Message.From.ID
	chat, ok := bot.chats[userID]
	if !ok || chat == nil {
		chatCtx, cancel := newChatContext(ctx, userID)
		chatCtx.Deps().Logger().Info("creating new chat")
		chat = NewChat(userID, cancel, bot)
		bot.chats[userID] = chat
		go bot.goChat(chatCtx, chat)
	}
	return chat
}

func (bot *Bot) closeChat(ctx deps.Context, chat *Chat) {
	bot.chats[chat.userID] = nil
	chat.cancel()
}

func (bot *Bot) goChat(ctx deps.Context, chat *Chat) {
	defer func() {
		if err := recover(); err != nil {
			ctx.Deps().Logger().With(logger.ERROR, err).Error("panic in goChat")
			bot.closeChat(ctx, chat)
		}
	}()
	chat.Handle(ctx)
}
