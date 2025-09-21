package client

import (
	"context"
	"fmt"
	"time"

	dbproto "github.com/yhonda-ohishi/db_service/src/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// DBServiceClient wraps gRPC clients for db_service
type DBServiceClient struct {
	conn                      *grpc.ClientConn
	etcMeisaiClient          dbproto.ETCMeisaiServiceClient
	dtakoUriageKeihiClient   dbproto.DTakoUriageKeihiServiceClient
	dtakoFerryRowsClient     dbproto.DTakoFerryRowsServiceClient
	etcMeisaiMappingClient   dbproto.ETCMeisaiMappingServiceClient
}

// NewDBServiceClient creates a new client for db_service
func NewDBServiceClient(address string) (*DBServiceClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Connect to db_service gRPC server
	conn, err := grpc.DialContext(ctx, address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to db_service: %w", err)
	}

	return &DBServiceClient{
		conn:                      conn,
		etcMeisaiClient:          dbproto.NewETCMeisaiServiceClient(conn),
		dtakoUriageKeihiClient:   dbproto.NewDTakoUriageKeihiServiceClient(conn),
		dtakoFerryRowsClient:     dbproto.NewDTakoFerryRowsServiceClient(conn),
		etcMeisaiMappingClient:   dbproto.NewETCMeisaiMappingServiceClient(conn),
	}, nil
}

// GetETCMeisaiClient returns the ETC明細 service client
func (c *DBServiceClient) GetETCMeisaiClient() dbproto.ETCMeisaiServiceClient {
	return c.etcMeisaiClient
}

// GetDTakoUriageKeihiClient returns the 経費精算 service client
func (c *DBServiceClient) GetDTakoUriageKeihiClient() dbproto.DTakoUriageKeihiServiceClient {
	return c.dtakoUriageKeihiClient
}

// GetDTakoFerryRowsClient returns the フェリー運行 service client
func (c *DBServiceClient) GetDTakoFerryRowsClient() dbproto.DTakoFerryRowsServiceClient {
	return c.dtakoFerryRowsClient
}

// GetETCMeisaiMappingClient returns the ETC明細マッピング service client
func (c *DBServiceClient) GetETCMeisaiMappingClient() dbproto.ETCMeisaiMappingServiceClient {
	return c.etcMeisaiMappingClient
}

// Close closes the gRPC connection
func (c *DBServiceClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}