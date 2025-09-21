package metrics

import (
	"context"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

// ExampleUsage demonstrates how to use the metrics package
func ExampleUsage() {
	// Create a new metrics service with custom configuration
	config := Config{
		Namespace: "myapp",
		Subsystem: "api",
		DurationBuckets: []float64{
			0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0,
		},
		SizeBuckets: []float64{
			100, 1000, 10000, 100000, 1000000,
		},
		ExcludeLabels:      []string{"user_agent"},
		MaxPathCardinality: 50,
	}

	metricsService := NewService(config)

	// Create Fiber app
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"error": true,
				"message": err.Error(),
			})
		},
	})

	// Add standard middleware
	app.Use(recover.New())
	app.Use(cors.New())

	// Add metrics middleware with custom configuration
	metricsConfig := MiddlewareConfig{
		Service:   metricsService,
		SkipPaths: []string{"/health", "/metrics"},
		PathNormalizer: func(path string) string {
			// Example: normalize paths with IDs
			// /api/users/123 -> /api/users/:id
			// /api/orders/456/items/789 -> /api/orders/:id/items/:id
			return normalizePath(path)
		},
	}
	app.Use(MiddlewareWithConfig(metricsConfig))

	// Register custom business metrics
	setupCustomMetrics(metricsService)

	// Add routes
	setupRoutes(app, metricsService)

	// Expose metrics endpoint
	app.Get("/metrics", metricsService.Handler())

	log.Println("Server starting on :8080")
	log.Println("Metrics available at http://localhost:8080/metrics")
	log.Fatal(app.Listen(":8080"))
}

// setupCustomMetrics demonstrates how to register custom business metrics
func setupCustomMetrics(service *Service) {
	// Business-specific metrics
	businessMetrics := service.NewBusinessMetrics()

	// Example: Register custom application metrics
	service.RegisterCounter(
		"feature_usage_total",
		"Total number of feature usages",
		[]string{"feature", "user_type"},
	)

	service.RegisterHistogram(
		"external_api_call_duration_seconds",
		"Duration of external API calls",
		[]string{"service", "endpoint", "status"},
		[]float64{0.1, 0.5, 1.0, 2.5, 5.0, 10.0},
	)

	service.RegisterGauge(
		"active_connections",
		"Current number of active connections",
		[]string{"type"},
	)

	// Example of using business metrics
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// Simulate business metrics
				businessMetrics.SetActiveUsers(float64(42 + time.Now().Second()%20))
				businessMetrics.IncrementUserAction("dashboard_view", "user123")
			}
		}
	}()
}

// setupRoutes demonstrates various route handlers with metrics integration
func setupRoutes(app *fiber.App, metricsService *Service) {
	businessMetrics := metricsService.NewBusinessMetrics()

	// Health check endpoint (excluded from metrics)
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":    "healthy",
			"timestamp": time.Now().Unix(),
		})
	})

	// API routes
	api := app.Group("/api/v1")

	// Users endpoints
	users := api.Group("/users")
	users.Get("/", func(c *fiber.Ctx) error {
		// Simulate database operation
		start := time.Now()
		time.Sleep(10 * time.Millisecond) // Simulate DB query
		businessMetrics.RecordDatabaseOperation("SELECT", "users", time.Since(start), true)

		// Record custom feature usage
		if counter, ok := metricsService.GetCounter("feature_usage_total"); ok {
			counter.WithLabelValues("list_users", "standard").Inc()
		}

		return c.JSON(fiber.Map{
			"users": []string{"user1", "user2", "user3"},
		})
	})

	users.Get("/:id", func(c *fiber.Ctx) error {
		userID := c.Params("id")

		// Simulate database operation
		start := time.Now()
		time.Sleep(5 * time.Millisecond) // Simulate DB query
		businessMetrics.RecordDatabaseOperation("SELECT", "users", time.Since(start), true)

		// Record user action
		businessMetrics.IncrementUserAction("profile_view", userID)

		return c.JSON(fiber.Map{
			"id":   userID,
			"name": "John Doe",
		})
	})

	users.Post("/", func(c *fiber.Ctx) error {
		// Simulate user creation
		start := time.Now()
		time.Sleep(50 * time.Millisecond) // Simulate DB insert
		businessMetrics.RecordDatabaseOperation("INSERT", "users", time.Since(start), true)

		// Record business transaction
		businessMetrics.RecordBusinessTransaction("user_registration", 0, time.Since(start))

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"id":      "new-user-id",
			"message": "User created successfully",
		})
	})

	// Orders endpoints
	orders := api.Group("/orders")
	orders.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"orders": []string{"order1", "order2"},
		})
	})

	orders.Post("/", func(c *fiber.Ctx) error {
		// Simulate external payment API call
		start := time.Now()
		success := simulateExternalAPICall("payment", "/charge")
		duration := time.Since(start)

		// Record external API call metrics
		if histogram, ok := metricsService.GetHistogram("external_api_call_duration_seconds"); ok {
			status := "success"
			if !success {
				status = "error"
			}
			histogram.WithLabelValues("payment", "/charge", status).Observe(duration.Seconds())
		}

		if !success {
			return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
				"error": "Payment service unavailable",
			})
		}

		// Record successful business transaction
		businessMetrics.RecordBusinessTransaction("order_creation", 99.99, duration)

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"id":      "new-order-id",
			"amount":  99.99,
			"message": "Order created successfully",
		})
	})

	// Admin endpoints
	admin := api.Group("/admin")
	admin.Get("/stats", func(c *fiber.Ctx) error {
		// Record admin feature usage
		if counter, ok := metricsService.GetCounter("feature_usage_total"); ok {
			counter.WithLabelValues("admin_stats", "admin").Inc()
		}

		return c.JSON(fiber.Map{
			"total_users":  1000,
			"total_orders": 500,
		})
	})

	// Simulate error endpoint
	api.Get("/error", func(c *fiber.Ctx) error {
		// This will be recorded as a 500 error in metrics
		return fiber.NewError(fiber.StatusInternalServerError, "Simulated error")
	})

	// Slow endpoint for testing duration metrics
	api.Get("/slow", func(c *fiber.Ctx) error {
		time.Sleep(2 * time.Second)
		return c.JSON(fiber.Map{"message": "slow response"})
	})
}

