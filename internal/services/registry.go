package services

import (
	"database/sql"
	"log"
	"os"

	pb "github.com/yhonda-ohishi/db-handler-server/proto"
	"github.com/yhonda-ohishi/db-handler-server/internal/client"
	dbproto "github.com/yhonda-ohishi/db_service/src/proto"
	etcpb "github.com/yhonda-ohishi/etc_meisai_scraper/src/pb"
	etcservices "github.com/yhonda-ohishi/etc_meisai_scraper/src/services"
	_ "github.com/go-sql-driver/mysql"
	"google.golang.org/grpc"
)

// ServiceRegistry holds all gRPC service implementations
type ServiceRegistry struct {
	UserService        *UserService
	TransactionService *TransactionService
	CardService        *CardService
	PaymentService     *PaymentService
	ETCService         *ETCServiceServer
	DBServiceClient    *client.DBServiceClient
	// DB services for single mode - can be either mock or real implementations
	ETCMeisaiService        dbproto.ETCMeisaiServiceServer
	DTakoUriageKeihiService dbproto.DTakoUriageKeihiServiceServer
	DTakoFerryRowsService   dbproto.DTakoFerryRowsServiceServer
	ETCMeisaiMappingService dbproto.ETCMeisaiMappingServiceServer
	// etc_meisai_scraper services
	DownloadService         etcpb.DownloadServiceServer
	IsSingleMode            bool
}

// NewServiceRegistry creates a new service registry with all services initialized
func NewServiceRegistry() *ServiceRegistry {
	return &ServiceRegistry{
		UserService:        NewUserService(),
		TransactionService: NewTransactionService(),
		CardService:        NewCardService(),
		PaymentService:     NewPaymentService(),
		ETCService:         NewETCServiceServer(),
	}
}

// NewServiceRegistryForSingleMode creates a service registry with DB services for single mode
func NewServiceRegistryForSingleMode() *ServiceRegistry {
	reg := NewServiceRegistryWithRealDB()
	if reg != nil {
		return reg
	}

	// Fallback to basic registry without db_service
	log.Printf("Warning: Failed to initialize db_service, running without database services")
	return &ServiceRegistry{
		UserService:        NewUserService(),
		TransactionService: NewTransactionService(),
		CardService:        NewCardService(),
		PaymentService:     NewPaymentService(),
		ETCService:         NewETCServiceServer(),
		IsSingleMode:       true,
	}
}

// NewServiceRegistryWithRealDB creates a service registry with db_service client
func NewServiceRegistryWithRealDB() *ServiceRegistry {
	// db_service is accessed via gRPC/bufconn, not direct database connection
	// The db_service handles all database operations

	// Initialize download service for etc_meisai_scraper
	// For now, using a dummy DB connection - in production this would be configured properly
	var downloadServiceServer etcpb.DownloadServiceServer

	// Try to create the download service
	if dbDSN := os.Getenv("DATABASE_URL"); dbDSN != "" {
		if db, err := sql.Open("mysql", dbDSN); err == nil {
			logger := log.New(os.Stdout, "[DownloadService] ", log.LstdFlags)
			downloadServiceServer = etcservices.NewDownloadServiceGRPC(db, logger)
		}
	}

	// If no DB configured, create without download service
	if downloadServiceServer == nil {
		log.Println("Warning: DownloadService not initialized (no DATABASE_URL)")
	}

	return &ServiceRegistry{
		UserService:             NewUserService(),
		TransactionService:      NewTransactionService(),
		CardService:             NewCardService(),
		PaymentService:          NewPaymentService(),
		ETCService:              NewETCServiceServer(),
		DownloadService:         downloadServiceServer,
		IsSingleMode:            true,
	}
}

// RegisterAll registers all services to a gRPC server
// This method supports both single mode (register directly to server) and separate mode
func (r *ServiceRegistry) RegisterAll(server *grpc.Server) {
	// Register all services with the gRPC server
	pb.RegisterUserServiceServer(server, r.UserService)
	pb.RegisterTransactionServiceServer(server, r.TransactionService)
	pb.RegisterCardServiceServer(server, r.CardService)
	pb.RegisterPaymentServiceServer(server, r.PaymentService)
	pb.RegisterETCServiceServer(server, r.ETCService)

	// In single mode, also register db_service services directly
	// These are accessed via bufconn in-memory, not external connection
	if r.IsSingleMode {
		if r.ETCMeisaiService != nil {
			dbproto.RegisterETCMeisaiServiceServer(server, r.ETCMeisaiService)
		}
		if r.DTakoUriageKeihiService != nil {
			dbproto.RegisterDTakoUriageKeihiServiceServer(server, r.DTakoUriageKeihiService)
		}
		if r.DTakoFerryRowsService != nil {
			dbproto.RegisterDTakoFerryRowsServiceServer(server, r.DTakoFerryRowsService)
		}
		if r.ETCMeisaiMappingService != nil {
			dbproto.RegisterETCMeisaiMappingServiceServer(server, r.ETCMeisaiMappingService)
		}

		// Register etc_meisai_scraper services
		if r.DownloadService != nil {
			etcpb.RegisterDownloadServiceServer(server, r.DownloadService)
		}
	}
}

