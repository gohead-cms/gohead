package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gohead-cms/gohead/internal/api/middleware"
	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/auth"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/testutils"

	"github.com/gin-gonic/gin"
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

func TestProtectedHandlerWithPermissions(t *testing.T) {
	// Setup the test database and server with middleware
	router, db := testutils.SetupTestServer()
	// Apply the ResponseWrapper middleware to ensure a consistent response format.
	router.Use(middleware.ResponseWrapper())
	defer testutils.CleanupTestDB()

	// Apply migrations
	assert.NoError(t, db.AutoMigrate(&models.User{}, &models.UserRole{}, &models.Collection{}, &models.Attribute{}))

	// Seed roles with the required permissions for the test cases.
	adminRole := models.UserRole{Name: "admin", Permissions: models.JSONMap{"create": true}}
	userRole := models.UserRole{Name: "user", Permissions: models.JSONMap{"read": true}}
	assert.NoError(t, db.Create(&adminRole).Error)
	assert.NoError(t, db.Create(&userRole).Error)

	// Create a test collection
	collection := models.Collection{
		Name: "protected_items",
		Attributes: []models.Attribute{
			{Name: "title", Type: "string", Required: true},
			{Name: "content", Type: "richtext", Required: true},
		},
	}
	assert.NoError(t, db.Create(&collection).Error)

	// Register the handler with AuthMiddleware and Role Authorization.
	gin.SetMode(gin.TestMode)
	protected := router.Group("/")
	protected.Use(middleware.AuthMiddleware(), middleware.AuthorizeRole("admin"))
	{
		protected.POST("/:collection", DynamicCollectionHandler)
	}

	// Generate tokens for different roles
	adminToken, err := auth.GenerateJWT("admin_user", "admin")
	assert.NoError(t, err)
	userToken, err := auth.GenerateJWT("regular_user", "user")
	assert.NoError(t, err)

	// Define test cases
	testCases := []struct {
		name           string
		token          string
		inputData      map[string]any
		expectedStatus int
		expectedData   any
		expectedError  string
	}{
		{
			name:  "Valid Admin Token",
			token: adminToken,
			inputData: map[string]any{
				"title":   "Admin Content",
				"content": "This content is created by an admin.",
			},
			expectedStatus: http.StatusCreated,
			expectedData: map[string]any{
				"title":   "Admin Content",
				"content": "This content is created by an admin.",
			},
			expectedError: "",
		},
		{
			name:  "Valid User Token - Insufficient Permissions",
			token: userToken,
			inputData: map[string]any{
				"title":   "User Content",
				"content": "This content is created by a regular user.",
			},
			expectedStatus: http.StatusForbidden,
			expectedData:   nil,
			expectedError:  "insufficient permissions",
		},
		{
			name:           "Missing Token",
			token:          "",
			inputData:      map[string]any{},
			expectedStatus: http.StatusUnauthorized,
			expectedData:   nil,
			expectedError:  "Authorization header required",
		},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Prepare the request body
			requestBody := map[string]any{"data": tc.inputData}
			body, _ := json.Marshal(requestBody)
			req, _ := http.NewRequest(http.MethodPost, "/protected_items", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			if tc.token != "" {
				req.Header.Set("Authorization", "Bearer "+tc.token)
			}

			// Create a response recorder
			rr := httptest.NewRecorder()

			// Serve the HTTP request
			router.ServeHTTP(rr, req)

			// Assert the response status
			assert.Equal(t, tc.expectedStatus, rr.Code)

			// Assert the response body contains the expected content
			var response map[string]any
			err := json.Unmarshal(rr.Body.Bytes(), &response)
			assert.NoError(t, err)

			if tc.expectedStatus >= 400 {
				// Assert error response
				errorData, ok := response["error"].(map[string]any)
				assert.True(t, ok, "Response should have an 'error' key")
				assert.Contains(t, errorData["message"], tc.expectedError)
			} else {
				// Assert successful data response
				data, ok := response["data"].(map[string]any)
				assert.True(t, ok, "Response should have a 'data' key")
				assert.Equal(t, tc.expectedData.(map[string]any)["title"], data["title"])
				assert.Equal(t, tc.expectedData.(map[string]any)["content"], data["content"])
			}
		})
	}
}
