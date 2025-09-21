package gateway

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// BenchmarkResult holds the results of a performance benchmark
type BenchmarkResult struct {
	TotalRequests      int64         `json:"total_requests"`
	SuccessfulRequests int64         `json:"successful_requests"`
	FailedRequests     int64         `json:"failed_requests"`
	Duration           time.Duration `json:"duration"`
	RequestsPerSecond  float64       `json:"requests_per_second"`
	AverageLatency     time.Duration `json:"average_latency"`
	MinLatency         time.Duration `json:"min_latency"`
	MaxLatency         time.Duration `json:"max_latency"`
	P50Latency         time.Duration `json:"p50_latency"`
	P95Latency         time.Duration `json:"p95_latency"`
	P99Latency         time.Duration `json:"p99_latency"`
	ErrorRate          float64       `json:"error_rate"`
	MemoryUsage        int64         `json:"memory_usage_bytes"`
}

// BenchmarkConfig holds configuration for benchmarking
type BenchmarkConfig struct {
	Concurrency   int           `json:"concurrency"`
	Duration      time.Duration `json:"duration"`
	RequestCount  int           `json:"request_count"`
	Endpoint      string        `json:"endpoint"`
	Method        string        `json:"method"`
	Payload       []byte        `json:"payload"`
	Headers       map[string]string `json:"headers"`
	WarmupTime    time.Duration `json:"warmup_time"`
}

// LatencyTracker tracks request latencies for statistical analysis
type LatencyTracker struct {
	mu        sync.Mutex
	latencies []time.Duration
}

func NewLatencyTracker() *LatencyTracker {
	return &LatencyTracker{
		latencies: make([]time.Duration, 0, 10000),
	}
}

func (lt *LatencyTracker) Record(latency time.Duration) {
	lt.mu.Lock()
	defer lt.mu.Unlock()
	lt.latencies = append(lt.latencies, latency)
}

func (lt *LatencyTracker) GetStats() (min, max, avg, p50, p95, p99 time.Duration) {
	lt.mu.Lock()
	defer lt.mu.Unlock()

	if len(lt.latencies) == 0 {
		return 0, 0, 0, 0, 0, 0
	}

	// Sort latencies for percentile calculations
	sorted := make([]time.Duration, len(lt.latencies))
	copy(sorted, lt.latencies)

	// Simple bubble sort for small datasets (optimize for production)
	for i := 0; i < len(sorted); i++ {
		for j := 0; j < len(sorted)-1-i; j++ {
			if sorted[j] > sorted[j+1] {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	min = sorted[0]
	max = sorted[len(sorted)-1]

	// Calculate average
	var total time.Duration
	for _, lat := range sorted {
		total += lat
	}
	avg = total / time.Duration(len(sorted))

	// Calculate percentiles
	p50 = sorted[len(sorted)*50/100]
	p95 = sorted[len(sorted)*95/100]
	p99 = sorted[len(sorted)*99/100]

	return min, max, avg, p50, p95, p99
}

// PerformanceBenchmark runs performance tests on the gateway
type PerformanceBenchmark struct {
	gateway    Gateway
	config     *BenchmarkConfig
	results    *BenchmarkResult
	latencyTracker *LatencyTracker
}

// Gateway interface for benchmarking
type Gateway interface {
	GetHTTPHandler() interface{}
	GetPerformanceStats() map[string]interface{}
}

// NewPerformanceBenchmark creates a new benchmark instance
func NewPerformanceBenchmark(gateway Gateway, config *BenchmarkConfig) *PerformanceBenchmark {
	return &PerformanceBenchmark{
		gateway:        gateway,
		config:         config,
		latencyTracker: NewLatencyTracker(),
		results: &BenchmarkResult{},
	}
}

// Run executes the performance benchmark
func (pb *PerformanceBenchmark) Run(ctx context.Context) (*BenchmarkResult, error) {
	// Warmup phase
	if pb.config.WarmupTime > 0 {
		fmt.Printf("Warming up for %v...\n", pb.config.WarmupTime)
		time.Sleep(pb.config.WarmupTime)
	}

	fmt.Printf("Starting benchmark: %d concurrent workers for %v\n",
		pb.config.Concurrency, pb.config.Duration)

	var wg sync.WaitGroup
	var successCount, errorCount int64
	startTime := time.Now()

	// Create context with timeout
	benchCtx, cancel := context.WithTimeout(ctx, pb.config.Duration)
	defer cancel()

	// Start concurrent workers
	for i := 0; i < pb.config.Concurrency; i++ {
		wg.Add(1)
		go pb.worker(benchCtx, &wg, &successCount, &errorCount)
	}

	// Wait for all workers to complete
	wg.Wait()
	endTime := time.Now()

	// Calculate results
	duration := endTime.Sub(startTime)
	totalRequests := successCount + errorCount

	pb.results = &BenchmarkResult{
		TotalRequests:      totalRequests,
		SuccessfulRequests: successCount,
		FailedRequests:     errorCount,
		Duration:           duration,
		RequestsPerSecond:  float64(totalRequests) / duration.Seconds(),
		ErrorRate:          float64(errorCount) / float64(totalRequests) * 100,
	}

	// Calculate latency statistics
	min, max, avg, p50, p95, p99 := pb.latencyTracker.GetStats()
	pb.results.MinLatency = min
	pb.results.MaxLatency = max
	pb.results.AverageLatency = avg
	pb.results.P50Latency = p50
	pb.results.P95Latency = p95
	pb.results.P99Latency = p99

	return pb.results, nil
}

// worker performs the actual benchmark requests
func (pb *PerformanceBenchmark) worker(ctx context.Context, wg *sync.WaitGroup, successCount, errorCount *int64) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			start := time.Now()

			// Simulate request based on configuration
			success := pb.makeRequest()

			latency := time.Since(start)
			pb.latencyTracker.Record(latency)

			if success {
				atomic.AddInt64(successCount, 1)
			} else {
				atomic.AddInt64(errorCount, 1)
			}
		}
	}
}

