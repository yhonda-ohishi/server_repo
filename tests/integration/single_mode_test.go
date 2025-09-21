package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yhonda-ohishi/db-handler-server/internal/client"
	"github.com/yhonda-ohishi/db-handler-server/internal/config"
	"github.com/yhonda-ohishi/db-handler-server/internal/gateway"
	"github.com/yhonda-ohishi/db-handler-server/internal/services"
	pb "github.com/yhonda-ohishi/db-handler-server/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// TestSingleModeIntegration tests the complete single mode flow with all protocols
func TestSingleModeIntegration(t *testing.T) {
	// Setup configuration for single mode
	cfg := &config.Config{
		Deployment: config.DeploymentConfig{Mode: "single"},
		Server:     config.ServerConfig{HTTPPort: 8080, GRPCPort: 9090},
	}

	// Setup bufconn for in-memory gRPC server
	bufconnClient := client.NewBufconnClient()

	// Create gRPC server with all services
	grpcServer := grpc.NewServer()
	registry := services.NewServiceRegistry()
	registry.RegisterAll(grpcServer)
	reflection.Register(grpcServer)

	// Start gRPC server
	listener := bufconnClient.GetListener()
	go func() {
		_ = grpcServer.Serve(listener)
	}()

	// Setup gateway with bufconn client
	gw := gateway.NewSimpleGateway(cfg)
	gw.SetBufconnClient(bufconnClient)

	// Initialize gateway
	err := gw.Initialize()
	require.NoError(t, err)

	// Create HTTP handler for testing
	app := gw.GetHTTPHandler()

	t.Run("Single Mode E2E User Workflow", func(t *testing.T) {
		// Test complete user lifecycle via REST API
		var userID string

		// 1. Create user
		t.Run("Create user via REST", func(t *testing.T) {
			payload := `{
				"email": "integration@example.com",
				"name": "Integration Test User",
				"phone_number": "090-1234-5678",
				"address": "Tokyo, Japan"
			}`

			req := httptest.NewRequest("POST", "/api/v1/users", strings.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusCreated, resp.StatusCode)

			var result map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&result)
			require.NoError(t, err)

			userID = result["id"].(string)
			assert.NotEmpty(t, userID)
			assert.Equal(t, "integration@example.com", result["email"])
		})

		// 2. Get user by ID
		t.Run("Get user via REST", func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/users/"+userID, nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var result map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&result)
			require.NoError(t, err)

			assert.Equal(t, userID, result["id"])
			assert.Equal(t, "integration@example.com", result["email"])
		})

		// 3. Update user
		t.Run("Update user via REST", func(t *testing.T) {
			payload := `{
				"email": "updated@example.com",
				"name": "Updated Name",
				"phone_number": "090-9999-8888",
				"address": "Osaka, Japan"
			}`

			req := httptest.NewRequest("PUT", "/api/v1/users/"+userID, strings.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var result map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&result)
			require.NoError(t, err)

			assert.Equal(t, "updated@example.com", result["email"])
			assert.Equal(t, "Updated Name", result["name"])
		})

		// 4. List users
		t.Run("List users via REST", func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/users", nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var result map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&result)
			require.NoError(t, err)

			users := result["users"].([]interface{})
			assert.GreaterOrEqual(t, len(users), 1)
		})

		// 5. Delete user
		t.Run("Delete user via REST", func(t *testing.T) {
			req := httptest.NewRequest("DELETE", "/api/v1/users/"+userID, nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusNoContent, resp.StatusCode)
		})
	})

	t.Run("Single Mode gRPC Direct Access", func(t *testing.T) {
		// Test direct gRPC access using bufconn
		ctx := context.Background()
		conn, err := bufconnClient.GetConnection(ctx)
		require.NoError(t, err)
		defer conn.Close()

		userClient := pb.NewUserServiceClient(conn)

		// Test user creation via gRPC
		t.Run("Create user via gRPC", func(t *testing.T) {
			req := &pb.CreateUserRequest{
				Email:       "grpc@example.com",
				Name:        "gRPC Test User",
				PhoneNumber: "090-1111-2222",
				Address:     "Kyoto, Japan",
			}

			resp, err := userClient.CreateUser(ctx, req)
			require.NoError(t, err)
			assert.NotEmpty(t, resp.Id)
			assert.Equal(t, req.Email, resp.Email)
		})
	})

	t.Run("Single Mode Transaction Flow", func(t *testing.T) {
		// Test transaction retrieval
		t.Run("Get transaction via REST", func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/transactions/txn-1", nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var result map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&result)
			require.NoError(t, err)

			assert.Equal(t, "txn-1", result["id"])
			assert.Contains(t, result, "card_id")
			assert.Contains(t, result, "toll_amount")
		})

		// Test transaction history
		t.Run("Get transaction history via REST", func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/transactions?card_id=card-1", nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var result map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&result)
			require.NoError(t, err)

			assert.Contains(t, result, "transactions")
			assert.Contains(t, result, "total_amount")
		})
	})

	t.Run("Single Mode JSON-RPC Access", func(t *testing.T) {
		// Test JSON-RPC 2.0 protocol
		t.Run("User get via JSON-RPC", func(t *testing.T) {
			payload := `{
				"jsonrpc": "2.0",
				"method": "user.get",
				"params": {"id": "test-user"},
				"id": 1
			}`

			req := httptest.NewRequest("POST", "/jsonrpc", strings.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var result map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&result)
			require.NoError(t, err)

			assert.Equal(t, "2.0", result["jsonrpc"])
			assert.Contains(t, result, "result")
		})
	})

	t.Run("Single Mode Health Checks", func(t *testing.T) {
		// Test health endpoints
		t.Run("Health endpoint", func(t *testing.T) {
			req := httptest.NewRequest("GET", "/health", nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)
		})

		t.Run("Readiness endpoint", func(t *testing.T) {
			req := httptest.NewRequest("GET", "/ready", nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)
		})
	})

	// Cleanup
	grpcServer.GracefulStop()
}

