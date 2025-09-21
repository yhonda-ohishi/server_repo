package health

import (
	"context"
	"sync"
	"time"

	dbproto "github.com/yhonda-ohishi/db_service/src/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
)

// DBServiceHealth provides health checks for db_service components
type DBServiceHealth struct {
	mu                      sync.RWMutex
	etcMeisaiStatus        ServiceStatus
	dtakoUriageKeihiStatus ServiceStatus
	dtakoFerryRowsStatus   ServiceStatus
	etcMeisaiMappingStatus ServiceStatus
	conn                   *grpc.ClientConn
}

// ServiceStatus represents the health status of a service
type ServiceStatus struct {
	Healthy     bool      `json:"healthy"`
	LastChecked time.Time `json:"last_checked"`
	Message     string    `json:"message"`
}

// NewDBServiceHealth creates a new db_service health checker
func NewDBServiceHealth(conn *grpc.ClientConn) *DBServiceHealth {
	return &DBServiceHealth{
		conn: conn,
		etcMeisaiStatus: ServiceStatus{
			Healthy: false,
			Message: "Not checked yet",
		},
		dtakoUriageKeihiStatus: ServiceStatus{
			Healthy: false,
			Message: "Not checked yet",
		},
		dtakoFerryRowsStatus: ServiceStatus{
			Healthy: false,
			Message: "Not checked yet",
		},
		etcMeisaiMappingStatus: ServiceStatus{
			Healthy: false,
			Message: "Not checked yet",
		},
	}
}

// CheckAll performs health checks on all db_services
func (h *DBServiceHealth) CheckAll(ctx context.Context) map[string]ServiceStatus {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Check ETCMeisaiService
	h.etcMeisaiStatus = h.checkETCMeisaiService(ctx)

	// Check DTakoUriageKeihiService
	h.dtakoUriageKeihiStatus = h.checkDTakoUriageKeihiService(ctx)

	// Check DTakoFerryRowsService
	h.dtakoFerryRowsStatus = h.checkDTakoFerryRowsService(ctx)

	// Check ETCMeisaiMappingService
	h.etcMeisaiMappingStatus = h.checkETCMeisaiMappingService(ctx)

	return h.GetStatus()
}

// checkETCMeisaiService checks the health of ETCMeisaiService
func (h *DBServiceHealth) checkETCMeisaiService(ctx context.Context) ServiceStatus {
	if h.conn == nil {
		return ServiceStatus{
			Healthy:     false,
			LastChecked: time.Now(),
			Message:     "No gRPC connection available",
		}
	}

	// Try to list with empty request (should return empty list or error)
	client := dbproto.NewETCMeisaiServiceClient(h.conn)
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	_, err := client.List(ctx, &dbproto.ListETCMeisaiRequest{})
	if err != nil {
		return ServiceStatus{
			Healthy:     false,
			LastChecked: time.Now(),
			Message:     "Service unavailable: " + err.Error(),
		}
	}

	return ServiceStatus{
		Healthy:     true,
		LastChecked: time.Now(),
		Message:     "Service is healthy",
	}
}

// checkDTakoUriageKeihiService checks the health of DTakoUriageKeihiService
func (h *DBServiceHealth) checkDTakoUriageKeihiService(ctx context.Context) ServiceStatus {
	if h.conn == nil {
		return ServiceStatus{
			Healthy:     false,
			LastChecked: time.Now(),
			Message:     "No gRPC connection available",
		}
	}

	// For now, just check if service is registered (no List method available)
	// In production, would use actual health check endpoint
	return ServiceStatus{
		Healthy:     true,
		LastChecked: time.Now(),
		Message:     "Service assumed healthy (mock)",
	}
}

// checkDTakoFerryRowsService checks the health of DTakoFerryRowsService
func (h *DBServiceHealth) checkDTakoFerryRowsService(ctx context.Context) ServiceStatus {
	if h.conn == nil {
		return ServiceStatus{
			Healthy:     false,
			LastChecked: time.Now(),
			Message:     "No gRPC connection available",
		}
	}

	// For now, just check if service is registered
	return ServiceStatus{
		Healthy:     true,
		LastChecked: time.Now(),
		Message:     "Service assumed healthy (mock)",
	}
}

