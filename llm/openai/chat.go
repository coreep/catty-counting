package openai

import (
	"context"
	"fmt"

	"github.com/EPecherkin/catty-counting/db"
	"github.com/EPecherkin/catty-counting/deps"
	"github.com/EPecherkin/catty-counting/log"
	"github.com/openai/openai-go/v2"
	"github.com/pkg/errors"
)

const PROMPT_SYSTEM_ASSISTANT = `You are an accounting helping assistant, which is capable of processing docs, receipts, building statistics and giving advices. Receiving a message from user, you should analyze if you have necessary details in your context to provide good answer. In case you need any more data about the user - ask user about it. You also have access to tools to retrieve stored information about the user and past interations. Keep your answers reasonably short.`

type Chat struct {
	userID  uint
	oClient *openai.Client
	deps    deps.Deps
	history []openai.ChatCompletionMessageParamUnion
}

func newChat(userID uint, oClient *openai.Client, deps deps.Deps) *Chat {
	deps.Logger = deps.Logger.With(log.CALLER, "openai.Chat").With(log.USER_ID, userID)
	deps.Logger.Debug("Creating openai chat")
	return &Chat{userID: userID, oClient: oClient, deps: deps}
}

func (chat *Chat) Talk(ctx context.Context, message db.Message, responseChan chan<- string) {
	chat.deps.Logger = chat.deps.Logger.With(log.MESSAGE_ID, message.ID)
	chat.deps.Logger.Debug("starting talking")
	defer func() {
		if err := recover(); err != nil {
			chat.deps.Logger.With(log.ERROR, err).Error("failed talking")
		} else {
			chat.deps.Logger.Debug("finished talking")
		}
	}()

	// TODO: graceful shutdown
	if err := chat.loadHistory(); err != nil {
		chat.deps.Logger.With(log.ERROR, errors.WithStack(err)).Error("failed to load user's history")
	}

	if err := chat.deps.DBC.Preload("Files.ExposedFile").Preload("Files").First(&message, message.ID).Error; err != nil {
		chat.deps.Logger.With(log.ERROR, errors.WithStack(err)).Error("failed to preload message files, falling back to provided message")
	}

	if err := chat.handleFiles(ctx, &message); err != nil {
		chat.deps.Logger.With(log.ERROR, err).Error("failed to process provided files")
		// TODO: move to constant
		responseChan <- "Sorry, I failed to handle your request... Could you try again, please?"
		return
	}

	response, err := chat.handleResponse(ctx, &message)
	if err != nil {
		chat.deps.Logger.With(log.ERROR, err).Error("failed to handle response")
		// TODO: move to constant
		responseChan <- "Sorry, I failed to handle your request... Could you try again, please?"
		return
	}
	responseChan <- response
}

func (chat *Chat) loadHistory() error {
	if len(chat.history) > 0 {
		return nil
	}

	var messages []db.Message
	if err := chat.deps.DBC.Where("user_id = ?", chat.userID).Order("created_at asc").Find(&messages).Error; err != nil {
		return fmt.Errorf("loading messages from db: %w", errors.WithStack(err))
	}

	chat.history = []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(PROMPT_SYSTEM_ASSISTANT),
	}

	for _, msg := range messages {
		if msg.Direction == db.MessageDirectionFromUser {
			chat.history = append(chat.history, openai.UserMessage(
				[]openai.ChatCompletionContentPartUnionParam{
					openai.TextContentPart(msg.Text),
				},
			))
		} else if msg.Direction == db.MessageDirectionToUser {
			chat.history = append(chat.history, openai.AssistantMessage(msg.Text))
		}
	}

	return nil
}
