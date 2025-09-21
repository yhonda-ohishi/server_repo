package logger

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
)

// This file contains examples of how to use the logger in your application
// These functions are for demonstration purposes

// ExampleInitializeFromConfig shows how to initialize the logger from application config
func ExampleInitializeFromConfig() error {
	// Example using the config package (adjust import as needed)
	// cfg, err := config.Load()
	// if err != nil {
	//     return err
	// }

	// Initialize logger from config
	loggerConfig := Config{
		Level:  "info", // cfg.Logging.Level
		Format: "json", // cfg.Logging.Format
	}

	return Initialize(loggerConfig)
}

// ExampleFiberAppSetup shows how to set up a Fiber app with logging middleware
func ExampleFiberAppSetup() *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: ErrorHandler,
	})

	// Add recovery middleware
	app.Use(RecoveryMiddleware())

	// Add request logging middleware
	app.Use(RequestLogger())

	// Add user context middleware (if you have authentication)
	app.Use(UserContextMiddleware())

	// Example route with logging
	app.Get("/api/users/:id", func(c *fiber.Ctx) error {
		ctx := c.UserContext()
		userID := c.Params("id")

		// Log business event
		LogBusinessEvent(ctx, "user_fetch_requested", map[string]interface{}{
			"user_id": userID,
		})

		// Simulate database operation
		start := time.Now()
		// ... database call ...
		LogDatabaseOperation(ctx, "SELECT", "users", time.Since(start), nil)

		// Log successful completion
		WithContext(ctx).WithField("user_id", userID).Info("User fetched successfully")

		return c.JSON(fiber.Map{
			"user_id": userID,
			"name":    "John Doe",
		})
	})

	return app
}

// ExampleErrorHandling shows various error handling patterns
func ExampleErrorHandling(ctx context.Context) {
	// Simple error logging
	err := someOperation()
	if err != nil {
		LogError(ctx, err, "Operation failed", nil)
		return
	}

	// Error with additional context
	err = anotherOperation()
	if err != nil {
		LogError(ctx, err, "Another operation failed", map[string]interface{}{
			"operation": "another_op",
			"retry":     3,
		})
		return
	}

	// Using logger methods directly
	logger := WithContext(ctx)
	err = complexOperation()
	if err != nil {
		logger.WithError(err).WithFields(map[string]interface{}{
			"component": "complex_system",
			"attempt":   1,
		}).Error("Complex operation failed")
		return
	}
}

// ExampleExternalAPILogging shows how to log external API calls
func ExampleExternalAPILogging(ctx context.Context) {
	start := time.Now()

	// Simulate external API call
	statusCode := 200
	var err error

	// Log the external API call
	LogExternalAPICall(ctx, "payment_service", "/api/v1/charge", statusCode, time.Since(start), err)
}

// ExampleStructuredLogging shows various structured logging patterns
func ExampleStructuredLogging(ctx context.Context) {
	// Basic structured logging
	WithContext(ctx).WithFields(map[string]interface{}{
		"user_id":    "12345",
		"operation":  "payment",
		"amount":     99.99,
		"currency":   "USD",
	}).Info("Payment processing started")

	// Building logger with multiple fields
	logger := WithContext(ctx).
		WithField("service", "payment").
		WithField("version", "v1.2.3")

	logger.Info("Service initialized")

	// Chaining with error
	err := processPayment()
	if err != nil {
		logger.WithError(err).Error("Payment processing failed")
	} else {
		logger.Info("Payment processing completed")
	}
}

// ExampleCustomMiddleware shows how to create custom logging middleware
func ExampleCustomMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		ctx := c.UserContext()

		// Pre-request logging
		WithContext(ctx).WithFields(map[string]interface{}{
			"method": c.Method(),
			"path":   c.Path(),
			"ip":     c.IP(),
		}).Debug("Request started")

		err := c.Next()

		// Post-request logging
		duration := time.Since(start)
		statusCode := c.Response().StatusCode()

		logEntry := WithContext(ctx).WithFields(map[string]interface{}{
			"method":      c.Method(),
			"path":        c.Path(),
			"status_code": statusCode,
			"duration_ms": duration.Milliseconds(),
			"response_size": len(c.Response().Body()),
		})

		if err != nil {
			logEntry.WithError(err).Error("Request completed with error")
		} else if statusCode >= 400 {
			logEntry.Warn("Request completed with error status")
		} else {
			logEntry.Info("Request completed successfully")
		}

		return err
	}
}

// Helper functions for examples (placeholders)
func someOperation() error {
	return nil
}

func anotherOperation() error {
	return nil
}

func complexOperation() error {
	return nil
}

func processPayment() error {
	return nil
}