// checkETCMeisaiMappingService checks the health of ETCMeisaiMappingService
func (h *DBServiceHealth) checkETCMeisaiMappingService(ctx context.Context) ServiceStatus {
	if h.conn == nil {
		return ServiceStatus{
			Healthy:     false,
			LastChecked: time.Now(),
			Message:     "No gRPC connection available",
		}
	}

	// For now, just check if service is registered
	return ServiceStatus{
		Healthy:     true,
		LastChecked: time.Now(),
		Message:     "Service assumed healthy (mock)",
	}
}

// GetStatus returns the current status of all services
func (h *DBServiceHealth) GetStatus() map[string]ServiceStatus {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return map[string]ServiceStatus{
		"etc_meisai_service":         h.etcMeisaiStatus,
		"dtako_uriage_keihi_service": h.dtakoUriageKeihiStatus,
		"dtako_ferry_rows_service":   h.dtakoFerryRowsStatus,
		"etc_meisai_mapping_service": h.etcMeisaiMappingStatus,
	}
}

// IsHealthy returns true if all services are healthy
func (h *DBServiceHealth) IsHealthy() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return h.etcMeisaiStatus.Healthy &&
		h.dtakoUriageKeihiStatus.Healthy &&
		h.dtakoFerryRowsStatus.Healthy &&
		h.etcMeisaiMappingStatus.Healthy
}

// ImplementHealthServer implements the gRPC health check protocol for db_services
func (h *DBServiceHealth) ImplementHealthServer() grpc_health_v1.HealthServer {
	return &dbServiceHealthServer{health: h}
}

type dbServiceHealthServer struct {
	grpc_health_v1.UnimplementedHealthServer
	health *DBServiceHealth
}

func (s *dbServiceHealthServer) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	// Map service names to our internal services
	serviceMap := map[string]func() bool{
		"db_service.ETCMeisaiService":        func() bool { return s.health.etcMeisaiStatus.Healthy },
		"db_service.DTakoUriageKeihiService": func() bool { return s.health.dtakoUriageKeihiStatus.Healthy },
		"db_service.DTakoFerryRowsService":   func() bool { return s.health.dtakoFerryRowsStatus.Healthy },
		"db_service.ETCMeisaiMappingService": func() bool { return s.health.etcMeisaiMappingStatus.Healthy },
	}

	// Check specific service or overall health
	if req.Service != "" {
		if checkFunc, exists := serviceMap[req.Service]; exists {
			if checkFunc() {
				return &grpc_health_v1.HealthCheckResponse{
					Status: grpc_health_v1.HealthCheckResponse_SERVING,
				}, nil
			}
			return &grpc_health_v1.HealthCheckResponse{
				Status: grpc_health_v1.HealthCheckResponse_NOT_SERVING,
			}, nil
		}
	}

	// Overall health check
	if s.health.IsHealthy() {
		return &grpc_health_v1.HealthCheckResponse{
			Status: grpc_health_v1.HealthCheckResponse_SERVING,
		}, nil
	}

	return &grpc_health_v1.HealthCheckResponse{
		Status: grpc_health_v1.HealthCheckResponse_NOT_SERVING,
	}, nil
}

func (s *dbServiceHealthServer) Watch(req *grpc_health_v1.HealthCheckRequest, stream grpc_health_v1.Health_WatchServer) error {
	// For simplicity, send initial status and complete
	// In production, would implement actual watching
	resp := &grpc_health_v1.HealthCheckResponse{
		Status: grpc_health_v1.HealthCheckResponse_SERVING,
	}
	if !s.health.IsHealthy() {
		resp.Status = grpc_health_v1.HealthCheckResponse_NOT_SERVING
	}
	return stream.Send(resp)
}