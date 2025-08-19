package openai

import (
	"context"
	"fmt"

	"github.com/EPecherkin/catty-counting/db"
	"github.com/EPecherkin/catty-counting/log"
	"github.com/openai/openai-go/v2"
	"github.com/pkg/errors"
)

func (chat *Chat) handleResponse(ctx context.Context, message *db.Message) error {

	params := openai.ChatCompletionNewParams{
		Model: openai.ChatModelGPT5,
		Messages: chat.history,
	}
	resp, err := chat.oClient.Chat.Completions.New(ctx, params)
	if err != nil {
		return fmt.Errorf("getting to user response: %w", errors.WithStack(err))
	}
	chat.deps.Logger.Debug("Parsing finished")
	assistantText := ""
	if len(resp.Choices) > 0 && resp.Choices[0].Message.Content != "" {
		assistantText = resp.Choices[0].Message.Content
	}
	if assistantText == "" {
		return errors.New("empty to user response")
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

	return nil
}
