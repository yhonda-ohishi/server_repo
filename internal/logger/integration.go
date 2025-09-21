package logger

import (
	"fmt"

	"github.com/yhonda-ohishi/db-handler-server/internal/config"
)

// InitializeFromAppConfig initializes the logger using the application's config
func InitializeFromAppConfig(cfg *config.Config) error {
	if cfg == nil {
		return fmt.Errorf("config cannot be nil")
	}

	loggerConfig := Config{
		Level:  cfg.Logging.Level,
		Format: cfg.Logging.Format,
		Output: nil, // Use default (stdout)
	}

	// Validate configuration
	if err := ValidateConfig(loggerConfig); err != nil {
		return fmt.Errorf("invalid logger configuration: %w", err)
	}

	// Initialize logger
	if err := Initialize(loggerConfig); err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}

	// Log successful initialization
	Info("Logger initialized successfully")
	Infof("Logger configuration: level=%s, format=%s", cfg.Logging.Level, cfg.Logging.Format)

	return nil
}

// MustInitializeFromAppConfig initializes the logger and panics on error
func MustInitializeFromAppConfig(cfg *config.Config) {
	if err := InitializeFromAppConfig(cfg); err != nil {
		panic(fmt.Sprintf("Failed to initialize logger: %v", err))
	}
}