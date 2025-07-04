package main

import (
	"context"
	"fmt"
	"os"

	"github.com/EPecherkin/catty-counting/telegram"
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

	ctx := context.Background()

	switch mode {
	case "telegram":
		telegram.RunBot(ctx)
	// case "api":
	// 	Api.Run()
	default:
		fmt.Println("Unknown mode: " + mode)
	}
}
