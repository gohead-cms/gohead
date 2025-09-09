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
	router.Use(middleware.ResponseWrapper())
	defer testutils.CleanupTestDB()

	assert.NoError(t, db.AutoMigrate(&models.User{}, &models.UserRole{}, &models.Collection{}, &models.Attribute{}, &models.Item{}))

	adminRole := models.UserRole{
		Name: "admin",
		Permissions: models.JSONMap{
			"create": true,
			"read":   true,
			"update": true,
			"delete": true,
		},
	}
	assert.NoError(t, db.Create(&adminRole).Error)

	// Define a test collection
	collection := models.Collection{
		Name: "articles",
		Attributes: []models.Attribute{
			{Name: "title", Type: "string", Required: true},
			{Name: "content", Type: "richtext", Required: true},
		},
	}
	assert.NoError(t, storage.SaveCollection(&collection))

	protected := router.Group("/")
	protected.Use(middleware.AuthMiddleware())
	{
		protected.POST("/:collection", DynamicCollectionHandler)
		protected.GET("/:collection/:id", DynamicCollectionHandler)
	}

	t.Run("Create Content Item", func(t *testing.T) {
		// --- Generate a valid JWT for the "admin" role we just created ---
		token, err := auth.GenerateJWT("test_user", "admin")
		assert.NoError(t, err)

		// --- Request body ---
		requestBody := map[string]any{
			"data": map[string]interface{}{
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

		// --- Assert status ---
		assert.Equal(t, http.StatusCreated, rr.Code, "Expected status 201 because admin has 'create' permission")

		// --- Assert response body ---
		var response map[string]interface{}
		err = json.Unmarshal(rr.Body.Bytes(), &response)
		assert.NoError(t, err)
		data := response["data"].(map[string]any)
		assert.Equal(t, "Test Article", data["title"])

		// --- Check DB storage ---
		_, total, err := storage.GetItems(collection.ID, 1, 15)
		assert.NoError(t, err)
		assert.Equal(t, 1, total)
	})
}

func TestRetrieveContentItem(t *testing.T) {
	router, db := testutils.SetupTestServer()
	defer testutils.CleanupTestDB()

	assert.NoError(t, db.AutoMigrate(&models.User{}, &models.UserRole{}, &models.Collection{}, &models.Attribute{}, &models.Item{}))

	// ## FIX 1: Seed roles with permissions the handler will check for ##
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
	assert.NoError(t, storage.SaveItem(&contentItem))

	// ## FIX 2: Apply AuthMiddleware to protected routes ##
	protected := router.Group("/")
	protected.Use(middleware.AuthMiddleware())
	{
		protected.GET("/:collection/:id", DynamicCollectionHandler)
	}

	// --- Generate a valid JWT for the "viewer" role ---
	token, err := auth.GenerateJWT("test_viewer", "viewer")
	assert.NoError(t, err)

	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/articles/%d", contentItem.ID), nil)
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Assert status
	assert.Equal(t, http.StatusOK, rr.Code, "Expected status 200 because viewer has 'read' permission")

	// Assert body
	var response map[string]any
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err)
	data := response["data"].(map[string]any)
	assert.Equal(t, "Existing Article", data["title"])
}
