// internal/api/middleware/authorization_middleware_test.go
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gitlab.com/sudo.bngz/gohead/pkg/auth"
	"gitlab.com/sudo.bngz/gohead/pkg/logger"
)

// Mock handler to simulate an actual endpoint
func mockProtectedHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Access granted"})
}

func TestAuthorizationMiddleware(t *testing.T) {
	logger.InitLogger("debug")
	// Initialize JWT with a test secret
	auth.InitializeJWT("test-secret")

	// Create the Gin router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(AuthMiddleware())
	router.GET("/protected", mockProtectedHandler)

	// Generate a valid token
	validToken, err := auth.GenerateJWT("test_user", "user")
	assert.NoError(t, err)

	// Define test cases
	testCases := []struct {
		name           string
		token          string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Valid Token",
			token:          validToken,
			expectedStatus: http.StatusOK,
			expectedBody:   `{"message":"Access granted"}`,
		},
		{
			name:           "Missing Token",
			token:          "",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"Authorization header required"}`,
		},
		{
			name:           "Invalid Token",
			token:          "invalid-token",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"Invalid token"}`,
		},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
			if tc.token != "" {
				req.Header.Set("Authorization", "Bearer "+tc.token)
			}

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			assert.Equal(t, tc.expectedStatus, rr.Code)
			assert.JSONEq(t, tc.expectedBody, rr.Body.String())
		})
	}
}

func TestAuthorizeRoleMiddleware(t *testing.T) {
	// Initialize JWT with a test secret
	auth.InitializeJWT("test-secret")

	// Create the Gin router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(AuthMiddleware())
	router.Use(AuthorizeRole("admin"))
	router.GET("/admin", mockProtectedHandler)

	// Generate tokens for different roles
	adminToken, err := auth.GenerateJWT("admin_user", "admin")
	assert.NoError(t, err)
	userToken, err := auth.GenerateJWT("user", "user")
	assert.NoError(t, err)

	// Define test cases
	testCases := []struct {
		name           string
		token          string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Valid Admin Token",
			token:          adminToken,
			expectedStatus: http.StatusOK,
			expectedBody:   `{"message":"Access granted"}`,
		},
		{
			name:           "Non-Admin Token",
			token:          userToken,
			expectedStatus: http.StatusForbidden,
			expectedBody:   `{"error":"insufficient permissions"}`,
		},
		{
			name:           "Missing Token",
			token:          "",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"Authorization header required"}`,
		},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, "/admin", nil)
			if tc.token != "" {
				req.Header.Set("Authorization", "Bearer "+tc.token)
			}

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			assert.Equal(t, tc.expectedStatus, rr.Code)
			assert.JSONEq(t, tc.expectedBody, rr.Body.String())
		})
	}
}
