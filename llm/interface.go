package llm

import (
	"context"
	"log/slog"
)

type Client interface {
	CreateChat(ctx context.Context, lgr *slog.Logger) (*Chat, error)
}

type Chat interface {
	GoTalk(ctx context.Context, message string, responseChan chan<- string, errorChan chan<- error)
}
