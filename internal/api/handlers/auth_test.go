// internal/api/handlers/auth_test.go
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

func TestRegister(t *testing.T) {
	logger.InitLogger("debug")
	// Initialize in-memory test database
	db, err := database.InitDatabase("sqlite://:memory:")
	assert.NoError(t, err, "Failed to initialize in-memory database")
	database.DB = db

	// Apply migrations
	err = db.AutoMigrate(&models.User{})
	assert.NoError(t, err, "Failed to apply migrations")

	// Set Gin to Test Mode
	gin.SetMode(gin.TestMode)

	// Create a Gin router and register the route
	router := gin.Default()
	router.POST("/auth/register", Register)

	// Define test user data
	testUser := map[string]string{
		"username": "testuser",
		"password": "testpass",
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
