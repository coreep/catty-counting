package llm

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/EPecherkin/catty-counting/config"
	"github.com/pkg/errors"
	"google.golang.org/genai"
)

func NewClient(ctx context.Context, logger *slog.Logger) (*genai.Client, error) {
	apiKey := config.GeminiApiKey()

	client, err := genai.NewClient(ctx, &genai.ClientConfig{APIKey: apiKey, Backend: genai.BackendGeminiAPI})
	if err != nil {
		return nil, fmt.Errorf("creating gemini client: %w", errors.WithStack(err))
	}

	return client, nil
}
