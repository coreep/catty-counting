package llm

import (
	"context"
	"fmt"
	"io"

	"github.com/EPecherkin/catty-counting/deps"
	"github.com/EPecherkin/catty-counting/logger"
	"github.com/google/generative-ai-go/genai"
	"github.com/pkg/errors"
)

const systemPrompt = `You are an accounting helping assistant, which is capable of processing docs, receipts, building statistics and giving advices.`

func GoTalk(ctx deps.Context, message string, responseChan chan<- string, errorChan chan<- error) {
	defer func() {
		close(responseChan)
		if err := recover(); err != nil {
			ctx.Deps().Logger().With(logger.ERROR, err).Error("failed talking")
		}
	}()

	cs := ctx.Deps().LLM().StartChat()
	cs.History = []*genai.Content{
		{
			Parts: []genai.Part{
				genai.Text(systemPrompt),
			},
			Role: "user",
		},
		{
			Parts: []genai.Part{
				genai.Text("Ok"),
			},
			Role: "model",
		},
	}

	iter := cs.SendMessageStream(context.Background(), genai.Text(message))

	for {
		select {
		case <-ctx.Done():
			return
		default:
			resp, err := iter.Next()
			if err == io.EOF {
				return
			}
			if err != nil {
				errorChan <- fmt.Errorf("iterating response: %w", errors.WithStack(err))
				return
			}

			if len(resp.Candidates) > 0 && len(resp.Candidates[0].Content.Parts) > 0 {
				if part, ok := resp.Candidates[0].Content.Parts[0].(genai.Text); ok {
					responseChan <- string(part)
				}
			}
		}
	}
}

// .toai done[
// configure a connection with gemini api. There is ENV variable GEMINI_API_KEY
// create a basic prompt which should be an instriuctions put at the beginning of each request, defining the purpose of the bot and what it should and shouldn't do
// create a function `HandleUserMessage`. It is supposed to answer to user's message. It should also receive a list of strings, which contains of previous messages from user and your answers, to improve your context knowledge. The response from llm should be "streamed" outside, so it could be displayed at the recepient
// ]
