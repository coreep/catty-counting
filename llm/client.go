package llm

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/EPecherkin/catty-counting/config"
	"github.com/pkg/errors"
	"google.golang.org/genai"
)

const model = "gemini-1.5-flash"

type Client struct {
	lgr       *slog.Logger
	llmClient *genai.Client
}

func CreateClient(ctx context.Context, lgr *slog.Logger) (*Client, error) {
	lgr.Debug("Creating new LLM Gemini Client")
	apiKey := config.GeminiApiKey()

	llmClient, err := genai.NewClient(ctx, &genai.ClientConfig{APIKey: apiKey, Backend: genai.BackendGeminiAPI})
	if err != nil {
		return nil, fmt.Errorf("creating gemini client: %w", errors.WithStack(err))
	}

	return &Client{llmClient: llmClient, lgr: lgr}, nil
}

func (client *Client) CreateChat(ctx context.Context, lgr *slog.Logger) (*Chat, error) {
	config := &genai.GenerateContentConfig{SystemInstruction: &genai.Content{Parts: []*genai.Part{{Text: systemPrompt}}}}
	llmChat, err := client.llmClient.Chats.Create(ctx, model, config, nil)
	if err != nil {
		return nil, fmt.Errorf("creating new chat with llm: %w", errors.WithStack(err))
	}
	return newChat(llmChat, lgr), nil
}
