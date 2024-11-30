// internal/api/handlers/content_item_test.go
package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gitlab.com/sudo.bngz/gohead/internal/models"
	"gitlab.com/sudo.bngz/gohead/pkg/database"
	"gitlab.com/sudo.bngz/gohead/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestCreateContentItemIntegration(t *testing.T) {
	logger.InitLogger("debug")

	// Initialize in-memory test database
	db, err := database.InitDatabase("sqlite://:memory:")
	assert.NoError(t, err, "Failed to initialize in-memory database")

	// Apply migrations for all necessary models
	err = db.AutoMigrate(&models.ContentItem{})
	assert.NoError(t, err, "Failed to apply migrations for ContentItem")

	// Create a test content type
	ct := models.ContentType{
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
	router := gin.Default()

	router.POST("/articles", CreateContentItem(ct))

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
	var response models.ContentItem
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Test Article", response.Data["title"])
}
