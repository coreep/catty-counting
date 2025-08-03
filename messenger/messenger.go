package messenger

import (
	"github.com/EPecherkin/catty-counting/messenger/base"
	"github.com/EPecherkin/catty-counting/messenger/telegram"
)

type Client interface {
	base.Client
}

var CreateTelegramClient = telegram.CreateClient
