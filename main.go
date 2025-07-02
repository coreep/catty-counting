package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/EPecherkin/catty-counting/telegram"
	"github.com/joho/godotenv"
	// "github.com/EPecherkin/catty-counting/api"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	defer func() {
		if err := recover(); err != nil {
			logger.Error("Panic in main", slog.Any("error", err))
		}
	}()
	// initialize
	err := godotenv.Load()
	if err != nil {
		logger.Error("Error loading .env file")
		return
	}

	mode := "telegram"
	if len(os.Args) > 1 {
		mode = os.Args[1]
	}

	ctx := context.Background()

	switch mode {
	case "telegram":
		telegram.Run(ctx)
	// case "api":
	// 	Api.Run()
	default:
		logger.Error("Unknown mode: " + mode)
	}
}
