package llm

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/EPecherkin/catty-counting/config"
	"github.com/EPecherkin/catty-counting/deps"
	"github.com/pkg/errors"
	"google.golang.org/genai"
)

const model ="gemini-1.5-flash"

func NewClient(ctx context.Context, logger *slog.Logger) (*genai.Client, error) {
	apiKey := config.GeminiApiKey()

	client, err := genai.NewClient(ctx, &genai.ClientConfig{APIKey: apiKey, Backend: genai.BackendGeminiAPI})
	if err != nil {
		return nil, fmt.Errorf("creating gemini client: %w", errors.WithStack(err))
	}

	return client, nil
}

func NewChat(ctx deps.Context) (*genai.Chat, error) {
	config := &genai.GenerateContentConfig{SystemInstruction: &genai.Content{Parts: []*genai.Part{{Text: systemPrompt}}}}
	chat, err := ctx.Deps().LLM().Chats.Create(ctx, model, config, nil)
	if err != nil {
		return nil, fmt.Errorf("creating new chat with llm: %w", errors.WithStack(err))
	}
	return chat, nil
}
