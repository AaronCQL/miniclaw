package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"goclaw/internal"
)

func main() {
	homeDir := internal.HomeDir()
	agentDir := internal.AgentDir()
	cfg := internal.LoadConfig(homeDir, agentDir)
	app := internal.NewApp(cfg)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := app.Start(ctx); err != nil {
		log.Fatal(err)
	}
}
