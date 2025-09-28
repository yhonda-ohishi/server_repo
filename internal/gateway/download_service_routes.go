package gateway

import (
	"context"
	"encoding/json"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	etcpb "github.com/yhonda-ohishi/etc_meisai_scraper/src/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// DownloadServiceRoutes handles REST routes for etc_meisai_scraper DownloadService
type DownloadServiceRoutes struct {
	conn *grpc.ClientConn
}

// NewDownloadServiceRoutes creates a new download service route handler
func NewDownloadServiceRoutes(conn *grpc.ClientConn) *DownloadServiceRoutes {
	return &DownloadServiceRoutes{
		conn: conn,
	}
}

// RegisterRoutes registers all download service REST endpoints
func (r *DownloadServiceRoutes) RegisterRoutes(app *fiber.App) {
	// Create API group for download service
	api := app.Group("/etc_meisai_scraper/v1")

	// Download endpoints
	api.Post("/download/sync", r.downloadSync)
	api.Post("/download/async", r.downloadAsync)
	api.Get("/download/jobs/:job_id", r.getJobStatus)
	api.Get("/accounts", r.getAllAccountIDs)
}

// downloadSync handles synchronous download
func (r *DownloadServiceRoutes) downloadSync(c *fiber.Ctx) error {
	var req etcpb.DownloadRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body: " + err.Error(),
		})
	}

	client := etcpb.NewDownloadServiceClient(r.conn)
	resp, err := client.DownloadSync(context.Background(), &req)
	if err != nil {
		st, _ := status.FromError(err)
		if st.Code() == codes.NotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": st.Message(),
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(resp)
}

// downloadAsync handles asynchronous download
func (r *DownloadServiceRoutes) downloadAsync(c *fiber.Ctx) error {
	var req etcpb.DownloadRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body: " + err.Error(),
		})
	}

	// Convert accounts array to full credentials if provided
	if len(req.Accounts) > 0 {
		for i, account := range req.Accounts {
			fullCredentials := r.getFullAccountCredentials(account)
			if fullCredentials != "" {
				req.Accounts[i] = fullCredentials
			}
		}
	}

	client := etcpb.NewDownloadServiceClient(r.conn)
	resp, err := client.DownloadAsync(context.Background(), &req)
	if err != nil {
		st, _ := status.FromError(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": st.Message(),
		})
	}

	return c.JSON(resp)
}

// getJobStatus retrieves job status
func (r *DownloadServiceRoutes) getJobStatus(c *fiber.Ctx) error {
	jobID := c.Params("job_id")
	if jobID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Job ID is required",
		})
	}

	req := &etcpb.GetJobStatusRequest{
		JobId: jobID,
	}

	client := etcpb.NewDownloadServiceClient(r.conn)
	resp, err := client.GetJobStatus(context.Background(), req)
	if err != nil {
		st, _ := status.FromError(err)
		if st.Code() == codes.NotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Job not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(resp)
}

// getAllAccountIDs retrieves all account IDs
func (r *DownloadServiceRoutes) getAllAccountIDs(c *fiber.Ctx) error {
	req := &etcpb.GetAllAccountIDsRequest{}

	client := etcpb.NewDownloadServiceClient(r.conn)
	resp, err := client.GetAllAccountIDs(context.Background(), req)
	if err != nil {
		st, _ := status.FromError(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": st.Message(),
		})
	}

	return c.JSON(resp)
}

// getFullAccountCredentials looks up full credentials from environment variables
func (r *DownloadServiceRoutes) getFullAccountCredentials(accountID string) string {
	// If already in full format, return as-is
	if strings.Contains(accountID, ":") {
		return accountID
	}

	// Check corporate accounts
	corporateAccounts := os.Getenv("ETC_CORPORATE_ACCOUNTS")
	if corporateAccounts != "" {
		for _, accountStr := range strings.Split(corporateAccounts, ",") {
			parts := strings.Split(accountStr, ":")
			if len(parts) >= 2 && parts[0] == accountID {
				return accountStr // Return full "accountID:password" format
			}
		}
	}

	// Check personal accounts
	personalAccounts := os.Getenv("ETC_PERSONAL_ACCOUNTS")
	if personalAccounts != "" {
		for _, accountStr := range strings.Split(personalAccounts, ",") {
			parts := strings.Split(accountStr, ":")
			if len(parts) >= 2 && parts[0] == accountID {
				return accountStr // Return full "accountID:password" format
			}
		}
	}

	// If not found, return original accountID
	return accountID
}

// Helper function to convert protobuf message to JSON
func toJSON(msg interface{}) ([]byte, error) {
	return json.Marshal(msg)
}