// internal/api/middleware/authorization_middleware_test.go
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"gohead/pkg/auth"
	"gohead/pkg/logger"
	"gohead/pkg/testutils"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// Mock handler to simulate an actual endpoint
func mockProtectedHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Access granted"})
}

func TestAuthorizationMiddleware(t *testing.T) {
	testutils.SetupTestServer() // Initializes DB and roles
	defer testutils.CleanupTestDB()
	logger.InitLogger("info")
	// Initialize JWT with a test secret
	auth.InitializeJWT("test-secret")

	// Create the Gin router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(AuthMiddleware())
	router.GET("/protected", mockProtectedHandler)

	// Generate a valid token
	validToken, err := auth.GenerateJWT("test_user", "viewer")
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
			token:          "Bearer invalid-token",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":{"status":401,"name":"InvalidTokenError","message":"Invalid token","details":null}}`,
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
	testutils.SetupTestServer() // Initializes DB and roles
	defer testutils.CleanupTestDB()
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
	userToken, err := auth.GenerateJWT("user", "viewer")
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
			expectedBody:   `{"error":{"status":403,"name":"ForbiddenError","message":"insufficient permissions","details":null}}`,
		},
		{
			name:           "Missing Token",
			token:          "",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":{"status":401,"name":"UnauthorizedError","message":"Authorization header required","details":null}}`,
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
