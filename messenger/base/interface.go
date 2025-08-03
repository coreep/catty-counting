package base

import (
	"context"

	"github.com/EPecherkin/catty-counting/db"
)

type OnMessageCallback func(ctx context.Context, msg db.Message, response chan<- string)

type Client interface {
	Listen(ctx context.Context)
	OnMessage(callback OnMessageCallback)
}
