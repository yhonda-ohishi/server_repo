package metrics

import (
	"bytes"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
)

// Service provides metrics collection and reporting functionality
type Service struct {
	registry *prometheus.Registry

	// HTTP metrics
	requestCount      *prometheus.CounterVec
	requestDuration   *prometheus.HistogramVec
	requestSize       *prometheus.HistogramVec
	responseSize      *prometheus.HistogramVec

	// Custom metrics storage
	customMetrics sync.Map

	// Configuration
	config Config
}

// Config holds configuration for metrics service
type Config struct {
	// Namespace for metrics (default: "http")
	Namespace string
	// Subsystem for metrics (default: "server")
	Subsystem string
	// Buckets for duration histogram (in seconds)
	DurationBuckets []float64
	// Buckets for size histogram (in bytes)
	SizeBuckets []float64
	// Labels to exclude from metrics (for cardinality control)
	ExcludeLabels []string
	// MaxPathCardinality limits the number of unique paths tracked
	MaxPathCardinality int
}

// DefaultConfig returns default metrics configuration
func DefaultConfig() Config {
	return Config{
		Namespace: "http",
		Subsystem: "server",
		DurationBuckets: []float64{
			0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0,
		},
		SizeBuckets: []float64{
			100, 1000, 10000, 100000, 1000000, 10000000,
		},
		ExcludeLabels:      []string{},
		MaxPathCardinality: 100,
	}
}

// NewService creates a new metrics service
func NewService(config Config) *Service {
	if config.Namespace == "" {
		config = DefaultConfig()
	}

	registry := prometheus.NewRegistry()

	// Create HTTP metrics
	requestCount := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "requests_total",
			Help:      "Total number of HTTP requests by method, path, and status code",
		},
		[]string{"method", "path", "status"},
	)

	requestDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "request_duration_seconds",
			Help:      "HTTP request duration in seconds",
			Buckets:   config.DurationBuckets,
		},
		[]string{"method", "path", "status"},
	)

	requestSize := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "request_size_bytes",
			Help:      "HTTP request size in bytes",
			Buckets:   config.SizeBuckets,
		},
		[]string{"method", "path"},
	)

	responseSize := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "response_size_bytes",
			Help:      "HTTP response size in bytes",
			Buckets:   config.SizeBuckets,
		},
		[]string{"method", "path", "status"},
	)

	// Register metrics
	registry.MustRegister(requestCount)
	registry.MustRegister(requestDuration)
	registry.MustRegister(requestSize)
	registry.MustRegister(responseSize)

	// Register Go runtime metrics
	registry.MustRegister(prometheus.NewGoCollector())
	registry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))

	return &Service{
		registry:        registry,
		requestCount:    requestCount,
		requestDuration: requestDuration,
		requestSize:     requestSize,
		responseSize:    responseSize,
		config:          config,
	}
}

// NewServiceWithDefaults creates a new metrics service with default configuration
func NewServiceWithDefaults() *Service {
	return NewService(DefaultConfig())
}

// RecordRequest records HTTP request metrics
func (s *Service) RecordRequest(method, path string, statusCode int, duration time.Duration, requestSize, responseSize int64) {
	status := strconv.Itoa(statusCode)

	// Normalize path to control cardinality
	normalizedPath := s.normalizePath(path)

	// Record metrics
	s.requestCount.WithLabelValues(method, normalizedPath, status).Inc()
	s.requestDuration.WithLabelValues(method, normalizedPath, status).Observe(duration.Seconds())
	s.requestSize.WithLabelValues(method, normalizedPath).Observe(float64(requestSize))
	s.responseSize.WithLabelValues(method, normalizedPath, status).Observe(float64(responseSize))
}

// normalizePath normalizes URL paths to control metric cardinality
func (s *Service) normalizePath(path string) string {
	// Simple normalization - replace IDs with placeholders
	// In a real implementation, you might want more sophisticated path normalization
	if len(path) > 100 {
		return "/long_path"
	}

	// You can add more sophisticated path normalization here
	// For example, replacing UUIDs, numeric IDs, etc.
	return path
}

// RegisterCounter registers a custom counter metric
func (s *Service) RegisterCounter(name, help string, labels []string) *prometheus.CounterVec {
	counter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: s.config.Namespace,
			Subsystem: s.config.Subsystem,
			Name:      name,
			Help:      help,
		},
		labels,
	)

	s.registry.MustRegister(counter)
	s.customMetrics.Store(name, counter)
	return counter
}

