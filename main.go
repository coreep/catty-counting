package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/EPecherkin/catty-counting/api"
	"github.com/EPecherkin/catty-counting/chatter"
	"github.com/EPecherkin/catty-counting/config"
	"github.com/EPecherkin/catty-counting/db"
	"github.com/EPecherkin/catty-counting/deps"
	"github.com/EPecherkin/catty-counting/llm"
	"github.com/EPecherkin/catty-counting/logger"
	"github.com/EPecherkin/catty-counting/messenger"
	"github.com/pkg/errors"
	"gocloud.dev/blob"
	_ "gocloud.dev/blob/fileblob"
	"gorm.io/gorm"
)

func main() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("{\"error\": \"panic in main: %v\"}\n", err)
		}
	}()

	ctx := context.Background()
	lgr, dbc, files, llmc, msgc, err := initialize(ctx)
	if err != nil {
		lgr.With(logger.ERROR, err).Error("Initialization failed")
		os.Exit(1)
	}

	var wg sync.WaitGroup
	d := deps.Deps{Logger: lgr, DBC: dbc, Files: files}
	go func() {
		wg.Add(1)
		defer wg.Done()
		chatter.NewChatter(msgc, llmc, d).Run(ctx)
	}()
	go func() {
		wg.Add(1)
		defer wg.Done()
		api.NewApi(d).Run(ctx)
	}()

	wg.Wait()
}

func initialize(ctx context.Context) (lgr *slog.Logger, databaseConnection *gorm.DB, filesBucket *blob.Bucket, _ llm.Client, _ messenger.Client, _ error) {
	if err := config.Init(); err != nil {
		return logger.NewLogger(), nil, nil, nil, nil, fmt.Errorf("initializing config: %w", err)
	}

	lgr = logger.NewLogger()

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
		return lgr, nil, nil, nil, nil, fmt.Errorf("initializing llm client: %w", err)
	}

	msgc, err := messenger.CreateTelegramClient(deps.Deps{Logger: lgr, DBC: dbc, Files: files})
	if err != nil {
		return lgr, nil, nil, nil, nil, fmt.Errorf("initializing messenger client: %w", err)
	}
	return lgr, dbc, files, llmc, msgc, nil
}
