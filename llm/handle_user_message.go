package llm

import (
	"context"
	"fmt"
	"io"

	"github.com/EPecherkin/catty-counting/deps"
	"github.com/google/generative-ai-go/genai"
)

const systemPrompt = `You are an accounting helping assistant, which is capable of processing docs, receipts, building statistics and giving advices.`

func HandleUserMessage(ctx deps.Context, message string, responseChannel chan<- string, errorChannel chan<- error) {
	defer close(responseChannel)

	client, err := newClient(context.Background())
	if err != nil {
		errorChannel<- fmt.Errorf("getting llm client: %w", err)
		return
	}

	cs := client.StartChat()
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

	for {
		resp, err := iter.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			// handle error
			return
		}

		if len(resp.Candidates) > 0 && len(resp.Candidates[0].Content.Parts) > 0 {
			if part, ok := resp.Candidates[0].Content.Parts[0].(genai.Text); ok {
				responseChannel <- string(part)
			}
		}
	}
}

// .toai done[
// configure a connection with gemini api. There is ENV variable GEMINI_API_KEY
// create a basic prompt which should be an instriuctions put at the beginning of each request, defining the purpose of the bot and what it should and shouldn't do
// create a function `HandleUserMessage`. It is supposed to answer to user's message. It should also receive a list of strings, which contains of previous messages from user and your answers, to improve your context knowledge. The response from llm should be "streamed" outside, so it could be displayed at the recepient
// ]
