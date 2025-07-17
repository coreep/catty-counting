package main

import (
	"context"
	"fmt"
	"os"

	"github.com/EPecherkin/catty-counting/config"
	"github.com/EPecherkin/catty-counting/deps"
	"github.com/EPecherkin/catty-counting/llm"
	"github.com/EPecherkin/catty-counting/logger"
	"github.com/EPecherkin/catty-counting/telegram"
)

func main() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("{\"error\": \"panic in main: %w\"\n", err)
		}
	}()

	ctx, llmClient, err := initialize()
	if err != nil {
		ctx.Deps().Logger().With(logger.ERROR, err).Error("Initialization failed")
		os.Exit(1)
	}

	mode := "telegram"
	if len(os.Args) > 1 {
		mode = os.Args[1]
	}
	ctx.Deps().Logger().Info("running in '%s' mode\n", mode)

	switch mode {
	case "telegram":
		telegram.NewBot(llmClient).Run(ctx)
	// case "api":
	// 	Api.Run()
	default:
		fmt.Println("unknown mode: " + mode)
	}
}

func initialize() (deps.Context, *llm.Client, error) {
	lgr := logger.NewLogger()
	ctx := context.Background()
	myCtx := deps.NewContext(
		ctx,
		deps.NewDeps(
			lgr,
			// .toai todo[add db.NewConnection()]
		),
	)

	if err := config.Init(); err != nil {
		return myCtx, nil, fmt.Errorf("initializing config: %w", err)
	}
	llmClient, err := llm.NewClient(myCtx)
	if err != nil {
		return myCtx, nil, fmt.Errorf("initializing llmClient: %w", err)
	}
	return myCtx, llmClient, nil
}
