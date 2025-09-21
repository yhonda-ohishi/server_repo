# Metrics Integration Guide

This guide shows how to integrate the metrics package into your existing Fiber application.

## Quick Integration

### 1. Basic Setup

Add to your main server file (e.g., `cmd/server/main.go`):

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

    // Add metrics middleware (should be one of the first middleware)
    app.Use(metricsService.Middleware())

    // Your existing routes...

    // Expose metrics endpoint
    app.Get("/metrics", metricsService.Handler())

    app.Listen(":8080")
}
```

### 2. Custom Configuration

For production use, configure the metrics service:

```go
config := metrics.Config{
    Namespace: "myapp",           // Your app name
    Subsystem: "api",             // Subsystem (e.g., api, worker, etc.)
    DurationBuckets: []float64{   // Response time buckets (seconds)
        0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0,
    },
    SizeBuckets: []float64{       // Request/response size buckets (bytes)
        100, 1000, 10000, 100000, 1000000,
    },
    MaxPathCardinality: 100,      // Limit unique paths to control memory
}

metricsService := metrics.NewService(config)
```

### 3. Middleware Configuration

Control which paths are monitored:

```go
middlewareConfig := metrics.MiddlewareConfig{
    Service:   metricsService,
    SkipPaths: []string{"/health", "/metrics", "/favicon.ico"},
    PathNormalizer: func(path string) string {
        // Normalize paths to control cardinality
        // Example: /api/users/123 -> /api/users/:id
        return normalizePath(path)
    },
}

app.Use(metrics.MiddlewareWithConfig(middlewareConfig))
```

## Integration with Existing Components

### Health Check Integration

If you have an existing health check system:

```go
// In your health check handler
func healthHandler(c *fiber.Ctx) error {
    // Your existing health checks...

    // Optionally record health check metrics
    if businessMetrics != nil {
        businessMetrics.IncrementUserAction("health_check", "system")
    }

    return c.JSON(fiber.Map{"status": "healthy"})
}
```

### Logger Integration

Combine with your existing logger middleware:

```go
app.Use(logger.RequestLogger())  // Your existing logger
app.Use(metricsService.Middleware())  // Add metrics after logger
```

### Database Integration

Record database operation metrics:

```go
func getUserFromDB(userID string) (*User, error) {
    start := time.Now()

    // Your database operation
    user, err := db.GetUser(userID)

    // Record metrics
    success := err == nil
    businessMetrics.RecordDatabaseOperation("SELECT", "users", time.Since(start), success)

    return user, err
}
```

### gRPC Integration

For gRPC services, record metrics in interceptors:

```go
func metricsUnaryInterceptor(metricsService *metrics.Service) grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
        start := time.Now()

        resp, err := handler(ctx, req)

        // Record gRPC metrics
        status := "success"
        if err != nil {
            status = "error"
        }

        // Use custom metrics for gRPC
        if counter, ok := metricsService.GetCounter("grpc_requests_total"); ok {
            counter.WithLabelValues(info.FullMethod, status).Inc()
        }

        if histogram, ok := metricsService.GetHistogram("grpc_request_duration_seconds"); ok {
            histogram.WithLabelValues(info.FullMethod).Observe(time.Since(start).Seconds())
        }

        return resp, err
    }
}
```

## Business Metrics Examples

### User Actions

```go
// In your user handlers
func loginHandler(c *fiber.Ctx) error {
    userID := extractUserID(c)

    // Your login logic...

    // Record user action
    businessMetrics.IncrementUserAction("login", userID)

    return c.JSON(fiber.Map{"status": "logged in"})
}
```

### Payment Processing

```go
func processPayment(amount float64) error {
    start := time.Now()

    // Your payment processing logic...

    // Record business transaction
    businessMetrics.RecordBusinessTransaction("payment", amount, time.Since(start))

    return nil
}
```

### Active Users

```go
// Update periodically (e.g., every minute)
func updateActiveUserCount() {
    count := getActiveUserCount() // Your logic
    businessMetrics.SetActiveUsers(float64(count))
}
```

## Monitoring Integration

### Prometheus Configuration

Add to your `prometheus.yml`:

```yaml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'myapp'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
    scrape_interval: 15s
    scrape_timeout: 5s
