package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ConfigFromAppConfig creates a logger config from the application config
func ConfigFromAppConfig(level, format string) Config {
	return Config{
		Level:  level,
		Format: format,
		Output: nil, // Use default (stdout)
	}
}

// ConfigWithFile creates a logger config that writes to a file
func ConfigWithFile(level, format, filename string) (Config, error) {
	// Ensure directory exists
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return Config{}, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open file for writing
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return Config{}, fmt.Errorf("failed to open log file: %w", err)
	}

	return Config{
		Level:  level,
		Format: format,
		Output: file,
	}, nil
}

// ConfigWithFileAndConsole creates a logger config that writes to both file and console
func ConfigWithFileAndConsole(level, format, filename string) (Config, error) {
	// Ensure directory exists
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return Config{}, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open file for writing
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return Config{}, fmt.Errorf("failed to open log file: %w", err)
	}

	// Create multi-writer for both file and console
	multiWriter := io.MultiWriter(os.Stdout, file)

	return Config{
		Level:  level,
		Format: format,
		Output: multiWriter,
	}, nil
}

// ValidateConfig validates logger configuration
func ValidateConfig(config Config) error {
	// Validate log level
	_, err := parseLogLevel(config.Level)
	if err != nil {
		return fmt.Errorf("invalid log level: %w", err)
	}

	// Validate format
	format := strings.ToLower(config.Format)
	if format != "json" && format != "console" {
		return fmt.Errorf("invalid log format: %s, must be 'json' or 'console'", config.Format)
	}

	return nil
}

// Environment-based configuration helpers

// GetConfigFromEnv creates logger config from environment variables
func GetConfigFromEnv() Config {
	level := os.Getenv("LOG_LEVEL")
	if level == "" {
		level = "info"
	}

	format := os.Getenv("LOG_FORMAT")
	if format == "" {
		format = "json"
	}

	return Config{
		Level:  level,
		Format: format,
		Output: nil,
	}
}

// IsProductionEnv checks if we're running in production
func IsProductionEnv() bool {
	env := strings.ToLower(os.Getenv("ENVIRONMENT"))
	return env == "production" || env == "prod"
}

// IsDevelopmentEnv checks if we're running in development
func IsDevelopmentEnv() bool {
	env := strings.ToLower(os.Getenv("ENVIRONMENT"))
	return env == "development" || env == "dev" || env == ""
}

// GetRecommendedConfig returns recommended configuration based on environment
func GetRecommendedConfig() Config {
	if IsProductionEnv() {
		// Production: JSON format, info level
		return Config{
			Level:  "info",
			Format: "json",
			Output: nil,
		}
	} else {
		// Development: Console format, debug level
		return Config{
			Level:  "debug",
			Format: "console",
			Output: nil,
		}
	}
}