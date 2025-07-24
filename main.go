package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/EPecherkin/catty-counting/config"
	"github.com/EPecherkin/catty-counting/db"
	"github.com/EPecherkin/catty-counting/llm"
	"github.com/EPecherkin/catty-counting/logger"
	"github.com/EPecherkin/catty-counting/messenger"
	"github.com/EPecherkin/catty-counting/server"
	"github.com/pkg/errors"
	"gocloud.dev/blob"
	_ "gocloud.dev/blob/fileblob"
	"gorm.io/gorm"
)

func main() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("{\"error\": \"panic in main: %w\"\n", err)
		}
	}()

	ctx := context.Background()
	lgr, dbc, files, llmc, msgc, err := initialize(ctx)
	if err != nil {
		lgr.With(logger.ERROR, err).Error("Initialization failed")
		os.Exit(1)
	}

	mode := "server"
	if len(os.Args) > 1 {
		mode = os.Args[1]
	}
	lgr.Info("running in '" + mode + "' mode")

	switch mode {
	case "server":
		server.NewServer(msgc, llmc, server.NewServerDeps(lgr, dbc, files)).Run(ctx)
	default:
		fmt.Println("unknown mode: " + mode)
	}
}

func initialize(ctx context.Context) (lgr *slog.Logger, databaseConnection *gorm.DB, filesBucket *blob.Bucket, _ llm.Client, _ messenger.Client, _ error) {
	lgr = logger.NewLogger()

	if err := config.Init(); err != nil {
		return lgr, nil, nil, nil, nil, fmt.Errorf("initializing config: %w", err)
	}

	dbc, err := db.NewConnection()
	if err != nil {
		return lgr, nil, nil, nil, nil, fmt.Errorf("initializing database connection: %w", err)
	}

	files, err := blob.OpenBucket(ctx, config.FileBucket())
	if err != nil {
		return lgr, nil, nil, nil, nil, fmt.Errorf("initializing file blob: %w", errors.WithStack(err))
	}

	llmc, err := llm.CreateOpenaiClient(ctx, lgr)
	if err != nil {
		return lgr, nil, nil, nil, nil, fmt.Errorf("", err)
	}

	msgc, err := messenger.CreateTelegramClient(lgr, dbc, files)
	return lgr, dbc, files, llmc, msgc, nil
}