// makeRequest simulates making a request to the gateway
func (pb *PerformanceBenchmark) makeRequest() bool {
	// This would normally make an actual HTTP request
	// For now, simulate request processing time
	time.Sleep(time.Microsecond * time.Duration(100 + (time.Now().UnixNano() % 1000)))

	// Simulate 95% success rate
	return time.Now().UnixNano() % 100 < 95
}

// GetResults returns the current benchmark results
func (pb *PerformanceBenchmark) GetResults() *BenchmarkResult {
	return pb.results
}

// PrintResults prints a formatted report of the benchmark results
func (pb *PerformanceBenchmark) PrintResults() {
	fmt.Println("\n=== Performance Benchmark Results ===")
	fmt.Printf("Duration: %v\n", pb.results.Duration)
	fmt.Printf("Total Requests: %d\n", pb.results.TotalRequests)
	fmt.Printf("Successful Requests: %d\n", pb.results.SuccessfulRequests)
	fmt.Printf("Failed Requests: %d\n", pb.results.FailedRequests)
	fmt.Printf("Requests/sec: %.2f\n", pb.results.RequestsPerSecond)
	fmt.Printf("Error Rate: %.2f%%\n", pb.results.ErrorRate)
	fmt.Println("\n--- Latency Statistics ---")
	fmt.Printf("Average: %v\n", pb.results.AverageLatency)
	fmt.Printf("Min: %v\n", pb.results.MinLatency)
	fmt.Printf("Max: %v\n", pb.results.MaxLatency)
	fmt.Printf("P50: %v\n", pb.results.P50Latency)
	fmt.Printf("P95: %v\n", pb.results.P95Latency)
	fmt.Printf("P99: %v\n", pb.results.P99Latency)
	fmt.Println("=====================================")
}

// CompareResults compares two benchmark results and shows the difference
func CompareResults(baseline, current *BenchmarkResult) {
	fmt.Println("\n=== Performance Comparison ===")

	rpsImprovement := ((current.RequestsPerSecond - baseline.RequestsPerSecond) / baseline.RequestsPerSecond) * 100
	latencyImprovement := ((baseline.AverageLatency.Nanoseconds() - current.AverageLatency.Nanoseconds()) / baseline.AverageLatency.Nanoseconds()) * 100

	fmt.Printf("Requests/sec: %.2f -> %.2f (%.2f%% change)\n",
		baseline.RequestsPerSecond, current.RequestsPerSecond, rpsImprovement)
	fmt.Printf("Average Latency: %v -> %v (%.2f%% improvement)\n",
		baseline.AverageLatency, current.AverageLatency, float64(latencyImprovement))
	fmt.Printf("P95 Latency: %v -> %v\n",
		baseline.P95Latency, current.P95Latency)
	fmt.Printf("Error Rate: %.2f%% -> %.2f%%\n",
		baseline.ErrorRate, current.ErrorRate)

	if rpsImprovement > 0 {
		fmt.Printf("✅ Performance improved by %.2f%%\n", rpsImprovement)
	} else {
		fmt.Printf("⚠️  Performance decreased by %.2f%%\n", -rpsImprovement)
	}
	fmt.Println("===============================")
}

// DefaultBenchmarkConfigs returns common benchmark configurations
func DefaultBenchmarkConfigs() map[string]*BenchmarkConfig {
	return map[string]*BenchmarkConfig{
		"light_load": {
			Concurrency:  10,
			Duration:     30 * time.Second,
			Endpoint:     "/api/v1/users",
			Method:       "GET",
			WarmupTime:   5 * time.Second,
		},
		"medium_load": {
			Concurrency:  50,
			Duration:     60 * time.Second,
			Endpoint:     "/api/v1/users",
			Method:       "GET",
			WarmupTime:   10 * time.Second,
		},
		"heavy_load": {
			Concurrency:  200,
			Duration:     120 * time.Second,
			Endpoint:     "/api/v1/users",
			Method:       "GET",
			WarmupTime:   15 * time.Second,
		},
		"spike_test": {
			Concurrency:  500,
			Duration:     30 * time.Second,
			Endpoint:     "/api/v1/users",
			Method:       "GET",
			WarmupTime:   5 * time.Second,
		},
	}
}