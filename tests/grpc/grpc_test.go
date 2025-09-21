package grpc_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yhonda-ohishi/db-handler-server/internal/client"
	"github.com/yhonda-ohishi/db-handler-server/internal/services"
	pb "github.com/yhonda-ohishi/db-handler-server/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// TestGRPCProtocol tests native gRPC communication
func TestGRPCProtocol(t *testing.T) {
	// Setup bufconn server for testing
	bufconnClient := client.NewBufconnClient()

	// Create gRPC server
	server := grpc.NewServer()

	// Register services
	registry := services.NewServiceRegistry()
	registry.RegisterAll(server)

	// Enable reflection for testing
	reflection.Register(server)

	// Start server
	listener := bufconnClient.GetListener()
	go func() {
		_ = server.Serve(listener)
	}()

	// Create client connection
	ctx := context.Background()
	conn, err := bufconnClient.GetConnection(ctx)
	require.NoError(t, err)
	defer conn.Close()

	// Test User Service
	t.Run("UserService gRPC Tests", func(t *testing.T) {
		userClient := pb.NewUserServiceClient(conn)

		t.Run("CreateUser", func(t *testing.T) {
			req := &pb.CreateUserRequest{
				Email:       "grpc@example.com",
				Name:        "gRPC Test User",
				PhoneNumber: "090-1111-2222",
				Address:     "Tokyo, Japan",
			}

			resp, err := userClient.CreateUser(ctx, req)
			require.NoError(t, err)
			assert.NotEmpty(t, resp.Id)
			assert.Equal(t, req.Email, resp.Email)
			assert.Equal(t, req.Name, resp.Name)
			assert.Equal(t, pb.UserStatus_USER_STATUS_ACTIVE, resp.Status)
		})

		t.Run("GetUser", func(t *testing.T) {
			// First create a user
			createReq := &pb.CreateUserRequest{
				Email:       "get@example.com",
				Name:        "Get Test User",
				PhoneNumber: "090-3333-4444",
				Address:     "Osaka, Japan",
			}
			createResp, err := userClient.CreateUser(ctx, createReq)
			require.NoError(t, err)

			// Then get the user
			getReq := &pb.GetUserRequest{Id: createResp.Id}
			getResp, err := userClient.GetUser(ctx, getReq)
			require.NoError(t, err)

			assert.Equal(t, createResp.Id, getResp.Id)
			assert.Equal(t, createResp.Email, getResp.Email)
			assert.Equal(t, createResp.Name, getResp.Name)
		})

		t.Run("UpdateUser", func(t *testing.T) {
			// First create a user
			createReq := &pb.CreateUserRequest{
				Email:       "update@example.com",
				Name:        "Update Test User",
				PhoneNumber: "090-5555-6666",
				Address:     "Kyoto, Japan",
			}
			createResp, err := userClient.CreateUser(ctx, createReq)
			require.NoError(t, err)

			// Then update the user
			updateReq := &pb.UpdateUserRequest{
				Id:          createResp.Id,
				Email:       "updated@example.com",
				Name:        "Updated Name",
				PhoneNumber: "090-7777-8888",
				Address:     "Updated Address",
			}
			updateResp, err := userClient.UpdateUser(ctx, updateReq)
			require.NoError(t, err)

			assert.Equal(t, updateReq.Id, updateResp.Id)
			assert.Equal(t, updateReq.Email, updateResp.Email)
			assert.Equal(t, updateReq.Name, updateResp.Name)
			assert.Equal(t, updateReq.PhoneNumber, updateResp.PhoneNumber)
			assert.Equal(t, updateReq.Address, updateResp.Address)
		})

		t.Run("ListUsers", func(t *testing.T) {
			req := &pb.ListUsersRequest{
				PageSize: 10,
			}

			resp, err := userClient.ListUsers(ctx, req)
			require.NoError(t, err)
			assert.NotNil(t, resp.Users)
			assert.GreaterOrEqual(t, len(resp.Users), 0)
		})

		t.Run("DeleteUser", func(t *testing.T) {
			// First create a user
			createReq := &pb.CreateUserRequest{
				Email:       "delete@example.com",
				Name:        "Delete Test User",
				PhoneNumber: "090-9999-0000",
				Address:     "Fukuoka, Japan",
			}
			createResp, err := userClient.CreateUser(ctx, createReq)
			require.NoError(t, err)

			// Then delete the user
			deleteReq := &pb.DeleteUserRequest{Id: createResp.Id}
			_, err = userClient.DeleteUser(ctx, deleteReq)
			require.NoError(t, err)

			// Verify user is deleted by trying to get it
			getReq := &pb.GetUserRequest{Id: createResp.Id}
			_, err = userClient.GetUser(ctx, getReq)
			assert.Error(t, err) // Should return not found error
		})
	})

	// Test Transaction Service
	t.Run("TransactionService gRPC Tests", func(t *testing.T) {
		txnClient := pb.NewTransactionServiceClient(conn)

		t.Run("GetTransaction", func(t *testing.T) {
			req := &pb.GetTransactionRequest{
				Id: "txn-1",
			}

			resp, err := txnClient.GetTransaction(ctx, req)
			require.NoError(t, err)
			assert.NotEmpty(t, resp.Id)
			assert.NotEmpty(t, resp.CardId)
			assert.Greater(t, resp.TollAmount, int64(0))
		})

		t.Run("GetTransactionHistory", func(t *testing.T) {
			now := time.Now()
			startDate := timestamppb.New(now.Add(-7 * 24 * time.Hour))
			endDate := timestamppb.New(now)

			req := &pb.GetTransactionHistoryRequest{
				CardId:    "card-1",
				StartDate: startDate,
				EndDate:   endDate,
				PageSize:  10,
			}

			resp, err := txnClient.GetTransactionHistory(ctx, req)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(resp.Transactions), 0)
			assert.GreaterOrEqual(t, resp.TotalAmount, int64(0))

			// Validate transaction structure if any exist
			if len(resp.Transactions) > 0 {
				txn := resp.Transactions[0]
				assert.NotEmpty(t, txn.Id)
				assert.NotEmpty(t, txn.CardId)
				assert.NotEmpty(t, txn.EntryGateId)
				assert.NotEmpty(t, txn.ExitGateId)
				assert.NotNil(t, txn.EntryTime)
				assert.NotNil(t, txn.ExitTime)
				assert.Greater(t, txn.Distance, 0.0)
				assert.GreaterOrEqual(t, txn.TollAmount, int64(0))
				assert.GreaterOrEqual(t, txn.FinalAmount, int64(0))
			}
		})
	})

	// Test error handling
	t.Run("Error Handling", func(t *testing.T) {
		userClient := pb.NewUserServiceClient(conn)

		t.Run("GetUser with invalid ID", func(t *testing.T) {
			req := &pb.GetUserRequest{Id: ""}
			_, err := userClient.GetUser(ctx, req)
			assert.Error(t, err)
		})

		t.Run("CreateUser with invalid data", func(t *testing.T) {
			req := &pb.CreateUserRequest{
				Email: "", // Invalid empty email
				Name:  "",
			}
			_, err := userClient.CreateUser(ctx, req)
			assert.Error(t, err)
		})
	})

	// Test connection
	t.Run("Connection Test", func(t *testing.T) {
		t.Run("Connection state", func(t *testing.T) {
			state := conn.GetState()
			assert.NotEqual(t, "SHUTDOWN", state.String())
		})
	})

	// Cleanup
	server.GracefulStop()
}