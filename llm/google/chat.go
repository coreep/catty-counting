package google

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/EPecherkin/catty-counting/logger"
	"github.com/pkg/errors"
	"google.golang.org/genai"
)

type Chat struct {
	gChat *genai.Chat
	lgr   *slog.Logger
}

func newChat(gChat *genai.Chat, lgr *slog.Logger) *Chat {
	lgr = lgr.With(logger.CALLER, "gemini chat")
	return &Chat{gChat: gChat, lgr: lgr}
}

const systemPrompt = `You are an accounting helping assistant, which is capable of processing docs, receipts, building statistics and giving advices.`

func (chat *Chat) GoTalk(ctx context.Context, message string, responseChan chan<- string, errorChan chan<- error) {
			chat.lgr.Debug("starting talking")
	defer func() {
		close(responseChan)
		if err := recover(); err != nil {
			chat.lgr.With(logger.ERROR, err).Error("failed talking")
		} else {
			chat.lgr.Debug("finished talking")
		}
	}()

	for resp, err := range chat.gChat.SendMessageStream(ctx, genai.Part{Text: message}) {
		if ctx.Err() != nil || err == io.EOF {
			break
		}
		if err != nil {
			errorChan <- fmt.Errorf("iterating response: %w", errors.WithStack(err))
			break
		}
		responseChan <- resp.Text()
	}
}
