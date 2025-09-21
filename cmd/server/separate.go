package main

import (
	"context"
	"fmt"

	"github.com/yhonda-ohishi/db-handler-server/internal/config"
	"github.com/yhonda-ohishi/db-handler-server/internal/gateway"
)

// RunSeparateMode runs the server in separate process mode with network connections
func RunSeparateMode(cfg *config.Config) error {
	fmt.Println("Starting server in separate mode")

	// Create and start the simple gateway
	gw := gateway.NewSimpleGateway(cfg)

	ctx := context.Background()
	if err := gw.Start(ctx); err != nil {
		return fmt.Errorf("failed to start gateway: %w", err)
	}

	fmt.Printf("Gateway started successfully on port %d (mode: separate)\n", cfg.Server.HTTPPort)
	return nil
}