package llm

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/pkg/errors"
	"google.golang.org/genai"
)

func NewClient(ctx context.Context, logger *slog.Logger) (*genai.Client, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, errors.New("GEMINI_API_KEY is not set")
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{APIKey: apiKey, Backend: genai.BackendGeminiAPI})
	if err != nil {
		return nil, fmt.Errorf("creating gemini client: %w", errors.WithStack(err))
	}

	return client, nil
}
