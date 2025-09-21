package rest_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUserServiceContract tests the REST API contract for UserService
func TestUserServiceContract(t *testing.T) {
	// Setup Fiber test app
	app := fiber.New()

	// Add test routes that mirror our expected API
	app.Get("/api/v1/users", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"users": []fiber.Map{
				{
					"id":           "user-1",
					"email":        "test1@example.com",
					"name":         "Test User 1",
					"phone_number": "090-1234-5678",
					"address":      "Tokyo, Japan",
					"status":       "active",
					"created_at":   time.Now().Format(time.RFC3339),
					"updated_at":   time.Now().Format(time.RFC3339),
				},
				{
					"id":           "user-2",
					"email":        "test2@example.com",
					"name":         "Test User 2",
					"phone_number": "090-8765-4321",
					"address":      "Osaka, Japan",
					"status":       "active",
					"created_at":   time.Now().Format(time.RFC3339),
					"updated_at":   time.Now().Format(time.RFC3339),
				},
			},
			"next_page_token": "",
		})
	})

	app.Get("/api/v1/users/:id", func(c *fiber.Ctx) error {
		userID := c.Params("id")
		if userID == "" {
			return c.Status(400).JSON(fiber.Map{"error": "user ID required"})
		}

		if userID == "not-found" {
			return c.Status(404).JSON(fiber.Map{"error": "user not found"})
		}

		return c.JSON(fiber.Map{
			"id":           userID,
			"email":        "test@example.com",
			"name":         "Test User",
			"phone_number": "090-1234-5678",
			"address":      "Tokyo, Japan",
			"status":       "active",
			"created_at":   time.Now().Format(time.RFC3339),
			"updated_at":   time.Now().Format(time.RFC3339),
		})
	})

	app.Post("/api/v1/users", func(c *fiber.Ctx) error {
		var req map[string]interface{}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
		}

		// Validate required fields
		if req["email"] == nil || req["name"] == nil {
			return c.Status(400).JSON(fiber.Map{"error": "email and name are required"})
		}

		// Return created user
		return c.Status(201).JSON(fiber.Map{
			"id":           "user-new",
			"email":        req["email"],
			"name":         req["name"],
			"phone_number": req["phone_number"],
			"address":      req["address"],
			"status":       "active",
			"created_at":   time.Now().Format(time.RFC3339),
			"updated_at":   time.Now().Format(time.RFC3339),
		})
	})

	app.Put("/api/v1/users/:id", func(c *fiber.Ctx) error {
		userID := c.Params("id")
		if userID == "not-found" {
			return c.Status(404).JSON(fiber.Map{"error": "user not found"})
		}

		var req map[string]interface{}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
		}

		return c.JSON(fiber.Map{
			"id":           userID,
			"email":        req["email"],
			"name":         req["name"],
			"phone_number": req["phone_number"],
			"address":      req["address"],
			"status":       "active",
			"created_at":   time.Now().Add(-24*time.Hour).Format(time.RFC3339),
			"updated_at":   time.Now().Format(time.RFC3339),
		})
	})

	app.Delete("/api/v1/users/:id", func(c *fiber.Ctx) error {
		userID := c.Params("id")
		if userID == "not-found" {
			return c.Status(404).JSON(fiber.Map{"error": "user not found"})
		}

		return c.Status(204).Send(nil)
	})

	t.Run("GET /api/v1/users - List users", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/users", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		assert.Contains(t, result, "users")
		users := result["users"].([]interface{})
		assert.Len(t, users, 2)

		// Validate user structure
		user := users[0].(map[string]interface{})
		assert.Contains(t, user, "id")
		assert.Contains(t, user, "email")
		assert.Contains(t, user, "name")
		assert.Contains(t, user, "status")
	})

	t.Run("GET /api/v1/users/:id - Get user by ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/users/user-123", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		assert.Equal(t, "user-123", result["id"])
		assert.Contains(t, result, "email")
		assert.Contains(t, result, "name")
	})

	t.Run("GET /api/v1/users/:id - User not found", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/users/not-found", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("POST /api/v1/users - Create user", func(t *testing.T) {
		payload := map[string]interface{}{
			"email":        "new@example.com",
			"name":         "New User",
			"phone_number": "090-9999-8888",
			"address":      "Kyoto, Japan",
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/api/v1/users", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		assert.Equal(t, "new@example.com", result["email"])
		assert.Equal(t, "New User", result["name"])
		assert.Contains(t, result, "id")
		assert.Equal(t, "active", result["status"])
	})

	t.Run("PUT /api/v1/users/:id - Update user", func(t *testing.T) {
		payload := map[string]interface{}{
			"email":        "updated@example.com",
			"name":         "Updated User",
			"phone_number": "090-7777-6666",
			"address":      "Nagoya, Japan",
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest("PUT", "/api/v1/users/user-123", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		assert.Equal(t, "user-123", result["id"])
		assert.Equal(t, "updated@example.com", result["email"])
		assert.Equal(t, "Updated User", result["name"])
	})

	t.Run("DELETE /api/v1/users/:id - Delete user", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/v1/users/user-123", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	})

	t.Run("POST /api/v1/users - Validation error", func(t *testing.T) {
		payload := map[string]interface{}{
			"name": "Missing Email",
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/api/v1/users", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}