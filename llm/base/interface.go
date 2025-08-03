package base

import (
	"context"

	"github.com/EPecherkin/catty-counting/db"
)

type Client interface {
	HandleMessage(ctx context.Context, message db.Message, response chan<- string)
}
