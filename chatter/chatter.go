package chatter

import (
	"context"
	"fmt"

	"github.com/EPecherkin/catty-counting/deps"
	"github.com/EPecherkin/catty-counting/llm"
	"github.com/EPecherkin/catty-counting/logger"
	"github.com/EPecherkin/catty-counting/messenger"
	"github.com/EPecherkin/catty-counting/messenger/base"
)

type Chatter struct {
	chatter messenger.Client
	llmc llm.Client

	deps deps.Deps
}

func NewChatter(msgc messenger.Client, llmc llm.Client, deps deps.Deps) *Chatter {
	deps.Logger = deps.Logger.With(logger.CALLER, "chatter")
	deps.Logger.Debug("Creating chatter")
	return &Chatter{
		chatter: msgc,
		llmc: llmc,
		deps: deps,
	}
}

func (chatter *Chatter) Run(ctx context.Context) {
	chatter.deps.Logger.Debug("Running chatter")

	go chatter.chatter.GoTalk(ctx)
	for {
		select {
		case messageRequest := <-chatter.chatter.Messages():
			go chatter.goHandleMessageRequest(ctx, messageRequest)
		case <-ctx.Done():
			chatter.deps.Logger.Info("Chatter message wait interrupted")
			return
		}
	}
}

func (chatter *Chatter) goHandleMessageRequest(ctx context.Context, messageRequest *base.MessageRequest) {
	chatter.deps.Logger.Debug(fmt.Sprintf("received message MessageRequest files %d", len(messageRequest.Message.Files)))
	chatter.deps.Logger.Debug(fmt.Sprintf("received message MessageRequest %v", messageRequest.Message))
	messageRequest.Response <- "handled"
	close(messageRequest.Response)
}
