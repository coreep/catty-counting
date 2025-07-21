package google

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
	lgr     *slog.Logger
	gClient *genai.Client
}

func CreateClient(ctx context.Context, lgr *slog.Logger) (*Client, error) {
	lgr.Debug("Creating new LLM Gemini Client")
	apiKey := config.GeminiApiKey()

	gClient, err := genai.NewClient(ctx, &genai.ClientConfig{APIKey: apiKey, Backend: genai.BackendGeminiAPI})
	if err != nil {
		return nil, fmt.Errorf("creating gemini client: %w", errors.WithStack(err))
	}

	return &Client{gClient: gClient, lgr: lgr}, nil
}

func (client *Client) CreateChat(ctx context.Context, lgr *slog.Logger) (*Chat, error) {
	config := &genai.GenerateContentConfig{SystemInstruction: &genai.Content{Parts: []*genai.Part{{Text: systemPrompt}}}}
	gemini, err := client.gClient.Chats.Create(ctx, model, config, nil)
	if err != nil {
		return nil, fmt.Errorf("creating new chat with llm: %w", errors.WithStack(err))
	}
	return newChat(gemini, lgr), nil
}
