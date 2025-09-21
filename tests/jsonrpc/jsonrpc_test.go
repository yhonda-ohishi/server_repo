package jsonrpc_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// JSONRPCRequest represents a JSON-RPC 2.0 request
type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
	ID      interface{} `json:"id,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response
type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
	ID      interface{} `json:"id,omitempty"`
}

// JSONRPCError represents a JSON-RPC 2.0 error
type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// TestJSONRPC20Protocol tests JSON-RPC 2.0 protocol implementation
func TestJSONRPC20Protocol(t *testing.T) {
	// Setup test server
	app := fiber.New()

	// Mock JSON-RPC endpoint
	app.Post("/jsonrpc", func(c *fiber.Ctx) error {
		var req JSONRPCRequest
		if err := c.BodyParser(&req); err != nil {
			return c.JSON(JSONRPCResponse{
				JSONRPC: "2.0",
				Error: &JSONRPCError{
					Code:    -32700,
					Message: "Parse error",
				},
				ID: nil,
			})
		}

		// Validate JSON-RPC version
		if req.JSONRPC != "2.0" {
			return c.JSON(JSONRPCResponse{
				JSONRPC: "2.0",
				Error: &JSONRPCError{
					Code:    -32600,
					Message: "Invalid Request",
				},
				ID: req.ID,
			})
		}

		// Handle different methods
		switch req.Method {
		case "user.get":
			params := req.Params.(map[string]interface{})
			userID := params["id"].(string)

			if userID == "not-found" {
				return c.JSON(JSONRPCResponse{
					JSONRPC: "2.0",
					Error: &JSONRPCError{
						Code:    -32000,
						Message: "User not found",
					},
					ID: req.ID,
				})
			}

			return c.JSON(JSONRPCResponse{
				JSONRPC: "2.0",
				Result: map[string]interface{}{
					"id":           userID,
					"email":        "jsonrpc@example.com",
					"name":         "JSON-RPC User",
					"phone_number": "090-1234-5678",
					"address":      "Tokyo, Japan",
					"status":       "active",
				},
				ID: req.ID,
			})

		case "user.create":
			params := req.Params.(map[string]interface{})

			// Validate required fields
			if params["email"] == nil || params["name"] == nil {
				return c.JSON(JSONRPCResponse{
					JSONRPC: "2.0",
					Error: &JSONRPCError{
						Code:    -32602,
						Message: "Invalid params",
						Data:    "email and name are required",
					},
					ID: req.ID,
				})
			}

			return c.JSON(JSONRPCResponse{
				JSONRPC: "2.0",
				Result: map[string]interface{}{
					"id":           "new-user-id",
					"email":        params["email"],
					"name":         params["name"],
					"phone_number": params["phone_number"],
					"address":      params["address"],
					"status":       "active",
				},
				ID: req.ID,
			})

		case "user.list":
			return c.JSON(JSONRPCResponse{
				JSONRPC: "2.0",
				Result: map[string]interface{}{
					"users": []map[string]interface{}{
						{
							"id":    "user-1",
							"email": "user1@example.com",
							"name":  "User 1",
						},
						{
							"id":    "user-2",
							"email": "user2@example.com",
							"name":  "User 2",
						},
					},
					"next_page_token": "",
				},
				ID: req.ID,
			})

		case "transaction.get":
			params := req.Params.(map[string]interface{})
			txnID := params["id"].(string)

			return c.JSON(JSONRPCResponse{
				JSONRPC: "2.0",
				Result: map[string]interface{}{
					"id":               txnID,
					"card_id":          "card-1",
					"entry_gate_id":    "gate-001",
					"exit_gate_id":     "gate-002",
					"entry_time":       "2024-01-15T08:30:00Z",
					"exit_time":        "2024-01-15T09:15:00Z",
					"distance":         45.5,
					"toll_amount":      1200,
					"discount_amount":  100,
					"final_amount":     1100,
					"payment_status":   "completed",
					"transaction_date": "2024-01-15T09:15:00Z",
				},
				ID: req.ID,
			})

		case "transaction.history":
			params := req.Params.(map[string]interface{})
			cardID := params["card_id"].(string)

			return c.JSON(JSONRPCResponse{
				JSONRPC: "2.0",
				Result: map[string]interface{}{
					"transactions": []map[string]interface{}{
						{
							"id":               "txn-1",
							"card_id":          cardID,
							"entry_gate_id":    "gate-001",
							"exit_gate_id":     "gate-002",
							"distance":         45.5,
							"toll_amount":      1200,
							"final_amount":     1100,
							"payment_status":   "completed",
						},
					},
					"next_page_token": "",
					"total_amount":    1100,
				},
				ID: req.ID,
			})

		default:
			return c.JSON(JSONRPCResponse{
				JSONRPC: "2.0",
				Error: &JSONRPCError{
					Code:    -32601,
					Message: "Method not found",
				},
				ID: req.ID,
			})
		}
	})

	// Handle batch requests
	app.Post("/jsonrpc", func(c *fiber.Ctx) error {
		var reqs []JSONRPCRequest
		if err := c.BodyParser(&reqs); err != nil {
			// If not array, try single request (handled above)
			return c.Next()
		}

		var responses []JSONRPCResponse
		for _, req := range reqs {
			// Process each request (simplified for test)
			if req.Method == "user.get" && req.Params != nil {
				responses = append(responses, JSONRPCResponse{
					JSONRPC: "2.0",
					Result: map[string]interface{}{
						"id":    "batch-user",
						"email": "batch@example.com",
						"name":  "Batch User",
					},
					ID: req.ID,
				})
			}
		}

		return c.JSON(responses)
	})

	t.Run("Single JSON-RPC Request - user.get", func(t *testing.T) {
		req := JSONRPCRequest{
			JSONRPC: "2.0",
			Method:  "user.get",
			Params: map[string]interface{}{
				"id": "user-123",
			},
			ID: 1,
		}

		body, _ := json.Marshal(req)
		httpReq := httptest.NewRequest("POST", "/jsonrpc", bytes.NewReader(body))
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(httpReq)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var jsonResp JSONRPCResponse
		err = json.NewDecoder(resp.Body).Decode(&jsonResp)
		require.NoError(t, err)

		assert.Equal(t, "2.0", jsonResp.JSONRPC)
		assert.Equal(t, float64(1), jsonResp.ID)
		assert.Nil(t, jsonResp.Error)
		assert.NotNil(t, jsonResp.Result)

		result := jsonResp.Result.(map[string]interface{})
		assert.Equal(t, "user-123", result["id"])
		assert.Contains(t, result, "email")
		assert.Contains(t, result, "name")
	})

	t.Run("Single JSON-RPC Request - user.create", func(t *testing.T) {
		req := JSONRPCRequest{
			JSONRPC: "2.0",
			Method:  "user.create",
			Params: map[string]interface{}{
				"email":        "newuser@example.com",
				"name":         "New User",
				"phone_number": "090-9999-8888",
				"address":      "Kyoto, Japan",
			},
			ID: 2,
		}

		body, _ := json.Marshal(req)
		httpReq := httptest.NewRequest("POST", "/jsonrpc", bytes.NewReader(body))
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(httpReq)
		require.NoError(t, err)
		defer resp.Body.Close()

		var jsonResp JSONRPCResponse
		err = json.NewDecoder(resp.Body).Decode(&jsonResp)
		require.NoError(t, err)

		assert.Equal(t, "2.0", jsonResp.JSONRPC)
		assert.Equal(t, float64(2), jsonResp.ID)
		assert.Nil(t, jsonResp.Error)

		result := jsonResp.Result.(map[string]interface{})
		assert.Equal(t, "newuser@example.com", result["email"])
		assert.Equal(t, "New User", result["name"])
	})

	t.Run("JSON-RPC Error - Method not found", func(t *testing.T) {
		req := JSONRPCRequest{
			JSONRPC: "2.0",
			Method:  "nonexistent.method",
			ID:      3,
		}

		body, _ := json.Marshal(req)
		httpReq := httptest.NewRequest("POST", "/jsonrpc", bytes.NewReader(body))
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(httpReq)
		require.NoError(t, err)
		defer resp.Body.Close()

		var jsonResp JSONRPCResponse
		err = json.NewDecoder(resp.Body).Decode(&jsonResp)
		require.NoError(t, err)

		assert.Equal(t, "2.0", jsonResp.JSONRPC)
		assert.Equal(t, float64(3), jsonResp.ID)
		assert.NotNil(t, jsonResp.Error)
		assert.Equal(t, -32601, jsonResp.Error.Code)
		assert.Contains(t, jsonResp.Error.Message, "Method not found")
	})

	t.Run("JSON-RPC Error - Invalid params", func(t *testing.T) {
		req := JSONRPCRequest{
			JSONRPC: "2.0",
			Method:  "user.create",
			Params: map[string]interface{}{
				"name": "Missing Email",
			},
			ID: 4,
		}

		body, _ := json.Marshal(req)
		httpReq := httptest.NewRequest("POST", "/jsonrpc", bytes.NewReader(body))
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(httpReq)
		require.NoError(t, err)
		defer resp.Body.Close()

		var jsonResp JSONRPCResponse
		err = json.NewDecoder(resp.Body).Decode(&jsonResp)
		require.NoError(t, err)

		assert.NotNil(t, jsonResp.Error)
		assert.Equal(t, -32602, jsonResp.Error.Code)
		assert.Contains(t, jsonResp.Error.Message, "Invalid params")
	})

	t.Run("JSON-RPC Transaction Methods", func(t *testing.T) {
		req := JSONRPCRequest{
			JSONRPC: "2.0",
			Method:  "transaction.get",
			Params: map[string]interface{}{
				"id": "txn-123",
			},
			ID: 5,
		}

		body, _ := json.Marshal(req)
		httpReq := httptest.NewRequest("POST", "/jsonrpc", bytes.NewReader(body))
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(httpReq)
		require.NoError(t, err)
		defer resp.Body.Close()

		var jsonResp JSONRPCResponse
		err = json.NewDecoder(resp.Body).Decode(&jsonResp)
		require.NoError(t, err)

		assert.Nil(t, jsonResp.Error)
		result := jsonResp.Result.(map[string]interface{})
		assert.Equal(t, "txn-123", result["id"])
		assert.Contains(t, result, "card_id")
		assert.Contains(t, result, "toll_amount")
	})

	t.Run("JSON-RPC Notification (no ID)", func(t *testing.T) {
		req := JSONRPCRequest{
			JSONRPC: "2.0",
			Method:  "user.get",
			Params: map[string]interface{}{
				"id": "user-notification",
			},
			// No ID - this is a notification
		}

		body, _ := json.Marshal(req)
		httpReq := httptest.NewRequest("POST", "/jsonrpc", bytes.NewReader(body))
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(httpReq)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Notifications should not return a response, or return empty response
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Invalid JSON-RPC version", func(t *testing.T) {
		req := JSONRPCRequest{
			JSONRPC: "1.0", // Invalid version
			Method:  "user.get",
			ID:      6,
		}

		body, _ := json.Marshal(req)
		httpReq := httptest.NewRequest("POST", "/jsonrpc", bytes.NewReader(body))
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(httpReq)
		require.NoError(t, err)
		defer resp.Body.Close()

		var jsonResp JSONRPCResponse
		err = json.NewDecoder(resp.Body).Decode(&jsonResp)
		require.NoError(t, err)

		assert.NotNil(t, jsonResp.Error)
		assert.Equal(t, -32600, jsonResp.Error.Code)
		assert.Contains(t, jsonResp.Error.Message, "Invalid Request")
	})
}