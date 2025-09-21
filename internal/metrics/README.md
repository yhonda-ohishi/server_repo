# Metrics Package

This package provides comprehensive metrics collection and reporting functionality using Prometheus for HTTP services built with Fiber.

## Features

- **Standard HTTP Metrics**: Automatic collection of request count, duration, and size metrics
- **Custom Business Metrics**: Support for registering and using custom application-specific metrics
- **Fiber Middleware**: Automatic metrics collection with minimal setup
- **Prometheus Integration**: Full Prometheus compatibility with standard metric types
- **Low Cardinality**: Built-in path normalization to control metric cardinality
- **Business Metrics Helper**: Pre-built helpers for common business metrics

## Quick Start

### Basic Setup

```go
package main

import (
    "github.com/gofiber/fiber/v2"
    "github.com/yhonda-ohishi/db-handler-server/internal/metrics"
)

func main() {
    // Create metrics service
    metricsService := metrics.NewServiceWithDefaults()

    // Create Fiber app
    app := fiber.New()

    // Add metrics middleware
    app.Use(metricsService.Middleware())

    // Add your routes
    app.Get("/api/users", func(c *fiber.Ctx) error {
        return c.JSON(fiber.Map{"users": []string{"user1", "user2"}})
    })

    // Expose metrics endpoint
    app.Get("/metrics", metricsService.Handler())

    app.Listen(":8080")
}
```

### Custom Configuration

```go
config := metrics.Config{
    Namespace: "myapp",
    Subsystem: "api",
    DurationBuckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0},
    SizeBuckets: []float64{100, 1000, 10000, 100000},
    MaxPathCardinality: 50,
}

metricsService := metrics.NewService(config)
```

### Middleware Configuration

```go
middlewareConfig := metrics.MiddlewareConfig{
    Service:   metricsService,
    SkipPaths: []string{"/health", "/metrics"},
    PathNormalizer: func(path string) string {
        // Normalize /api/users/123 to /api/users/:id
        return normalizePath(path)
    },
}

app.Use(metrics.MiddlewareWithConfig(middlewareConfig))
```

## Standard HTTP Metrics

The package automatically collects the following HTTP metrics:

### Counter Metrics
- `http_server_requests_total`: Total number of HTTP requests
  - Labels: `method`, `path`, `status`

### Histogram Metrics
- `http_server_request_duration_seconds`: HTTP request duration
  - Labels: `method`, `path`, `status`
- `http_server_request_size_bytes`: HTTP request size
  - Labels: `method`, `path`
- `http_server_response_size_bytes`: HTTP response size
  - Labels: `method`, `path`, `status`

## Custom Metrics

### Registering Custom Metrics

```go
// Counter
userActions := metricsService.RegisterCounter(
    "user_actions_total",
    "Total number of user actions",
    []string{"action", "user_id"},
)

// Gauge
activeUsers := metricsService.RegisterGauge(
    "active_users",
    "Current number of active users",
    []string{},
)

// Histogram
apiLatency := metricsService.RegisterHistogram(
    "external_api_duration_seconds",
    "External API call duration",
    []string{"service", "endpoint"},
    []float64{0.1, 0.5, 1.0, 2.5, 5.0},
)

// Summary
processingTime := metricsService.RegisterSummary(
    "processing_duration_seconds",
    "Processing time summary",
    []string{"processor"},
    map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
)
```

### Using Custom Metrics

```go
// Increment counter
userActions.WithLabelValues("login", "user123").Inc()

// Set gauge value
activeUsers.WithLabelValues().Set(42)

// Record histogram observation
start := time.Now()
// ... do some work ...
apiLatency.WithLabelValues("payment", "/charge").Observe(time.Since(start).Seconds())

// Record summary observation
processingTime.WithLabelValues("image_processor").Observe(duration.Seconds())
```

### Retrieving Registered Metrics

```go
if counter, ok := metricsService.GetCounter("user_actions_total"); ok {
    counter.WithLabelValues("logout", "user456").Inc()
}
```

## Business Metrics Helper

The package includes a business metrics helper for common patterns:

```go
businessMetrics := metricsService.NewBusinessMetrics()

// User actions
businessMetrics.IncrementUserAction("login", "user123")

// Business transactions
businessMetrics.RecordBusinessTransaction("payment", 99.99, 200*time.Millisecond)

// Active users
businessMetrics.SetActiveUsers(42)

// Database operations
businessMetrics.RecordDatabaseOperation("SELECT", "users", 10*time.Millisecond, true)
```

## Path Normalization

To control metric cardinality, implement path normalization:

```go
func normalizePath(path string) string {
    // Replace UUIDs and numeric IDs with placeholders
    re := regexp.MustCompile(`/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)
    path = re.ReplaceAllString(path, "/:uuid")

    re = regexp.MustCompile(`/\d+`)
    path = re.ReplaceAllString(path, "/:id")

    return path
}
```

## Metrics Endpoint

The metrics endpoint exposes all collected metrics in Prometheus format:

```bash
curl http://localhost:8080/metrics
```

Example output:
```
# HELP http_server_requests_total Total number of HTTP requests by method, path, and status code
# TYPE http_server_requests_total counter
http_server_requests_total{method="GET",path="/api/users",status="200"} 1

# HELP http_server_request_duration_seconds HTTP request duration in seconds
# TYPE http_server_request_duration_seconds histogram
http_server_request_duration_seconds_bucket{method="GET",path="/api/users",status="200",le="0.001"} 0
http_server_request_duration_seconds_bucket{method="GET",path="/api/users",status="200",le="0.005"} 1
...
```

## Integration with Monitoring Systems

### Prometheus Configuration

Add this to your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'my-api'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
    scrape_interval: 15s
```

### Grafana Dashboard

Create dashboards using these metrics:

- **Request Rate**: `rate(http_server_requests_total[5m])`
- **Error Rate**: `rate(http_server_requests_total{status=~"5.."}[5m])`
- **Response Time**: `histogram_quantile(0.95, rate(http_server_request_duration_seconds_bucket[5m]))`
- **Request Size**: `histogram_quantile(0.95, rate(http_server_request_size_bytes_bucket[5m]))`

## Performance Considerations

- Metrics collection has minimal overhead (~1-2µs per request)
- Use path normalization to control cardinality
- Consider excluding high-frequency, low-value endpoints
- Monitor memory usage with high-cardinality labels

## Best Practices

1. **Limit Label Cardinality**: Keep the number of unique label combinations low
2. **Use Appropriate Metric Types**:
   - Counters for things that only increase
   - Gauges for things that can go up and down
   - Histograms for distributions of values
3. **Normalize Paths**: Replace IDs and UUIDs with placeholders
4. **Skip Monitoring Endpoints**: Exclude `/health` and `/metrics` from collection
5. **Use Business Metrics**: Track business-relevant metrics alongside technical ones

## Testing

Run the test suite:

```bash
go test ./internal/metrics/...
```

Run benchmarks:

```bash
go test -bench=. ./internal/metrics/...
```

## Examples

See `example.go` for comprehensive usage examples including:
- Custom metric registration
- Business metrics usage
- Path normalization
- External API monitoring
- Advanced metric patterns