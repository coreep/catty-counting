package llm

import (
	"github.com/EPecherkin/catty-counting/llm/base"
	"github.com/EPecherkin/catty-counting/llm/openai"
)

type Client interface {
	base.Client
}

var CreateOpenaiClient = openai.CreateClient
