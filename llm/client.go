package llm

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/google/generative-ai-go/genai"
	"github.com/pkg/errors"
	"google.golang.org/api/option"
)

func NewClient(ctx context.Context, logger *slog.Logger) (*genai.GenerativeModel, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, errors.New("GEMINI_API_KEY is not set")
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("creating gemini client: %w", errors.WithStack(err))
	}

	model := client.GenerativeModel("gemini-1.5-flash")
	return model, nil
}
