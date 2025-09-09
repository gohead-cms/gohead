// internal/api/handlers/type_test.go
package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gohead-cms/gohead/internal/api/middleware"
	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/testutils"

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
	router.Use(middleware.ResponseWrapper())
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
		inputData      map[string]any
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Valid Preferences Collection",
			inputData: map[string]any{
				"name": "preferences",
				"kind": "collection",
				"attributes": map[string]any{
					"language": map[string]any{
						"type":     "text",
						"required": true,
						"enum":     []string{"English", "French", "Spanish", "German", "Chinese"},
					},
					"timezone": map[string]any{
						"type":     "text",
						"required": true,
						"enum": []string{
							"UTC-12:00", "UTC-11:00", "UTC-10:00", "UTC-09:00", "UTC-08:00", "UTC-07:00",
							"UTC-06:00", "UTC-05:00", "UTC-04:00", "UTC-03:00", "UTC-02:00", "UTC-01:00",
							"UTC+00:00", "UTC+01:00", "UTC+02:00", "UTC+03:00", "UTC+04:00", "UTC+05:00",
							"UTC+06:00", "UTC+07:00", "UTC+08:00", "UTC+09:00", "UTC+10:00", "UTC+11:00",
							"UTC+12:00",
						},
					},
				},
			},
			expectedStatus: http.StatusCreated,
			expectedBody:   `Collection created successfully`,
		},
		{
			name: "Missing Name Field",
			inputData: map[string]any{
				"kind": "collection",
				"attributes": map[string]any{
					"language": map[string]any{
						"type": "text",
					},
				},
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `missing or invalid field`,
		},
		{
			name: "Missing Kind Field",
			inputData: map[string]any{
				"name": "preferences",
				"attributes": map[string]any{
					"language": map[string]any{
						"type": "text",
					},
				},
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `missing or invalid field`,
		},
		{
			name: "Empty Attributes Object",
			inputData: map[string]any{
				"name":       "preferences",
				"kind":       "collection",
				"attributes": map[string]any{},
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `attributes array cannot be empty`,
		},
		{
			name: "Attribute Missing Type",
			inputData: map[string]any{
				"name": "preferences",
				"kind": "collection",
				"attributes": map[string]any{
					"language": map[string]any{
						"required": true,
					},
				},
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `failed to parse attribute`,
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
