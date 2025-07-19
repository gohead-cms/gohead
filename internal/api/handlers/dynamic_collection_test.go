package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"gohead/internal/models"
	"gohead/pkg/auth"
	"gohead/pkg/logger"
	"gohead/pkg/storage"
	"gohead/pkg/testutils"

	"github.com/gin-gonic/gin"
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

func TestDynamicContentHandler(t *testing.T) {
	// Setup the test database
	router, db := testutils.SetupTestServer()
	defer testutils.CleanupTestDB()

	// Apply migrations
	assert.NoError(t, db.AutoMigrate(&models.User{}, &models.UserRole{}, &models.Collection{}, &models.Attribute{}))

	// Define a test content type
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
	assert.NoError(t, storage.SaveCollection(&collection))

	// Register the dynamic handler
	router.POST("/:collection", DynamicCollectionHandler)
	router.GET("/:collection/:id", DynamicCollectionHandler)

	// Define test cases
	t.Run("Create Content Item", func(t *testing.T) {
		testCreateContentItem(router, t)
	})
}

func testCreateContentItem(router *gin.Engine, t *testing.T) {
	// --- Setup authentication token (assuming JWT auth) ---
	// Generate a valid JWT for an "admin" user (adapt role/username as needed)
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
	req.Header.Set("Authorization", "Bearer "+token) // <-- add auth header

	// --- Response recorder ---
	rr := httptest.NewRecorder()

	// --- Serve request ---
	router.ServeHTTP(rr, req)

	// --- Assert status ---
	assert.Equal(t, http.StatusCreated, rr.Code)

	// --- Assert response body ---
	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err)
	data := response["data"].(map[string]any)
	assert.Equal(t, "Test Article", data["title"])
	assert.Equal(t, "This is the content of the test article.", data["content"])

	// --- Check DB storage ---
	collection, err := storage.GetCollectionByName("articles")
	assert.NoError(t, err)

	items, total, err := storage.GetItems(collection.ID, 1, 15)
	assert.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, 1, len(items))
	assert.Equal(t, "Test Article", items[0].Data["title"])
}

func TestRetrieveContentItem(t *testing.T) {
	// Setup the test database and router
	router, db := testutils.SetupTestServer()
	defer testutils.CleanupTestDB()

	// Apply migrations
	assert.NoError(t, db.AutoMigrate(&models.User{}, &models.UserRole{}, &models.Collection{}, &models.Attribute{}))

	// Prepare a collection using storage layer
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
	assert.NoError(t, storage.SaveCollection(&collection))

	// Prepare a content item to retrieve
	contentItem := models.Item{
		CollectionID: collection.ID,
		Data: models.JSONMap{
			"title":   "Existing Article",
			"content": "This is an existing article.",
		},
	}
	assert.NoError(t, storage.SaveItem(&contentItem))

	// Register the dynamic handler
	router.GET("/:collection/:id", DynamicCollectionHandler)

	// Send a GET request to retrieve the content item
	url := fmt.Sprintf("/articles/%d", contentItem.ID)
	req, _ := http.NewRequest(http.MethodGet, url, nil)

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Serve the HTTP request
	router.ServeHTTP(rr, req)

	// Assert the response status
	assert.Equal(t, http.StatusOK, rr.Code)

	// Assert the response body
	var response map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Existing Article", response["data"].(map[string]interface{})["title"])
	assert.Equal(t, "This is an existing article.", response["data"].(map[string]interface{})["content"])
}
