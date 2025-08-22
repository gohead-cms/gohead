// internal/api/handlers/auth_test.go
package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gohead-cms/gohead/internal/models"

	"github.com/gohead-cms/gohead/pkg/database"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/testutils"

	"golang.org/x/crypto/bcrypt"

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
	// Setup the test server and database
	r, db := testutils.SetupTestServer()
	defer testutils.CleanupTestDB()

	// Apply necessary migrations
	assert.NoError(t, db.AutoMigrate(&models.User{}))

	// Define the test route and handler
	r.POST("/auth/register", Register)

	// Create an example user payload for testing
	examplePayload := map[string]string{
		"username":  "testreader",
		"password":  "securepassword",
		"email":     "testreader@example.com",
		"role_name": "viewer",
	}
	// Marshal the payload to JSON
	userJSON, err := json.Marshal(examplePayload)
	assert.NoError(t, err)

	// Create the HTTP request
	req, err := http.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(string(userJSON)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// I don't need to test more, integration tests will do the rest of the edge cases
	assert.NoError(t, err)
	assert.Equal(t, 200, w.Code)
}

func TestLogin(t *testing.T) {
	// Setup the test database
	router, db := testutils.SetupTestServer()
	defer testutils.CleanupTestDB()

	// Apply migrations
	assert.NoError(t, db.AutoMigrate(&models.User{}, &models.UserRole{}))

	// Seed roles
	adminRole := models.UserRole{Name: "admin", Description: "Administrator", Permissions: models.JSONMap{"manage_users": true}}
	assert.NoError(t, db.Create(&adminRole).Error)

	// Seed a user with a hashed password
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	user := models.User{
		Username: "testadmin",
		Password: string(hashedPassword), // Use the hashed password
		Email:    "testadmin@example.com",
		Role:     adminRole,
		Slug:     "testadmin",
	}
	assert.NoError(t, db.Create(&user).Error)

	// Ensure the test database is used globally
	database.DB = db

	// Initialize the router and attach the handler
	router.POST("/auth/login", Login)

	// Test valid login
	payload := map[string]string{
		"username": "testadmin",
		"password": "password123",
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}
