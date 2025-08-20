package openai

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/EPecherkin/catty-counting/db"
	"github.com/EPecherkin/catty-counting/log"
	"github.com/openai/openai-go/v2"
	"github.com/pkg/errors"
)

func (chat *Chat) handleResponse(ctx context.Context, message *db.Message) (responseFromLLmToUser string, err error) {
	chat.deps.Logger.Debug("requesting response to user")
	userMessageParts := []openai.ChatCompletionContentPartUnionParam{}
	if message.Text != "" {
		userMessageParts = append(userMessageParts, openai.TextContentPart(message.Text))
		chat.deps.Logger.With("message", message.Text).Debug("text appended to request for response")
	}
	// TODO: files seems to be empty
	// TODO: need separate entities to serialize/deserialize to
	for _, f := range message.Files {
		fileDetails, err := json.Marshal(f)
		if err != nil {
			chat.deps.Logger.With(log.ERROR, errors.WithStack(err)).Error("unable to marshal file")
			continue
		}
		chat.deps.Logger.With("file", string(fileDetails)).Debug("file appended to request for response")
		userMessageParts = append(userMessageParts, openai.TextContentPart("File provided: "+string(fileDetails)))
	}

	chat.history = append(chat.history, openai.UserMessage(userMessageParts))

	params := openai.ChatCompletionNewParams{
		Model:    openai.ChatModelGPT5,
		Messages: chat.history,
	}
	resp, err := chat.oClient.Chat.Completions.New(ctx, params)
	if err != nil {
		return "", fmt.Errorf("getting to user response: %w", errors.WithStack(err))
	}
	chat.deps.Logger.Debug("request response to user complete")
	assistantText := ""
	if len(resp.Choices) > 0 && resp.Choices[0].Message.Content != "" {
		assistantText = resp.Choices[0].Message.Content
	}
	if assistantText == "" {
		return "", errors.New("empty response to user")
	}

	responseMessage := db.Message{
		UserID:    message.UserID,
		Text:      assistantText,
		Direction: db.MessageDirectionToUser,
	}
	if err := chat.deps.DBC.Create(&responseMessage).Error; err != nil {
		chat.deps.Logger.With(log.ERROR, errors.WithStack(err)).Error("failed to save summary message")
	}

	assistantMessage := openai.AssistantMessage(assistantText)
	chat.history = append(chat.history, assistantMessage)

	return assistantText, nil
}
