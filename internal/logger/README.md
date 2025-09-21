# Logger Package

A comprehensive logging implementation using zerolog for structured logging with context support, request ID tracking, and Fiber middleware integration.

## Features

- **Structured Logging**: Uses zerolog for high-performance structured logging
- **Multiple Output Formats**: Supports both JSON and console output formats
- **Configurable Log Levels**: Debug, Info, Warn, Error, Fatal, and Panic levels
- **Context-Based Logging**: Support for request IDs and user IDs via context
- **Fiber Middleware**: Built-in middleware for HTTP request logging
- **Specialized Logging Methods**: Helper methods for common logging scenarios
- **Global Logger Instance**: Easy-to-use global logger with convenience functions

## Quick Start

### 1. Initialize the Logger

```go
import "github.com/yhonda-ohishi/db-handler-server/internal/logger"

// Initialize with basic configuration
config := logger.Config{
    Level:  "info",
    Format: "json",
}

err := logger.Initialize(config)
if err != nil {
    log.Fatal("Failed to initialize logger:", err)
}
```

### 2. Integration with Application Config

```go
import (
    "github.com/yhonda-ohishi/db-handler-server/internal/config"
    "github.com/yhonda-ohishi/db-handler-server/internal/logger"
)

// Load application config
cfg, err := config.Load()
if err != nil {
    log.Fatal("Failed to load config:", err)
}

// Initialize logger from config
loggerConfig := logger.ConfigFromAppConfig(cfg.Logging.Level, cfg.Logging.Format)
err = logger.Initialize(loggerConfig)
if err != nil {
    log.Fatal("Failed to initialize logger:", err)
}
```

### 3. Fiber App Setup

```go
import (
    "github.com/gofiber/fiber/v2"
    "github.com/yhonda-ohishi/db-handler-server/internal/logger"
)

app := fiber.New(fiber.Config{
    ErrorHandler: logger.ErrorHandler,
})

// Add middleware
app.Use(logger.RecoveryMiddleware())
app.Use(logger.RequestLogger())
app.Use(logger.UserContextMiddleware())
```

## Usage Examples

### Basic Logging

```go
import "github.com/yhonda-ohishi/db-handler-server/internal/logger"

// Simple logging
logger.Info("Application started")
logger.Error("Something went wrong")
logger.Debug("Debug information")

// Formatted logging
logger.Infof("User %s logged in", userID)
logger.Errorf("Failed to process order %d: %v", orderID, err)
```

### Context-Based Logging

```go
// Add request ID to context
ctx := logger.ContextWithRequestID(context.Background(), "req-123")

// Log with context
logger.WithContext(ctx).Info("Processing request")

// Add user ID
ctx = logger.ContextWithUserID(ctx, "user-456")
logger.WithContext(ctx).Info("User action performed")
```

### Structured Logging

```go
// Single field
logger.WithField("user_id", "12345").Info("User action")

// Multiple fields
logger.WithFields(map[string]interface{}{
    "user_id":   "12345",
    "action":    "purchase",
    "amount":    99.99,
    "currency":  "USD",
}).Info("Purchase completed")

// With error
logger.WithError(err).Error("Operation failed")
```

### Specialized Logging Methods

```go
// HTTP request logging
logger.LogRequest(ctx, "GET", "/api/users", 200, 150*time.Millisecond)

// Error logging with context
logger.LogError(ctx, err, "Database operation failed", map[string]interface{}{
    "table": "users",
    "operation": "INSERT",
})

// Business event logging
logger.LogBusinessEvent(ctx, "user_registration", map[string]interface{}{
    "user_id": "12345",
    "source": "web",
})

// Database operation logging
logger.LogDatabaseOperation(ctx, "SELECT", "users", 50*time.Millisecond, nil)

// External API call logging
logger.LogExternalAPICall(ctx, "payment_service", "/charge", 200, 200*time.Millisecond, nil)
```

### Fiber Route Example

```go
app.Get("/api/users/:id", func(c *fiber.Ctx) error {
    ctx := c.UserContext()
    userID := c.Params("id")

    // Log business event
    logger.LogBusinessEvent(ctx, "user_fetch_requested", map[string]interface{}{
        "user_id": userID,
    })

    // Simulate database operation
    start := time.Now()
    user, err := getUserFromDB(userID)
    logger.LogDatabaseOperation(ctx, "SELECT", "users", time.Since(start), err)

    if err != nil {
        logger.LogError(ctx, err, "Failed to fetch user", map[string]interface{}{
            "user_id": userID,
        })
        return fiber.ErrInternalServerError
    }

    // Log success
    logger.WithContext(ctx).WithField("user_id", userID).Info("User fetched successfully")

    return c.JSON(user)
})
```

## Configuration Options

### Logger Config

```go
type Config struct {
    Level  string    // "debug", "info", "warn", "error"
    Format string    // "json", "console"
    Output io.Writer // Optional custom output writer
}
```

### Middleware Config

```go
config := logger.MiddlewareConfig{
    SkipPaths:         []string{"/health", "/metrics"},
    SkipSuccessfulGET: false,
    RequestIDHeader:   "X-Request-ID",
}

app.Use(logger.RequestLoggerWithConfig(config))
```

### Environment-Based Configuration

```go
// Get config from environment variables
config := logger.GetConfigFromEnv()

// Get recommended config based on environment
config := logger.GetRecommendedConfig()

// Production: JSON format, info level
// Development: Console format, debug level
```

## File Output

```go
// Write to file
config, err := logger.ConfigWithFile("info", "json", "/var/log/app.log")
if err != nil {
    log.Fatal(err)
}

// Write to both file and console
config, err := logger.ConfigWithFileAndConsole("info", "json", "/var/log/app.log")
if err != nil {
    log.Fatal(err)
}
```

## Testing

The package includes comprehensive tests:

```bash
go test ./internal/logger/... -v
```

## Log Levels

- **Debug**: Detailed information for debugging
- **Info**: General application flow information
- **Warn**: Warning messages for potentially harmful situations
- **Error**: Error events that might still allow the application to continue
- **Fatal**: Severe error events that will presumably lead the application to abort
- **Panic**: Very severe error events that will cause the application to panic

## JSON Output Format

When using JSON format, logs are structured as:

```json
{
  "level": "info",
  "time": "2024-01-15T10:30:00Z",
  "message": "User action performed",
  "request_id": "req-123",
  "user_id": "user-456",
  "action": "purchase",
  "amount": 99.99
}
```

## Console Output Format

When using console format, logs are human-readable:

```
2024-01-15T10:30:00Z INF User action performed action=purchase amount=99.99 request_id=req-123 user_id=user-456
```

## Best Practices

1. **Always use context**: Pass context through your application to enable request tracing
2. **Use structured logging**: Prefer fields over formatted strings for better searchability
3. **Log at appropriate levels**: Use debug for development, info for important events, error for failures
4. **Include relevant context**: Add user IDs, request IDs, and other relevant metadata
5. **Don't log sensitive information**: Avoid logging passwords, tokens, or personal data
6. **Use specialized methods**: Use the provided helper methods for common scenarios
7. **Test your logging**: Ensure logs contain the expected information and format

## Dependencies

- [zerolog](https://github.com/rs/zerolog): High-performance structured logging
- [fiber/v2](https://github.com/gofiber/fiber): Web framework (for middleware)
- [google/uuid](https://github.com/google/uuid): UUID generation for request IDs