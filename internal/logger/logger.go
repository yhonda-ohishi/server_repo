package logger

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// ContextKey is used for context-based values
type ContextKey string

const (
	// RequestIDKey is the context key for request ID
	RequestIDKey ContextKey = "request_id"
	// UserIDKey is the context key for user ID
	UserIDKey ContextKey = "user_id"
)

// Logger wraps zerolog.Logger with additional functionality
type Logger struct {
	logger zerolog.Logger
}

// Config holds logger configuration
type Config struct {
	Level  string // debug, info, warn, error
	Format string // json, console
	Output io.Writer
}

var (
	// Global logger instance
	globalLogger *Logger
)

// Initialize sets up the global logger with the provided configuration
func Initialize(config Config) error {
	level, err := parseLogLevel(config.Level)
	if err != nil {
		return fmt.Errorf("invalid log level: %w", err)
	}

	// Set global log level
	zerolog.SetGlobalLevel(level)

	// Configure output writer
	var output io.Writer = os.Stdout
	if config.Output != nil {
		output = config.Output
	}

	// Configure format
	var logger zerolog.Logger
	switch strings.ToLower(config.Format) {
	case "console":
		logger = zerolog.New(zerolog.ConsoleWriter{
			Out:        output,
			TimeFormat: time.RFC3339,
			NoColor:    false,
		}).With().Timestamp().Logger()
	case "json":
		fallthrough
	default:
		logger = zerolog.New(output).With().Timestamp().Logger()
	}

	globalLogger = &Logger{logger: logger}

	// Set zerolog global logger
	log.Logger = logger

	return nil
}

// GetLogger returns the global logger instance
func GetLogger() *Logger {
	if globalLogger == nil {
		// Initialize with default config if not already initialized
		_ = Initialize(Config{
			Level:  "info",
			Format: "json",
		})
	}
	return globalLogger
}

// parseLogLevel converts string level to zerolog.Level
func parseLogLevel(level string) (zerolog.Level, error) {
	switch strings.ToLower(level) {
	case "debug":
		return zerolog.DebugLevel, nil
	case "info":
		return zerolog.InfoLevel, nil
	case "warn", "warning":
		return zerolog.WarnLevel, nil
	case "error":
		return zerolog.ErrorLevel, nil
	case "fatal":
		return zerolog.FatalLevel, nil
	case "panic":
		return zerolog.PanicLevel, nil
	case "disabled":
		return zerolog.Disabled, nil
	default:
		return zerolog.InfoLevel, fmt.Errorf("unknown log level: %s", level)
	}
}

// WithContext returns a new logger with context values
func (l *Logger) WithContext(ctx context.Context) *Logger {
	logger := l.logger

	// Add request ID if present
	if requestID := ctx.Value(RequestIDKey); requestID != nil {
		logger = logger.With().Str("request_id", requestID.(string)).Logger()
	}

	// Add user ID if present
	if userID := ctx.Value(UserIDKey); userID != nil {
		logger = logger.With().Str("user_id", userID.(string)).Logger()
	}

	return &Logger{logger: logger}
}

// WithFields returns a new logger with additional fields
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	event := l.logger.With()
	for key, value := range fields {
		event = event.Interface(key, value)
	}
	return &Logger{logger: event.Logger()}
}

// WithField returns a new logger with an additional field
func (l *Logger) WithField(key string, value interface{}) *Logger {
	return &Logger{logger: l.logger.With().Interface(key, value).Logger()}
}

// WithError returns a new logger with error field
func (l *Logger) WithError(err error) *Logger {
	return &Logger{logger: l.logger.With().Err(err).Logger()}
}

// Debug logs a debug message
func (l *Logger) Debug(msg string) {
	l.logger.Debug().Msg(msg)
}

// Debugf logs a formatted debug message
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.logger.Debug().Msgf(format, args...)
}

// Info logs an info message
func (l *Logger) Info(msg string) {
	l.logger.Info().Msg(msg)
}

// Infof logs a formatted info message
func (l *Logger) Infof(format string, args ...interface{}) {
	l.logger.Info().Msgf(format, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string) {
	l.logger.Warn().Msg(msg)
}

// Warnf logs a formatted warning message
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.logger.Warn().Msgf(format, args...)
}

// Error logs an error message
func (l *Logger) Error(msg string) {
	l.logger.Error().Msg(msg)
}

// Errorf logs a formatted error message
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.logger.Error().Msgf(format, args...)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(msg string) {
	l.logger.Fatal().Msg(msg)
}

// Fatalf logs a formatted fatal message and exits
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.logger.Fatal().Msgf(format, args...)
}

// Panic logs a panic message and panics
func (l *Logger) Panic(msg string) {
	l.logger.Panic().Msg(msg)
}

// Panicf logs a formatted panic message and panics
func (l *Logger) Panicf(format string, args ...interface{}) {
	l.logger.Panic().Msgf(format, args...)
}

// Global convenience functions using the global logger

// Debug logs a debug message using the global logger
func Debug(msg string) {
	GetLogger().Debug(msg)
}

// Debugf logs a formatted debug message using the global logger
func Debugf(format string, args ...interface{}) {
	GetLogger().Debugf(format, args...)
}

// Info logs an info message using the global logger
func Info(msg string) {
	GetLogger().Info(msg)
}

