package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
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

// TestMultiProtocolConcurrent tests concurrent requests across all protocols
func TestMultiProtocolConcurrent(t *testing.T) {
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

	t.Run("Concurrent User Operations", func(t *testing.T) {
		const numGoroutines = 50
		const operationsPerGoroutine = 10

		var wg sync.WaitGroup
		var successCount int64
		var errorCount int64

		// Test concurrent user creation across protocols
		t.Run("Concurrent user creation", func(t *testing.T) {
			wg.Add(numGoroutines)

			for i := 0; i < numGoroutines; i++ {
				go func(id int) {
					defer wg.Done()

					for j := 0; j < operationsPerGoroutine; j++ {
						// Alternate between protocols
						switch (id + j) % 3 {
						case 0: // REST
							payload := fmt.Sprintf(`{
								"email": "concurrent%d_%d@example.com",
								"name": "Concurrent User %d_%d",
								"phone_number": "090-%04d-%04d",
								"address": "Concurrent City %d_%d"
							}`, id, j, id, j, id, j, id, j)

							req := httptest.NewRequest("POST", "/api/v1/users", strings.NewReader(payload))
							req.Header.Set("Content-Type", "application/json")
							resp, err := app.Test(req)
							if err == nil && resp.StatusCode == http.StatusCreated {
								atomic.AddInt64(&successCount, 1)
							} else {
								atomic.AddInt64(&errorCount, 1)
							}
							if resp != nil {
								resp.Body.Close()
							}

						case 1: // gRPC
							grpcReq := &pb.CreateUserRequest{
								Email:       fmt.Sprintf("grpc%d_%d@example.com", id, j),
								Name:        fmt.Sprintf("gRPC User %d_%d", id, j),
								PhoneNumber: fmt.Sprintf("090-%04d-%04d", id, j),
								Address:     fmt.Sprintf("gRPC City %d_%d", id, j),
							}

							_, err := userClient.CreateUser(ctx, grpcReq)
							if err == nil {
								atomic.AddInt64(&successCount, 1)
							} else {
								atomic.AddInt64(&errorCount, 1)
							}

						case 2: // JSON-RPC
							payload := fmt.Sprintf(`{
								"jsonrpc": "2.0",
								"method": "user.create",
								"params": {
									"email": "jsonrpc%d_%d@example.com",
									"name": "JSON-RPC User %d_%d",
									"phone_number": "090-%04d-%04d",
									"address": "JSON-RPC City %d_%d"
								},
								"id": %d
							}`, id, j, id, j, id, j, id, j, id*1000+j)

							req := httptest.NewRequest("POST", "/jsonrpc", strings.NewReader(payload))
							req.Header.Set("Content-Type", "application/json")
							resp, err := app.Test(req)
							if err == nil && resp.StatusCode == http.StatusOK {
								var result map[string]interface{}
								err = json.NewDecoder(resp.Body).Decode(&result)
								if err == nil && result["error"] == nil {
									atomic.AddInt64(&successCount, 1)
								} else {
									atomic.AddInt64(&errorCount, 1)
								}
							} else {
								atomic.AddInt64(&errorCount, 1)
							}
							if resp != nil {
								resp.Body.Close()
							}
						}
					}
				}(i)
			}

			wg.Wait()

			totalOperations := int64(numGoroutines * operationsPerGoroutine)
			t.Logf("Total operations: %d, Successes: %d, Errors: %d", totalOperations, successCount, errorCount)

			// At least 80% of operations should succeed
			successRate := float64(successCount) / float64(totalOperations)
			assert.GreaterOrEqual(t, successRate, 0.8, "Success rate should be at least 80%")
		})

		// Reset counters
		atomic.StoreInt64(&successCount, 0)
		atomic.StoreInt64(&errorCount, 0)

		// Test concurrent read operations
		t.Run("Concurrent read operations", func(t *testing.T) {
			wg.Add(numGoroutines)

			for i := 0; i < numGoroutines; i++ {
				go func(id int) {
					defer wg.Done()

					for j := 0; j < operationsPerGoroutine; j++ {
						// Alternate between protocols for user list operations
						switch (id + j) % 3 {
						case 0: // REST user list
							req := httptest.NewRequest("GET", "/api/v1/users", nil)
							resp, err := app.Test(req)
							if err == nil && resp.StatusCode == http.StatusOK {
								atomic.AddInt64(&successCount, 1)
							} else {
								atomic.AddInt64(&errorCount, 1)
							}
							if resp != nil {
								resp.Body.Close()
							}

						case 1: // gRPC user list
							grpcReq := &pb.ListUsersRequest{PageSize: 10}
							_, err := userClient.ListUsers(ctx, grpcReq)
							if err == nil {
								atomic.AddInt64(&successCount, 1)
							} else {
								atomic.AddInt64(&errorCount, 1)
							}

						case 2: // JSON-RPC user list
							payload := `{
								"jsonrpc": "2.0",
								"method": "user.list",
								"id": 1
							}`

							req := httptest.NewRequest("POST", "/jsonrpc", strings.NewReader(payload))
							req.Header.Set("Content-Type", "application/json")
							resp, err := app.Test(req)
							if err == nil && resp.StatusCode == http.StatusOK {
								atomic.AddInt64(&successCount, 1)
							} else {
								atomic.AddInt64(&errorCount, 1)
							}
							if resp != nil {
								resp.Body.Close()
							}
						}
					}
				}(i)
			}

			wg.Wait()

			totalOperations := int64(numGoroutines * operationsPerGoroutine)
			t.Logf("Read operations - Total: %d, Successes: %d, Errors: %d", totalOperations, successCount, errorCount)

			// Read operations should have very high success rate
			successRate := float64(successCount) / float64(totalOperations)
			assert.GreaterOrEqual(t, successRate, 0.95, "Read success rate should be at least 95%")
		})
	})

	t.Run("Concurrent Transaction Operations", func(t *testing.T) {
		const numGoroutines = 30
		var wg sync.WaitGroup
		var successCount int64

		wg.Add(numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer wg.Done()

				// Test transaction retrieval
				req := httptest.NewRequest("GET", "/api/v1/transactions/txn-1", nil)
				resp, err := app.Test(req)
				if err == nil && resp.StatusCode == http.StatusOK {
					atomic.AddInt64(&successCount, 1)
				}
				if resp != nil {
					resp.Body.Close()
				}

				// Test transaction history
				req = httptest.NewRequest("GET", "/api/v1/transactions?card_id=card-1", nil)
				resp, err = app.Test(req)
				if err == nil && resp.StatusCode == http.StatusOK {
					atomic.AddInt64(&successCount, 1)
				}
				if resp != nil {
					resp.Body.Close()
				}
			}(i)
		}

		wg.Wait()

		expectedSuccesses := int64(numGoroutines * 2) // 2 operations per goroutine
		t.Logf("Transaction operations - Expected: %d, Actual: %d", expectedSuccesses, successCount)
		assert.GreaterOrEqual(t, successCount, expectedSuccesses*8/10) // At least 80% success
	})

	t.Run("Mixed Protocol Load Test", func(t *testing.T) {
		const duration = 5 * time.Second
		const numWorkers = 20

		var wg sync.WaitGroup
		var requestCount int64
		var errorCount int64

		ctx, cancel := context.WithTimeout(context.Background(), duration)
		defer cancel()

		// Start workers
		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()

				for {
					select {
					case <-ctx.Done():
						return
					default:
						atomic.AddInt64(&requestCount, 1)

						// Randomly choose protocol and operation
						operation := atomic.LoadInt64(&requestCount) % 6

						switch operation {
						case 0: // REST user list
							req := httptest.NewRequest("GET", "/api/v1/users", nil)
							resp, err := app.Test(req)
							if err != nil || resp.StatusCode != http.StatusOK {
								atomic.AddInt64(&errorCount, 1)
							}
							if resp != nil {
								resp.Body.Close()
							}

						case 1: // REST transaction
							req := httptest.NewRequest("GET", "/api/v1/transactions/txn-1", nil)
							resp, err := app.Test(req)
							if err != nil || resp.StatusCode != http.StatusOK {
								atomic.AddInt64(&errorCount, 1)
							}
							if resp != nil {
								resp.Body.Close()
							}

						case 2: // JSON-RPC user list
							payload := `{"jsonrpc": "2.0", "method": "user.list", "id": 1}`
							req := httptest.NewRequest("POST", "/jsonrpc", strings.NewReader(payload))
							req.Header.Set("Content-Type", "application/json")
							resp, err := app.Test(req)
							if err != nil || resp.StatusCode != http.StatusOK {
								atomic.AddInt64(&errorCount, 1)
							}
							if resp != nil {
								resp.Body.Close()
							}

						case 3: // JSON-RPC transaction
							payload := `{"jsonrpc": "2.0", "method": "transaction.get", "params": {"id": "txn-1"}, "id": 1}`
							req := httptest.NewRequest("POST", "/jsonrpc", strings.NewReader(payload))
							req.Header.Set("Content-Type", "application/json")
							resp, err := app.Test(req)
							if err != nil || resp.StatusCode != http.StatusOK {
								atomic.AddInt64(&errorCount, 1)
							}
							if resp != nil {
								resp.Body.Close()
							}

						case 4: // gRPC user list
							grpcReq := &pb.ListUsersRequest{PageSize: 10}
							_, err := userClient.ListUsers(context.Background(), grpcReq)
							if err != nil {
								atomic.AddInt64(&errorCount, 1)
							}

						case 5: // Health check
							req := httptest.NewRequest("GET", "/health", nil)
							resp, err := app.Test(req)
							if err != nil || resp.StatusCode != http.StatusOK {
								atomic.AddInt64(&errorCount, 1)
							}
							if resp != nil {
								resp.Body.Close()
							}
						}

						// Small delay to prevent overwhelming
						time.Sleep(time.Millisecond * 10)
					}
				}
			}(i)
		}

		wg.Wait()

		totalRequests := atomic.LoadInt64(&requestCount)
		totalErrors := atomic.LoadInt64(&errorCount)
		errorRate := float64(totalErrors) / float64(totalRequests)

		t.Logf("Load test results:")
		t.Logf("  Duration: %v", duration)
		t.Logf("  Workers: %d", numWorkers)
		t.Logf("  Total requests: %d", totalRequests)
		t.Logf("  Total errors: %d", totalErrors)
		t.Logf("  Error rate: %.2f%%", errorRate*100)
		t.Logf("  Requests per second: %.2f", float64(totalRequests)/duration.Seconds())

		// Error rate should be low
		assert.Less(t, errorRate, 0.05, "Error rate should be less than 5%")

		// Should handle reasonable throughput
		rps := float64(totalRequests) / duration.Seconds()
		assert.Greater(t, rps, 50.0, "Should handle at least 50 requests per second")
	})

	t.Run("Race Condition Detection", func(t *testing.T) {
		// Test for race conditions in user creation/modification
		const numGoroutines = 10
		var wg sync.WaitGroup

		// Create a user that multiple goroutines will try to modify
		createReq := &pb.CreateUserRequest{
			Email:       "race-test@example.com",
			Name:        "Race Test User",
			PhoneNumber: "090-0000-0000",
			Address:     "Race Test City",
		}

		user, err := userClient.CreateUser(ctx, createReq)
		require.NoError(t, err)
		userID := user.Id

		wg.Add(numGoroutines)

		// Multiple goroutines try to update the same user
		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer wg.Done()

				payload := fmt.Sprintf(`{
					"email": "race-test-%d@example.com",
					"name": "Race Test User %d",
					"phone_number": "090-0000-%04d",
					"address": "Race Test City %d"
				}`, id, id, id, id)

				req := httptest.NewRequest("PUT", "/api/v1/users/"+userID, strings.NewReader(payload))
				req.Header.Set("Content-Type", "application/json")
				resp, err := app.Test(req)
				if resp != nil {
					resp.Body.Close()
				}

				// We don't assert success here because race conditions might cause some to fail
				// The important thing is that the system doesn't crash
				_ = err
			}(i)
		}

		wg.Wait()

		// Verify that the user still exists and is in a valid state
		getReq := &pb.GetUserRequest{Id: userID}
		finalUser, err := userClient.GetUser(ctx, getReq)
		require.NoError(t, err)
		assert.NotNil(t, finalUser)
		assert.Equal(t, userID, finalUser.Id)
		assert.NotEmpty(t, finalUser.Email)
		assert.NotEmpty(t, finalUser.Name)
	})

	// Cleanup
	grpcServer.GracefulStop()
}