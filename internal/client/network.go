package client

import (
	"context"
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

// NetworkClient manages network-based gRPC connections
type NetworkClient struct {
	address string
	conn    *grpc.ClientConn
	mu      sync.RWMutex
	opts    []grpc.DialOption
}

// NetworkClientConfig holds configuration for network client
type NetworkClientConfig struct {
	Address         string
	MaxRetries      int
	Timeout         time.Duration
	KeepAlive       time.Duration
	MaxMessageSize  int
	WithInsecure    bool
	BackoffMultiplier float64
}

// DefaultNetworkConfig returns default network client configuration
func DefaultNetworkConfig(address string) *NetworkClientConfig {
	return &NetworkClientConfig{
		Address:         address,
		MaxRetries:      3,
		Timeout:         10 * time.Second,
		KeepAlive:       30 * time.Second,
		MaxMessageSize:  4 * 1024 * 1024, // 4MB
		WithInsecure:    true,
		BackoffMultiplier: 1.5,
	}
}

// NewNetworkClient creates a new network-based gRPC client
func NewNetworkClient(config *NetworkClientConfig) *NetworkClient {
	opts := buildDialOptions(config)

	return &NetworkClient{
		address: config.Address,
		opts:    opts,
	}
}

// buildDialOptions constructs gRPC dial options from config
func buildDialOptions(config *NetworkClientConfig) []grpc.DialOption {
	opts := []grpc.DialOption{}

	// Security
	if config.WithInsecure {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	// Keepalive
	if config.KeepAlive > 0 {
		keepaliveParams := keepalive.ClientParameters{
			Time:                config.KeepAlive,
			Timeout:             config.Timeout,
			PermitWithoutStream: true,
		}
		opts = append(opts, grpc.WithKeepaliveParams(keepaliveParams))
	}

	// Backoff config for retries
	backoffConfig := backoff.Config{
		BaseDelay:  1.0 * time.Second,
		Multiplier: config.BackoffMultiplier,
		Jitter:     0.2,
		MaxDelay:   120 * time.Second,
	}
	opts = append(opts, grpc.WithConnectParams(grpc.ConnectParams{
		Backoff:           backoffConfig,
		MinConnectTimeout: 20 * time.Second,
	}))

	// Message size
	if config.MaxMessageSize > 0 {
		opts = append(opts, grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(config.MaxMessageSize),
			grpc.MaxCallSendMsgSize(config.MaxMessageSize),
		))
	}

	// Unary interceptor for logging/metrics
	opts = append(opts, grpc.WithUnaryInterceptor(unaryClientInterceptor()))

	// Stream interceptor for logging/metrics
	opts = append(opts, grpc.WithStreamInterceptor(streamClientInterceptor()))

	return opts
}

// Connect establishes a connection to the gRPC server
func (n *NetworkClient) Connect(ctx context.Context) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.conn != nil {
		state := n.conn.GetState()
		if state == connectivity.Ready || state == connectivity.Idle {
			return nil
		}
	}

	conn, err := grpc.DialContext(ctx, n.address, n.opts...)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", n.address, err)
	}

	n.conn = conn
	return nil
}

// GetConnection returns the gRPC connection
func (n *NetworkClient) GetConnection(ctx context.Context) (*grpc.ClientConn, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.conn == nil {
		return nil, fmt.Errorf("not connected")
	}

	return n.conn, nil
}

// WaitForReady waits for the connection to be ready
func (n *NetworkClient) WaitForReady(ctx context.Context) error {
	conn, err := n.GetConnection(ctx)
	if err != nil {
		return err
	}

	for {
		state := conn.GetState()
		if state == connectivity.Ready {
			return nil
		}

		if !conn.WaitForStateChange(ctx, state) {
			return ctx.Err()
		}
	}
}

// Close closes the gRPC connection
func (n *NetworkClient) Close() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.conn != nil {
		err := n.conn.Close()
		n.conn = nil
		return err
	}

	return nil
}

// IsHealthy checks if the connection is healthy
func (n *NetworkClient) IsHealthy() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.conn == nil {
		return false
	}

	state := n.conn.GetState()
	return state == connectivity.Ready || state == connectivity.Idle
}

// unaryClientInterceptor provides logging and metrics for unary calls
func unaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		start := time.Now()

		err := invoker(ctx, method, req, reply, cc, opts...)

		duration := time.Since(start)
		// TODO: Add metrics and logging here
		_ = duration

		return err
	}
}

// streamClientInterceptor provides logging and metrics for stream calls
func streamClientInterceptor() grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		start := time.Now()

		stream, err := streamer(ctx, desc, cc, method, opts...)

		duration := time.Since(start)
		// TODO: Add metrics and logging here
		_ = duration

		return stream, err
	}
}

// ConnectionPool manages a pool of gRPC connections
type ConnectionPool struct {
	connections []*NetworkClient
	current     int
	mu          sync.RWMutex
	config      *NetworkClientConfig
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(config *NetworkClientConfig, size int) (*ConnectionPool, error) {
	if size <= 0 {
		return nil, fmt.Errorf("pool size must be positive")
	}

	pool := &ConnectionPool{
		connections: make([]*NetworkClient, size),
		config:      config,
	}

	for i := 0; i < size; i++ {
		pool.connections[i] = NewNetworkClient(config)
	}

	return pool, nil
}

// GetConnection returns a connection from the pool using round-robin
func (p *ConnectionPool) GetConnection(ctx context.Context) (*grpc.ClientConn, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	client := p.connections[p.current]
	p.current = (p.current + 1) % len(p.connections)

	if err := client.Connect(ctx); err != nil {
		return nil, err
	}

	return client.GetConnection(ctx)
}

// Close closes all connections in the pool
func (p *ConnectionPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var errs []error
	for _, client := range p.connections {
		if err := client.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to close connections: %v", errs)
	}

	return nil
}