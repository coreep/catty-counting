package llm

import (
	"context"
	"fmt"

	"github.com/EPecherkin/catty-counting/config"
	"github.com/pkg/errors"
	"google.golang.org/genai"
)

const model = "gemini-1.5-flash"

type Client struct {
	llmClient *genai.Client
}

func CreateClient(ctx deps.Context) (*Client, error) {
	ctx.Deps().Logger().Debug("Creating new LLM Gemini Client")
	apiKey := config.GeminiApiKey()

	llmClient, err := genai.NewClient(ctx, &genai.ClientConfig{APIKey: apiKey, Backend: genai.BackendGeminiAPI})
	if err != nil {
		return nil, fmt.Errorf("creating gemini client: %w", errors.WithStack(err))
	}

	return &Client{llmClient: llmClient}, nil
}

func (client *Client) CreateChat(ctx deps.Context) (*Chat, error) {
	config := &genai.GenerateContentConfig{SystemInstruction: &genai.Content{Parts: []*genai.Part{{Text: systemPrompt}}}}
	llmChat, err := client.llmClient.Chats.Create(ctx, model, config, nil)
	if err != nil {
		return nil, fmt.Errorf("creating new chat with llm: %w", errors.WithStack(err))
	}
	return newChat(llmChat), nil
}
