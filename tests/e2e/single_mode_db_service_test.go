package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSingleModeDBServiceE2E tests db_service endpoints in single mode
func TestSingleModeDBServiceE2E(t *testing.T) {
	// Skip in CI or if not explicitly enabled
	if os.Getenv("RUN_E2E_TESTS") != "true" {
		t.Skip("Skipping E2E test. Set RUN_E2E_TESTS=true to run")
	}

	// Assume server is running on port 8086
	baseURL := "http://localhost:8086"

	// Wait for server to be ready
	waitForServer(t, baseURL)

	t.Run("ETCMeisai_CRUD_Operations", func(t *testing.T) {
		// List all ETCMeisai
		resp, err := http.Get(baseURL + "/api/v1/db/etc-meisai")
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var listResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&listResp)
		require.NoError(t, err)
		resp.Body.Close()

		initialCount := int(listResp["total_count"].(float64))
		assert.GreaterOrEqual(t, initialCount, 2, "Should have initial mock data")

		// Get specific ETCMeisai
		resp, err = http.Get(baseURL + "/api/v1/db/etc-meisai/1")
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var getMeisai map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&getMeisai)
		require.NoError(t, err)
		resp.Body.Close()

		assert.Equal(t, float64(1), getMeisai["id"])
		assert.Equal(t, "東京IC", getMeisai["ic_fr"])

		// Create new ETCMeisai
		newMeisai := map[string]interface{}{
			"date_to":      "2024-04-01",
			"date_to_date": "2024-04-01",
			"ic_fr":        "福岡IC",
			"ic_to":        "北九州IC",
			"price":        3500,
			"shashu":       2,
			"etc_num":      "9999-8888-7777-6666",
			"hash":         "e2e_test_hash",
		}

		jsonData, _ := json.Marshal(newMeisai)
		resp, err = http.Post(
			baseURL+"/api/v1/db/etc-meisai",
			"application/json",
			bytes.NewBuffer(jsonData),
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var createdMeisai map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&createdMeisai)
		require.NoError(t, err)
		resp.Body.Close()

		createdID := createdMeisai["id"].(float64)
		assert.Greater(t, createdID, float64(0))

		// Update the created ETCMeisai
		updateMeisai := map[string]interface{}{
			"date_to":      "2024-04-01",
			"date_to_date": "2024-04-01",
			"ic_fr":        "福岡IC",
			"ic_to":        "北九州IC",
			"price":        4000, // Updated price
			"shashu":       2,
			"etc_num":      "9999-8888-7777-6666",
			"hash":         "e2e_test_hash_updated",
		}

		jsonData, _ = json.Marshal(updateMeisai)
		req, err := http.NewRequest(
			http.MethodPut,
			fmt.Sprintf("%s/api/v1/db/etc-meisai/%d", baseURL, int(createdID)),
			bytes.NewBuffer(jsonData),
		)
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err = client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var updatedMeisai map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&updatedMeisai)
		require.NoError(t, err)
		resp.Body.Close()

		assert.Equal(t, float64(4000), updatedMeisai["price"])

		// Delete the created ETCMeisai
		req, err = http.NewRequest(
			http.MethodDelete,
			fmt.Sprintf("%s/api/v1/db/etc-meisai/%d", baseURL, int(createdID)),
			nil,
		)
		require.NoError(t, err)

		resp, err = client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
		resp.Body.Close()

		// Verify deletion
		resp, err = http.Get(fmt.Sprintf("%s/api/v1/db/etc-meisai/%d", baseURL, int(createdID)))
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		resp.Body.Close()
	})

	t.Run("DTakoUriageKeihi_Create", func(t *testing.T) {
		newKeihi := map[string]interface{}{
			"datetime":        "2024-04-01T15:30:00Z",
			"keihi_c":         300,
			"price":           25000.75,
			"dtako_row_id":    "DTAKO_E2E_001",
			"dtako_row_id_r":  "DTAKO_E2E_001R",
			"start_srch_id":   "START001",
			"start_srch_time": "2024-04-01T15:00:00Z",
		}

		jsonData, _ := json.Marshal(newKeihi)
		resp, err := http.Post(
			baseURL+"/api/v1/db/dtako-uriage-keihi",
			"application/json",
			bytes.NewBuffer(jsonData),
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var created map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&created)
		require.NoError(t, err)
		resp.Body.Close()

		assert.NotEmpty(t, created["srch_id"])
		assert.Contains(t, created["srch_id"].(string), "SRCH", "Should auto-generate SRCH ID")
		assert.Equal(t, float64(25000.75), created["price"])
	})

	t.Run("DTakoFerryRows_Create", func(t *testing.T) {
		newFerry := map[string]interface{}{
			"unko_no":        "FE2E001",
			"unko_date":      "2024-04-01",
			"yomitori_date":  "2024-04-02",
			"jigyosho_cd":    10,
			"jigyosho_name":  "E2Eテスト事業所",
			"sharyo_cd":      200,
			"sharyo_name":    "テストフェリー",
			"jomuin_cd1":     2001,
			"jomuin_name1":   "テスト乗務員",
		}

		jsonData, _ := json.Marshal(newFerry)
		resp, err := http.Post(
			baseURL+"/api/v1/db/dtako-ferry-rows",
			"application/json",
			bytes.NewBuffer(jsonData),
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var created map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&created)
		require.NoError(t, err)
		resp.Body.Close()

		assert.Greater(t, created["id"].(float64), float64(0), "ID should be assigned")
		assert.Equal(t, "FE2E001", created["unko_no"])
		assert.Equal(t, float64(10), created["jigyosho_cd"])
	})

	t.Run("ETCMeisaiMapping_Create", func(t *testing.T) {
		newMapping := map[string]interface{}{
			"etc_meisai_hash": "e2e_test_hash",
			"dtako_row_id":    "DTAKO_E2E_001",
			"created_at":      "2024-04-01T15:00:00Z",
			"updated_at":      "2024-04-01T15:00:00Z",
			"created_by":      "e2e_test_user",
			"notes":           "E2E test mapping",
		}

		jsonData, _ := json.Marshal(newMapping)
		resp, err := http.Post(
			baseURL+"/api/v1/db/etc-meisai-mapping",
			"application/json",
			bytes.NewBuffer(jsonData),
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var created map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&created)
		require.NoError(t, err)
		resp.Body.Close()

		assert.Greater(t, created["id"].(float64), float64(0), "ID should be assigned")
		assert.Equal(t, "e2e_test_hash", created["etc_meisai_hash"])
		assert.Equal(t, "DTAKO_E2E_001", created["dtako_row_id"])
	})
}

// waitForServer waits for the server to be ready
func waitForServer(t *testing.T, baseURL string) {
	maxRetries := 30
	for i := 0; i < maxRetries; i++ {
		resp, err := http.Get(baseURL + "/health/ready")
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(1 * time.Second)
	}
	t.Fatal("Server did not become ready in time")
}