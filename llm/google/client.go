package google

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/EPecherkin/catty-counting/config"
	"github.com/EPecherkin/catty-counting/llm/base"
	"github.com/EPecherkin/catty-counting/logger"
	"github.com/pkg/errors"
	"google.golang.org/genai"
)

const model = "gemini-1.5-flash"

type Client struct {
	gClient *genai.Client
	lgr     *slog.Logger
}

func CreateClient(ctx context.Context, lgr *slog.Logger) (base.Client, error) {
	lgr = lgr.With(logger.CALLER, "gemini client")
	lgr.Debug("Creating gemini client")
	apiKey := config.GeminiApiKey()

	gClient, err := genai.NewClient(ctx, &genai.ClientConfig{APIKey: apiKey, Backend: genai.BackendGeminiAPI})
	if err != nil {
		return nil, fmt.Errorf("creating gemini client: %w", errors.WithStack(err))
	}

	return &Client{gClient: gClient, lgr: lgr}, nil
}

func (client *Client) CreateChat(ctx context.Context, lgr *slog.Logger) (base.Chat, error) {
	lgr.Debug("Creating gemini chat")
	config := &genai.GenerateContentConfig{SystemInstruction: &genai.Content{Parts: []*genai.Part{{Text: systemPrompt}}}}
	gemini, err := client.gClient.Chats.Create(ctx, model, config, nil)
	if err != nil {
		return nil, fmt.Errorf("creating new chat with llm: %w", errors.WithStack(err))
	}
	return newChat(gemini, lgr), nil
}