// RegisterGauge registers a custom gauge metric
func (s *Service) RegisterGauge(name, help string, labels []string) *prometheus.GaugeVec {
	gauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: s.config.Namespace,
			Subsystem: s.config.Subsystem,
			Name:      name,
			Help:      help,
		},
		labels,
	)

	s.registry.MustRegister(gauge)
	s.customMetrics.Store(name, gauge)
	return gauge
}

// RegisterHistogram registers a custom histogram metric
func (s *Service) RegisterHistogram(name, help string, labels []string, buckets []float64) *prometheus.HistogramVec {
	if buckets == nil {
		buckets = prometheus.DefBuckets
	}

	histogram := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: s.config.Namespace,
			Subsystem: s.config.Subsystem,
			Name:      name,
			Help:      help,
			Buckets:   buckets,
		},
		labels,
	)

	s.registry.MustRegister(histogram)
	s.customMetrics.Store(name, histogram)
	return histogram
}

// RegisterSummary registers a custom summary metric
func (s *Service) RegisterSummary(name, help string, labels []string, objectives map[float64]float64) *prometheus.SummaryVec {
	if objectives == nil {
		objectives = map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001}
	}

	summary := prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  s.config.Namespace,
			Subsystem:  s.config.Subsystem,
			Name:       name,
			Help:       help,
			Objectives: objectives,
		},
		labels,
	)

	s.registry.MustRegister(summary)
	s.customMetrics.Store(name, summary)
	return summary
}

// GetCounter retrieves a registered counter metric
func (s *Service) GetCounter(name string) (*prometheus.CounterVec, bool) {
	if metric, ok := s.customMetrics.Load(name); ok {
		if counter, ok := metric.(*prometheus.CounterVec); ok {
			return counter, true
		}
	}
	return nil, false
}

// GetGauge retrieves a registered gauge metric
func (s *Service) GetGauge(name string) (*prometheus.GaugeVec, bool) {
	if metric, ok := s.customMetrics.Load(name); ok {
		if gauge, ok := metric.(*prometheus.GaugeVec); ok {
			return gauge, true
		}
	}
	return nil, false
}

// GetHistogram retrieves a registered histogram metric
func (s *Service) GetHistogram(name string) (*prometheus.HistogramVec, bool) {
	if metric, ok := s.customMetrics.Load(name); ok {
		if histogram, ok := metric.(*prometheus.HistogramVec); ok {
			return histogram, true
		}
	}
	return nil, false
}

// GetSummary retrieves a registered summary metric
func (s *Service) GetSummary(name string) (*prometheus.SummaryVec, bool) {
	if metric, ok := s.customMetrics.Load(name); ok {
		if summary, ok := metric.(*prometheus.SummaryVec); ok {
			return summary, true
		}
	}
	return nil, false
}

// Handler returns the Prometheus metrics handler for exposing metrics
func (s *Service) Handler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Gather metrics
		metricFamilies, err := s.registry.Gather()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Error gathering metrics")
		}

		// Create buffer for output
		buf := &bytes.Buffer{}

		// Use Prometheus exposition format
		encoder := expfmt.NewEncoder(buf, expfmt.FmtText)
		for _, mf := range metricFamilies {
			if err := encoder.Encode(mf); err != nil {
				return c.Status(fiber.StatusInternalServerError).SendString("Error encoding metrics")
			}
		}

		// Set content type and return metrics
		c.Set("Content-Type", string(expfmt.FmtText))
		return c.SendString(buf.String())
	}
}

// Helper function to format float values
func formatFloat(f float64) string {
	return fmt.Sprintf("%g", f)
}

// Helper function to format uint values
func formatUint(u uint64) string {
	return fmt.Sprintf("%d", u)
}

// MiddlewareConfig holds configuration for metrics middleware
type MiddlewareConfig struct {
	// Service is the metrics service instance
	Service *Service
	// SkipPaths defines paths to skip metrics collection
	SkipPaths []string
	// PathNormalizer is a function to normalize paths for metrics
	PathNormalizer func(string) string
}

// DefaultMiddlewareConfig returns default middleware configuration
func DefaultMiddlewareConfig(service *Service) MiddlewareConfig {
	return MiddlewareConfig{
		Service:   service,
		SkipPaths: []string{"/metrics", "/health"},
		PathNormalizer: func(path string) string {
			return path
		},
	}
}

