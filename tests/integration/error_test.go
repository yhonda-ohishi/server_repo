package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yhonda-ohishi/db-handler-server/internal/client"
	"github.com/yhonda-ohishi/db-handler-server/internal/config"
	"github.com/yhonda-ohishi/db-handler-server/internal/gateway"
	"github.com/yhonda-ohishi/db-handler-server/internal/services"
	pb "github.com/yhonda-ohishi/db-handler-server/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

// TestErrorHandling tests error scenarios across all protocols
func TestErrorHandling(t *testing.T) {
	// Setup test environment
	cfg := &config.Config{
		Deployment: config.DeploymentConfig{Mode: "single"},
		Server:     config.ServerConfig{HTTPPort: 8080, GRPCPort: 9090},
	}

	bufconnClient := client.NewBufconnClient()
	grpcServer := grpc.NewServer()
	registry := services.NewServiceRegistry()
	registry.RegisterAll(grpcServer)
	reflection.Register(grpcServer)

	listener := bufconnClient.GetListener()
	go func() {
		_ = grpcServer.Serve(listener)
	}()

	gw := gateway.NewSimpleGateway(cfg)
	gw.SetBufconnClient(bufconnClient)
	err := gw.Initialize()
	require.NoError(t, err)

	app := gw.GetHTTPHandler()

	// Get gRPC connection
	ctx := context.Background()
	conn, err := bufconnClient.GetConnection(ctx)
	require.NoError(t, err)
	defer conn.Close()

	userClient := pb.NewUserServiceClient(conn)
	txnClient := pb.NewTransactionServiceClient(conn)

	t.Run("404 Not Found Errors", func(t *testing.T) {
		nonExistentID := "definitely-does-not-exist"

		t.Run("REST 404 errors", func(t *testing.T) {
			// User not found
			req := httptest.NewRequest("GET", "/api/v1/users/"+nonExistentID, nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusNotFound, resp.StatusCode)

			var result map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&result)
			require.NoError(t, err)
			assert.Contains(t, result, "error")

			// Transaction not found
			req = httptest.NewRequest("GET", "/api/v1/transactions/"+nonExistentID, nil)
			resp, err = app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		})

		t.Run("gRPC NotFound errors", func(t *testing.T) {
			// User not found
			getUserReq := &pb.GetUserRequest{Id: nonExistentID}
			_, err := userClient.GetUser(ctx, getUserReq)
			assert.Error(t, err)

			grpcStatus, ok := status.FromError(err)
			assert.True(t, ok)
			assert.Equal(t, codes.NotFound, grpcStatus.Code())
			assert.Contains(t, grpcStatus.Message(), "not found")

			// Transaction not found
			getTxnReq := &pb.GetTransactionRequest{Id: nonExistentID}
			_, err = txnClient.GetTransaction(ctx, getTxnReq)
			assert.Error(t, err)

			grpcStatus, ok = status.FromError(err)
			assert.True(t, ok)
			assert.Equal(t, codes.NotFound, grpcStatus.Code())
		})

		t.Run("JSON-RPC not found errors", func(t *testing.T) {
			// User not found
			payload := `{
				"jsonrpc": "2.0",
				"method": "user.get",
				"params": {"id": "not-found"},
				"id": 1
			}`

			req := httptest.NewRequest("POST", "/jsonrpc", strings.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode) // JSON-RPC errors still return 200

			var result map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&result)
			require.NoError(t, err)

			assert.Equal(t, "2.0", result["jsonrpc"])
			assert.NotNil(t, result["error"])

			errorObj := result["error"].(map[string]interface{})
			assert.Equal(t, float64(-32000), errorObj["code"]) // Custom application error
			assert.Contains(t, errorObj["message"], "not found")
		})
	})

	t.Run("400 Bad Request Errors", func(t *testing.T) {
		t.Run("REST validation errors", func(t *testing.T) {
			// Missing required fields
			payload := `{
				"name": "Missing Email User"
			}`

			req := httptest.NewRequest("POST", "/api/v1/users", strings.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

			var result map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&result)
			require.NoError(t, err)
			assert.Contains(t, result, "error")

			// Empty user ID in path
			req = httptest.NewRequest("GET", "/api/v1/users/", nil)
			resp, err = app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			// Should either be 400 or 404, both are acceptable
			assert.True(t, resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusNotFound)

			// Missing query parameters
			req = httptest.NewRequest("GET", "/api/v1/transactions", nil) // Missing card_id
			resp, err = app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})

		t.Run("gRPC validation errors", func(t *testing.T) {
			// Empty email
			createReq := &pb.CreateUserRequest{
				Email: "",
				Name:  "",
			}

			_, err := userClient.CreateUser(ctx, createReq)
			assert.Error(t, err)

			grpcStatus, ok := status.FromError(err)
			assert.True(t, ok)
			assert.Equal(t, codes.InvalidArgument, grpcStatus.Code())

			// Empty transaction ID
			getTxnReq := &pb.GetTransactionRequest{Id: ""}
			_, err = txnClient.GetTransaction(ctx, getTxnReq)
			assert.Error(t, err)

			grpcStatus, ok = status.FromError(err)
			assert.True(t, ok)
			assert.Equal(t, codes.InvalidArgument, grpcStatus.Code())
		})

		t.Run("JSON-RPC validation errors", func(t *testing.T) {
			// Invalid parameters
			payload := `{
				"jsonrpc": "2.0",
				"method": "user.create",
				"params": {
					"name": "Missing Email"
				},
				"id": 1
			}`

			req := httptest.NewRequest("POST", "/jsonrpc", strings.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			var result map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&result)
			require.NoError(t, err)

			assert.NotNil(t, result["error"])
			errorObj := result["error"].(map[string]interface{})
			assert.Equal(t, float64(-32602), errorObj["code"]) // Invalid params
		})
	})

	t.Run("Malformed Request Errors", func(t *testing.T) {
		t.Run("Invalid JSON in REST", func(t *testing.T) {
			// Malformed JSON
			req := httptest.NewRequest("POST", "/api/v1/users", strings.NewReader(`{"invalid": json}`))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})

		t.Run("Invalid JSON-RPC requests", func(t *testing.T) {
			// Malformed JSON
			req := httptest.NewRequest("POST", "/jsonrpc", strings.NewReader(`{"invalid": json}`))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			var result map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&result)
			require.NoError(t, err)

			assert.NotNil(t, result["error"])
			errorObj := result["error"].(map[string]interface{})
			assert.Equal(t, float64(-32700), errorObj["code"]) // Parse error

			// Missing JSON-RPC version
			payload := `{
				"method": "user.get",
				"params": {"id": "test"},
				"id": 1
			}`

			req = httptest.NewRequest("POST", "/jsonrpc", strings.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			resp, err = app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			err = json.NewDecoder(resp.Body).Decode(&result)
			require.NoError(t, err)

			assert.NotNil(t, result["error"])
			errorObj = result["error"].(map[string]interface{})
			assert.Equal(t, float64(-32600), errorObj["code"]) // Invalid Request

			// Invalid JSON-RPC version
			payload = `{
				"jsonrpc": "1.0",
				"method": "user.get",
				"params": {"id": "test"},
				"id": 1
			}`

			req = httptest.NewRequest("POST", "/jsonrpc", strings.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			resp, err = app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			err = json.NewDecoder(resp.Body).Decode(&result)
			require.NoError(t, err)

			assert.NotNil(t, result["error"])
			errorObj = result["error"].(map[string]interface{})
			assert.Equal(t, float64(-32600), errorObj["code"]) // Invalid Request
		})
	})

	t.Run("Method Not Found Errors", func(t *testing.T) {
		t.Run("REST method not allowed", func(t *testing.T) {
			// PATCH method not implemented
			req := httptest.NewRequest("PATCH", "/api/v1/users/test-user", strings.NewReader(`{}`))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			// Should be either 405 Method Not Allowed or 404 Not Found
			assert.True(t, resp.StatusCode == http.StatusMethodNotAllowed || resp.StatusCode == http.StatusNotFound)
		})

		t.Run("JSON-RPC method not found", func(t *testing.T) {
			payload := `{
				"jsonrpc": "2.0",
				"method": "nonexistent.method",
				"params": {},
				"id": 1
			}`

			req := httptest.NewRequest("POST", "/jsonrpc", strings.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			var result map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&result)
			require.NoError(t, err)

			assert.NotNil(t, result["error"])
			errorObj := result["error"].(map[string]interface{})
			assert.Equal(t, float64(-32601), errorObj["code"]) // Method not found
			assert.Contains(t, errorObj["message"], "Method not found")
		})
	})

	t.Run("Content Type Errors", func(t *testing.T) {
		t.Run("Wrong content type for REST", func(t *testing.T) {
			// Send JSON data with wrong content type
			req := httptest.NewRequest("POST", "/api/v1/users", strings.NewReader(`{"email": "test@example.com", "name": "Test"}`))
			req.Header.Set("Content-Type", "text/plain")
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			// Should result in a parse error or bad request
			assert.True(t, resp.StatusCode >= 400)
		})

		t.Run("Missing content type for JSON-RPC", func(t *testing.T) {
			payload := `{
				"jsonrpc": "2.0",
				"method": "user.get",
				"params": {"id": "test"},
				"id": 1
			}`

			req := httptest.NewRequest("POST", "/jsonrpc", strings.NewReader(payload))
			// No Content-Type header
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			// May still work in some implementations, but should handle gracefully
			assert.True(t, resp.StatusCode < 500) // Should not be server error
		})
	})

	t.Run("Error Response Format Consistency", func(t *testing.T) {
		t.Run("REST error format", func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/users/not-found", nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			var result map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&result)
			require.NoError(t, err)

			// Should have an error field
			assert.Contains(t, result, "error")
			assert.IsType(t, "", result["error"])
		})

		t.Run("JSON-RPC error format", func(t *testing.T) {
			payload := `{
				"jsonrpc": "2.0",
				"method": "user.get",
				"params": {"id": "not-found"},
				"id": 1
			}`

			req := httptest.NewRequest("POST", "/jsonrpc", strings.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			var result map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&result)
			require.NoError(t, err)

			// Should follow JSON-RPC 2.0 error format
			assert.Equal(t, "2.0", result["jsonrpc"])
			assert.NotNil(t, result["error"])
			assert.Equal(t, float64(1), result["id"])

			errorObj := result["error"].(map[string]interface{})
			assert.Contains(t, errorObj, "code")
			assert.Contains(t, errorObj, "message")
			assert.IsType(t, float64(0), errorObj["code"])
			assert.IsType(t, "", errorObj["message"])
		})
	})

	t.Run("Server Error Handling", func(t *testing.T) {
		// Test that 500 errors are handled gracefully
		t.Run("Graceful degradation", func(t *testing.T) {
			// Test health endpoints during normal operation
			req := httptest.NewRequest("GET", "/health", nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)

			// Test readiness
			req = httptest.NewRequest("GET", "/ready", nil)
			resp, err = app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)
		})
	})

	t.Run("Rate Limiting and Resource Protection", func(t *testing.T) {
		// Test large request handling
		t.Run("Large request body", func(t *testing.T) {
			// Create a very large user name
			largeString := strings.Repeat("a", 10000)
			payload := `{
				"email": "large@example.com",
				"name": "` + largeString + `",
				"phone_number": "090-0000-0000",
				"address": "Large City"
			}`

			req := httptest.NewRequest("POST", "/api/v1/users", strings.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			// Should either accept it or reject with 413 (too large) or 400 (validation error)
			assert.True(t, resp.StatusCode == http.StatusCreated ||
						resp.StatusCode == http.StatusBadRequest ||
						resp.StatusCode == http.StatusRequestEntityTooLarge)
		})
	})

	// Cleanup
	grpcServer.GracefulStop()
}