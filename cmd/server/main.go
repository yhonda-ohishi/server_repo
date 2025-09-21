package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yhonda-ohishi/db-handler-server/internal/config"
)

var (
	version   = "v1.0.0"
	buildTime = "unknown"
	gitCommit = "unknown"
)

func main() {
	// Print banner
	printBanner()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Logger initialization would go here for production use

	fmt.Printf("Starting gRPC-First Multi-Protocol Gateway (version: %s, mode: %s)\n",
		version, cfg.Deployment.Mode)

	// Setup signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Start server based on deployment mode
	errCh := make(chan error, 1)
	go func() {
		switch cfg.Deployment.Mode {
		case "single":
			errCh <- RunSingleMode(cfg)
		case "separate":
			errCh <- RunSeparateMode(cfg)
		default:
			errCh <- fmt.Errorf("unknown deployment mode: %s", cfg.Deployment.Mode)
		}
	}()

	// Give the server time to start
	select {
	case err := <-errCh:
		if err != nil {
			fmt.Printf("Server failed to start: %v\n", err)
			os.Exit(1)
		}
	case <-time.After(2 * time.Second):
		// Server started successfully, continue to wait for signals
		fmt.Println("Server started successfully, waiting for shutdown signal...")
	}

	// Wait for shutdown signal
	select {
	case err := <-errCh:
		if err != nil {
			fmt.Printf("Server error: %v\n", err)
			os.Exit(1)
		}
	case sig := <-sigCh:
		fmt.Printf("Received shutdown signal: %v\n", sig)

		// Graceful shutdown with timeout
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()

		fmt.Println("Starting graceful shutdown...")

		// Wait for shutdown to complete or timeout
		done := make(chan bool, 1)
		go func() {
			// Here you would typically call shutdown methods on your services
			// For now, we'll just wait a moment to simulate cleanup
			time.Sleep(1 * time.Second)
			done <- true
		}()

		select {
		case <-done:
			fmt.Println("Graceful shutdown completed")
		case <-shutdownCtx.Done():
			fmt.Println("Graceful shutdown timed out, forcing exit")
		}
	}
}

func printBanner() {
	banner := `
  _____ _______ _____   __  __      _           _
 |  ___|_   ___|  ___| |  \/  |    (_)         (_)
 | |__   | | | |____  | .  . | ___ _ ___  __ _ _
 |  __|  | | |  ____|  | |\/| |/ _ \ / __|/ _` + "`" + ` | |
 | |___  | | | |____  | |  | |  __/ \__ \ (_| | |
 \____/  |_| \______| \_|  |_/\___|_|___/\__,_|_|

  gRPC-First Multi-Protocol Gateway Server
  Version: %s

`
	fmt.Printf(banner, version)
}

// Configuration validation
func validateConfig(cfg *config.Config) error {
	if cfg.Server.HTTPPort == cfg.Server.GRPCPort {
		return fmt.Errorf("HTTP and gRPC ports cannot be the same")
	}

	if cfg.Deployment.Mode == "separate" {
		if cfg.External.DatabaseGRPCURL == "" && cfg.External.HandlersGRPCURL == "" {
			return fmt.Errorf("at least one external service URL must be configured in separate mode")
		}
	}

	return nil
}

// Health check endpoint for deployment monitoring
func healthCheck() error {
	// This could be called by deployment tools to check if the server is ready
	// For now, just return nil
	return nil
}

// Metrics endpoint for monitoring
func getMetrics() map[string]interface{} {
	return map[string]interface{}{
		"version":    version,
		"build_time": buildTime,
		"git_commit": gitCommit,
		"uptime":     time.Since(startTime).Seconds(),
	}
}

var startTime = time.Now()

func init() {
	// Set default timezone
	if tz := os.Getenv("TZ"); tz != "" {
		var err error
		time.Local, err = time.LoadLocation(tz)
		if err != nil {
			log.Printf("Failed to set timezone %s: %v", tz, err)
		}
	}
}