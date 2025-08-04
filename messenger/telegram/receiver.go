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

type Receiver struct {
	telegramUserID int64
	close          func()
	client         *Client
	onMessage      base.OnMessageCallback

	updates chan tgbotapi.Update
	user    db.User

	responder *Responder

	deps deps.Deps
}

func NewReceiver(tUserID int64, closeF func(), client *Client, onMessage base.OnMessageCallback, deps deps.Deps) *Receiver {
	deps.Logger = deps.Logger.With(logger.CALLER, "messenger.telegram.Receiver").With(logger.TELEGRAM_USER_ID, tUserID)
	return &Receiver{telegramUserID: tUserID, close: closeF, client: client, onMessage: onMessage, updates: make(chan tgbotapi.Update), deps: deps}
}

func (receiver *Receiver) GoReceiveMessages(ctx context.Context) {
	receiver.deps.Logger.Debug("receiving updates and accumulating messages")
	defer func() {
		receiver.deps.Logger.Debug("stopping receiving updates")
		if err := recover(); err != nil {
			receiver.deps.Logger.With(logger.ERROR, err).Error("panic in GoReceiveMessages")
		}
		receiver.close()
		close(receiver.updates)
	}()

	var user db.User
	if err := receiver.deps.DBC.First(&user, db.User{TelegramID: receiver.telegramUserID}).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			receiver.deps.Logger.With(logger.ERROR, err).Error("Failed to query user")
			return
		} else {
			receiver.deps.Logger.Debug("Creating new user from telegram")
			user = db.User{TelegramID: receiver.telegramUserID}
			if err := receiver.deps.DBC.Create(&user).Error; err != nil {
				receiver.deps.Logger.With(logger.ERROR, err).Error("Failed to create user")
				return
			}
		}
	}
	receiver.user = user
	receiver.deps.Logger = receiver.deps.Logger.With(logger.USER_ID, user.ID)

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
		case update := <-receiver.updates:
			tMessage := update.Message
			receiver.deps.Logger = receiver.deps.Logger.
				With(logger.TELEGRAM_UPDATE_ID, update.UpdateID).
				With(logger.TELEGRAM_CHAT_ID, tMessage.Chat.ID).
				With(logger.TELEGRAM_MESSAGE_ID, tMessage.MessageID)

			if receiver.responder != nil {
				receiver.deps.Logger.Info("receiver received update during another exchange. Interrupting...")
				receiver.responder.close()
			}

			if tMessage.Audio != nil || len(tMessage.Entities) > 0 || tMessage.Voice != nil || tMessage.Video != nil || tMessage.VideoNote != nil || tMessage.Sticker != nil || tMessage.Contact != nil || tMessage.Location != nil || tMessage.Venue != nil || tMessage.Poll != nil || tMessage.Dice != nil || tMessage.Invoice != nil {
				receiver.deps.Logger.With("update", update).Warn("received unusual content")
				preResponse = "Sorry, I don't yet know how to work with that, but I'll do my best."
			}

			if message == nil {
				receiver.deps.Logger.Debug("building new message")
				message = &db.Message{UserID: receiver.user.ID, TelegramIDs: []int{tMessage.MessageID}, Direction: db.MessageDirectionFromUser}
				if err := receiver.deps.DBC.Create(message).Error; err != nil {
					receiver.deps.Logger.With(logger.ERROR, errors.WithStack(err)).Error("Failed to create message")
					// NOTE: important failure
				}
				receiver.deps.Logger = receiver.deps.Logger.With(logger.MESSAGE_ID, message.ID)
			} else {
				receiver.deps.Logger.Debug("appending to message")
				message.TelegramIDs = append(message.TelegramIDs, tMessage.MessageID)
			}

			message.Text += tMessage.Text
			if err := receiver.deps.DBC.Save(message).Error; err != nil {
				receiver.deps.Logger.With(logger.ERROR, errors.WithStack(err)).Error("Failed to save message")
				// NOTE: important failure
			}

			if tMessage.Document != nil {
				lgr := receiver.deps.Logger.With(logger.TELEGRAM_DOCUMENT_ID, tMessage.Document.FileID)
				lgr.Debug("Received document")
				blobKey, err := receiver.downloadFile(ctx, tMessage.Document.FileID)
				if err != nil {
					lgr.With(logger.ERROR, errors.WithStack(err)).Error("Filed to download document")
				}
				file := db.File{
					MessageID:    message.ID,
					TelegramID:   tMessage.Document.FileID,
					OriginalName: tMessage.Document.FileName,
					MimeType:     tMessage.Document.MimeType,
					Size:         int64(tMessage.Document.FileSize),
					BlobKey:      blobKey,
				}
				message.Files = append(message.Files, file)
				if err := receiver.deps.DBC.Save(&file).Error; err != nil {
					lgr.With(logger.ERROR, errors.WithStack(err)).Error("Filed to save file with document")
					// NOTE: important failure
				}
			}

			if tMessage.Photo != nil {
				for _, photo := range tMessage.Photo {
					lgr := receiver.deps.Logger.With(logger.TELEGRAM_PHOTO_ID, photo.FileID)
					lgr.Debug("Received photo")
					blobKey, err := receiver.downloadFile(ctx, photo.FileID)
					if err != nil {
						lgr.With(logger.ERROR, errors.WithStack(err)).Error("Filed to download photo")
					}
					file := db.File{
						MessageID:  message.ID,
						TelegramID: photo.FileID,
						Size:       int64(photo.FileSize),
						BlobKey:    blobKey,
					}
					message.Files = append(message.Files, file)
					if err := receiver.deps.DBC.Save(&file).Error; err != nil {
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
			receiver.deps.Logger.Debug("Message built. Initiating response")
			responseCtx, cancel := context.WithCancel(ctx)
			closeF := func() {
				if receiver.responder != nil {
					receiver.responder.deps.Logger.Debug("closing responder")
				}
				receiver.responder = nil
				cancel()
			}
			responder := NewResponder(closeF, *message, receiver.onMessage, receiver, receiver.deps)
			receiver.responder = responder
			go responder.GoRespond(responseCtx)
			responder.response <- preResponse
			receiver.deps.Logger.Debug("responder started working")
			message = nil
		case <-ctx.Done():
			lgr := receiver.deps.Logger
			if err := ctx.Err(); err != nil {
				lgr = lgr.With(logger.ERROR, errors.WithStack(err))
			}
			lgr.Debug("telegram receiver context closed")
			return
		}
	}
}

func (receiver *Receiver) downloadFile(ctx context.Context, telegramID string) (blobKey string, _ error) {
	fileUrl, err := receiver.client.tgbot.GetFileDirectURL(telegramID)
	if err != nil {
		return "", fmt.Errorf("getting file direct url: %w", err)
	}

	resp, err := http.Get(fileUrl)
	if err != nil {
		return "", fmt.Errorf("downloading file: %w", err)
	}
	defer resp.Body.Close()

	blobKey = uuid.New().String()
	w, err := receiver.deps.Files.NewWriter(ctx, blobKey, nil)
	if err != nil {
		return "", fmt.Errorf("creating new file writer: %w", err)
	}
	if _, err := io.Copy(w, resp.Body); err != nil {
		return "", fmt.Errorf("copying file to blob: %w", err)
	}
	if err := w.Close(); err != nil {
		return "", fmt.Errorf("closing writer: %w", err)
	}

	return blobKey, nil
}
