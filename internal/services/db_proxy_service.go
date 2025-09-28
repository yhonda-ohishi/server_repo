package services

import (
	dbproto "github.com/yhonda-ohishi/db_service/src/proto"
	"github.com/yhonda-ohishi/db_service/src/service"
	// "github.com/yhonda-ohishi/db_service/src/repository"
	"google.golang.org/grpc"
)

// DBProxyService wraps db_service implementations
type DBProxyService struct {
	ETCMeisaiService        *service.ETCMeisaiService
	DTakoUriageKeihiService *service.DTakoUriageKeihiService
	DTakoFerryRowsService   *service.DTakoFerryRowsService
	ETCMeisaiMappingService *service.ETCMeisaiMappingService
}

// NewDBProxyService creates a new DB proxy service
// In single mode: creates in-memory/mock implementations
// In separate mode: would connect to external db_service
func NewDBProxyService(useMockData bool) *DBProxyService {
	if useMockData {
		// For mock data, return empty proxy - mock services are registered separately
		return &DBProxyService{}
	}

	// For production, would connect to real database
	// This is just a placeholder - you would configure real DB connection here
	return &DBProxyService{}
}

// NewDBProxyServiceWithDB creates services - db_service handles all DB connections via gRPC
func NewDBProxyServiceWithDB(dsn string) (*DBProxyService, error) {
	// db_service handles all database operations via gRPC
	// No direct database connection needed here
	return &DBProxyService{}, nil
}

// Close closes any resources
func (s *DBProxyService) Close() error {
	// No direct database connections to close
	// db_service manages its own connections
	return nil
}

// RegisterToServer registers all db_service implementations to gRPC server
func (s *DBProxyService) RegisterToServer(server interface{}) {
	// Type assertion to grpc.Server
	if grpcServer, ok := server.(dbproto.ETCMeisaiServiceServer); ok {
		// Services are already implementing the interfaces
		_ = grpcServer
	}

	// Register each service
	if grpcServer, ok := server.(*grpc.Server); ok {
		if s.ETCMeisaiService != nil {
			dbproto.RegisterETCMeisaiServiceServer(grpcServer, s.ETCMeisaiService)
		}
		if s.DTakoUriageKeihiService != nil {
			dbproto.RegisterDTakoUriageKeihiServiceServer(grpcServer, s.DTakoUriageKeihiService)
		}
		if s.DTakoFerryRowsService != nil {
			dbproto.RegisterDTakoFerryRowsServiceServer(grpcServer, s.DTakoFerryRowsService)
		}
		if s.ETCMeisaiMappingService != nil {
			dbproto.RegisterETCMeisaiMappingServiceServer(grpcServer, s.ETCMeisaiMappingService)
		}
	}
}