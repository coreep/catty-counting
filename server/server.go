package server

import (
	"context"
	"log/slog"

	"github.com/EPecherkin/catty-counting/llm"
	"github.com/EPecherkin/catty-counting/logger"
	"github.com/EPecherkin/catty-counting/messenger"
	"github.com/EPecherkin/catty-counting/messenger/base"
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
	msgc messenger.Client
	llmc llm.Client
	deps *ServerDeps
}

func NewServer(msgc messenger.Client, llmc llm.Client, deps *ServerDeps) *Server {
	deps.lgr = deps.lgr.With(logger.CALLER, "server")
	deps.lgr.Debug("Creating server")
	return &Server{
		msgc: msgc,
		llmc: llmc,
		deps: deps,
	}
}

func (server *Server) Run(ctx context.Context) {
	server.deps.lgr.Debug("Running server")

	server.msgc.GoTalk(ctx)
	for {
		select {
		case messageRequest := <-server.msgc.Messages():
			go server.goHandleMessageRequest(ctx, messageRequest)
		case <-ctx.Done():
			server.deps.lgr.Info("Server message wait interrupted")
		}
	}
}

func (server *Server) goHandleMessageRequest(ctx context.Context, messageRequest *base.MessageRequest) {

}
