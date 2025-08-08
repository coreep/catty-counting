package openai

import (
	"context"
	"sync"

	"github.com/EPecherkin/catty-counting/config"
	"github.com/EPecherkin/catty-counting/db"
	"github.com/EPecherkin/catty-counting/deps"
	"github.com/EPecherkin/catty-counting/llm/base"
	"github.com/EPecherkin/catty-counting/logger"
	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"
)

type Client struct {
	oClient *openai.Client
	deps    deps.Deps

	chatPerUser map[uint]*Chat
	mu          sync.Mutex
}

func CreateClient(deps deps.Deps) (base.Client, error) {
	deps.Logger = deps.Logger.With(logger.CALLER, "openai client")
	deps.Logger.Debug("Creating openai client")
	apiKey := config.OpenAiApiKey()
	oClient := openai.NewClient(option.WithAPIKey(apiKey))

	return &Client{oClient: &oClient, deps: deps, chatPerUser: make(map[uint]*Chat)}, nil
}

func (client *Client) HandleMessage(ctx context.Context, message db.Message, response chan<- string) {
	client.mu.Lock()
	chat, ok := client.chatPerUser[message.UserID]
	if !ok {
		chat = newChat(client.oClient, client.deps)
		client.chatPerUser[message.UserID] = chat
	}
	client.mu.Unlock()

	chat.Talk(ctx, message, response)
}
