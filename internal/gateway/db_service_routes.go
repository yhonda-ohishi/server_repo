package gateway

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/gofiber/fiber/v2"
	dbproto "github.com/yhonda-ohishi/db_service/src/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// DBServiceRoutes handles REST routes for db_service
type DBServiceRoutes struct {
	conn *grpc.ClientConn
}

// NewDBServiceRoutes creates a new db_service route handler
func NewDBServiceRoutes(conn *grpc.ClientConn) *DBServiceRoutes {
	return &DBServiceRoutes{
		conn: conn,
	}
}

// RegisterRoutes registers all db_service REST endpoints
func (r *DBServiceRoutes) RegisterRoutes(app *fiber.App) {
	// Create API group for db_service
	api := app.Group("/api/v1/db")

	// ETCMeisai endpoints
	api.Get("/etc-meisai", r.listETCMeisai)
	api.Get("/etc-meisai/:id", r.getETCMeisai)
	api.Post("/etc-meisai", r.createETCMeisai)
	api.Put("/etc-meisai/:id", r.updateETCMeisai)
	api.Delete("/etc-meisai/:id", r.deleteETCMeisai)

	// DTakoUriageKeihi endpoints
	api.Post("/dtako-uriage-keihi", r.createDTakoUriageKeihi)

	// DTakoFerryRows endpoints
	api.Post("/dtako-ferry-rows", r.createDTakoFerryRows)

	// ETCMeisaiMapping endpoints
	api.Post("/etc-meisai-mapping", r.createETCMeisaiMapping)
}

// ETCMeisai handlers

func (r *DBServiceRoutes) listETCMeisai(c *fiber.Ctx) error {
	if r.conn == nil {
		return c.Status(503).JSON(fiber.Map{
			"error": "Service unavailable",
		})
	}

	client := dbproto.NewETCMeisaiServiceClient(r.conn)

	// Parse query parameters
	req := &dbproto.ListETCMeisaiRequest{}
	if hash := c.Query("hash"); hash != "" {
		req.Hash = &hash
	}
	if startDate := c.Query("start_date"); startDate != "" {
		req.StartDate = &startDate
	}
	if endDate := c.Query("end_date"); endDate != "" {
		req.EndDate = &endDate
	}

	resp, err := client.List(context.Background(), req)
	if err != nil {
		return handleGRPCError(c, err)
	}

	return c.JSON(fiber.Map{
		"items":       resp.Items,
		"total_count": resp.TotalCount,
	})
}

func (r *DBServiceRoutes) getETCMeisai(c *fiber.Ctx) error {
	if r.conn == nil {
		return c.Status(503).JSON(fiber.Map{
			"error": "Service unavailable",
		})
	}

	idStr := c.Params("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid ID format",
		})
	}

	client := dbproto.NewETCMeisaiServiceClient(r.conn)
	resp, err := client.Get(context.Background(), &dbproto.GetETCMeisaiRequest{
		Id: id,
	})
	if err != nil {
		return handleGRPCError(c, err)
	}

	return c.JSON(resp.EtcMeisai)
}

func (r *DBServiceRoutes) createETCMeisai(c *fiber.Ctx) error {
	if r.conn == nil {
		return c.Status(503).JSON(fiber.Map{
			"error": "Service unavailable",
		})
	}

	var etcMeisai dbproto.ETCMeisai
	if err := c.BodyParser(&etcMeisai); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	client := dbproto.NewETCMeisaiServiceClient(r.conn)
	resp, err := client.Create(context.Background(), &dbproto.CreateETCMeisaiRequest{
		EtcMeisai: &etcMeisai,
	})
	if err != nil {
		return handleGRPCError(c, err)
	}

	return c.Status(201).JSON(resp.EtcMeisai)
}

func (r *DBServiceRoutes) updateETCMeisai(c *fiber.Ctx) error {
	if r.conn == nil {
		return c.Status(503).JSON(fiber.Map{
			"error": "Service unavailable",
		})
	}

	idStr := c.Params("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid ID format",
		})
	}

	var etcMeisai dbproto.ETCMeisai
	if err := c.BodyParser(&etcMeisai); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	etcMeisai.Id = id

	client := dbproto.NewETCMeisaiServiceClient(r.conn)
	resp, err := client.Update(context.Background(), &dbproto.UpdateETCMeisaiRequest{
		EtcMeisai: &etcMeisai,
	})
	if err != nil {
		return handleGRPCError(c, err)
	}

	return c.JSON(resp.EtcMeisai)
}

func (r *DBServiceRoutes) deleteETCMeisai(c *fiber.Ctx) error {
	if r.conn == nil {
		return c.Status(503).JSON(fiber.Map{
			"error": "Service unavailable",
		})
	}

	idStr := c.Params("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid ID format",
		})
	}

	client := dbproto.NewETCMeisaiServiceClient(r.conn)
	_, err = client.Delete(context.Background(), &dbproto.DeleteETCMeisaiRequest{
		Id: id,
	})
	if err != nil {
		return handleGRPCError(c, err)
	}

	return c.SendStatus(204)
}

