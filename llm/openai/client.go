package openai

import (
	"context"
	"log/slog"

	"github.com/EPecherkin/catty-counting/config"
	"github.com/EPecherkin/catty-counting/llm/base"
	"github.com/EPecherkin/catty-counting/logger"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

type Client struct {
	oClient openai.Client
	lgr     *slog.Logger
}

func CreateClient(ctx context.Context, lgr *slog.Logger) (base.Client, error) {
	lgr = lgr.With(logger.CALLER, "openai client")
	lgr.Debug("Creating openai client")
	apiKey := config.OpenAiApiKey()
	oClient := openai.NewClient(option.WithAPIKey(apiKey))

	return &Client{oClient: oClient, lgr: lgr}, nil
}

func (client *Client) CreateChat(ctx context.Context, lgr *slog.Logger) (base.Chat, error) {
	return newChat(&client.oClient, lgr), nil
}
