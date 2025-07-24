package llm

import (
	"github.com/EPecherkin/catty-counting/llm/base"
	"github.com/EPecherkin/catty-counting/llm/openai"
)

type Client interface {
	base.Client
}

type Chat interface {
	base.Chat
}

var CreateOpenaiClient = openai.CreateClient
