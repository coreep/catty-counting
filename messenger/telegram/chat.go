package telegram

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/EPecherkin/catty-counting/db"
	"github.com/EPecherkin/catty-counting/deps"
	"github.com/EPecherkin/catty-counting/logger"
	"github.com/EPecherkin/catty-counting/messenger/base"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"
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

	deps deps.Deps
}

func NewChat(userID int64, closeF func(), client *Client, deps deps.Deps) *Chat {
	deps.Logger = deps.Logger.With(logger.CALLER, "messenger.telegram.Chat").With(logger.TELEGRAM_USER_ID, userID)
	return &Chat{telegramUserID: userID, close: closeF, client: client, updates: make(chan tgbotapi.Update), deps: deps}
}

func (chat *Chat) GoReceiveMessages(ctx context.Context) {
	chat.deps.Logger.Debug("receiving updates and accumulating messages")
	defer func() {
		chat.deps.Logger.Debug("stopping receiving updates")
		if err := recover(); err != nil {
			chat.deps.Logger.With(logger.ERROR, err).Error("panic in GoReceiveMessages")
		}
		chat.close()
		close(chat.updates)
	}()

	var user db.User
	if err := chat.deps.DBC.Find(user, db.User{TelegramID: chat.telegramUserID}).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			chat.deps.Logger.With(logger.ERROR, err).Error("Failed to query user")
			return
		} else {
			chat.deps.Logger.Debug("Creating new user from telegram")
			user = db.User{TelegramID: chat.telegramUserID}
			if err := chat.deps.DBC.Create(&user).Error; err != nil {
				chat.deps.Logger.With(logger.ERROR, err).Error("Failed to create user")
				return
			}
		}
	}
	chat.user = user
	chat.deps.Logger = chat.deps.Logger.With(logger.USER_ID, user.ID)

	var timeouter *time.Ticker
	refreshTimeouter := func() {
		if timeouter != nil {
			timeouter.Stop()
		}
		timeouter = time.NewTicker(WAIT_FOR_MESSAGE)
	}
	refreshTimeouter()
	defer func() { timeouter.Stop() }()

	var message *db.Message
	var preResponse string

	for {
		select {
		case <-ctx.Done():
			lgr := chat.deps.Logger
			if err := ctx.Err(); err != nil {
				lgr = lgr.With(logger.ERROR, errors.WithStack(err))
			}
			lgr.Debug("telegram chat context closed")
			return
		case update := <-chat.updates:
			tMessage := update.Message
			chat.deps.Logger = chat.deps.Logger.
				With(logger.TELEGRAM_UPDATE_ID, update.UpdateID).
				With(logger.TELEGRAM_CHAT_ID, tMessage.Chat.ID).
				With(logger.TELEGRAM_MESSAGE_ID, tMessage.MessageID)

			if chat.responder != nil {
				chat.deps.Logger.Info("chat received update during another exchange. Interrupting...")
				chat.responder.close()
			}

			if tMessage.Audio != nil || len(tMessage.Entities) > 0 || tMessage.Voice != nil || tMessage.Video != nil || tMessage.VideoNote != nil || tMessage.Sticker != nil || tMessage.Contact != nil || tMessage.Location != nil || tMessage.Venue != nil || tMessage.Poll != nil || tMessage.Dice != nil || tMessage.Invoice != nil {
				chat.deps.Logger.With("update", update).Warn("received unusual content")
				preResponse = "Sorry, I don't yet know how to work with that, but I'll do my best."
			}

			if message == nil {
				chat.deps.Logger.Debug("building new message")
				message = &db.Message{UserID: chat.user.ID, TelegramIDs: []int{tMessage.MessageID}}
				if err := chat.deps.DBC.Create(message).Error; err != nil {
					chat.deps.Logger.With(logger.ERROR, errors.WithStack(err)).Error("Failed to create message")
					// NOTE: important failure
				}
				chat.deps.Logger = chat.deps.Logger.With(logger.MESSAGE_ID, message.ID)
			} else {
				chat.deps.Logger.Debug("appending to message")
				message.TelegramIDs = append(message.TelegramIDs, tMessage.MessageID)
			}

			message.Text += tMessage.Text
			if err := chat.deps.DBC.Save(message).Error; err != nil {
				chat.deps.Logger.With(logger.ERROR, errors.WithStack(err)).Error("Failed to save message")
				// NOTE: important failure
			}

			if tMessage.Document != nil {
				lgr := chat.deps.Logger.With(logger.TELEGRAM_DOCUMENT_ID, tMessage.Document.FileID)
				lgr.Debug("Received document")
				// DONE: TOAI: check content of db/models.go .
				// DONE: TOAI: create new db.File for correspnding chat.user with fields based on data from the update. Add created file to message.files
				// DONE: TOAI: download file using tgbotapi instsance from chat.client. Generate uniq UUID and use it as a key to store the file using blob storage from chat.deps.Files. Set key to db.File
				if err := chat.downloadFile(ctx, &file); err != nil {
					lgr.With(logger.ERROR, errors.WithStack(err)).Error("Filed to download file with document")
				}
				file := db.File{
					MessageID:    message.ID,
					TelegramID:   tMessage.Document.FileID,
					OriginalName: tMessage.Document.FileName,
					MimeType:     tMessage.Document.MimeType,
					Size:         int64(tMessage.Document.FileSize),
				}
				message.Files = append(message.Files, file)
				if err := chat.deps.DBC.Save(&file).Error; err != nil {
					lgr.With(logger.ERROR, errors.WithStack(err)).Error("Filed to save file with document")
					// NOTE: important failure
				}
			}

			if tMessage.Photo != nil {
				for _, photo := range tMessage.Photo {
					lgr := chat.deps.Logger.With(logger.TELEGRAM_PHOTO_ID, photo.FileID)
					lgr.Debug("Received photo")
					// DONE: TOAI: check content of db/models.go .
					// DONE: TOAI: Create new db.File for correspnding chat.user with fields based on data from the update. Add created file to message.files
					// DONE: TOAI: download file using tgbotapi instsance from chat.client. Generate uniq UUID and use it as a key to store the file using blob storage from chat.deps.Files. Set key to db.File
					file := db.File{
						MessageID:  message.ID,
						TelegramID: photo.FileID,
						Size:       int64(photo.FileSize),
					}
					if err := chat.downloadFile(ctx, &file); err != nil {
						lgr.With(logger.ERROR, errors.WithStack(err)).Error("Filed to download file with photo")
					}
					message.Files = append(message.Files, file)
					if err := chat.deps.DBC.Save(&file).Error; err != nil {
						lgr.With(logger.ERROR, errors.WithStack(err)).Error("Failed to save file with photo")
						// NOTE: important failure
					}
				}
			}

			refreshTimeouter()
		case <-timeouter.C:
			if message == nil {
				continue
			}
			chat.deps.Logger.Debug("Message built. Initiating response")
			responseCtx, cancel := context.WithCancel(ctx)
			closeF := func() {
				if chat.responder != nil {
					chat.responder.deps.Logger.Debug("closing responder")
				}
				chat.responder = nil
				cancel()
			}
			responder := NewResponder(closeF, chat, chat.deps)
			chat.responder = responder
			go responder.GoRespond(responseCtx)
			responder.response <- preResponse
			chat.deps.Logger.Debug("Sending message request")
			chat.client.messages <- base.NewMessageRequest(*message, responder.response)
			message = nil
		}
	}
}

func (chat *Chat) downloadFile(ctx context.Context, file *db.File) error {
	fileUrl, err := chat.client.tgbot.GetFileDirectURL(file.TelegramID)
	if err != nil {
		return fmt.Errorf("getting file direct url: %w", err)
	}

	resp, err := http.Get(fileUrl)
	if err != nil {
		return errors.Wrap(err, "downloading file")
	}
	defer resp.Body.Close()

	key := uuid.New().String()
	w, err := chat.deps.Files.NewWriter(ctx, key, nil)
	if err != nil {
		return errors.Wrap(err, "creating new file writer")
	}
	if _, err := io.Copy(w, resp.Body); err != nil {
		return errors.Wrap(err, "copying file to blob")
	}
	if err := w.Close(); err != nil {
		return errors.Wrap(err, "closing writer")
	}

	exposedFile := db.ExposedFile{
		FileID: file.ID,
		Key:    key,
	}
	if err := chat.deps.DBC.Create(&exposedFile).Error; err != nil {
		return errors.Wrap(err, "creating exposed file record")
	}
	file.ExposedFile = &exposedFile
	return nil
}