// DTakoUriageKeihi handlers

func (r *DBServiceRoutes) createDTakoUriageKeihi(c *fiber.Ctx) error {
	if r.conn == nil {
		return c.Status(503).JSON(fiber.Map{
			"error": "Service unavailable",
		})
	}

	var dtakoUriageKeihi dbproto.DTakoUriageKeihi
	if err := c.BodyParser(&dtakoUriageKeihi); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	client := dbproto.NewDTakoUriageKeihiServiceClient(r.conn)
	resp, err := client.Create(context.Background(), &dbproto.CreateDTakoUriageKeihiRequest{
		DtakoUriageKeihi: &dtakoUriageKeihi,
	})
	if err != nil {
		return handleGRPCError(c, err)
	}

	return c.Status(201).JSON(resp.DtakoUriageKeihi)
}

// DTakoFerryRows handlers

func (r *DBServiceRoutes) createDTakoFerryRows(c *fiber.Ctx) error {
	if r.conn == nil {
		return c.Status(503).JSON(fiber.Map{
			"error": "Service unavailable",
		})
	}

	// Parse JSON body manually to handle field types correctly
	var body map[string]interface{}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid JSON body",
		})
	}

	// Create DTakoFerryRows from parsed data
	dtakoFerryRows := &dbproto.DTakoFerryRows{}

	if v, ok := body["unko_no"].(string); ok {
		dtakoFerryRows.UnkoNo = v
	}
	if v, ok := body["unko_date"].(string); ok {
		dtakoFerryRows.UnkoDate = v
	}
	if v, ok := body["yomitori_date"].(string); ok {
		dtakoFerryRows.YomitoriDate = v
	}
	if v, ok := body["jigyosho_cd"].(float64); ok {
		dtakoFerryRows.JigyoshoCd = int32(v)
	}
	if v, ok := body["jigyosho_name"].(string); ok {
		dtakoFerryRows.JigyoshoName = v
	}
	if v, ok := body["sharyo_cd"].(float64); ok {
		dtakoFerryRows.SharyoCd = int32(v)
	}
	if v, ok := body["sharyo_name"].(string); ok {
		dtakoFerryRows.SharyoName = v
	}
	if v, ok := body["jomuin_cd1"].(float64); ok {
		dtakoFerryRows.JomuinCd1 = int32(v)
	}
	if v, ok := body["jomuin_name1"].(string); ok {
		dtakoFerryRows.JomuinName1 = v
	}

	client := dbproto.NewDTakoFerryRowsServiceClient(r.conn)
	resp, err := client.Create(context.Background(), &dbproto.CreateDTakoFerryRowsRequest{
		DtakoFerryRows: dtakoFerryRows,
	})
	if err != nil {
		return handleGRPCError(c, err)
	}

	return c.Status(201).JSON(resp.DtakoFerryRows)
}

// ETCMeisaiMapping handlers

func (r *DBServiceRoutes) createETCMeisaiMapping(c *fiber.Ctx) error {
	if r.conn == nil {
		return c.Status(503).JSON(fiber.Map{
			"error": "Service unavailable",
		})
	}

	var etcMeisaiMapping dbproto.ETCMeisaiMapping
	if err := c.BodyParser(&etcMeisaiMapping); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	client := dbproto.NewETCMeisaiMappingServiceClient(r.conn)
	resp, err := client.Create(context.Background(), &dbproto.CreateETCMeisaiMappingRequest{
		EtcMeisaiMapping: &etcMeisaiMapping,
	})
	if err != nil {
		return handleGRPCError(c, err)
	}

	return c.Status(201).JSON(resp.EtcMeisaiMapping)
}

// handleGRPCError converts gRPC errors to HTTP status codes
func handleGRPCError(c *fiber.Ctx, err error) error {
	st, ok := status.FromError(err)
	if !ok {
		return c.Status(500).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	var httpStatus int
	switch st.Code() {
	case codes.NotFound:
		httpStatus = 404
	case codes.InvalidArgument:
		httpStatus = 400
	case codes.AlreadyExists:
		httpStatus = 409
	case codes.PermissionDenied:
		httpStatus = 403
	case codes.Unauthenticated:
		httpStatus = 401
	case codes.ResourceExhausted:
		httpStatus = 429
	case codes.FailedPrecondition:
		httpStatus = 412
	case codes.Unimplemented:
		httpStatus = 501
	case codes.Unavailable:
		httpStatus = 503
	default:
		httpStatus = 500
	}

	return c.Status(httpStatus).JSON(fiber.Map{
		"error": st.Message(),
		"code":  st.Code().String(),
	})
}