```

### Grafana Dashboard

Key metrics to monitor:

```promql
# Request Rate (requests per second)
rate(myapp_api_requests_total[5m])

# Error Rate (percentage)
(
  rate(myapp_api_requests_total{status=~"5.."}[5m]) /
  rate(myapp_api_requests_total[5m])
) * 100

# Response Time (95th percentile)
histogram_quantile(0.95,
  rate(myapp_api_request_duration_seconds_bucket[5m])
)

# Request Size (95th percentile)
histogram_quantile(0.95,
  rate(myapp_api_request_size_bytes_bucket[5m])
)

# Active Users
myapp_api_active_users

# Database Operation Rate
rate(myapp_api_database_operations_total[5m])

# Business Transaction Rate
rate(myapp_api_business_transactions_total[5m])
```

### Alerting Rules

Example Prometheus alerting rules:

```yaml
groups:
  - name: myapp.rules
    rules:
      - alert: HighErrorRate
        expr: |
          (
            rate(myapp_api_requests_total{status=~"5.."}[5m]) /
            rate(myapp_api_requests_total[5m])
          ) * 100 > 5
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "High error rate detected"

      - alert: HighResponseTime
        expr: |
          histogram_quantile(0.95,
            rate(myapp_api_request_duration_seconds_bucket[5m])
          ) > 1.0
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High response time detected"

      - alert: DatabaseErrors
        expr: |
          rate(myapp_api_database_operations_total{status="error"}[5m]) > 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Database errors detected"
```

## Performance Considerations

### Memory Usage

- Each unique label combination creates a new time series
- Use path normalization to limit cardinality
- Monitor memory usage with high-cardinality metrics

### CPU Overhead

- Metrics collection adds ~1-2µs per request
- Histogram observations are more expensive than counter increments
- Consider sampling for very high-frequency operations

### Storage

- Prometheus stores all time series in memory
- Plan for ~1-8 bytes per sample per time series
- Use recording rules for frequently queried aggregations

## Best Practices

### 1. Metric Naming

Follow Prometheus naming conventions:

- Use snake_case: `http_requests_total`
- Include units: `duration_seconds`, `size_bytes`
- Use consistent prefixes: `myapp_api_*`

### 2. Label Usage

- Keep label cardinality low (< 1000 unique combinations per metric)
- Use labels for dimensions you want to aggregate/filter by
- Avoid user IDs or other high-cardinality data in labels

### 3. Path Normalization

```go
func normalizePath(path string) string {
    // Replace UUIDs
    re := regexp.MustCompile(`/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)
    path = re.ReplaceAllString(path, "/:uuid")

    // Replace numeric IDs
    re = regexp.MustCompile(`/\d+`)
    path = re.ReplaceAllString(path, "/:id")

    // Limit path length
    if len(path) > 100 {
        return "/long_path"
    }

    return path
}
```

### 4. Testing

Write tests for your metrics:

```go
func TestMetricsRecording(t *testing.T) {
    metricsService := metrics.NewServiceWithDefaults()

    // Your test logic...

    // Verify metrics were recorded
    expected := `
        # HELP myapp_api_requests_total Total number of requests
        # TYPE myapp_api_requests_total counter
        myapp_api_requests_total{method="GET",path="/test",status="200"} 1
    `

    err := testutil.GatherAndCompare(metricsService.registry,
        strings.NewReader(expected), "myapp_api_requests_total")
    require.NoError(t, err)
}
```

## Troubleshooting

### Common Issues

1. **High memory usage**: Check metric cardinality with `/metrics` endpoint
2. **Missing metrics**: Ensure middleware is properly configured
3. **Incorrect values**: Verify path normalization and label usage

### Debug Commands

```bash
# Check metric cardinality
curl http://localhost:8080/metrics | grep -c "^myapp_"

# Monitor memory usage
curl http://localhost:8080/metrics | grep "go_memstats_alloc_bytes"

# Check for high-cardinality metrics
curl http://localhost:8080/metrics | grep "myapp_api_requests_total" | wc -l
```