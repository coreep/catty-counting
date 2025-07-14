package llm

import (
	"fmt"
	"io"

	"github.com/EPecherkin/catty-counting/deps"
	"github.com/EPecherkin/catty-counting/logger"
	"github.com/pkg/errors"
	"google.golang.org/genai"
)

const systemPrompt = `You are an accounting helping assistant, which is capable of processing docs, receipts, building statistics and giving advices.`

func GoTalk(ctx deps.Context, message string, responseChan chan<- string, errorChan chan<- error) {
	defer func() {
		close(responseChan)
		if err := recover(); err != nil {
			ctx.Deps().Logger().With(logger.ERROR, err).Error("failed talking")
		}
	}()

	config := &genai.GenerateContentConfig{SystemInstruction: &genai.Content{Parts: []*genai.Part{{Text: systemPrompt}}}}
	iter := ctx.Deps().LLM().Models.GenerateContentStream(ctx, "gemini-1.5-flash", genai.Text(message), config)

	for resp, err := range iter {
		if ctx.Err() != nil {
			break
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			errorChan <- fmt.Errorf("iterating response: %w", errors.WithStack(err))
			break
		}
		responseChan <- resp.Text()
	}
}

// .toai done[
// configure a connection with gemini api. There is ENV variable GEMINI_API_KEY
// create a basic prompt which should be an instriuctions put at the beginning of each request, defining the purpose of the bot and what it should and shouldn't do
// create a function `HandleUserMessage`. It is supposed to answer to user's message. It should also receive a list of strings, which contains of previous messages from user and your answers, to improve your context knowledge. The response from llm should be "streamed" outside, so it could be displayed at the recepient
// ]
