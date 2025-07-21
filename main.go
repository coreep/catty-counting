package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/EPecherkin/catty-counting/config"
	"github.com/EPecherkin/catty-counting/llm"
	"github.com/EPecherkin/catty-counting/llm/google"
	"github.com/EPecherkin/catty-counting/logger"
	"github.com/EPecherkin/catty-counting/telegram"
	"github.com/pkg/errors"
	"gocloud.dev/blob"
)

func main() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("{\"error\": \"panic in main: %w\"\n", err)
		}
	}()

	ctx := context.Background()
	lgr, fileBucket, llmClient, err := initialize(ctx)
	if err != nil {
		lgr.With(logger.ERROR, err).Error("Initialization failed")
		os.Exit(1)
	}

	mode := "telegram"
	if len(os.Args) > 1 {
		mode = os.Args[1]
	}
	lgr.Info("running in '" + mode + "' mode")

	switch mode {
	case "telegram":
		telegram.NewBot(telegram.NewBotDeps(lgr, fileBucket, llmClient)).Run(ctx)
	// case "api":
	// 	Api.Run()
	default:
		fmt.Println("unknown mode: " + mode)
	}
}

func initialize(ctx context.Context) (*slog.Logger, *blob.Bucket, *llm.Client, error) {
	lgr := logger.NewLogger()

	if err := config.Init(); err != nil {
		return lgr, nil, nil, fmt.Errorf("initializing config: %w", err)
	}

	fileBucket, err := blob.OpenBucket(ctx, config.FileBucket())
	if err != nil {
		return lgr, nil, nil, fmt.Errorf("initializing file blob: %w", errors.WithStack(err))
	}

	llmClient, err := google.CreateClient(ctx, lgr)
	if err != nil {
		return lgr, nil, nil, fmt.Errorf("", err)
	}
	return lgr, fileBucket, llmClient, nil
}
