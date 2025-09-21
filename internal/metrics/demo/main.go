package main

import (
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/yhonda-ohishi/db-handler-server/internal/metrics"
)

func main() {
	// Create a new metrics service with custom configuration
	config := metrics.Config{
		Namespace: "demo",
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

	metricsService := metrics.NewService(config)

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

	// Add metrics middleware
	app.Use(metricsService.Middleware())

	// Register custom business metrics
	setupCustomMetrics(metricsService)

	// Add routes
	setupRoutes(app, metricsService)

	// Expose metrics endpoint
	app.Get("/metrics", metricsService.Handler())

	log.Println("Demo server starting on :8080")
	log.Println("Metrics available at http://localhost:8080/metrics")
	log.Println("Test endpoints:")
	log.Println("  GET  http://localhost:8080/health")
	log.Println("  GET  http://localhost:8080/api/users")
	log.Println("  GET  http://localhost:8080/api/users/123")
	log.Println("  POST http://localhost:8080/api/users")
	log.Println("  GET  http://localhost:8080/api/slow")
	log.Println("  GET  http://localhost:8080/api/error")

	log.Fatal(app.Listen(":8080"))
}

func setupCustomMetrics(service *metrics.Service) {
	// Business-specific metrics
	businessMetrics := service.NewBusinessMetrics()

	// Register custom application metrics
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

	// Example of periodic business metrics updates
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// Simulate business metrics
				businessMetrics.SetActiveUsers(float64(42 + time.Now().Second()%20))
				businessMetrics.IncrementUserAction("dashboard_view", "demo_user")
			}
		}
	}()
}

func setupRoutes(app *fiber.App, metricsService *metrics.Service) {
	businessMetrics := metricsService.NewBusinessMetrics()

	// Health check endpoint (excluded from metrics)
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":    "healthy",
			"timestamp": time.Now().Unix(),
		})
	})

	// API routes
	api := app.Group("/api")

	// Users endpoints
	api.Get("/users", func(c *fiber.Ctx) error {
		// Simulate database operation
		start := time.Now()
		time.Sleep(10 * time.Millisecond) // Simulate DB query
		businessMetrics.RecordDatabaseOperation("SELECT", "users", time.Since(start), true)

		// Record custom feature usage
		if counter, ok := metricsService.GetCounter("feature_usage_total"); ok {
			counter.WithLabelValues("list_users", "standard").Inc()
		}

		return c.JSON(fiber.Map{
			"users": []map[string]string{
				{"id": "1", "name": "Alice"},
				{"id": "2", "name": "Bob"},
				{"id": "3", "name": "Charlie"},
			},
		})
	})

	api.Get("/users/:id", func(c *fiber.Ctx) error {
		userID := c.Params("id")

		// Simulate database operation
		start := time.Now()
		time.Sleep(5 * time.Millisecond) // Simulate DB query
		businessMetrics.RecordDatabaseOperation("SELECT", "users", time.Since(start), true)

		// Record user action
		businessMetrics.IncrementUserAction("profile_view", userID)

		return c.JSON(fiber.Map{
			"id":   userID,
			"name": "User " + userID,
		})
	})

	api.Post("/users", func(c *fiber.Ctx) error {
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

	// Slow endpoint for testing duration metrics
	api.Get("/slow", func(c *fiber.Ctx) error {
		time.Sleep(2 * time.Second)
		return c.JSON(fiber.Map{"message": "slow response"})
	})

	// Error endpoint for testing error metrics
	api.Get("/error", func(c *fiber.Ctx) error {
		return fiber.NewError(fiber.StatusInternalServerError, "Simulated error")
	})

	// Custom metrics endpoint
	api.Get("/custom-metric", func(c *fiber.Ctx) error {
		// Example of using custom metrics in handlers
		if counter, ok := metricsService.GetCounter("feature_usage_total"); ok {
			counter.WithLabelValues("custom_endpoint", "premium").Inc()
		}

		// Simulate external API call
		start := time.Now()
		time.Sleep(time.Duration(100+time.Now().UnixNano()%200) * time.Millisecond)
		success := time.Now().UnixNano()%10 > 1 // 90% success rate

		if histogram, ok := metricsService.GetHistogram("external_api_call_duration_seconds"); ok {
			status := "success"
			if !success {
				status = "error"
			}
			histogram.WithLabelValues("payment", "/charge", status).Observe(time.Since(start).Seconds())
		}

		if !success {
			return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
				"error": "External service unavailable",
			})
		}

		return c.JSON(fiber.Map{
			"message": "Custom metric recorded",
			"duration": time.Since(start).String(),
		})
	})
}