package base

import (
	"context"

	"github.com/EPecherkin/catty-counting/db"
)

// Represents every message received from messenger.
// Creates `response` channel with the ability to write response sequentially. Send NEW chunks, not the entire text.
// `response` should be canceled after the processing, signaling the end of response.
type MessageRequest struct {
	user     db.User
	message  db.Message
	response chan<- string
}

type Client interface {
	GoTalk(ctx context.Context)
	Messages() <-chan *MessageRequest
}
