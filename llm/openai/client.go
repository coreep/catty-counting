package openai

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/EPecherkin/catty-counting/config"
	"github.com/EPecherkin/catty-counting/llm"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/pkg/errors"
	"google.golang.org/genai"
)

type Client struct {
	lgr     *slog.Logger
	oClient *openai.Client
}

func CreateClient(ctx context.Context, lgr *slog.Logger) (*llm.Client, error) {
	lgr.Debug("Creating new LLM OpenAI Client")
	apiKey := config.OpenAiApiKey()
	oClient := openai.NewClient(option.WithAPIKey(apiKey))

	return &Client{oClient: &oClient, lgr: lgr}, nil
}

func (client *Client) CreateChat(ctx context.Context, lgr *slog.Logger) (*Chat, error) {
	config := &genai.GenerateContentConfig{SystemInstruction: &genai.Content{Parts: []*genai.Part{{Text: systemPrompt}}}}
	gemini, err := client.oClient.Chats.Create(ctx, model, config, nil)
	if err != nil {
		return nil, fmt.Errorf("creating new chat with llm: %w", errors.WithStack(err))
	}
	return newChat(gemini, lgr), nil
}
