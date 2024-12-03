// internal/api/handlers/content_item_test.go
package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gitlab.com/sudo.bngz/gohead/internal/models"
	"gitlab.com/sudo.bngz/gohead/pkg/logger"
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

func TestCreateItemIntegration(t *testing.T) {
	// Initialize in-memory test database
	router, db := testutils.SetupTestServer()

	// Apply migrations for all necessary models
	err := db.AutoMigrate(&models.Item{})
	assert.NoError(t, err, "Failed to apply migrations for Item")

	// Create a test content type
	ct := models.Collection{
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
	db.Create(&ct)

	// Prepare the Gin router
	gin.SetMode(gin.TestMode)
	router.POST("/articles", CreateItem(ct))

	// Prepare the test request
	itemData := map[string]interface{}{
		"title":   "Test Article",
		"content": "This is a test.",
	}
	body, _ := json.Marshal(itemData)
	req, _ := http.NewRequest(http.MethodPost, "/articles", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Serve the request
	router.ServeHTTP(rr, req)

	// Check the response
	assert.Equal(t, http.StatusCreated, rr.Code)
	var response models.Item
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Test Article", response.Data["title"])
}
