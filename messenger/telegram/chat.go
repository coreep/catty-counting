package telegram

import (
	"context"
	"log/slog"
	"time"

	"github.com/EPecherkin/catty-counting/db"
	"github.com/EPecherkin/catty-counting/logger"
	"github.com/EPecherkin/catty-counting/messenger/base"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

const (
	WAIT_FOR_MESSAGE = 1 * time.Second
)

type Chat struct {
	telegramUserID int64
	close          func()
	client         *Client

	updates  chan tgbotapi.Update
	response chan string
	user     db.User

	responder *Responder

	lgr *slog.Logger
}

func NewChat(userID int64, closeF func(), client *Client) *Chat {
	lgr := client.lgr.With(logger.CALLER, "messenger.telegram.Chat").With(logger.TELEGRAM_USER_ID, userID)
	return &Chat{telegramUserID: userID, close: closeF, client: client, updates: make(chan tgbotapi.Update), lgr: lgr}
}

func (chat *Chat) GoReceiveMessages(ctx context.Context) {
	chat.lgr.Debug("message builder is accumulating")
	defer func() {
		if err := recover(); err != nil {
			chat.lgr.With(logger.ERROR, err).Error("panic in GoReceiveMessages")
		}
		chat.close()
		close(chat.updates)
	}()

	var user db.User
	if err := chat.client.dbc.Find(user, db.User{TelegramID: chat.telegramUserID}).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			chat.lgr.With(logger.ERROR, err).Error("Failed to query user")
			return
		} else {
			chat.lgr.Debug("Creating new user from telegram")
			user = db.User{TelegramID: chat.telegramUserID}
			if err := chat.client.dbc.Create(&user).Error; err != nil {
				chat.lgr.With(logger.ERROR, err).Error("Failed to create user")
				return
			}
		}
	}
	chat.user = user

	timeouter := time.NewTicker(WAIT_FOR_MESSAGE)
	defer func() { timeouter.Stop() }()

	var message *db.Message
	var preResponse string

	for {
		select {
		case <-ctx.Done():
			lgr := chat.lgr
			if err := ctx.Err(); err != nil {
				lgr = lgr.With(logger.ERROR, errors.WithStack(err))
			}
			lgr.Debug("telegram chat context closed")
			return
		case update := <-chat.updates:
			tMessage := update.Message
			chat.lgr = chat.lgr.
				With(logger.TELEGRAM_UPDATE_ID, update.UpdateID).
				With(logger.TELEGRAM_CHAT_ID, tMessage.Chat.ID).
				With(logger.TELEGRAM_MESSAGE_ID, tMessage.MessageID)

			if tMessage.Audio != nil || len(tMessage.Entities) > 0 || tMessage.Voice != nil || tMessage.Video != nil || tMessage.VideoNote != nil || tMessage.Sticker != nil || tMessage.Contact != nil || tMessage.Location != nil || tMessage.Venue != nil || tMessage.Poll != nil || tMessage.Dice != nil || tMessage.Invoice != nil {
				chat.lgr.With("update", update).Warn("received unusual content")
				preResponse = "Sorry, I don't yet know how to work with that, but I'll do my best."
			}

			if chat.responder != nil {
				chat.lgr.Info("chat received update during another exchange. Interrupting...")
				chat.responder.close()
			}
			if message == nil {
				chat.lgr.Debug("received new update")
				message = &db.Message{TelegramIDs: []int{tMessage.MessageID}}
				if err := chat.client.dbc.Create(message).Error; err != nil {
					chat.lgr.With(logger.ERROR, err).Error("Failed to create message")
				}
				chat.lgr = chat.lgr.With(logger.MESSAGE_ID, message.ID)
			} else {
				chat.lgr.Debug("received extra update for existing message")
				message.TelegramIDs = append(message.TelegramIDs, tMessage.MessageID)
			}
			message.Text += tMessage.Text
			if err := chat.client.dbc.Save(message).Error; err != nil {
				chat.lgr.With(logger.ERROR, err).Error("Failed to save message")
			}
			if tMessage.Document != nil {
				tMessage.Document.FileID
			}
			if tMessage.Photo != nil {
			}

			timeouter.Stop()
			timeouter = time.NewTicker(WAIT_FOR_MESSAGE)
		case <-timeouter.C:
			chat.lgr.Debug("Initiating response")
			responseCtx, cancel := context.WithCancel(ctx)
			closeF := func() {
				chat.response = nil
				cancel()
			}
			responder := NewResponder(closeF, chat)
			chat.responder = responder
			go responder.GoRespond(responseCtx)
			responder.response <- preResponse
			chat.client.messages <- base.NewMessageRequest(*message, responder.response)
			return
		}
	}
}
