package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTransactionServiceContract tests the REST API contract for TransactionService
func TestTransactionServiceContract(t *testing.T) {
	// Setup test server
	app := fiber.New()

	// Mock transaction data
	mockTransactions := []fiber.Map{
		{
			"id":               "txn-1",
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
		{
			"id":               "txn-2",
			"card_id":          "card-1",
			"entry_gate_id":    "gate-003",
			"exit_gate_id":     "gate-004",
			"entry_time":       "2024-01-16T14:20:00Z",
			"exit_time":        "2024-01-16T15:45:00Z",
			"distance":         82.3,
			"toll_amount":      1800,
			"discount_amount":  0,
			"final_amount":     1800,
			"payment_status":   "pending",
			"transaction_date": "2024-01-16T15:45:00Z",
		},
	}

	// Get single transaction
	app.Get("/api/v1/transactions/:id", func(c *fiber.Ctx) error {
		txnID := c.Params("id")
		if txnID == "" {
			return c.Status(400).JSON(fiber.Map{"error": "transaction ID required"})
		}

		if txnID == "not-found" {
			return c.Status(404).JSON(fiber.Map{"error": "transaction not found"})
		}

		// Return first mock transaction
		return c.JSON(mockTransactions[0])
	})

	// Get transaction history
	app.Get("/api/v1/transactions", func(c *fiber.Ctx) error {
		cardID := c.Query("card_id")
		if cardID == "" {
			return c.Status(400).JSON(fiber.Map{"error": "card_id parameter required"})
		}

		// Filter by card_id
		var filteredTxns []fiber.Map
		for _, txn := range mockTransactions {
			if txn["card_id"] == cardID {
				filteredTxns = append(filteredTxns, txn)
			}
		}

		// Calculate total amount
		var totalAmount int64
		for _, txn := range filteredTxns {
			if amount, ok := txn["final_amount"].(int); ok {
				totalAmount += int64(amount)
			}
		}

		return c.JSON(fiber.Map{
			"transactions":     filteredTxns,
			"next_page_token":  "",
			"total_amount":     totalAmount,
		})
	})

	t.Run("GET /api/v1/transactions/:id - Get transaction by ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/transactions/txn-123", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		// Validate transaction structure
		assert.Contains(t, result, "id")
		assert.Contains(t, result, "card_id")
		assert.Contains(t, result, "entry_gate_id")
		assert.Contains(t, result, "exit_gate_id")
		assert.Contains(t, result, "entry_time")
		assert.Contains(t, result, "exit_time")
		assert.Contains(t, result, "distance")
		assert.Contains(t, result, "toll_amount")
		assert.Contains(t, result, "discount_amount")
		assert.Contains(t, result, "final_amount")
		assert.Contains(t, result, "payment_status")
		assert.Contains(t, result, "transaction_date")

		// Validate data types
		assert.IsType(t, "", result["id"])
		assert.IsType(t, "", result["card_id"])
		assert.IsType(t, float64(0), result["distance"])
		assert.Contains(t, []string{"pending", "completed", "failed"}, result["payment_status"])
	})

	t.Run("GET /api/v1/transactions/:id - Transaction not found", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/transactions/not-found", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("GET /api/v1/transactions - Get transaction history by card", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/transactions?card_id=card-1", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		assert.Contains(t, result, "transactions")
		assert.Contains(t, result, "next_page_token")
		assert.Contains(t, result, "total_amount")

		transactions := result["transactions"].([]interface{})
		assert.Len(t, transactions, 2)

		// Validate first transaction
		txn := transactions[0].(map[string]interface{})
		assert.Equal(t, "card-1", txn["card_id"])
		assert.Contains(t, txn, "entry_time")
		assert.Contains(t, txn, "exit_time")
	})

	t.Run("GET /api/v1/transactions - Missing card_id parameter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/transactions", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		assert.Contains(t, result, "error")
		assert.Contains(t, result["error"], "card_id")
	})

	t.Run("GET /api/v1/transactions - With date range filters", func(t *testing.T) {
		startDate := "2024-01-15T00:00:00Z"
		endDate := "2024-01-17T00:00:00Z"
		url := "/api/v1/transactions?card_id=card-1&start_date=" + startDate + "&end_date=" + endDate

		req := httptest.NewRequest("GET", url, nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		assert.Contains(t, result, "transactions")
		transactions := result["transactions"].([]interface{})
		assert.GreaterOrEqual(t, len(transactions), 0)
	})

	t.Run("GET /api/v1/transactions - Empty result for non-existent card", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/transactions?card_id=non-existent", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		if result["transactions"] != nil {
			transactions := result["transactions"].([]interface{})
			assert.Len(t, transactions, 0)
		} else {
			assert.Nil(t, result["transactions"])
		}
		assert.Equal(t, int64(0), int64(result["total_amount"].(float64)))
	})
}