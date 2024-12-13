package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"gitlab.com/sudo.bngz/gohead/internal/models"
	"gitlab.com/sudo.bngz/gohead/pkg/logger"
	"gitlab.com/sudo.bngz/gohead/pkg/storage"
	"gitlab.com/sudo.bngz/gohead/pkg/testutils"

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
	db := testutils.SetupTestDB()
	defer testutils.CleanupTestDB()

	// Apply migrations
	assert.NoError(t, db.AutoMigrate(&models.User{}, &models.UserRole{}, &models.Collection{}, &models.Field{}, &models.Relationship{}))

	// Define a test content type
	collection := models.Collection{
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
	assert.NoError(t, storage.SaveCollection(&collection))

	// Create the Gin router
	gin.SetMode(gin.TestMode)
	router := gin.Default()

	// Register the dynamic handler
	router.POST("/:collection", DynamicCollectionHandler)
	router.GET("/:collection/:id", DynamicCollectionHandler)

	// Define test cases
	t.Run("Create Content Item", func(t *testing.T) {
		testCreateContentItem(router, t)
	})

	//.t.Run("Retrieve Content Item", func(t *testing.T) {
	//	testRetrieveContentItem(router, t)
	//})
}

func testCreateContentItem(router *gin.Engine, t *testing.T) {

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

	// Check that the item is stored in the database using storage layer
	collection, err := storage.GetCollectionByName("articles")
	assert.NoError(t, err)

	items, err := storage.GetItems(collection.ID)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(items))
	assert.Equal(t, "Test Article", items[0].Data["title"])
}

func testRetrieveContentItem(router *gin.Engine, t *testing.T) {
	// Prepare a collection using storage layer
	collection := models.Collection{
		Name: "articles",
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
