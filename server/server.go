package server

import (
	"context"
	"fmt"

	"github.com/EPecherkin/catty-counting/deps"
	"github.com/EPecherkin/catty-counting/llm"
	"github.com/EPecherkin/catty-counting/logger"
	"github.com/EPecherkin/catty-counting/messenger"
	"github.com/EPecherkin/catty-counting/messenger/base"
)

type Server struct {
	msgc messenger.Client
	llmc llm.Client

	deps deps.Deps
}

func NewServer(msgc messenger.Client, llmc llm.Client, deps deps.Deps) *Server {
	deps.Logger = deps.Logger.With(logger.CALLER, "server")
	deps.Logger.Debug("Creating server")
	return &Server{
		msgc: msgc,
		llmc: llmc,
		deps: deps,
	}
}

func (server *Server) Run(ctx context.Context) {
	server.deps.Logger.Debug("Running server")

	go server.msgc.GoTalk(ctx)
	for {
		select {
		case messageRequest := <-server.msgc.Messages():
			go server.goHandleMessageRequest(ctx, messageRequest)
		case <-ctx.Done():
			server.deps.Logger.Info("Server message wait interrupted")
			return
		}
	}
}

func (server *Server) goHandleMessageRequest(ctx context.Context, messageRequest *base.MessageRequest) {
	server.deps.Logger.Debug(fmt.Sprintf("received message MessageRequest %v", messageRequest))
	messageRequest.Response <- "handled"
	close(messageRequest.Response)
}