// TestSingleModePerformance tests performance characteristics of single mode
func TestSingleModePerformance(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Deployment: config.DeploymentConfig{Mode: "single"},
		Server:     config.ServerConfig{HTTPPort: 8080, GRPCPort: 9090},
	}

	bufconnClient := client.NewBufconnClient()
	grpcServer := grpc.NewServer()
	registry := services.NewServiceRegistry()
	registry.RegisterAll(grpcServer)

	listener := bufconnClient.GetListener()
	go func() {
		_ = grpcServer.Serve(listener)
	}()

	gw := gateway.NewSimpleGateway(cfg)
	gw.SetBufconnClient(bufconnClient)
	err := gw.Initialize()
	require.NoError(t, err)

	app := gw.GetHTTPHandler()

	t.Run("Protocol conversion overhead", func(t *testing.T) {
		iterations := 100

		// Measure REST API latency
		start := time.Now()
		for i := 0; i < iterations; i++ {
			req := httptest.NewRequest("GET", "/api/v1/users", nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			resp.Body.Close()
		}
		restDuration := time.Since(start)

		// Measure JSON-RPC latency
		start = time.Now()
		for i := 0; i < iterations; i++ {
			payload := `{"jsonrpc": "2.0", "method": "user.list", "id": 1}`
			req := httptest.NewRequest("POST", "/jsonrpc", strings.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req)
			require.NoError(t, err)
			resp.Body.Close()
		}
		jsonrpcDuration := time.Since(start)

		t.Logf("REST API average latency: %v", restDuration/time.Duration(iterations))
		t.Logf("JSON-RPC average latency: %v", jsonrpcDuration/time.Duration(iterations))

		// Assert that protocol conversion overhead is reasonable (< 10ms per request)
		avgRestLatency := restDuration / time.Duration(iterations)
		avgJsonrpcLatency := jsonrpcDuration / time.Duration(iterations)

		assert.Less(t, avgRestLatency, 10*time.Millisecond, "REST API latency should be < 10ms")
		assert.Less(t, avgJsonrpcLatency, 10*time.Millisecond, "JSON-RPC latency should be < 10ms")
	})

	// Cleanup
	grpcServer.GracefulStop()
}