package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/zono819/hyperliquid-bot/internal/infrastructure/config"
)

var (
	version   = "dev"
	buildTime = "unknown"
)

func main() {
	// Parse flags
	configPath := flag.String("config", "config/config.yaml", "path to config file")
	showVersion := flag.Bool("version", false, "show version")
	flag.Parse()

	if *showVersion {
		fmt.Printf("hyperliquid-bot %s (built: %s)\n", version, buildTime)
		os.Exit(0)
	}

	// Load config
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		fmt.Printf("\nReceived signal: %v, shutting down...\n", sig)
		cancel()
	}()

	// Run bot
	if err := run(ctx, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "bot error: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, cfg *config.Config) error {
	fmt.Printf("Starting %s in %s mode\n", cfg.App.Name, cfg.App.Environment)
	fmt.Printf("Strategy: %s, Symbol: %s\n", cfg.Strategy.Name, cfg.Strategy.Symbol)

	// TODO: Initialize exchange gateway
	// TODO: Initialize strategy
	// TODO: Create and start bot use case

	// Wait for context cancellation
	<-ctx.Done()

	fmt.Println("Bot stopped")
	return nil
}
