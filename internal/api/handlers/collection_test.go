// internal/api/handlers/type_test.go
package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gohead/internal/models"
	"gohead/pkg/logger"
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

func TestCreateCollectionHandler(t *testing.T) {
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

	// Initialize the router and attach the handler
	router.POST("/auth/register", Register)
	// Load test configuration
	// Create the Gin router
	gin.SetMode(gin.TestMode)

	// Register the handler
	router.POST("/collections", CreateCollection)

	// Define test cases
	testCases := []struct {
		name           string
		inputData      map[string]interface{}
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Valid Content Type",
			inputData: map[string]interface{}{
				"name": "articles",
				"fields": []map[string]interface{}{
					{
						"name":     "title",
						"type":     "string",
						"required": true,
					},
					{
						"name":     "content",
						"type":     "richtext",
						"required": true,
					},
				},
			},
			expectedStatus: http.StatusCreated,
			expectedBody:   `collection created successfully`,
		},
		{
			name: "Missing Name Field",
			inputData: map[string]interface{}{
				"fields": []map[string]interface{}{
					{
						"type":     "string",
						"required": true,
					},
				},
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `missing required field: 'name'`,
		},
		{
			name: "Empty Fields Array",
			inputData: map[string]interface{}{
				"name":   "users",
				"fields": []map[string]interface{}{},
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `fields array cannot be empty`,
		},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Prepare request body
			body, _ := json.Marshal(tc.inputData)
			req, _ := http.NewRequest(http.MethodPost, "/collections", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			// Create a response recorder
			rr := httptest.NewRecorder()

			// Serve the HTTP request
			router.ServeHTTP(rr, req)

			// Assert the response status
			assert.Equal(t, tc.expectedStatus, rr.Code)

			// Assert the response body contains expected data
			assert.Contains(t, rr.Body.String(), tc.expectedBody)
		})
	}
}
