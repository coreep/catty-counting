package openai

import (
	"context"
	"fmt"

	"github.com/EPecherkin/catty-counting/config"
	"github.com/EPecherkin/catty-counting/db"
	"github.com/EPecherkin/catty-counting/deps"
	"github.com/EPecherkin/catty-counting/logger"
	"github.com/openai/openai-go/v2"
	"github.com/pkg/errors"
)

const PROMPT_SYSTEM = `You are an accounting helping assistant, which is capable of processing docs, receipts, building statistics and giving advices.`

type Chat struct {
	oClient *openai.Client
	deps    deps.Deps
	history []openai.ChatCompletionMessageParamUnion
}

func newChat(oClient *openai.Client, deps deps.Deps) *Chat {
	deps.Logger = deps.Logger.With(logger.CALLER, "openai.Chat")
	deps.Logger.Debug("Creating openai chat")
	return &Chat{oClient: oClient, deps: deps}
}

func (chat *Chat) Talk(ctx context.Context, message db.Message, responseChan chan<- string) {
	chat.deps.Logger = chat.deps.Logger.With(logger.USER_ID, message.UserID, logger.MESSAGE_ID, message.ID)
	chat.deps.Logger.Debug("starting talking")
	defer func() {
		if err := recover(); err != nil {
			chat.deps.Logger.With(logger.ERROR, err).Error("failed talking")
		} else {
			chat.deps.Logger.Debug("finished talking")
		}
	}()

	if err := chat.deps.DBC.Preload("Files.ExposedFile").Preload("Files").First(&message, message.ID).Error; err != nil {
		chat.deps.Logger.With(logger.ERROR, errors.WithStack(err)).Error("failed to preload message files, falling back to provided message")
	}

	receipts, err := chat.processFiles(ctx, message)
	if err != nil {
		chat.deps.Logger.With(logger.ERROR, err).Error("failed to process provided files")
	}
	chat.deps.Logger.With("count", len(receipts)).Debug("Files processed, receipts created")
	// TODO:send analysis result to LLM and answer user's request
	summaryText := fmt.Sprintf("Processed %d files, created %d receipts.", len(message.Files), len(receipts))

	responseMessage := db.Message{
		UserID:    message.UserID,
		Text:      summaryText,
		Direction: db.MessageDirectionToUser,
	}
	if err := chat.deps.DBC.Create(&responseMessage).Error; err != nil {
		chat.deps.Logger.With(logger.ERROR, errors.WithStack(err)).Error("failed to save summary message")
	}

	responseChan <- summaryText
	assistantMessage := openai.AssistantMessage(responseMessage.Text)
	chat.history = append(chat.history, assistantMessage)
}

func (chat *Chat) loadHistory(userID uint) error {
	if len(chat.history) > 0 {
		return nil
	}

	var messages []db.Message
	if err := chat.deps.DBC.Where("user_id = ?", userID).Order("created_at asc").Find(&messages).Error; err != nil {
		return fmt.Errorf("loading messages from db: %w", errors.WithStack(err))
	}

	chat.history = []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(PROMPT_SYSTEM),
	}

	for _, msg := range messages {
		if msg.Direction == db.MessageDirectionFromUser {
			chat.history = append(chat.history, openai.UserMessage(
				[]openai.ChatCompletionContentPartUnionParam{
					openai.TextContentPart(msg.Text),
				},
			))
		} else {
			chat.history = append(chat.history, openai.AssistantMessage(msg.Text))
		}
	}

	return nil
}

func (chat *Chat) handleError(err error, details string) string {
	chat.deps.Logger.With(logger.ERROR, err).Error(details)
	if config.LogDebug() {
		return fmt.Sprintf(details+": %+v", err)
	} else {
		return "Sorry, something went wrong"
	}
}
