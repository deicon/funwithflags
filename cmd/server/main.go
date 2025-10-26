package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/deicon/funwithflags/internal/app"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	serverApp, err := app.New()
	if err != nil {
		log.Fatalf("init app: %v", err)
	}

	if err := serverApp.Run(ctx); err != nil {
		log.Fatalf("server stopped: %v", err)
	}

	os.Exit(0)
}
