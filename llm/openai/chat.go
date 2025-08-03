package openai

import (
	"context"
	"fmt"
	"strings"

	"github.com/EPecherkin/catty-counting/db"
	"github.com/EPecherkin/catty-counting/deps"
	"github.com/EPecherkin/catty-counting/logger"
	"github.com/openai/openai-go"
	"github.com/pkg/errors"
)

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

const systemPrompt = `You are an accounting helping assistant, which is capable of processing docs, receipts, building statistics and giving advices.`

func (chat *Chat) Talk(ctx context.Context, message db.Message, responseChan chan<- string) {
	lgr := chat.deps.Logger.With(logger.USER_ID, message.UserID, logger.MESSAGE_ID, message.ID)
	lgr.Debug("starting talking")
	defer func() {
		if err := recover(); err != nil {
			lgr.With(logger.ERROR, err).Error("failed talking")
		} else {
			lgr.Debug("finished talking")
		}
	}()

	if err := chat.loadHistory(message.UserID); err != nil {
		lgr.With(logger.ERROR, err).Error("failed to load history")
		responseChan <- "Internal error: failed to load message history."
		return
	}

	userMessage := openai.UserMessage(chat.buildContent(message))
	chat.history = append(chat.history, userMessage)

	params := openai.ChatCompletionNewParams{
		Model:    openai.ChatModelGPT4o,
		Messages: chat.history,
	}

	stream := chat.oClient.Chat.Completions.NewStreaming(ctx, params)
	defer func() {
		if err := stream.Close(); err != nil {
			lgr.With(logger.ERROR, errors.WithStack(err)).Error("failed to close talking stream")
		}
	}()

	var responseBuilder strings.Builder
	acc := openai.ChatCompletionAccumulator{}
	for stream.Next() {
		chunk := stream.Current()
		acc.AddChunk(chunk)

		if _, ok := acc.JustFinishedContent(); ok {
			lgr.Debug("stream finished")
			break
		}

		if tool, ok := acc.JustFinishedToolCall(); ok {
			lgr.Debug(fmt.Sprintf("took call finished %v %v %v", tool.Index, tool.Name, tool.Arguments))
			break
		}

		if refusal, ok := acc.JustFinishedRefusal(); ok {
			lgr.Debug(fmt.Sprintf("refusal finisheds %v", refusal))
			break
		}

		content := stream.Current().Choices[0].Delta.Content
		responseBuilder.WriteString(content)
		responseChan <- content
	}

	if err := stream.Err(); err != nil {
		lgr.With(logger.ERROR, errors.WithStack(err)).Error("stream error")
		responseChan <- "Internal error: chat stream failed."
		return
	}

	responseMessage := db.Message{
		UserID: message.UserID,
		Text:   responseBuilder.String(),
		Source: db.MessageSourceLlm,
	}
	if err := chat.deps.DBC.Create(&responseMessage).Error; err != nil {
		lgr.With(logger.ERROR, errors.WithStack(err)).Error("failed to save llm response")
	}

	assistantMessage := openai.AssistantMessage(responseMessage.Text)
	chat.history = append(chat.history, assistantMessage)
}

func (chat *Chat) loadHistory(userID uint) error {
	if len(chat.history) > 0 {
		return nil
	}

	var messages []db.Message
	if err := chat.deps.DBC.Where("user_id = ?", userID).Order("created_at asc").Find(&messages).Error; err != nil {
		return fmt.Errorf("loading messages from db: %w", err)
	}

	chat.history = []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(systemPrompt),
	}

	for _, msg := range messages {
		if msg.Source == db.MessageSourceUser {
			chat.history = append(chat.history, openai.UserMessage(chat.buildContent(msg)))
		} else {
			chat.history = append(chat.history, openai.AssistantMessage(msg.Text))
		}
	}

	return nil
}

func (chat *Chat) buildContent(message db.Message) []openai.ChatCompletionContentPartUnionParam {
	content := []openai.ChatCompletionContentPartUnionParam{
		openai.TextContentPart(message.Text),
	}

	// for _, file := range message.Files {
	// 	// Assuming files are images for now.
	// 	// In a real application, you'd need to handle different file types
	// 	// and potentially use different content parts (e.g., for text files).
	// 	if strings.HasPrefix(file.MimeType, "image/") {
	// 		// This is a placeholder for how you might get a public URL for a file.
	// 		// You would need to implement this part based on your file storage solution.
	// 		fileURL := fmt.Sprintf("https://your-file-storage-service.com/%s", file.BlobKey)
	// 		content = append(content, openai.ImageContentPart(openai.ChatCompletionContentPartImageImageURLParam{URL: fileURL}))
	// 	}
	// }

	return content
}
