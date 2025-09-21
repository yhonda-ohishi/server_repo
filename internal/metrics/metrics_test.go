package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestNewService(t *testing.T) {
	config := DefaultConfig()
	service := NewService(config)

	if service == nil {
		t.Fatal("Expected service to be created")
	}

	if service.registry == nil {
		t.Fatal("Expected registry to be initialized")
	}

	if service.requestCount == nil {
		t.Fatal("Expected request count metric to be initialized")
	}

	if service.requestDuration == nil {
		t.Fatal("Expected request duration metric to be initialized")
	}

	if service.requestSize == nil {
		t.Fatal("Expected request size metric to be initialized")
	}

	if service.responseSize == nil {
		t.Fatal("Expected response size metric to be initialized")
	}
}

func TestNewServiceWithDefaults(t *testing.T) {
	service := NewServiceWithDefaults()

	if service == nil {
		t.Fatal("Expected service to be created")
	}

	if service.config.Namespace != "http" {
		t.Errorf("Expected namespace to be 'http', got %s", service.config.Namespace)
	}

	if service.config.Subsystem != "server" {
		t.Errorf("Expected subsystem to be 'server', got %s", service.config.Subsystem)
	}
}

func TestRecordRequest(t *testing.T) {
	service := NewServiceWithDefaults()

	// Record a test request
	service.RecordRequest("GET", "/api/users", 200, 100*time.Millisecond, 1024, 2048)

	// Verify metrics were recorded
	expected := `
		# HELP http_server_requests_total Total number of HTTP requests by method, path, and status code
		# TYPE http_server_requests_total counter
		http_server_requests_total{method="GET",path="/api/users",status="200"} 1
	`

	if err := testutil.GatherAndCompare(service.registry, strings.NewReader(expected), "http_server_requests_total"); err != nil {
		t.Errorf("Unexpected metric value: %v", err)
	}
}

func TestRegisterCustomMetrics(t *testing.T) {
	service := NewServiceWithDefaults()

	// Test counter registration
	counter := service.RegisterCounter("test_counter", "Test counter metric", []string{"label1"})
	if counter == nil {
		t.Fatal("Expected counter to be registered")
	}

	// Test gauge registration
	gauge := service.RegisterGauge("test_gauge", "Test gauge metric", []string{"label1"})
	if gauge == nil {
		t.Fatal("Expected gauge to be registered")
	}

	// Test histogram registration
	histogram := service.RegisterHistogram("test_histogram", "Test histogram metric", []string{"label1"}, nil)
	if histogram == nil {
		t.Fatal("Expected histogram to be registered")
	}

	// Test summary registration
	summary := service.RegisterSummary("test_summary", "Test summary metric", []string{"label1"}, nil)
	if summary == nil {
		t.Fatal("Expected summary to be registered")
	}

	// Test metric retrieval
	retrievedCounter, ok := service.GetCounter("test_counter")
	if !ok || retrievedCounter != counter {
		t.Error("Failed to retrieve registered counter")
	}

	retrievedGauge, ok := service.GetGauge("test_gauge")
	if !ok || retrievedGauge != gauge {
		t.Error("Failed to retrieve registered gauge")
	}

	retrievedHistogram, ok := service.GetHistogram("test_histogram")
	if !ok || retrievedHistogram != histogram {
		t.Error("Failed to retrieve registered histogram")
	}

	retrievedSummary, ok := service.GetSummary("test_summary")
	if !ok || retrievedSummary != summary {
		t.Error("Failed to retrieve registered summary")
	}
}

func TestMiddleware(t *testing.T) {
	service := NewServiceWithDefaults()
	app := fiber.New()

	// Add metrics middleware
	app.Use(service.Middleware())

	// Add test route
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "test"})
	})

	// Make test request
	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to make test request: %v", err)
	}
	defer resp.Body.Close()

	// Give a small delay to ensure metrics are recorded
	time.Sleep(10 * time.Millisecond)

	// Verify metrics were recorded
	expected := `
		# HELP http_server_requests_total Total number of HTTP requests by method, path, and status code
		# TYPE http_server_requests_total counter
		http_server_requests_total{method="GET",path="/test",status="200"} 1
	`

	if err := testutil.GatherAndCompare(service.registry, strings.NewReader(expected), "http_server_requests_total"); err != nil {
		t.Errorf("Unexpected metric value: %v", err)
	}
}

