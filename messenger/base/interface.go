package base

import (
	"context"

	"github.com/EPecherkin/catty-counting/db"
)

// Represents every message received from messenger.
// Creates `response` channel with the ability to write response sequentially. Send NEW chunks, not the entire text.
// `response` should be canceled after the processing, signaling the end of response.
type MessageRequest struct {
	message  db.Message
	response chan<- string
}

func NewMessageRequest(message db.Message, response chan <- string) *MessageRequest {
	return &MessageRequest{message: message, response: response}
}

type Client interface {
	GoTalk(ctx context.Context)
	Messages() <-chan *MessageRequest
}
