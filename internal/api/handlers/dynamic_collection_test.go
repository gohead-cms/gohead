package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gohead-cms/gohead/internal/api/middleware"
	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/auth"
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

func TestDynamicContentHandler(t *testing.T) {
	router, db := testutils.SetupTestServer()
	// Apply the ResponseWrapper middleware to all routes.
	router.Use(middleware.ResponseWrapper())
	defer testutils.CleanupTestDB()

	assert.NoError(t, db.AutoMigrate(&models.User{}, &models.UserRole{}, &models.Collection{}, &models.Attribute{}, &models.Item{}))

	// Define a test collection to work with
	collection := models.Collection{
		Name: "articles",
		Attributes: []models.Attribute{
			{Name: "title", Type: "string", Required: true},
			{Name: "content", Type: "richtext", Required: true},
		},
	}
	assert.NoError(t, storage.SaveCollection(&collection))

	// Define a user role with permission to create content
	adminRole := models.UserRole{
		Name:        "admin",
		Permissions: models.JSONMap{"create": true},
	}
	assert.NoError(t, db.Create(&adminRole).Error)

	protected := router.Group("/")
	protected.Use(middleware.AuthMiddleware())
	{
		protected.POST("/:collection", DynamicCollectionHandler)
		protected.GET("/:collection/:id", DynamicCollectionHandler)
	}

	t.Run("Create Content Item", func(t *testing.T) {
		// Generate a valid JWT for the "admin" role
		token, err := auth.GenerateJWT("test_user", "admin")
		assert.NoError(t, err)

		// Request body
		requestBody := map[string]any{
			"data": map[string]any{
				"title":   "Test Article",
				"content": "This is the content of the test article.",
			},
		}
		body, _ := json.Marshal(requestBody)

		req, _ := http.NewRequest(http.MethodPost, "/articles", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		// Assert status
		assert.Equal(t, http.StatusCreated, rr.Code, "Expected status 201 because admin has 'create' permission")

		// Assert response body format from the ResponseWrapper middleware
		var response map[string]any
		err = json.Unmarshal(rr.Body.Bytes(), &response)
		assert.NoError(t, err)

		// The `data` key is where the actual response from the handler is stored
		_, ok := response["data"].(map[string]any)
		assert.True(t, ok, "Response should have a 'data' key")

		//assert.Equal(t, "Test Article", data["title"])

		// Check DB storage
		_, total, err := storage.GetItems(collection.ID, 1, 15)
		assert.NoError(t, err)
		assert.Equal(t, 1, total)
	})

	t.Run("Create Content Item - Validation Error", func(t *testing.T) {
		token, err := auth.GenerateJWT("test_user", "admin")
		assert.NoError(t, err)

		// Request body with missing required 'title' field
		requestBody := map[string]any{
			"data": map[string]any{
				"content": "This is the content of the test article.",
			},
		}
		body, _ := json.Marshal(requestBody)

		req, _ := http.NewRequest(http.MethodPost, "/articles", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		// Assert status
		assert.Equal(t, http.StatusBadRequest, rr.Code, "Expected status 400 for validation error")

		// Assert error response body from the ResponseWrapper middleware
		var response map[string]any
		err = json.Unmarshal(rr.Body.Bytes(), &response)
		assert.NoError(t, err)

		errorData, ok := response["error"].(map[string]any)
		assert.True(t, ok, "Response should have an 'error' key")

		assert.Equal(t, float64(http.StatusBadRequest), errorData["status"])
		assert.Equal(t, "ValidationError", errorData["name"])
		assert.Contains(t, errorData["message"], "missing required attribute")
	})
}

func TestRetrieveContentItem(t *testing.T) {
	router, db := testutils.SetupTestServer()
	// Apply the ResponseWrapper middleware to all routes.
	router.Use(middleware.ResponseWrapper())
	defer testutils.CleanupTestDB()

	assert.NoError(t, db.AutoMigrate(&models.User{}, &models.UserRole{}, &models.Collection{}, &models.Attribute{}, &models.Item{}))

	// Seed roles with permissions
	viewerRole := models.UserRole{
		Name:        "viewer",
		Permissions: models.JSONMap{"read": true},
	}
	assert.NoError(t, db.Create(&viewerRole).Error)

	// Prepare a collection
	collection := models.Collection{
		Name: "articles",
		Attributes: []models.Attribute{
			{Name: "title", Type: "string"},
			{Name: "content", Type: "richtext"},
		},
	}
	assert.NoError(t, storage.SaveCollection(&collection))

	// Prepare an item to retrieve
	contentItem := models.Item{
		CollectionID: collection.ID,
		Data:         models.JSONMap{"title": "Existing Article", "content": "This is an existing article."},
	}

	_, err := storage.SaveItem(*&collection, contentItem.Data)
	assert.NoError(t, err)
	// Apply AuthMiddleware to protected routes
	protected := router.Group("/")
	protected.Use(middleware.AuthMiddleware())
	{
		protected.GET("/:collection/:id", DynamicCollectionHandler)
	}

	// Generate a valid JWT for the "viewer" role
	token, err := auth.GenerateJWT("test_viewer", "viewer")
	assert.NoError(t, err)

	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/articles/%d", contentItem.ID), nil)
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Assert status
	assert.Equal(t, http.StatusOK, rr.Code, "Expected status 200 because viewer has 'read' permission")

	// Assert body format from the ResponseWrapper middleware
	var response map[string]any
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err)

	// The `data` key is where the actual response from the handler is stored
	data, ok := response["data"].(map[string]any)
	assert.True(t, ok, "Response should have a 'data' key")

	assert.Equal(t, "Existing Article", data["title"])
}
