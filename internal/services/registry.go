package services

import (
	"log"

	pb "github.com/yhonda-ohishi/db-handler-server/proto"
	"github.com/yhonda-ohishi/db-handler-server/internal/client"
	"github.com/yhonda-ohishi/db_service/src/config"
	dbproto "github.com/yhonda-ohishi/db_service/src/proto"
	"github.com/yhonda-ohishi/db_service/src/repository"
	"github.com/yhonda-ohishi/db_service/src/service"
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

// NewServiceRegistryWithRealDB creates a service registry with real db_service implementations
func NewServiceRegistryWithRealDB() *ServiceRegistry {
	// Load db_service configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Printf("Failed to load db_service config: %v", err)
		return nil
	}

	// Initialize database using db_service's config
	db, err := config.InitDatabase(cfg)
	if err != nil {
		log.Printf("Failed to initialize db_service database: %v", err)
		return nil
	}

	// Initialize repositories
	etcMeisaiRepo := repository.NewETCMeisaiRepository(db)
	dtakoUriageKeihiRepo := repository.NewDTakoUriageKeihiRepository(db)
	dtakoFerryRowsRepo := repository.NewDTakoFerryRowsRepository(db)
	etcMeisaiMappingRepo := repository.NewETCMeisaiMappingRepository(db)

	// Initialize services
	etcMeisaiService := service.NewETCMeisaiService(etcMeisaiRepo)
	dtakoUriageKeihiService := service.NewDTakoUriageKeihiService(dtakoUriageKeihiRepo)
	dtakoFerryRowsService := service.NewDTakoFerryRowsService(dtakoFerryRowsRepo)
	etcMeisaiMappingService := service.NewETCMeisaiMappingService(etcMeisaiMappingRepo)

	// Store real services
	return &ServiceRegistry{
		UserService:             NewUserService(),
		TransactionService:      NewTransactionService(),
		CardService:             NewCardService(),
		PaymentService:          NewPaymentService(),
		ETCService:              NewETCServiceServer(),
		ETCMeisaiService:        etcMeisaiService,
		DTakoUriageKeihiService: dtakoUriageKeihiService,
		DTakoFerryRowsService:   dtakoFerryRowsService,
		ETCMeisaiMappingService: etcMeisaiMappingService,
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