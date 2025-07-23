package server

import (
	"context"
	"log/slog"
	"os"

	"github.com/EPecherkin/catty-counting/llm"
	"github.com/EPecherkin/catty-counting/logger"
	"github.com/EPecherkin/catty-counting/messenger"
	"gocloud.dev/blob"
	"gorm.io/gorm"
)

type ServerDeps struct {
	lgr   *slog.Logger
	dbc   *gorm.DB
	files *blob.Bucket
}

func NewServerDeps(lgr *slog.Logger, dbc *gorm.DB, files *blob.Bucket) *ServerDeps {
	return &ServerDeps{lgr: lgr, dbc: dbc, files: files}
}

type Server struct {
	llmc llm.Client
	msgc messenger.Client
	deps *ServerDeps
}

func NewServer(llmc llm.Client, msgc messenger.Client, deps *ServerDeps) *Server {
	return &Server{
		deps: deps,
	}
}

func (server *Server) Run(ctx context.Context) {
	server.deps.lgr.Debug("Running server")
	err := bot.setup()
	if err != nil {
		bot.deps.lgr.With(logger.ERROR, err).Error("Failed to Bot")
		os.Exit(1)
	}

	bot.handleUpdates(ctx)

	telegram.NewBot(telegram.NewBotDeps(lgr, dbc))
}
