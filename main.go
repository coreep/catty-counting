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
			fmt.Printf("panic in main: %w\n", err)
		}
	}()

	mode := "telegram"
	if len(os.Args) > 1 {
		mode = os.Args[1]
	}
	fmt.Printf("running in '%s' mode\n", mode)

	ctx := context.Background()

	switch mode {
	case "telegram":
		telegram.RunBot(ctx)
	// case "api":
	// 	Api.Run()
	default:
		fmt.Println("unknown mode: " + mode)
	}
}
