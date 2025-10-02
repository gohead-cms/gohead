package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gohead-cms/gohead/internal/api/middleware"
	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/storage"
	"github.com/gohead-cms/gohead/pkg/testutils"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// Initialize logger for testing
func init() {
	var buffer bytes.Buffer
	logger.InitLogger("debug")
	logger.Log.SetOutput(&buffer)
	logger.Log.SetFormatter(&logrus.TextFormatter{})
}

func TestComponentHandlers(t *testing.T) {
	// Setup the test database and router
	router, db := testutils.SetupTestServer()
	router.Use(middleware.ResponseWrapper())
	defer testutils.CleanupTestDB()

	// Apply migrations
	assert.NoError(t, db.AutoMigrate(&models.Component{}, &models.ComponentAttribute{}))

	// Register component routes
	router.POST("/components", CreateComponent)
	router.GET("/components/:name", GetComponent)
	router.PUT("/components/:name", UpdateComponent)
	router.DELETE("/components/:name", DeleteComponent)

	// --- Test Data ---
	componentName := "seo"
	validComponentPayload := map[string]any{
		"name": "seo",
		"attributes": map[string]any{
			"meta_title": map[string]any{
				"type":     "string",
				"required": true,
			},
			"meta_description": map[string]any{
				"type": "text",
			},
		},
	}

	// --- Sub-test for Create ---
	t.Run("Create Component", func(t *testing.T) {
		body, _ := json.Marshal(validComponentPayload)
		req, _ := http.NewRequest(http.MethodPost, "/components", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)

		var response map[string]any
		json.Unmarshal(rr.Body.Bytes(), &response)
		assert.Contains(t, response["message"], "component created successfully")
		assert.Equal(t, componentName, response["component"])

		// Verify in DB
		_, err := storage.GetComponentByName(componentName)
		assert.NoError(t, err, "Component should exist in the database after creation")
	})

	// --- Sub-test for Get ---
	t.Run("Get Component", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/components/"+componentName, nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response models.Component
		json.Unmarshal(rr.Body.Bytes(), &response)
		assert.Equal(t, componentName, response.Name)
		assert.Len(t, response.Attributes, 2, "Expected component to have 2 attributes")
	})

	// --- Sub-test for Get Not Found ---
	t.Run("Get Component Not Found", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/components/nonexistent", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	// --- Sub-test for Update ---
	t.Run("Update Component", func(t *testing.T) {
		updatedPayload := map[string]any{
			"name": "seo", // Name usually doesn't change, but attributes do
			"attributes": map[string]any{
				"meta_title": map[string]any{
					"type": "string",
				},
				"canonical_url": map[string]any{ // Added new attribute
					"type": "string",
				},
			},
		}
		body, _ := json.Marshal(updatedPayload)
		req, _ := http.NewRequest(http.MethodPut, "/components/"+componentName, bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		// Verify update in DB
		cmp, err := storage.GetComponentByName(componentName)
		assert.NoError(t, err)
		assert.Len(t, cmp.Attributes, 2, "Expected updated component to have 2 attributes")
		// Check if one of the new attribute names exists
		var hasCanonicalUrl bool
		for _, attr := range cmp.Attributes {
			if attr.Name == "canonical_url" {
				hasCanonicalUrl = true
				break
			}
		}
		assert.True(t, hasCanonicalUrl, "Expected 'canonical_url' attribute after update")
	})

	// --- Sub-test for Delete ---
	t.Run("Delete Component", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodDelete, "/components/"+componentName, nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		// Verify deletion in DB
		_, err := storage.GetComponentByName(componentName)
		assert.Error(t, err, "Component should not exist in the database after deletion")
	})

	// --- Sub-test for Delete Not Found ---
	t.Run("Delete Component Not Found", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodDelete, "/components/nonexistent", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code, "Expected bad request for deleting non-existent component")
	})
}
