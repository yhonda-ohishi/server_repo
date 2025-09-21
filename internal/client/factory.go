package client

import (
	"context"
	"fmt"

	"github.com/yhonda-ohishi/db-handler-server/internal/config"
	"google.golang.org/grpc"
)

// ClientType represents the type of gRPC client
type ClientType string

const (
	ClientTypeBufconn ClientType = "bufconn"
	ClientTypeNetwork ClientType = "network"
)

// GRPCClient interface that both bufconn and network clients implement
type GRPCClient interface {
	GetConnection(ctx context.Context) (*grpc.ClientConn, error)
	Close() error
}

// Factory creates gRPC clients based on deployment mode
type Factory struct {
	config         *config.Config
	bufconnManager *BufconnManager
	networkClients map[string]*NetworkClient
}

// NewFactory creates a new client factory
func NewFactory(cfg *config.Config) *Factory {
	return &Factory{
		config:         cfg,
		bufconnManager: NewBufconnManager(),
		networkClients: make(map[string]*NetworkClient),
	}
}

// CreateClient creates a gRPC client based on deployment mode
func (f *Factory) CreateClient(ctx context.Context, serviceName string) (GRPCClient, error) {
	if f.config.IsSingleMode() {
		return f.createBufconnClient(serviceName)
	}
	return f.createNetworkClient(ctx, serviceName)
}

// createBufconnClient creates an in-memory bufconn client
func (f *Factory) createBufconnClient(serviceName string) (GRPCClient, error) {
	client := f.bufconnManager.CreateClient(serviceName)
	return &bufconnAdapter{client: client}, nil
}

// createNetworkClient creates a network-based gRPC client
func (f *Factory) createNetworkClient(ctx context.Context, serviceName string) (GRPCClient, error) {
	address := f.getServiceAddress(serviceName)
	if address == "" {
		return nil, fmt.Errorf("no address configured for service: %s", serviceName)
	}

	// Check if we already have a client for this service
	if client, ok := f.networkClients[serviceName]; ok {
		if client.IsHealthy() {
			return client, nil
		}
		// If not healthy, close and recreate
		_ = client.Close()
	}

	config := DefaultNetworkConfig(address)
	client := NewNetworkClient(config)

	if err := client.Connect(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", serviceName, err)
	}

	f.networkClients[serviceName] = client
	return client, nil
}

// getServiceAddress returns the configured address for a service
func (f *Factory) getServiceAddress(serviceName string) string {
	switch serviceName {
	case "database":
		return f.config.External.DatabaseGRPCURL
	case "handlers":
		return f.config.External.HandlersGRPCURL
	default:
		// For other services, could look up in a service registry
		return ""
	}
}

// GetBufconnManager returns the bufconn manager for single mode
func (f *Factory) GetBufconnManager() *BufconnManager {
	return f.bufconnManager
}

// CloseAll closes all managed clients
func (f *Factory) CloseAll() error {
	var errs []error

	// Close bufconn clients
	if err := f.bufconnManager.CloseAll(); err != nil {
		errs = append(errs, err)
	}

	// Close network clients
	for name, client := range f.networkClients {
		if err := client.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close network client %s: %w", name, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("close errors: %v", errs)
	}

	return nil
}

// bufconnAdapter adapts BufconnClient to GRPCClient interface
type bufconnAdapter struct {
	client *BufconnClient
}

func (b *bufconnAdapter) GetConnection(ctx context.Context) (*grpc.ClientConn, error) {
	return b.client.GetConnection(ctx)
}

func (b *bufconnAdapter) Close() error {
	return b.client.Close()
}

// ServiceConnection holds a connection to a specific service
type ServiceConnection struct {
	ServiceName string
	Client      GRPCClient
	Connection  *grpc.ClientConn
}

// ServiceManager manages connections to multiple services
type ServiceManager struct {
	factory     *Factory
	connections map[string]*ServiceConnection
}

// NewServiceManager creates a new service manager
func NewServiceManager(cfg *config.Config) *ServiceManager {
	return &ServiceManager{
		factory:     NewFactory(cfg),
		connections: make(map[string]*ServiceConnection),
	}
}

// GetConnection returns a connection to a service, creating it if necessary
func (s *ServiceManager) GetConnection(ctx context.Context, serviceName string) (*grpc.ClientConn, error) {
	if conn, ok := s.connections[serviceName]; ok && conn.Connection != nil {
		return conn.Connection, nil
	}

	client, err := s.factory.CreateClient(ctx, serviceName)
	if err != nil {
		return nil, err
	}

	conn, err := client.GetConnection(ctx)
	if err != nil {
		return nil, err
	}

	s.connections[serviceName] = &ServiceConnection{
		ServiceName: serviceName,
		Client:      client,
		Connection:  conn,
	}

	return conn, nil
}

// CloseAll closes all managed connections
func (s *ServiceManager) CloseAll() error {
	return s.factory.CloseAll()
}