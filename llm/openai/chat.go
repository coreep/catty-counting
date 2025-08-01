package openai

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/EPecherkin/catty-counting/logger"
	"github.com/openai/openai-go"
	"github.com/pkg/errors"
)

type Chat struct {
	oClient *openai.Client
	lgr     *slog.Logger
	history *[]openai.ChatCompletionMessageParamUnion
}

func newChat(oClient *openai.Client, lgr *slog.Logger) *Chat {
	lgr = lgr.With(logger.CALLER, "openai chat")
	lgr.Debug("Creating openai chat")
	return &Chat{oClient: oClient, lgr: lgr}
}

const systemPrompt = `You are an accounting helping assistant, which is capable of processing docs, receipts, building statistics and giving advices.`

func (chat *Chat) GoTalk(ctx context.Context, message string, responseChan chan<- string, errorChan chan<- error) {
	chat.lgr.Debug("starting talking")
	defer func() {
		close(responseChan)
		if err := recover(); err != nil {
			chat.lgr.With(logger.ERROR, err).Error("failed talking")
		} else {
			chat.lgr.Debug("finished talking")
		}
	}()

	params := openai.ChatCompletionNewParams{
		Model: openai.ChatModelGPT4o,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage([]openai.ChatCompletionContentPartUnionParam{
				openai.TextContentPart(message),
				// openai.ImageContentPart(openai.ChatCompletionContentPartImageImageURLParam{URL: imageUrl}),
			}),
		},
	}

	stream := chat.oClient.Chat.Completions.NewStreaming(ctx, params)
	defer func() {
		if err := stream.Close(); err != nil {
			chat.lgr.With(logger.ERROR, errors.WithStack(err)).Error("failed to close stream")
		}
	}()

	acc := openai.ChatCompletionAccumulator{}
	for stream.Next() {
		chunk := stream.Current()
		acc.AddChunk(chunk)

		if _, ok := acc.JustFinishedContent(); ok {
			chat.lgr.Debug("stream finished")
			return
		}

		if tool, ok := acc.JustFinishedToolCall(); ok {
			chat.lgr.Debug(fmt.Sprintf("took call finished %v %v %v", tool.Index, tool.Name, tool.Arguments))
			return
		}

		if refusal, ok := acc.JustFinishedRefusal(); ok {
			chat.lgr.Debug(fmt.Sprintf("refusal finisheds %v", refusal))
			return
		}

		responseChan <- stream.Current().Choices[0].Delta.Content
	}

	if err := stream.Err(); err != nil {
		errorChan <- fmt.Errorf("starting stream: %w", errors.WithStack(err))
		return
	}
}
