package health

import (
	"context"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusDegraded  Status = "degraded"
	StatusUnhealthy Status = "unhealthy"
)

type ComponentHealth struct {
	Name    string    `json:"name"`
	Status  Status    `json:"status"`
	Message string    `json:"message,omitempty"`
	LastCheck time.Time `json:"last_check"`
}

type HealthChecker interface {
	Check(ctx context.Context) error
	Name() string
}

type Service struct {
	mu       sync.RWMutex
	checkers map[string]HealthChecker
	status   map[string]*ComponentHealth
}

func NewService() *Service {
	return &Service{
		checkers: make(map[string]HealthChecker),
		status:   make(map[string]*ComponentHealth),
	}
}

func (s *Service) RegisterChecker(name string, checker HealthChecker) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checkers[name] = checker
	s.status[name] = &ComponentHealth{
		Name:   name,
		Status: StatusHealthy,
		LastCheck: time.Now(),
	}
}

func (s *Service) LivenessHandler(c *fiber.Ctx) error {
	// Simple liveness check - just return OK if the service is running
	return c.JSON(fiber.Map{
		"status": "alive",
		"timestamp": time.Now().Unix(),
	})
}

func (s *Service) ReadinessHandler(c *fiber.Ctx) error {
	ctx := c.Context()
	overallStatus := StatusHealthy
	components := make([]ComponentHealth, 0)

	s.mu.RLock()
	defer s.mu.RUnlock()

	for name, checker := range s.checkers {
		health := ComponentHealth{
			Name:      name,
			Status:    StatusHealthy,
			LastCheck: time.Now(),
		}

		if err := checker.Check(ctx); err != nil {
			health.Status = StatusUnhealthy
			health.Message = err.Error()
			overallStatus = StatusUnhealthy
		}

		s.status[name] = &health
		components = append(components, health)
	}

	statusCode := fiber.StatusOK
	if overallStatus == StatusUnhealthy {
		statusCode = fiber.StatusServiceUnavailable
	}

	return c.Status(statusCode).JSON(fiber.Map{
		"status":     overallStatus,
		"components": components,
		"timestamp":  time.Now().Unix(),
	})
}

func (s *Service) StatusHandler(c *fiber.Ctx) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	components := make([]ComponentHealth, 0, len(s.status))
	overallStatus := StatusHealthy

	for _, health := range s.status {
		components = append(components, *health)
		if health.Status == StatusUnhealthy {
			overallStatus = StatusUnhealthy
		} else if health.Status == StatusDegraded && overallStatus == StatusHealthy {
			overallStatus = StatusDegraded
		}
	}

	return c.JSON(fiber.Map{
		"status":     overallStatus,
		"components": components,
		"timestamp":  time.Now().Unix(),
		"version":    getVersion(),
		"uptime":     getUptime(),
	})
}

func (s *Service) StartBackgroundChecks(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				s.runChecks(ctx)
			}
		}
	}()
}

func (s *Service) runChecks(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for name, checker := range s.checkers {
		health := &ComponentHealth{
			Name:      name,
			Status:    StatusHealthy,
			LastCheck: time.Now(),
		}

		if err := checker.Check(ctx); err != nil {
			health.Status = StatusUnhealthy
			health.Message = err.Error()
		}

		s.status[name] = health
	}
}

var startTime = time.Now()

func getUptime() string {
	return time.Since(startTime).Round(time.Second).String()
}

func getVersion() string {
	// This would normally come from build flags
	return "v1.0.0"
}

// Example checker implementations

type DatabaseChecker struct {
	connectionString string
}

func NewDatabaseChecker(connStr string) *DatabaseChecker {
	return &DatabaseChecker{connectionString: connStr}
}

func (d *DatabaseChecker) Name() string {
	return "database"
}

func (d *DatabaseChecker) Check(ctx context.Context) error {
	// TODO: Implement actual database ping
	// For now, just return nil (healthy)
	return nil
}

type GRPCServiceChecker struct {
	serviceName string
	address     string
}

func NewGRPCServiceChecker(name, addr string) *GRPCServiceChecker {
	return &GRPCServiceChecker{
		serviceName: name,
		address:     addr,
	}
}

func (g *GRPCServiceChecker) Name() string {
	return g.serviceName
}

func (g *GRPCServiceChecker) Check(ctx context.Context) error {
	// TODO: Implement actual gRPC health check
	// For now, just return nil (healthy)
	return nil
}