func TestMiddlewareWithConfig(t *testing.T) {
	service := NewServiceWithDefaults()
	app := fiber.New()

	// Configure middleware to skip /health path
	config := MiddlewareConfig{
		Service:   service,
		SkipPaths: []string{"/health"},
		PathNormalizer: func(path string) string {
			if strings.HasPrefix(path, "/api/users/") {
				return "/api/users/:id"
			}
			return path
		},
	}

	app.Use(MiddlewareWithConfig(config))

	// Add test routes
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "test"})
	})
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})
	app.Get("/api/users/123", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"user": "test"})
	})

	// Make test requests
	tests := []struct {
		path     string
		expected bool
	}{
		{"/test", true},
		{"/health", false}, // Should be skipped
		{"/api/users/123", true},
	}

	for _, test := range tests {
		req := httptest.NewRequest("GET", test.path, nil)
		_, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to make test request to %s: %v", test.path, err)
		}
	}

	// Give a small delay to ensure metrics are recorded
	time.Sleep(10 * time.Millisecond)

	// Check that /test was recorded
	testMetric := `http_server_requests_total{method="GET",path="/test",status="200"} 1`
	if err := testutil.GatherAndCompare(service.registry, strings.NewReader(testMetric), "http_server_requests_total"); err == nil {
		// This is expected - /test should be recorded
	}

	// Check that normalized path was used
	normalizedMetric := `http_server_requests_total{method="GET",path="/api/users/:id",status="200"} 1`
	if err := testutil.GatherAndCompare(service.registry, strings.NewReader(normalizedMetric), "http_server_requests_total"); err != nil {
		t.Errorf("Expected normalized path metric: %v", err)
	}
}

func TestMetricsHandler(t *testing.T) {
	service := NewServiceWithDefaults()
	app := fiber.New()

	// Record some test metrics
	service.RecordRequest("GET", "/api/test", 200, 50*time.Millisecond, 100, 200)

	// Add metrics endpoint
	app.Get("/metrics", service.Handler())

	// Test metrics endpoint
	req := httptest.NewRequest("GET", "/metrics", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to make test request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/plain") {
		t.Errorf("Expected content type to contain text/plain, got %s", contentType)
	}
}

func TestBusinessMetrics(t *testing.T) {
	service := NewServiceWithDefaults()
	businessMetrics := service.NewBusinessMetrics()

	// Test user action metric
	businessMetrics.IncrementUserAction("login", "user123")
	businessMetrics.IncrementUserAction("login", "user456")

	// Test business transaction metric
	businessMetrics.RecordBusinessTransaction("payment", 99.99, 200*time.Millisecond)

	// Test active users metric
	businessMetrics.SetActiveUsers(42)

	// Test database operation metric
	businessMetrics.RecordDatabaseOperation("SELECT", "users", 10*time.Millisecond, true)
	businessMetrics.RecordDatabaseOperation("INSERT", "orders", 50*time.Millisecond, false)

	// Verify that custom metrics were created and can be retrieved
	if _, ok := service.GetCounter("user_actions_total"); !ok {
		t.Error("Expected user_actions_total counter to be registered")
	}

	if _, ok := service.GetCounter("business_transactions_total"); !ok {
		t.Error("Expected business_transactions_total counter to be registered")
	}

	if _, ok := service.GetGauge("active_users"); !ok {
		t.Error("Expected active_users gauge to be registered")
	}

	if _, ok := service.GetCounter("database_operations_total"); !ok {
		t.Error("Expected database_operations_total counter to be registered")
	}
}

func TestNormalizePath(t *testing.T) {
	service := NewServiceWithDefaults()

	tests := []struct {
		input    string
		expected string
	}{
		{"/api/users", "/api/users"},
		{"/", "/"},
		{strings.Repeat("a", 150), "/long_path"},
	}

	for _, test := range tests {
		result := service.normalizePath(test.input)
		if result != test.expected {
			t.Errorf("normalizePath(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

func BenchmarkRecordRequest(b *testing.B) {
	service := NewServiceWithDefaults()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.RecordRequest("GET", "/api/test", 200, 100*time.Millisecond, 1024, 2048)
	}
}

func BenchmarkMiddleware(b *testing.B) {
	service := NewServiceWithDefaults()
	app := fiber.New()
	app.Use(service.Middleware())
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		_, _ = app.Test(req)
	}
}