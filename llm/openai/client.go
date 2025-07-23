package openai

import (
	"context"
	"log/slog"

	"github.com/EPecherkin/catty-counting/config"
	"github.com/EPecherkin/catty-counting/llm/base"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

type Client struct {
	oClient openai.Client
	lgr     *slog.Logger
}

func CreateClient(ctx context.Context, lgr *slog.Logger) (base.Client, error) {
	lgr.Debug("Creating new LLM OpenAI Client")
	apiKey := config.OpenAiApiKey()
	oClient := openai.NewClient(option.WithAPIKey(apiKey))

	return &Client{oClient: oClient, lgr: lgr}, nil
}

func (client *Client) CreateChat(ctx context.Context, lgr *slog.Logger) (base.Chat, error) {
	return newChat(&client.oClient, lgr), nil
}