// MiddlewareWithConfig returns Fiber middleware for metrics collection with custom config
func MiddlewareWithConfig(config MiddlewareConfig) fiber.Handler {
	skipPaths := make(map[string]bool)
	for _, path := range config.SkipPaths {
		skipPaths[path] = true
	}

	return func(c *fiber.Ctx) error {
		// Skip metrics collection for specified paths
		if skipPaths[c.Path()] {
			return c.Next()
		}

		start := time.Now()

		// Get request size
		requestSize := int64(len(c.Request().Body()))

		// Continue with request
		err := c.Next()

		// Calculate metrics
		duration := time.Since(start)
		statusCode := c.Response().StatusCode()
		responseSize := int64(len(c.Response().Body()))

		// Normalize path
		path := c.Path()
		if config.PathNormalizer != nil {
			path = config.PathNormalizer(path)
		}

		// Record metrics
		config.Service.RecordRequest(
			c.Method(),
			path,
			statusCode,
			duration,
			requestSize,
			responseSize,
		)

		return err
	}
}

// Middleware returns Fiber middleware for metrics collection with default config
func (s *Service) Middleware() fiber.Handler {
	return MiddlewareWithConfig(DefaultMiddlewareConfig(s))
}

// BusinessMetrics provides helper methods for common business metrics
type BusinessMetrics struct {
	service *Service
}

// NewBusinessMetrics creates a new business metrics helper
func (s *Service) NewBusinessMetrics() *BusinessMetrics {
	return &BusinessMetrics{service: s}
}

// IncrementUserAction increments a counter for user actions
func (bm *BusinessMetrics) IncrementUserAction(action, userID string) {
	counter, exists := bm.service.GetCounter("user_actions_total")
	if !exists {
		counter = bm.service.RegisterCounter(
			"user_actions_total",
			"Total number of user actions",
			[]string{"action", "user_id"},
		)
	}
	counter.WithLabelValues(action, userID).Inc()
}

// RecordBusinessTransaction records a business transaction metric
func (bm *BusinessMetrics) RecordBusinessTransaction(transactionType string, amount float64, duration time.Duration) {
	// Transaction count
	counter, exists := bm.service.GetCounter("business_transactions_total")
	if !exists {
		counter = bm.service.RegisterCounter(
			"business_transactions_total",
			"Total number of business transactions",
			[]string{"type"},
		)
	}
	counter.WithLabelValues(transactionType).Inc()

	// Transaction amount
	histogram, exists := bm.service.GetHistogram("business_transaction_amount")
	if !exists {
		histogram = bm.service.RegisterHistogram(
			"business_transaction_amount",
			"Business transaction amounts",
			[]string{"type"},
			[]float64{1, 10, 100, 1000, 10000, 100000},
		)
	}
	histogram.WithLabelValues(transactionType).Observe(amount)

	// Transaction duration
	durationHist, exists := bm.service.GetHistogram("business_transaction_duration_seconds")
	if !exists {
		durationHist = bm.service.RegisterHistogram(
			"business_transaction_duration_seconds",
			"Business transaction duration in seconds",
			[]string{"type"},
			nil, // Use default buckets
		)
	}
	durationHist.WithLabelValues(transactionType).Observe(duration.Seconds())
}

// SetActiveUsers sets the current number of active users
func (bm *BusinessMetrics) SetActiveUsers(count float64) {
	gauge, exists := bm.service.GetGauge("active_users")
	if !exists {
		gauge = bm.service.RegisterGauge(
			"active_users",
			"Current number of active users",
			[]string{},
		)
	}
	gauge.WithLabelValues().Set(count)
}

// RecordDatabaseOperation records database operation metrics
func (bm *BusinessMetrics) RecordDatabaseOperation(operation, table string, duration time.Duration, success bool) {
	// Database operation count
	counter, exists := bm.service.GetCounter("database_operations_total")
	if !exists {
		counter = bm.service.RegisterCounter(
			"database_operations_total",
			"Total number of database operations",
			[]string{"operation", "table", "status"},
		)
	}

	status := "success"
	if !success {
		status = "error"
	}
	counter.WithLabelValues(operation, table, status).Inc()

	// Database operation duration
	histogram, exists := bm.service.GetHistogram("database_operation_duration_seconds")
	if !exists {
		histogram = bm.service.RegisterHistogram(
			"database_operation_duration_seconds",
			"Database operation duration in seconds",
			[]string{"operation", "table"},
			[]float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0},
		)
	}
	histogram.WithLabelValues(operation, table).Observe(duration.Seconds())
}