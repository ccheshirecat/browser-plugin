package main

import (
	"context"
	"log"
	"os"

	"github.com/volant-plugins/browser/internal/runtime/app"
)

func main() {
	ctx := context.Background()
	if err := app.Run(ctx); err != nil {
		log.Printf("exit due to error: %v", err)
		os.Exit(1)
	}
}
