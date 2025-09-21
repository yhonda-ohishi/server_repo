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
	"google.golang.org/grpc/reflection"
)

// TestProtocolConversion tests that all protocols return consistent data
func TestProtocolConversion(t *testing.T) {
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

	// Get gRPC connection for direct testing
	ctx := context.Background()
	conn, err := bufconnClient.GetConnection(ctx)
	require.NoError(t, err)
	defer conn.Close()

	userClient := pb.NewUserServiceClient(conn)
	txnClient := pb.NewTransactionServiceClient(conn)

	// Test data consistency across protocols
	t.Run("User Data Consistency", func(t *testing.T) {
		var userID string

		// Create user via gRPC and verify via REST
		t.Run("Create via gRPC, verify via REST", func(t *testing.T) {
			// Create user via gRPC
			grpcReq := &pb.CreateUserRequest{
				Email:       "consistency@example.com",
				Name:        "Consistency Test User",
				PhoneNumber: "090-5555-5555",
				Address:     "Consistency City",
			}

			grpcResp, err := userClient.CreateUser(ctx, grpcReq)
			require.NoError(t, err)
			userID = grpcResp.Id

			// Verify via REST
			req := httptest.NewRequest("GET", "/api/v1/users/"+userID, nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var restResult map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&restResult)
			require.NoError(t, err)

			// Verify data consistency
			assert.Equal(t, userID, restResult["id"])
			assert.Equal(t, grpcResp.Email, restResult["email"])
			assert.Equal(t, grpcResp.Name, restResult["name"])
			assert.Equal(t, grpcResp.PhoneNumber, restResult["phone_number"])
			assert.Equal(t, grpcResp.Address, restResult["address"])
		})

		// Verify same user via JSON-RPC
		t.Run("Verify via JSON-RPC", func(t *testing.T) {
			payload := `{
				"jsonrpc": "2.0",
				"method": "user.get",
				"params": {"id": "` + userID + `"},
				"id": 1
			}`

			req := httptest.NewRequest("POST", "/jsonrpc", strings.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var jsonrpcResult map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&jsonrpcResult)
			require.NoError(t, err)

			assert.Equal(t, "2.0", jsonrpcResult["jsonrpc"])
			assert.Nil(t, jsonrpcResult["error"])

			result := jsonrpcResult["result"].(map[string]interface{})
			assert.Equal(t, userID, result["id"])
			assert.Equal(t, "consistency@example.com", result["email"])
			assert.Equal(t, "Consistency Test User", result["name"])
		})

		// Update user via REST and verify via gRPC
		t.Run("Update via REST, verify via gRPC", func(t *testing.T) {
			// Update via REST
			payload := `{
				"email": "updated-consistency@example.com",
				"name": "Updated Consistency User",
				"phone_number": "090-7777-7777",
				"address": "Updated City"
			}`

			req := httptest.NewRequest("PUT", "/api/v1/users/"+userID, strings.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)

			// Verify via gRPC
			grpcReq := &pb.GetUserRequest{Id: userID}
			grpcResp, err := userClient.GetUser(ctx, grpcReq)
			require.NoError(t, err)

			assert.Equal(t, userID, grpcResp.Id)
			assert.Equal(t, "updated-consistency@example.com", grpcResp.Email)
			assert.Equal(t, "Updated Consistency User", grpcResp.Name)
			assert.Equal(t, "090-7777-7777", grpcResp.PhoneNumber)
			assert.Equal(t, "Updated City", grpcResp.Address)
		})
	})

	t.Run("Transaction Data Consistency", func(t *testing.T) {
		// Test transaction data across protocols
		t.Run("Transaction via all protocols", func(t *testing.T) {
			// Get transaction via gRPC
			grpcReq := &pb.GetTransactionRequest{Id: "txn-1"}
			grpcResp, err := txnClient.GetTransaction(ctx, grpcReq)
			require.NoError(t, err)

			// Get same transaction via REST
			req := httptest.NewRequest("GET", "/api/v1/transactions/txn-1", nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			var restResult map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&restResult)
			require.NoError(t, err)

			// Verify consistency
			assert.Equal(t, grpcResp.Id, restResult["id"])
			assert.Equal(t, grpcResp.CardId, restResult["card_id"])
			assert.Equal(t, grpcResp.EntryGateId, restResult["entry_gate_id"])
			assert.Equal(t, grpcResp.ExitGateId, restResult["exit_gate_id"])
			assert.Equal(t, float64(grpcResp.Distance), restResult["distance"])
			assert.Equal(t, float64(grpcResp.TollAmount), restResult["toll_amount"])
			assert.Equal(t, float64(grpcResp.FinalAmount), restResult["final_amount"])

			// Get same transaction via JSON-RPC
			payload := `{
				"jsonrpc": "2.0",
				"method": "transaction.get",
				"params": {"id": "txn-1"},
				"id": 1
			}`

			req = httptest.NewRequest("POST", "/jsonrpc", strings.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			resp, err = app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			var jsonrpcResult map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&jsonrpcResult)
			require.NoError(t, err)

			result := jsonrpcResult["result"].(map[string]interface{})
			assert.Equal(t, grpcResp.Id, result["id"])
			assert.Equal(t, grpcResp.CardId, result["card_id"])
		})
	})

	t.Run("List Operations Consistency", func(t *testing.T) {
		// Test user list consistency
		t.Run("User list via REST and JSON-RPC", func(t *testing.T) {
			// Get users via REST
			req := httptest.NewRequest("GET", "/api/v1/users", nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			var restResult map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&restResult)
			require.NoError(t, err)

			restUsers := restResult["users"].([]interface{})

			// Get users via JSON-RPC
			payload := `{
				"jsonrpc": "2.0",
				"method": "user.list",
				"id": 1
			}`

			req = httptest.NewRequest("POST", "/jsonrpc", strings.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			resp, err = app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			var jsonrpcResult map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&jsonrpcResult)
			require.NoError(t, err)

			jsonrpcResult2 := jsonrpcResult["result"].(map[string]interface{})
			jsonrpcUsers := jsonrpcResult2["users"].([]interface{})

			// Verify same number of users
			assert.Equal(t, len(restUsers), len(jsonrpcUsers))
		})

		// Test transaction history consistency
		t.Run("Transaction history via REST and gRPC", func(t *testing.T) {
			// Get via gRPC
			grpcReq := &pb.GetTransactionHistoryRequest{
				CardId:   "card-1",
				PageSize: 10,
			}
			grpcResp, err := txnClient.GetTransactionHistory(ctx, grpcReq)
			require.NoError(t, err)

			// Get via REST
			req := httptest.NewRequest("GET", "/api/v1/transactions?card_id=card-1", nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			var restResult map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&restResult)
			require.NoError(t, err)

			restTxns := restResult["transactions"].([]interface{})

			// Verify consistency
			assert.Equal(t, len(grpcResp.Transactions), len(restTxns))
			assert.Equal(t, float64(grpcResp.TotalAmount), restResult["total_amount"])
		})
	})

	t.Run("Error Handling Consistency", func(t *testing.T) {
		// Test 404 errors across protocols
		t.Run("Not found errors", func(t *testing.T) {
			nonExistentID := "non-existent-user"

			// Test REST 404
			req := httptest.NewRequest("GET", "/api/v1/users/"+nonExistentID, nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, http.StatusNotFound, resp.StatusCode)

			// Test gRPC NotFound
			grpcReq := &pb.GetUserRequest{Id: nonExistentID}
			_, err = userClient.GetUser(ctx, grpcReq)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "not found")

			// Test JSON-RPC error
			payload := `{
				"jsonrpc": "2.0",
				"method": "user.get",
				"params": {"id": "not-found"},
				"id": 1
			}`

			req = httptest.NewRequest("POST", "/jsonrpc", strings.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			resp, err = app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			var jsonrpcResult map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&jsonrpcResult)
			require.NoError(t, err)

			assert.NotNil(t, jsonrpcResult["error"])
			errorObj := jsonrpcResult["error"].(map[string]interface{})
			assert.Contains(t, errorObj["message"], "not found")
		})
	})

	t.Run("Field Type Consistency", func(t *testing.T) {
		// Test that numeric fields are handled consistently
		t.Run("Numeric field types", func(t *testing.T) {
			// Get transaction data to verify numeric fields
			req := httptest.NewRequest("GET", "/api/v1/transactions/txn-1", nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			var restResult map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&restResult)
			require.NoError(t, err)

			// Verify field types are appropriate
			assert.IsType(t, "", restResult["id"])
			assert.IsType(t, "", restResult["card_id"])
			assert.IsType(t, float64(0), restResult["distance"])
			assert.IsType(t, float64(0), restResult["toll_amount"])
			assert.IsType(t, float64(0), restResult["final_amount"])
		})
	})

	// Cleanup
	grpcServer.GracefulStop()
}