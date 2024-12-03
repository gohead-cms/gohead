// internal/api/handlers/permissions_handler_test.go
package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gitlab.com/sudo.bngz/gohead/internal/api/middleware"
	"gitlab.com/sudo.bngz/gohead/internal/models"
	"gitlab.com/sudo.bngz/gohead/pkg/auth"
	"gitlab.com/sudo.bngz/gohead/pkg/config"
	"gitlab.com/sudo.bngz/gohead/pkg/database"
	"gitlab.com/sudo.bngz/gohead/pkg/logger"
)

func TestProtectedHandlerWithPermissions(t *testing.T) {
	logger.InitLogger("debug")
	// Load test configuration
	cfg, err := config.LoadTestConfig()
	if err != nil {
		t.Fatalf("Failed to load test config: %v", err)
	}

	// Initialize JWT with a test secret
	auth.InitializeJWT(cfg.JWTSecret)

	// Set up a test content type
	contentType := models.Collection{
		Name: "protected_items",
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
	if err := database.DB.Create(&contentType).Error; err != nil {
		t.Fatalf("Failed to create content type: %v", err)
	}

	// Create the Gin router
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Register the handler with AuthMiddleware and Role Authorization
	router.POST("/:contentType", middleware.AuthMiddleware(), middleware.AuthorizeRole("admin"), CreateItem(contentType))

	// Generate tokens for different roles
	adminToken, err := auth.GenerateJWT("admin_user", "admin")
	assert.NoError(t, err)
	userToken, err := auth.GenerateJWT("regular_user", "user")
	assert.NoError(t, err)

	// Define test cases
	testCases := []struct {
		name           string
		token          string
		inputData      map[string]interface{}
		expectedStatus int
		expectedBody   string
	}{
		{
			name:  "Valid Admin Token",
			token: adminToken,
			inputData: map[string]interface{}{
				"title":   "Admin Content",
				"content": "This content is created by an admin.",
			},
			expectedStatus: http.StatusCreated,
			expectedBody:   `"title":"Admin Content"`,
		},
		{
			name:  "Valid User Token - Insufficient Permissions",
			token: userToken,
			inputData: map[string]interface{}{
				"title":   "User Content",
				"content": "This content is created by a regular user.",
			},
			expectedStatus: http.StatusForbidden,
			expectedBody:   `"error":"insufficient permissions"`,
		},
		{
			name:           "Missing Token",
			token:          "",
			inputData:      map[string]interface{}{},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `"error":"Authorization header required"`,
		},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Prepare the request body
			body, _ := json.Marshal(tc.inputData)
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
			assert.Contains(t, rr.Body.String(), tc.expectedBody)
		})
	}
}