// Infof logs a formatted info message using the global logger
func Infof(format string, args ...interface{}) {
	GetLogger().Infof(format, args...)
}

// Warn logs a warning message using the global logger
func Warn(msg string) {
	GetLogger().Warn(msg)
}

// Warnf logs a formatted warning message using the global logger
func Warnf(format string, args ...interface{}) {
	GetLogger().Warnf(format, args...)
}

// Error logs an error message using the global logger
func Error(msg string) {
	GetLogger().Error(msg)
}

// Errorf logs a formatted error message using the global logger
func Errorf(format string, args ...interface{}) {
	GetLogger().Errorf(format, args...)
}

// Fatal logs a fatal message using the global logger and exits
func Fatal(msg string) {
	GetLogger().Fatal(msg)
}

// Fatalf logs a formatted fatal message using the global logger and exits
func Fatalf(format string, args ...interface{}) {
	GetLogger().Fatalf(format, args...)
}

// WithContext returns a logger with context values using the global logger
func WithContext(ctx context.Context) *Logger {
	return GetLogger().WithContext(ctx)
}

// WithFields returns a logger with additional fields using the global logger
func WithFields(fields map[string]interface{}) *Logger {
	return GetLogger().WithFields(fields)
}

// WithField returns a logger with an additional field using the global logger
func WithField(key string, value interface{}) *Logger {
	return GetLogger().WithField(key, value)
}

// WithError returns a logger with error field using the global logger
func WithError(err error) *Logger {
	return GetLogger().WithError(err)
}

// Context helper functions

// NewRequestID generates a new request ID
func NewRequestID() string {
	return uuid.New().String()
}

// ContextWithRequestID adds a request ID to the context
func ContextWithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// ContextWithUserID adds a user ID to the context
func ContextWithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}

// GetRequestIDFromContext extracts request ID from context
func GetRequestIDFromContext(ctx context.Context) (string, bool) {
	requestID, ok := ctx.Value(RequestIDKey).(string)
	return requestID, ok
}

// GetUserIDFromContext extracts user ID from context
func GetUserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(UserIDKey).(string)
	return userID, ok
}

// Specialized logging methods for common scenarios

// LogRequest logs HTTP request details
func LogRequest(ctx context.Context, method, path string, statusCode int, duration time.Duration) {
	logger := WithContext(ctx).WithFields(map[string]interface{}{
		"method":      method,
		"path":        path,
		"status_code": statusCode,
		"duration_ms": duration.Milliseconds(),
	})

	if statusCode >= 500 {
		logger.Error("HTTP request completed with server error")
	} else if statusCode >= 400 {
		logger.Warn("HTTP request completed with client error")
	} else {
		logger.Info("HTTP request completed")
	}
}

// LogError logs an error with context and additional fields
func LogError(ctx context.Context, err error, msg string, fields map[string]interface{}) {
	logger := WithContext(ctx).WithError(err)
	if fields != nil {
		logger = logger.WithFields(fields)
	}
	logger.Error(msg)
}

// LogBusinessEvent logs a business event with structured data
func LogBusinessEvent(ctx context.Context, event string, data map[string]interface{}) {
	logger := WithContext(ctx).WithField("event", event)
	if data != nil {
		logger = logger.WithFields(data)
	}
	logger.Info("Business event")
}

// LogDatabaseOperation logs database operations
func LogDatabaseOperation(ctx context.Context, operation, table string, duration time.Duration, err error) {
	logger := WithContext(ctx).WithFields(map[string]interface{}{
		"operation":   operation,
		"table":       table,
		"duration_ms": duration.Milliseconds(),
	})

	if err != nil {
		logger.WithError(err).Error("Database operation failed")
	} else {
		logger.Debug("Database operation completed")
	}
}

// LogExternalAPICall logs external API calls
func LogExternalAPICall(ctx context.Context, service, endpoint string, statusCode int, duration time.Duration, err error) {
	logger := WithContext(ctx).WithFields(map[string]interface{}{
		"service":     service,
		"endpoint":    endpoint,
		"status_code": statusCode,
		"duration_ms": duration.Milliseconds(),
	})

	if err != nil {
		logger.WithError(err).Error("External API call failed")
	} else if statusCode >= 400 {
		logger.Warn("External API call returned error status")
	} else {
		logger.Info("External API call completed")
	}
}

// Fiber middleware

// FiberRequestLogger returns a Fiber middleware for request logging
func FiberRequestLogger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		// Generate request ID if not present
		requestID := c.Get("X-Request-ID")
		if requestID == "" {
			requestID = NewRequestID()
			c.Set("X-Request-ID", requestID)
		}

		// Add request ID to context
		ctx := ContextWithRequestID(c.Context(), requestID)
		c.SetUserContext(ctx)

		// Continue with request
		err := c.Next()

		// Log request completion
		duration := time.Since(start)
		LogRequest(ctx, c.Method(), c.Path(), c.Response().StatusCode(), duration)

		return err
	}
}

// FiberErrorLogger returns a Fiber middleware for error logging
func FiberErrorLogger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		err := c.Next()
		if err != nil {
			ctx := c.UserContext()
			LogError(ctx, err, "Request handler error", map[string]interface{}{
				"method": c.Method(),
				"path":   c.Path(),
			})
		}
		return err
	}
}