package client

import (
	"context"
	"fmt"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

// BufconnClient manages in-memory gRPC connections using bufconn
type BufconnClient struct {
	listener *bufconn.Listener
	server   *grpc.Server
	conn     *grpc.ClientConn
}

// NewBufconnClient creates a new bufconn client with an in-memory listener
func NewBufconnClient() *BufconnClient {
	return &BufconnClient{
		listener: bufconn.Listen(bufSize),
	}
}

// GetListener returns the bufconn listener for server registration
func (b *BufconnClient) GetListener() *bufconn.Listener {
	return b.listener
}

// StartServer starts the gRPC server with the provided options
func (b *BufconnClient) StartServer(opts ...grpc.ServerOption) (*grpc.Server, error) {
	if b.server != nil {
		return b.server, nil
	}

	b.server = grpc.NewServer(opts...)

	go func() {
		if err := b.server.Serve(b.listener); err != nil {
			// Log error but don't panic as server might be gracefully stopped
			fmt.Printf("bufconn server error: %v\n", err)
		}
	}()

	return b.server, nil
}

// GetConnection returns a client connection to the bufconn server
func (b *BufconnClient) GetConnection(ctx context.Context, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	if b.conn != nil {
		return b.conn, nil
	}

	// Create dialer function for bufconn
	bufDialer := func(context.Context, string) (net.Conn, error) {
		return b.listener.Dial()
	}

	// Default options for bufconn
	defaultOpts := []grpc.DialOption{
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	// Append user options
	defaultOpts = append(defaultOpts, opts...)

	conn, err := grpc.DialContext(ctx, "bufnet", defaultOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to dial bufconn: %w", err)
	}

	b.conn = conn
	return conn, nil
}

// Close closes the client connection and stops the server
func (b *BufconnClient) Close() error {
	var errs []error

	if b.conn != nil {
		if err := b.conn.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close client connection: %w", err))
		}
		b.conn = nil
	}

	if b.server != nil {
		b.server.GracefulStop()
		b.server = nil
	}

	if b.listener != nil {
		if err := b.listener.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close listener: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("close errors: %v", errs)
	}

	return nil
}

// BufconnManager manages multiple bufconn connections for different services
type BufconnManager struct {
	clients map[string]*BufconnClient
}

// NewBufconnManager creates a new bufconn manager
func NewBufconnManager() *BufconnManager {
	return &BufconnManager{
		clients: make(map[string]*BufconnClient),
	}
}

// CreateClient creates a new bufconn client for a service
func (m *BufconnManager) CreateClient(serviceName string) *BufconnClient {
	client := NewBufconnClient()
	m.clients[serviceName] = client
	return client
}

// GetClient returns a bufconn client for a service
func (m *BufconnManager) GetClient(serviceName string) (*BufconnClient, bool) {
	client, ok := m.clients[serviceName]
	return client, ok
}

// CloseAll closes all managed clients
func (m *BufconnManager) CloseAll() error {
	var errs []error
	for name, client := range m.clients {
		if err := client.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close client %s: %w", name, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("close errors: %v", errs)
	}

	return nil
}