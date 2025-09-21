package gateway

import (
	"context"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cache"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/monitor"
	"github.com/gofiber/fiber/v2/middleware/pprof"
	"github.com/gofiber/fiber/v2/utils"
	"github.com/yhonda-ohishi/db-handler-server/internal/config"
)

// PerformanceConfig holds performance optimization settings
type PerformanceConfig struct {
	EnableCompression bool
	EnableCaching     bool
	EnableRateLimit   bool
	EnableProfiling   bool
	EnableMonitoring  bool

	// Cache settings
	CacheDuration    time.Duration
	CacheMaxSize     int

	// Rate limiting
	RateLimit        int
	RateLimitWindow  time.Duration

	// Connection pooling
	MaxConnections   int
	KeepAlive        bool
	KeepAliveDuration time.Duration
}

// DefaultPerformanceConfig returns optimized performance settings
func DefaultPerformanceConfig() *PerformanceConfig {
	return &PerformanceConfig{
		EnableCompression: true,
		EnableCaching:     true,
		EnableRateLimit:   true,
		EnableProfiling:   false, // Enable only in debug mode
		EnableMonitoring:  true,

		CacheDuration:     5 * time.Minute,
		CacheMaxSize:      1000,

		RateLimit:         1000, // requests per minute
		RateLimitWindow:   time.Minute,

		MaxConnections:    10000,
		KeepAlive:        true,
		KeepAliveDuration: 30 * time.Second,
	}
}

// OptimizedGateway provides a performance-optimized gateway implementation
type OptimizedGateway struct {
	*SimpleGateway
	perfConfig    *PerformanceConfig
	connectionPool *ConnectionPool
	responseCache  *ResponseCache
}

// ConnectionPool manages connection reuse and pooling
type ConnectionPool struct {
	mu          sync.RWMutex
	connections map[string]*ConnectionEntry
	maxSize     int
	cleanupTicker *time.Ticker
}

type ConnectionEntry struct {
	lastUsed time.Time
	inUse    bool
	data     interface{}
}

// ResponseCache provides intelligent response caching
type ResponseCache struct {
	mu       sync.RWMutex
	entries  map[string]*CacheEntry
	maxSize  int
	duration time.Duration
}

type CacheEntry struct {
	data      []byte
	timestamp time.Time
	hits      int64
}

// NewOptimizedGateway creates a performance-optimized gateway
func NewOptimizedGateway(cfg *config.Config, perfConfig *PerformanceConfig) *OptimizedGateway {
	// Create base gateway with optimized fiber config
	app := fiber.New(fiber.Config{
		AppName:               "ETC Meisai Gateway (Optimized)",
		Prefork:               false, // Set to true for maximum performance in production
		CaseSensitive:         false,
		StrictRouting:         false,
		ServerHeader:          "Gateway/1.0",
		DisableKeepalive:     !perfConfig.KeepAlive,
		DisableDefaultDate:   true,
		DisableDefaultContentType: true,
		DisableHeaderNormalizing: true,
		ReduceMemoryUsage:    true,

		// Optimized timeouts
		ReadTimeout:          30 * time.Second,
		WriteTimeout:         30 * time.Second,
		IdleTimeout:          perfConfig.KeepAliveDuration,

		// Connection limits
		Concurrency:          perfConfig.MaxConnections,

		// Error handling
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"error": err.Error(),
				"code":  code,
			})
		},

		// JSON optimization disabled due to API changes
		// JSONEncoder: utils.UnsafeString,
		// JSONDecoder: utils.UnsafeBytes,
	})

	baseGateway := &SimpleGateway{
		config: cfg,
		app:    app,
	}

	optimized := &OptimizedGateway{
		SimpleGateway: baseGateway,
		perfConfig:    perfConfig,
		connectionPool: NewConnectionPool(perfConfig.MaxConnections),
		responseCache:  NewResponseCache(perfConfig.CacheMaxSize, perfConfig.CacheDuration),
	}

	optimized.setupPerformanceMiddleware()

	return optimized
}

// setupPerformanceMiddleware configures performance-oriented middleware
func (g *OptimizedGateway) setupPerformanceMiddleware() {
	// Recovery middleware (keep first)
	g.app.Use(func(c *fiber.Ctx) error {
		defer func() {
			if r := recover(); r != nil {
				c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Internal server error",
				})
			}
		}()
		return c.Next()
	})

	// Compression middleware
	if g.perfConfig.EnableCompression {
		g.app.Use(compress.New(compress.Config{
			Level: compress.LevelBestSpeed, // Favor speed over compression ratio
		}))
	}

	// Rate limiting middleware
	if g.perfConfig.EnableRateLimit {
		g.app.Use(limiter.New(limiter.Config{
			Max:        g.perfConfig.RateLimit,
			Expiration: g.perfConfig.RateLimitWindow,
			KeyGenerator: func(c *fiber.Ctx) string {
				return c.IP()
			},
			LimitReached: func(c *fiber.Ctx) error {
				return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
					"error": "Rate limit exceeded",
				})
			},
		}))
	}

	// Response caching middleware
	if g.perfConfig.EnableCaching {
		g.app.Use(cache.New(cache.Config{
			Expiration:   g.perfConfig.CacheDuration,
			CacheControl: true,
			KeyGenerator: func(c *fiber.Ctx) string {
				return utils.CopyString(c.OriginalURL())
			},
			// Only cache GET requests
			Next: func(c *fiber.Ctx) bool {
				return c.Method() != fiber.MethodGet
			},
		}))
	}

	// Performance monitoring
	if g.perfConfig.EnableMonitoring {
		g.app.Get("/debug/monitor", monitor.New(monitor.Config{
			Title: "gRPC Gateway Performance Monitor",
		}))
	}

	// Profiling endpoints (debug mode only)
	if g.perfConfig.EnableProfiling {
		g.app.Use(pprof.New())
	}

	// Request ID and timing middleware
	g.app.Use(func(c *fiber.Ctx) error {
		start := time.Now()
		requestID := utils.UUID()
		c.Set("X-Request-ID", requestID)
		c.Locals("requestID", requestID)
		c.Locals("startTime", start)

		err := c.Next()

		duration := time.Since(start)
		c.Set("X-Response-Time", duration.String())

		return err
	})
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(maxSize int) *ConnectionPool {
	pool := &ConnectionPool{
		connections: make(map[string]*ConnectionEntry),
		maxSize:     maxSize,
		cleanupTicker: time.NewTicker(5 * time.Minute),
	}

	// Start cleanup goroutine
	go pool.cleanup()

	return pool
}

