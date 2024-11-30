// internal/api/middleware/auth_test.go
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

// Mock handler to simulate a protected endpoint
func mockHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Access granted"})
}

func TestAuthMiddleware(t *testing.T) {
	logger.InitLogger("debug")
	// Initialize JWT with a test secret
	auth.InitializeJWT("test-secret")

	// Create the Gin router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(AuthMiddleware())
	router.GET("/protected", mockHandler)

	// Generate a valid token
	validToken, err := auth.GenerateJWT("test_user", "user")
	assert.NoError(t, err)

	// Define test cases
	testCases := []struct {
		name           string
		authHeader     string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Valid Token",
			authHeader:     "Bearer " + validToken,
			expectedStatus: http.StatusOK,
			expectedBody:   `{"message":"Access granted"}`,
		},
		{
			name:           "Missing Authorization Header",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"Authorization header required"}`,
		},
		{
			name:           "Malformed Authorization Header",
			authHeader:     validToken, // Missing "Bearer " prefix
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"Bearer token required"}`,
		},
		{
			name:           "Invalid Token",
			authHeader:     "Bearer invalid-token",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"Invalid token"}`,
		},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			assert.Equal(t, tc.expectedStatus, rr.Code)
			assert.JSONEq(t, tc.expectedBody, rr.Body.String())
		})
	}
}
