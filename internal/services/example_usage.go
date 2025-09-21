package services

import (
	"google.golang.org/grpc"
)

// Example of how to use the service registry with a gRPC server

// RegisterAllServices demonstrates how to register all services
func ExampleRegisterAllServices() *grpc.Server {
	// Create a new gRPC server
	server := grpc.NewServer()

	// Option 1: Use the convenience function to register all services
	registry := Register(server)

	// You can access individual services if needed
	_ = registry.GetUserServiceInstance()
	_ = registry.GetCardServiceInstance()
	_ = registry.GetTransactionServiceInstance()
	_ = registry.GetPaymentServiceInstance()

	return server
}

// RegisterSelectedServices demonstrates how to register specific services
func ExampleRegisterSelectedServices() *grpc.Server {
	// Create a new gRPC server
	server := grpc.NewServer()

	// Option 2: Register only specific services
	registry := RegisterServices(server, "user", "card")

	// Get service information
	info := registry.GetServiceInfo()
	_ = info // Use service info as needed

	return server
}

// RegisterWithCustomOptions demonstrates advanced usage
func ExampleRegisterWithCustomOptions() *grpc.Server {
	// Create a new gRPC server
	server := grpc.NewServer()

	// Option 3: Create registry with custom options
	registry := NewServiceRegistryWithOptions(WithMockData())

	// Register all services
	registry.RegisterAll(server)

	// Check health status
	health := registry.IsHealthy()
	for serviceName, isHealthy := range health {
		if !isHealthy {
			// Handle unhealthy service
			_ = serviceName
		}
	}

	return server
}

// StartServer demonstrates a complete server setup
func ExampleStartServer() {
	// This would be in your main function or server initialization code

	// 1. Create gRPC server
	server := grpc.NewServer()

	// 2. Register services
	registry := Register(server)

	// 3. Log service information
	info := registry.GetServiceInfo()
	for serviceName, serviceInfo := range info {
		// Log or print service information
		_ = serviceName
		_ = serviceInfo
	}

	// 4. Start server (example - not functional code)
	// lis, err := net.Listen("tcp", ":50051")
	// if err != nil {
	//     log.Fatalf("Failed to listen: %v", err)
	// }
	//
	// log.Println("Starting gRPC server on :50051")
	// if err := server.Serve(lis); err != nil {
	//     log.Fatalf("Failed to serve: %v", err)
	// }
}