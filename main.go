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
	"github.com/joho/godotenv"
	"github.com/pkg/errors"
)

func main() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("{\"error\": \"panic in main: %w\"\n", err)
		}
	}()

	ctx := initialize()

	mode := "telegram"
	if len(os.Args) > 1 {
		mode = os.Args[1]
	}
	ctx.Deps().Logger().Info("running in '%s' mode\n", mode)

	switch mode {
	case "telegram":
		telegram.RunBot(ctx)
	// case "api":
	// 	Api.Run()
	default:
		fmt.Println("unknown mode: " + mode)
	}
}

func initialize() deps.Context {
	lgr := logger.NewLogger()
	if err := godotenv.Load(); err != nil {
		lgr.With(logger.ERROR, errors.WithStack(err)).Error("can't load .env")
		os.Exit(1)
	}
	if err := config.Init(); err != nil {
		lgr.With(logger.ERROR, err).Error("failed to config")
	}
	ctx := context.Background()
	llm, err := llm.NewClient(ctx, lgr)
	if err != nil {
		lgr.With(logger.ERROR, err).Error("failed to create llm client")
		os.Exit(1)
	}
	return deps.NewContext(
		ctx,
		deps.NewDeps(
			lgr,
			llm,
			// .toai todo[add db.NewConnection()]
		),
	)
}
