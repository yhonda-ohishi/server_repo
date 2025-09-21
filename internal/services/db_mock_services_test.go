package services_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yhonda-ohishi/db-handler-server/internal/client"
	"github.com/yhonda-ohishi/db-handler-server/internal/services"
	dbproto "github.com/yhonda-ohishi/db_service/src/proto"
	"google.golang.org/grpc"
)

func TestDBServiceIntegrationViaBufconn(t *testing.T) {
	// Create bufconn client
	bufClient := client.NewBufconnClient()
	defer bufClient.Close()

	// Create gRPC server
	server := grpc.NewServer()

	// Register services using single mode registry
	registry := services.NewServiceRegistryForSingleMode()
	registry.RegisterAll(server)

	// Start server
	listener := bufClient.GetListener()
	go func() {
		if err := server.Serve(listener); err != nil {
			t.Logf("Server stopped: %v", err)
		}
	}()
	defer server.Stop()

	// Create client connection
	ctx := context.Background()
	conn, err := bufClient.GetConnection(ctx)
	require.NoError(t, err)
	defer conn.Close()

	t.Run("ETCMeisaiService_CRUD", func(t *testing.T) {
		client := dbproto.NewETCMeisaiServiceClient(conn)

		// Test List - should have initial mock data
		listResp, err := client.List(ctx, &dbproto.ListETCMeisaiRequest{})
		require.NoError(t, err)
		assert.NotNil(t, listResp)
		assert.GreaterOrEqual(t, len(listResp.Items), 2, "Should have at least 2 mock items")

		// Test Get - get first mock item
		getResp, err := client.Get(ctx, &dbproto.GetETCMeisaiRequest{Id: 1})
		require.NoError(t, err)
		assert.NotNil(t, getResp.EtcMeisai)
		assert.Equal(t, int64(1), getResp.EtcMeisai.Id)
		assert.Equal(t, "東京IC", getResp.EtcMeisai.IcFr)
		assert.Equal(t, "横浜IC", getResp.EtcMeisai.IcTo)
		assert.Equal(t, int32(1500), getResp.EtcMeisai.Price)

		// Test Create
		newMeisai := &dbproto.ETCMeisai{
			DateTo:     "2024-02-01",
			DateToDate: "2024-02-01",
			IcFr:       "京都IC",
			IcTo:       "神戸IC",
			Price:      2200,
			Shashu:     1,
			EtcNum:     "1111-2222-3333-4444",
			Hash:       "xyz789",
		}
		createResp, err := client.Create(ctx, &dbproto.CreateETCMeisaiRequest{
			EtcMeisai: newMeisai,
		})
		require.NoError(t, err)
		assert.NotNil(t, createResp.EtcMeisai)
		assert.Greater(t, createResp.EtcMeisai.Id, int64(2), "ID should be assigned")

		// Test Update
		updateMeisai := createResp.EtcMeisai
		updateMeisai.Price = 2500
		updateResp, err := client.Update(ctx, &dbproto.UpdateETCMeisaiRequest{
			EtcMeisai: updateMeisai,
		})
		require.NoError(t, err)
		assert.Equal(t, int32(2500), updateResp.EtcMeisai.Price)

		// Test Delete
		_, err = client.Delete(ctx, &dbproto.DeleteETCMeisaiRequest{
			Id: createResp.EtcMeisai.Id,
		})
		require.NoError(t, err)

		// Verify deletion
		_, err = client.Get(ctx, &dbproto.GetETCMeisaiRequest{Id: createResp.EtcMeisai.Id})
		assert.Error(t, err, "Should return not found error after deletion")
	})

	t.Run("DTakoUriageKeihiService_Create", func(t *testing.T) {
		client := dbproto.NewDTakoUriageKeihiServiceClient(conn)

		// Test Create with auto-generated SrchId
		newKeihi := &dbproto.DTakoUriageKeihi{
			Datetime:    "2024-01-15T10:30:00Z",
			KeihiC:      100,
			Price:       5000.50,
			DtakoRowId:  "DTAKO001",
			DtakoRowIdR: "DTAKO001R",
		}
		createResp, err := client.Create(ctx, &dbproto.CreateDTakoUriageKeihiRequest{
			DtakoUriageKeihi: newKeihi,
		})
		require.NoError(t, err)
		assert.NotNil(t, createResp.DtakoUriageKeihi)
		assert.NotEmpty(t, createResp.DtakoUriageKeihi.SrchId)
		assert.Contains(t, createResp.DtakoUriageKeihi.SrchId, "SRCH", "SrchId should be auto-generated")
	})

	t.Run("DTakoFerryRowsService_Create", func(t *testing.T) {
		client := dbproto.NewDTakoFerryRowsServiceClient(conn)

		// Test Create with auto-generated int32 ID
		newFerry := &dbproto.DTakoFerryRows{
			UnkoNo:       "F001",
			UnkoDate:     "2024-01-15",
			YomitoriDate: "2024-01-16",
			JigyoshoCd:   1,
			JigyoshoName: "東京事業所",
			SharyoCd:     100,
			SharyoName:   "フェリー1号",
			JomuinCd1:    1001,
			JomuinName1:  "山田太郎",
		}
		createResp, err := client.Create(ctx, &dbproto.CreateDTakoFerryRowsRequest{
			DtakoFerryRows: newFerry,
		})
		require.NoError(t, err)
		assert.NotNil(t, createResp.DtakoFerryRows)
		assert.Greater(t, createResp.DtakoFerryRows.Id, int32(0), "ID should be assigned")
		assert.Equal(t, "F001", createResp.DtakoFerryRows.UnkoNo)
	})

	t.Run("ETCMeisaiMappingService_Create", func(t *testing.T) {
		client := dbproto.NewETCMeisaiMappingServiceClient(conn)

		// Test Create mapping
		newMapping := &dbproto.ETCMeisaiMapping{
			EtcMeisaiHash: "abc123",
			DtakoRowId:    "DTAKO001",
			CreatedAt:     "2024-01-15T10:00:00Z",
			UpdatedAt:     "2024-01-15T10:00:00Z",
			CreatedBy:     "test_user",
		}
		createResp, err := client.Create(ctx, &dbproto.CreateETCMeisaiMappingRequest{
			EtcMeisaiMapping: newMapping,
		})
		require.NoError(t, err)
		assert.NotNil(t, createResp.EtcMeisaiMapping)
		assert.Greater(t, createResp.EtcMeisaiMapping.Id, int64(0), "ID should be assigned")
		assert.Equal(t, "abc123", createResp.EtcMeisaiMapping.EtcMeisaiHash)
	})
}