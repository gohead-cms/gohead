// internal/api/handlers/auth_test.go
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

func TestRegister(t *testing.T) {
	// Initialize in-memory test database
	router, db := testutils.SetupTestServer()

	// Apply migrations
	err := db.AutoMigrate(&models.User{})
	assert.NoError(t, err, "Failed to apply migrations")

	// Set Gin to Test Mode
	gin.SetMode(gin.TestMode)

	// Create a Gin router and register the route
	router.POST("/auth/register", Register)

	// Define test user data
	testUser := map[string]string{
		"username": "testuser",
		"password": "testpass",
		"email":    "joe@foo.com",
	}
	body, _ := json.Marshal(testUser)

	// Create a new HTTP request
	req, err := http.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(body))
	assert.NoError(t, err, "Failed to create HTTP request")
	req.Header.Set("Content-Type", "application/json")

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()

	// Serve the HTTP request
	router.ServeHTTP(rr, req)

	// Validate the response
	assert.Equal(t, http.StatusCreated, rr.Code, "Expected HTTP 201 Created")
	var response map[string]string
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err, "Failed to unmarshal response body")
	assert.Equal(t, "User registered successfully", response["message"])

	// Validate the user is stored in the database
	var user models.User
	err = db.First(&user, "username = ?", "testuser").Error
	assert.NoError(t, err, "User should exist in the database")
	assert.Equal(t, "testuser", user.Username, "Stored username should match test user")
}
