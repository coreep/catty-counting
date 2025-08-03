package chatter

import (
	"context"

	"github.com/EPecherkin/catty-counting/db"
	"github.com/EPecherkin/catty-counting/deps"
	"github.com/EPecherkin/catty-counting/llm"
	"github.com/EPecherkin/catty-counting/logger"
	"github.com/EPecherkin/catty-counting/messenger"
)

type Chatter struct {
	messengerc messenger.Client
	llmc       llm.Client

	deps deps.Deps
}

func NewChatter(msgc messenger.Client, llmc llm.Client, deps deps.Deps) *Chatter {
	deps.Logger = deps.Logger.With(logger.CALLER, "Chatter")
	deps.Logger.Debug("Creating chatter")
	return &Chatter{messengerc: msgc, llmc: llmc, deps: deps}
}

func (chatter *Chatter) Run(ctx context.Context) {
	chatter.deps.Logger.Debug("Running chatter")
	chatter.messengerc.OnMessage(chatter.handleMessage)
	chatter.messengerc.Listen(ctx)
	chatter.deps.Logger.Info("Chatter finished")
}

func (chatter *Chatter) handleMessage(ctx context.Context, message db.Message, response chan<- string) {
	lgr := chatter.deps.Logger.With(logger.USER_ID, message.UserID, logger.MESSAGE_ID, message.ID)
	lgr.Debug("llc handling message")
	chatter.llmc.HandleMessage(ctx, message, response)
}