// RegisterSeparately registers services individually to a gRPC server
// This provides more granular control over which services to register
func (r *ServiceRegistry) RegisterSeparately(server *grpc.Server, serviceNames ...string) {
	serviceMap := map[string]func(){
		"user": func() {
			pb.RegisterUserServiceServer(server, r.UserService)
		},
		"transaction": func() {
			pb.RegisterTransactionServiceServer(server, r.TransactionService)
		},
		"card": func() {
			pb.RegisterCardServiceServer(server, r.CardService)
		},
		"payment": func() {
			pb.RegisterPaymentServiceServer(server, r.PaymentService)
		},
		"etc": func() {
			pb.RegisterETCServiceServer(server, r.ETCService)
		},
	}

	// Register specified services
	for _, serviceName := range serviceNames {
		if registerFunc, exists := serviceMap[serviceName]; exists {
			registerFunc()
		}
	}
}

// GetServiceInfo returns information about all registered services
func (r *ServiceRegistry) GetServiceInfo() map[string]interface{} {
	return map[string]interface{}{
		"user_service": map[string]interface{}{
			"name":        "UserService",
			"description": "Manages user accounts and profiles",
			"methods":     []string{"GetUser", "CreateUser", "UpdateUser", "DeleteUser", "ListUsers"},
			"user_count":  r.UserService.GetUserCount(),
		},
		"transaction_service": map[string]interface{}{
			"name":              "TransactionService",
			"description":       "Handles ETC transaction history",
			"methods":           []string{"GetTransaction", "GetTransactionHistory"},
			"transaction_count": r.TransactionService.GetTransactionCount(),
		},
		"card_service": map[string]interface{}{
			"name":        "CardService",
			"description": "Manages ETC cards",
			"methods":     []string{"GetCard", "CreateCard", "UpdateCard", "DeleteCard", "ListCards"},
			"card_count":  r.CardService.GetCardCount(),
		},
		"payment_service": map[string]interface{}{
			"name":          "PaymentService",
			"description":   "Processes payments and generates statements",
			"methods":       []string{"GetPayment", "CreatePayment", "ListPayments", "GetMonthlyStatement"},
			"payment_count": r.PaymentService.GetPaymentCount(),
		},
	}
}

// Health check methods for each service
func (r *ServiceRegistry) IsHealthy() map[string]bool {
	return map[string]bool{
		"user_service":        r.UserService != nil,
		"transaction_service": r.TransactionService != nil,
		"card_service":        r.CardService != nil,
		"payment_service":     r.PaymentService != nil,
	}
}

// Register is a convenience function for registering all services
// This function provides a simple interface for external packages
func Register(server *grpc.Server) *ServiceRegistry {
	registry := NewServiceRegistry()
	registry.RegisterAll(server)
	return registry
}

// RegisterServices is an alternative registration function that allows
// specifying which services to register
func RegisterServices(server *grpc.Server, serviceNames ...string) *ServiceRegistry {
	registry := NewServiceRegistry()

	if len(serviceNames) == 0 {
		// If no specific services are specified, register all
		registry.RegisterAll(server)
	} else {
		// Register only specified services
		registry.RegisterSeparately(server, serviceNames...)
	}

	return registry
}

// ServiceOption represents a configuration option for service registration
type ServiceOption func(*ServiceRegistry)

// WithMockData is an option to initialize services with mock data (default behavior)
func WithMockData() ServiceOption {
	return func(r *ServiceRegistry) {
		// Mock data is already added by default in the New*Service functions
		// This option is here for explicit configuration and future extensions
	}
}

// NewServiceRegistryWithOptions creates a service registry with custom options
func NewServiceRegistryWithOptions(opts ...ServiceOption) *ServiceRegistry {
	registry := NewServiceRegistry()

	for _, opt := range opts {
		opt(registry)
	}

	return registry
}

// GetUserServiceInstance returns the user service instance for direct access
func (r *ServiceRegistry) GetUserServiceInstance() *UserService {
	return r.UserService
}

// GetTransactionServiceInstance returns the transaction service instance for direct access
func (r *ServiceRegistry) GetTransactionServiceInstance() *TransactionService {
	return r.TransactionService
}

// GetCardServiceInstance returns the card service instance for direct access
func (r *ServiceRegistry) GetCardServiceInstance() *CardService {
	return r.CardService
}

// GetPaymentServiceInstance returns the payment service instance for direct access
func (r *ServiceRegistry) GetPaymentServiceInstance() *PaymentService {
	return r.PaymentService
}