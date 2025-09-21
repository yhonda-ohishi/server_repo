package gateway

import (
	"context"
	"fmt"
	"sync"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/yhonda-ohishi/db-handler-server/internal/client"
	"github.com/yhonda-ohishi/db-handler-server/internal/config"
	"github.com/yhonda-ohishi/db-handler-server/internal/health"
	"github.com/yhonda-ohishi/db-handler-server/internal/services"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// SimpleGateway provides a basic working gateway implementation
type SimpleGateway struct {
	config         *config.Config
	app            *fiber.App
	grpcServer     *grpc.Server
	bufconnClient  *client.BufconnClient
	healthService  *health.Service
	serviceRegistry *services.ServiceRegistry
	wg             sync.WaitGroup
}

// NewSimpleGateway creates a new simple gateway
func NewSimpleGateway(cfg *config.Config) *SimpleGateway {
	app := fiber.New(fiber.Config{
		AppName: "ETC Meisai Gateway",
	})

	// Add middleware
	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Content-Type,Authorization",
	}))

	return &SimpleGateway{
		config: cfg,
		app:    app,
	}
}

// Start starts the gateway in the configured mode
func (g *SimpleGateway) Start(ctx context.Context) error {
	if g.config.IsSingleMode() {
		return g.startSingleMode(ctx)
	}
	return g.startSeparateMode(ctx)
}

// startSingleMode starts gateway with bufconn
func (g *SimpleGateway) startSingleMode(ctx context.Context) error {
	// Create bufconn client
	g.bufconnClient = client.NewBufconnClient()

	// Create gRPC server but don't start it yet
	g.grpcServer = grpc.NewServer()

	// Register services first - use single mode registry with mock DB services
	g.serviceRegistry = services.NewServiceRegistryForSingleMode()
	g.serviceRegistry.RegisterAll(g.grpcServer)

	// Enable reflection
	reflection.Register(g.grpcServer)

	// Now start the server with the listener
	listener := g.bufconnClient.GetListener()
	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		fmt.Println("Starting gRPC server on bufconn")
		if err := g.grpcServer.Serve(listener); err != nil {
			fmt.Printf("gRPC server error: %v\n", err)
		}
	}()

	// Get a connection to the bufconn server for REST proxy
	conn, err := g.bufconnClient.GetConnection(ctx)
	if err != nil {
		return fmt.Errorf("failed to get bufconn connection: %w", err)
	}

	// Setup basic REST endpoints
	g.setupBasicEndpoints()

	// Setup db_service REST routes
	dbRoutes := NewDBServiceRoutes(conn)
	dbRoutes.RegisterRoutes(g.app)

	// Setup Swagger UI
	g.SetupSwaggerUI()

	// Start HTTP server
	return g.startHTTPServer()
}

// startSeparateMode starts gateway with network connections
func (g *SimpleGateway) startSeparateMode(ctx context.Context) error {
	// Setup basic endpoints
	g.setupBasicEndpoints()

	// Setup Swagger UI
	g.SetupSwaggerUI()

	// Start HTTP server
	return g.startHTTPServer()
}

// setupBasicEndpoints sets up basic API endpoints
func (g *SimpleGateway) setupBasicEndpoints() {
	// Root endpoint for testing
	g.app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "ETC Meisai Gateway is running",
			"version": "v1.0.0",
			"mode":    g.config.Deployment.Mode,
		})
	})

	// Health endpoints
	g.app.Get("/health/live", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "alive"})
	})

	g.app.Get("/health/ready", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ready"})
	})

	// Info endpoint
	g.app.Get("/info", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"name":    "ETC Meisai Gateway",
			"version": "v1.0.0",
			"mode":    g.config.Deployment.Mode,
		})
	})

	// Basic API endpoints for testing
	api := g.app.Group("/api/v1")

	// Users endpoint
	api.Get("/users", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"users": []fiber.Map{
				{"id": "1", "name": "Test User 1", "email": "user1@example.com"},
				{"id": "2", "name": "Test User 2", "email": "user2@example.com"},
			},
		})
	})

	// Transactions endpoint
	api.Get("/transactions", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"transactions": []fiber.Map{
				{"id": "1", "amount": 1000, "card_id": "card1"},
				{"id": "2", "amount": 1500, "card_id": "card2"},
			},
		})
	})

	// ETC明細 endpoints
	api.Get("/etc/meisai", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"etc_meisai": []fiber.Map{
				{
					"id": "1",
					"date": "2024-01-15",
					"entrance_ic": "首都高速道路 入口",
					"exit_ic": "名神高速道路 出口",
					"toll_amount": 8500,
					"final_amount": 8000,
					"car_number": "品川 500 あ 1234",
				},
				{
					"id": "2",
					"date": "2024-01-20",
					"entrance_ic": "第三京浜道路 入口",
					"exit_ic": "東名高速道路 出口",
					"toll_amount": 6200,
					"final_amount": 5900,
					"car_number": "横浜 301 さ 5678",
				},
			},
		})
	})

	api.Get("/etc/summary", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"summary": fiber.Map{
				"total_transactions": 3,
				"total_amount": 28300,
				"total_toll": 30100,
				"total_discount": 1800,
			},
		})
	})
}

// startHTTPServer starts the HTTP server
func (g *SimpleGateway) startHTTPServer() error {
	address := fmt.Sprintf(":%d", g.config.Server.HTTPPort)

	fmt.Printf("Starting HTTP server on %s\n", address)
	// Start in goroutine and use a channel to signal when it's ready
	errCh := make(chan error, 1)

	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		if err := g.app.Listen(address); err != nil {
			errCh <- fmt.Errorf("HTTP server error: %w", err)
		}
	}()

	// Give the server a moment to start
	go func() {
		select {
		case err := <-errCh:
			fmt.Printf("Server start error: %v\n", err)
		}
	}()

	return nil
}

// Stop stops the gateway
func (g *SimpleGateway) Stop() error {
	fmt.Println("Stopping gateway...")

	if g.app != nil {
		_ = g.app.Shutdown()
	}

	if g.grpcServer != nil {
		g.grpcServer.GracefulStop()
	}

	if g.bufconnClient != nil {
		_ = g.bufconnClient.Close()
	}

	g.wg.Wait()
	fmt.Println("Gateway stopped")
	return nil
}