// normalizePath normalizes URL paths to control metric cardinality
func normalizePath(path string) string {
	// Example implementation - replace with your actual path normalization logic
	// This is a simple implementation that replaces common ID patterns

	// Handle common patterns:
	// /api/v1/users/123 -> /api/v1/users/:id
	// /api/v1/orders/456/items/789 -> /api/v1/orders/:id/items/:id

	// You would implement more sophisticated logic here based on your API structure
	// For now, just return the original path
	return path
}

// simulateExternalAPICall simulates an external API call
func simulateExternalAPICall(service, endpoint string) bool {
	// Simulate random success/failure and variable latency
	time.Sleep(time.Duration(100+time.Now().UnixNano()%400) * time.Millisecond)
	return time.Now().UnixNano()%10 > 1 // 90% success rate
}

// ExampleCustomMetricsUsage demonstrates advanced custom metrics usage
func ExampleCustomMetricsUsage() {
	service := NewServiceWithDefaults()

	// Register application-specific metrics
	cacheHits := service.RegisterCounter(
		"cache_hits_total",
		"Total number of cache hits",
		[]string{"cache_type", "key_type"},
	)

	cacheLatency := service.RegisterHistogram(
		"cache_operation_duration_seconds",
		"Cache operation duration",
		[]string{"operation", "cache_type"},
		[]float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.05},
	)

	queueSize := service.RegisterGauge(
		"queue_size",
		"Current queue size",
		[]string{"queue_name"},
	)

	processingSummary := service.RegisterSummary(
		"processing_duration_seconds",
		"Processing duration summary",
		[]string{"processor"},
		map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
	)

	// Example usage in application code
	ctx := context.Background()

	// Simulate cache operations
	go func() {
		for {
			// Cache hit
			cacheHits.WithLabelValues("redis", "user_session").Inc()

			// Cache operation timing
			start := time.Now()
			time.Sleep(time.Duration(1+time.Now().UnixNano()%5) * time.Millisecond)
			cacheLatency.WithLabelValues("get", "redis").Observe(time.Since(start).Seconds())

			time.Sleep(100 * time.Millisecond)
		}
	}()

	// Simulate queue monitoring
	go func() {
		for {
			// Simulate varying queue sizes
			size := float64(10 + time.Now().Unix()%50)
			queueSize.WithLabelValues("email_queue").Set(size)
			queueSize.WithLabelValues("notification_queue").Set(size / 2)

			time.Sleep(5 * time.Second)
		}
	}()

	// Simulate processing operations
	go func() {
		for {
			start := time.Now()
			time.Sleep(time.Duration(50+time.Now().UnixNano()%200) * time.Millisecond)
			processingSummary.WithLabelValues("image_processor").Observe(time.Since(start).Seconds())

			time.Sleep(500 * time.Millisecond)
		}
	}()

	// This would normally be part of your main application loop
	select {
	case <-ctx.Done():
		return
	}
}