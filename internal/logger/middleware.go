package logger

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

// MiddlewareConfig holds configuration for logging middleware
type MiddlewareConfig struct {
	// SkipPaths defines paths to skip logging (e.g., health checks)
	SkipPaths []string
	// SkipSuccessfulGET skips logging successful GET requests
	SkipSuccessfulGET bool
	// RequestIDHeader specifies the header name for request ID
	RequestIDHeader string
}

// DefaultMiddlewareConfig returns default middleware configuration
func DefaultMiddlewareConfig() MiddlewareConfig {
	return MiddlewareConfig{
		SkipPaths:         []string{"/health", "/metrics"},
		SkipSuccessfulGET: false,
		RequestIDHeader:   "X-Request-ID",
	}
}

// RequestLoggerWithConfig returns a Fiber middleware for request logging with custom config
func RequestLoggerWithConfig(config MiddlewareConfig) fiber.Handler {
	skipPaths := make(map[string]bool)
	for _, path := range config.SkipPaths {
		skipPaths[path] = true
	}

	return func(c *fiber.Ctx) error {
		// Skip logging for specified paths
		if skipPaths[c.Path()] {
			return c.Next()
		}

		start := time.Now()

		// Generate or get request ID
		requestID := c.Get(config.RequestIDHeader)
		if requestID == "" {
			requestID = NewRequestID()
			c.Set(config.RequestIDHeader, requestID)
		}

		// Add request ID to context
		ctx := ContextWithRequestID(c.Context(), requestID)
		c.SetUserContext(ctx)

		// Log incoming request
		WithContext(ctx).WithFields(map[string]interface{}{
			"method":     c.Method(),
			"path":       c.Path(),
			"user_agent": c.Get("User-Agent"),
			"remote_ip":  c.IP(),
		}).Info("Incoming request")

		// Continue with request
		err := c.Next()

		// Calculate duration
		duration := time.Since(start)
		statusCode := c.Response().StatusCode()

		// Skip logging successful GET requests if configured
		if config.SkipSuccessfulGET && c.Method() == "GET" && statusCode < 400 {
			return err
		}

		// Log request completion
		LogRequest(ctx, c.Method(), c.Path(), statusCode, duration)

		return err
	}
}

// RequestLogger returns a Fiber middleware for request logging with default config
func RequestLogger() fiber.Handler {
	return RequestLoggerWithConfig(DefaultMiddlewareConfig())
}

// UserContextMiddleware extracts user information and adds it to context
func UserContextMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := c.UserContext()

		// Extract user ID from JWT token or session
		// This is a placeholder - implement based on your authentication mechanism
		userID := extractUserIDFromRequest(c)
		if userID != "" {
			ctx = ContextWithUserID(ctx, userID)
			c.SetUserContext(ctx)
		}

		return c.Next()
	}
}

// extractUserIDFromRequest extracts user ID from the request
// This is a placeholder implementation - replace with your actual authentication logic
func extractUserIDFromRequest(c *fiber.Ctx) string {
	// Example: Extract from JWT token
	// token := c.Get("Authorization")
	// if token != "" {
	//     // Parse JWT and extract user ID
	//     return parseUserIDFromJWT(token)
	// }

	// Example: Extract from custom header
	return c.Get("X-User-ID")
}

// ErrorHandler is a custom Fiber error handler that logs errors
func ErrorHandler(ctx *fiber.Ctx, err error) error {
	// Default error response
	code := fiber.StatusInternalServerError
	message := "Internal Server Error"

	// Check if it's a Fiber error
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		message = e.Message
	}

	// Log the error
	LogError(ctx.UserContext(), err, "Request error", map[string]interface{}{
		"method":      ctx.Method(),
		"path":        ctx.Path(),
		"status_code": code,
	})

	// Send error response
	return ctx.Status(code).JSON(fiber.Map{
		"error":   true,
		"message": message,
	})
}

// RecoveryMiddleware recovers from panics and logs them
func RecoveryMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) (err error) {
		defer func() {
			if r := recover(); r != nil {
				var ok bool
				err, ok = r.(error)
				if !ok {
					err = fiber.NewError(fiber.StatusInternalServerError, "panic occurred")
				}

				// Log the panic
				LogError(c.UserContext(), err, "Panic recovered", map[string]interface{}{
					"method": c.Method(),
					"path":   c.Path(),
					"panic":  r,
				})
			}
		}()

		return c.Next()
	}
}