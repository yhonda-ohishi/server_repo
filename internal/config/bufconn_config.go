package config

import (
	"os"
	"strconv"
)

const (
	// DefaultBufconnSize is the default buffer size for bufconn (1MB)
	DefaultBufconnSize = 1024 * 1024
)

// BufconnConfig holds configuration for bufconn in-memory gRPC connections
type BufconnConfig struct {
	// BufferSize is the size of the in-memory buffer in bytes
	BufferSize int
	// Enabled indicates if bufconn should be used (single mode)
	Enabled bool
}

// NewBufconnConfig creates a new bufconn configuration
func NewBufconnConfig() *BufconnConfig {
	config := &BufconnConfig{
		BufferSize: DefaultBufconnSize,
		Enabled:    false,
	}

	// Check environment variable for buffer size
	if sizeStr := os.Getenv("BUFCONN_SIZE"); sizeStr != "" {
		if size, err := strconv.Atoi(sizeStr); err == nil && size > 0 {
			config.BufferSize = size
		}
	}

	// Enable bufconn in single mode
	if mode := os.Getenv("DEPLOYMENT_MODE"); mode == "single" {
		config.Enabled = true
	}

	return config
}

// GetBufferSize returns the configured buffer size
func (c *BufconnConfig) GetBufferSize() int {
	if c.BufferSize <= 0 {
		return DefaultBufconnSize
	}
	return c.BufferSize
}

// IsEnabled returns true if bufconn should be used
func (c *BufconnConfig) IsEnabled() bool {
	return c.Enabled
}