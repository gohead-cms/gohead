// internal/api/handlers/dynamic_content_test.go
package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"gitlab.com/sudo.bngz/gohead/internal/models"
	"gitlab.com/sudo.bngz/gohead/pkg/database"
	"gitlab.com/sudo.bngz/gohead/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestDynamicContentHandler(t *testing.T) {
	logger.InitLogger("debug")

	// Initialize in-memory test database
	db, err := database.InitDatabase("sqlite://:memory:")
	assert.NoError(t, err, "Failed to initialize in-memory database")

	// Apply migrations for all necessary models
	err = db.AutoMigrate(&models.ContentItem{})
	assert.NoError(t, err, "Failed to apply migrations for ContentItem")

	// Define a test content type
	contentType := models.ContentType{
		Name: "articles",
		Fields: []models.Field{
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
	if err := db.Create(&contentType).Error; err != nil {
		t.Fatalf("Failed to create content type: %v", err)
	}

	// Create the Gin router
	gin.SetMode(gin.TestMode)
	router := gin.Default()

	// Register the dynamic handler
	router.POST("/:contentType", DynamicContentHandler)
	router.GET("/:contentType/:id", DynamicContentHandler)

	// Define test cases
	t.Run("Create Content Item", func(t *testing.T) {
		testCreateContentItem(router, t)
	})

	t.Run("Retrieve Content Item", func(t *testing.T) {
		testRetrieveContentItem(router, t)
	})
}

func testCreateContentItem(router *gin.Engine, t *testing.T) {
	// Prepare the test request body
	requestBody := map[string]interface{}{
		"title":   "Test Article",
		"content": "This is the content of the test article.",
	}
	body, _ := json.Marshal(requestBody)

	req, _ := http.NewRequest(http.MethodPost, "/articles", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Serve the HTTP request
	router.ServeHTTP(rr, req)

	// Assert the response status
	assert.Equal(t, http.StatusCreated, rr.Code)

	// Assert the response body
	var response map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Test Article", response["data"].(map[string]interface{})["title"])
	assert.Equal(t, "This is the content of the test article.", response["data"].(map[string]interface{})["content"])

	// Check that the item is stored in the database
	var item models.ContentItem
	err = database.DB.Where("content_type = ? AND data ->> 'title' = ?", "articles", "Test Article").First(&item).Error
	assert.NoError(t, err)
}

func testRetrieveContentItem(router *gin.Engine, t *testing.T) {
	// Prepare a content item to retrieve
	contentItem := models.ContentItem{
		ContentType: "articles",
		Data: models.JSONMap{
			"title":   "Existing Article",
			"content": "This is an existing article.",
		},
	}
	if err := database.DB.Create(&contentItem).Error; err != nil {
		t.Fatalf("Failed to create content item: %v", err)
	}

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
