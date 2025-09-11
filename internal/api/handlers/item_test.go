package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gohead-cms/gohead/internal/api/middleware"
	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/auth"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/testutils"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// Initialize logger for testing
func init() {
	// Configure logger to write logs to a buffer for testing
	var buffer bytes.Buffer
	logger.InitLogger("debug")
	logger.Log.SetOutput(&buffer)
	logger.Log.SetFormatter(&logrus.TextFormatter{})
}

func TestCreateItemIntegration(t *testing.T) {
	// Setup the test database and server with middleware
	router, db := testutils.SetupTestServer()
	// Apply the ResponseWrapper and AuthMiddleware to the router
	router.Use(middleware.ResponseWrapper())
	protected := router.Group("/")
	protected.Use(middleware.AuthMiddleware())
	defer testutils.CleanupTestDB()

	// Apply migrations
	assert.NoError(t, db.AutoMigrate(&models.User{}, &models.UserRole{}, &models.Collection{}, &models.Attribute{}, &models.Item{}))

	// Seed roles with correct permissions for the test
	adminRole := models.UserRole{Name: "admin", Description: "Administrator", Permissions: models.JSONMap{"create": true}}
	assert.NoError(t, db.Create(&adminRole).Error)

	// Create a test collection
	collection := models.Collection{
		Name: "articles",
		Attributes: []models.Attribute{
			{
				Name:     "title",
				Type:     "string",
				Required: true,
			},
			{
				Name:     "content",
				Type:     "richtext",
				Required: true,
			},
		},
	}
	assert.NoError(t, db.Create(&collection).Error)

	// Register the handler with the protected group
	protected.POST("/:collection", DynamicCollectionHandler)

	t.Run("Create a valid item successfully", func(t *testing.T) {
		// Generate a valid JWT for an admin
		token, err := auth.GenerateJWT("test_user", "admin")
		assert.NoError(t, err)

		// Prepare the test request body
		itemData := map[string]any{
			"title":   "Test Article",
			"content": "This is a test.",
		}
		requestBody := map[string]any{"data": itemData}
		body, _ := json.Marshal(requestBody)

		req, _ := http.NewRequest(http.MethodPost, "/articles", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		// Assert status
		assert.Equal(t, http.StatusCreated, rr.Code)

		// Assert response body format from the ResponseWrapper middleware
		var response map[string]any
		err = json.Unmarshal(rr.Body.Bytes(), &response)
		assert.NoError(t, err)

		// The actual data is nested under the "data" key
		_, ok := response["data"].(map[string]any)
		assert.True(t, ok, "Response should have a 'data' key")

		//assert.Equal(t, "Test Article", data["title"])
	})

	t.Run("Fail to create item due to validation error", func(t *testing.T) {
		// Generate a valid JWT for an admin
		token, err := auth.GenerateJWT("test_user", "admin")
		assert.NoError(t, err)

		// Prepare an invalid request body with a missing required field
		itemData := map[string]any{
			"title": "Test Article",
			// "content" is missing, which is a required field
		}
		requestBody := map[string]any{"data": itemData}
		body, _ := json.Marshal(requestBody)

		req, _ := http.NewRequest(http.MethodPost, "/articles", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		// Assert status
		assert.Equal(t, http.StatusBadRequest, rr.Code)

		// Assert error response body from the ResponseWrapper middleware
		var response map[string]any
		err = json.Unmarshal(rr.Body.Bytes(), &response)
		assert.NoError(t, err)

		// The error details are nested under the "error" key
		errorData, ok := response["error"].(map[string]any)
		assert.True(t, ok, "Response should have an 'error' key")
		assert.Equal(t, float64(http.StatusBadRequest), errorData["status"])
		assert.Equal(t, "ValidationError", errorData["name"])
		assert.Contains(t, errorData["message"], "missing required attribute")
	})
}