// Get retrieves a connection from the pool
func (p *ConnectionPool) Get(key string) (interface{}, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if entry, exists := p.connections[key]; exists && !entry.inUse {
		entry.inUse = true
		entry.lastUsed = time.Now()
		return entry.data, true
	}

	return nil, false
}

// Put stores a connection in the pool
func (p *ConnectionPool) Put(key string, data interface{}) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.connections) >= p.maxSize {
		// Remove oldest unused connection
		p.evictOldest()
	}

	p.connections[key] = &ConnectionEntry{
		data:     data,
		lastUsed: time.Now(),
		inUse:    false,
	}
}

// Release marks a connection as available
func (p *ConnectionPool) Release(key string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if entry, exists := p.connections[key]; exists {
		entry.inUse = false
		entry.lastUsed = time.Now()
	}
}

// cleanup removes stale connections
func (p *ConnectionPool) cleanup() {
	for range p.cleanupTicker.C {
		p.mu.Lock()
		now := time.Now()
		for key, entry := range p.connections {
			if !entry.inUse && now.Sub(entry.lastUsed) > 10*time.Minute {
				delete(p.connections, key)
			}
		}
		p.mu.Unlock()
	}
}

// evictOldest removes the oldest unused connection
func (p *ConnectionPool) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range p.connections {
		if !entry.inUse && (oldestKey == "" || entry.lastUsed.Before(oldestTime)) {
			oldestKey = key
			oldestTime = entry.lastUsed
		}
	}

	if oldestKey != "" {
		delete(p.connections, oldestKey)
	}
}

// NewResponseCache creates a new response cache
func NewResponseCache(maxSize int, duration time.Duration) *ResponseCache {
	return &ResponseCache{
		entries:  make(map[string]*CacheEntry),
		maxSize:  maxSize,
		duration: duration,
	}
}

// Get retrieves a cached response
func (c *ResponseCache) Get(key string) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if entry, exists := c.entries[key]; exists {
		if time.Since(entry.timestamp) < c.duration {
			entry.hits++
			return entry.data, true
		}
		// Entry expired, will be cleaned up later
	}

	return nil, false
}

// Set stores a response in cache
func (c *ResponseCache) Set(key string, data []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.entries) >= c.maxSize {
		c.evictLeastUsed()
	}

	c.entries[key] = &CacheEntry{
		data:      data,
		timestamp: time.Now(),
		hits:      0,
	}
}

// evictLeastUsed removes the least frequently used entry
func (c *ResponseCache) evictLeastUsed() {
	var leastUsedKey string
	var leastHits int64 = -1

	for key, entry := range c.entries {
		if leastHits == -1 || entry.hits < leastHits {
			leastUsedKey = key
			leastHits = entry.hits
		}
	}

	if leastUsedKey != "" {
		delete(c.entries, leastUsedKey)
	}
}

// GetPerformanceStats returns current performance statistics
func (g *OptimizedGateway) GetPerformanceStats() map[string]interface{} {
	g.connectionPool.mu.RLock()
	poolSize := len(g.connectionPool.connections)
	g.connectionPool.mu.RUnlock()

	g.responseCache.mu.RLock()
	cacheSize := len(g.responseCache.entries)
	totalHits := int64(0)
	for _, entry := range g.responseCache.entries {
		totalHits += entry.hits
	}
	g.responseCache.mu.RUnlock()

	return map[string]interface{}{
		"connection_pool_size": poolSize,
		"cache_size":          cacheSize,
		"cache_hits":          totalHits,
		"compression_enabled": g.perfConfig.EnableCompression,
		"rate_limit_enabled":  g.perfConfig.EnableRateLimit,
		"max_connections":     g.perfConfig.MaxConnections,
	}
}

// Shutdown gracefully shuts down the optimized gateway
func (g *OptimizedGateway) Shutdown(ctx context.Context) error {
	if g.connectionPool.cleanupTicker != nil {
		g.connectionPool.cleanupTicker.Stop()
	}

	return g.SimpleGateway.app.Shutdown()
}