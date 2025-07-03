package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/EPecherkin/catty-counting/telegram"
	"github.com/joho/godotenv"
	// "github.com/EPecherkin/catty-counting/api"
)

func main() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("Panic in main: %w\n", err)
		}
	}()

	mode := "telegram"
	if len(os.Args) > 1 {
		mode = os.Args[1]
	}

	switch mode {
	case "telegram":
		telegram.RunBot()
	// case "api":
	// 	Api.Run()
	default:
		fmt.Println("Unknown mode: " + mode)
	}
}
