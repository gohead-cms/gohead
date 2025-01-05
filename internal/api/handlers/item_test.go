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
	// Setup the test database
	// Initialize in-memory test database
	router, db := testutils.SetupTestServer()
	defer testutils.CleanupTestDB()

	// Apply migrations
	assert.NoError(t, db.AutoMigrate(&models.User{}, &models.UserRole{}, &models.Collection{}, &models.Attribute{}))

	// Seed roles
	adminRole := models.UserRole{Name: "admin", Description: "Administrator", Permissions: models.JSONMap{"manage_users": true}}
	readerRole := models.UserRole{Name: "reader", Description: "Reader", Permissions: models.JSONMap{"read_content": true}}
	assert.NoError(t, db.Create(&adminRole).Error)
	assert.NoError(t, db.Create(&readerRole).Error)

	// Create a test content type
	ct := models.Collection{
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
	assert.NoError(t, db.Create(&ct).Error)

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
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Test Article", response.Data["title"])
}
