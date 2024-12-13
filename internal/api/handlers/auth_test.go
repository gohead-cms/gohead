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
	"gitlab.com/sudo.bngz/gohead/pkg/testutils"
	"golang.org/x/crypto/bcrypt"

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

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	return router
}

func TestRegister(t *testing.T) {
	// Setup the test database
	// Initialize in-memory test database
	db := testutils.SetupTestDB()
	defer testutils.CleanupTestDB()

	// Apply migrations
	assert.NoError(t, db.AutoMigrate(&models.User{}, &models.UserRole{}))

	// Seed roles
	adminRole := models.UserRole{Name: "admin", Description: "Administrator", Permissions: models.JSONMap{"manage_users": true}}
	readerRole := models.UserRole{Name: "reader", Description: "Reader", Permissions: models.JSONMap{"read_content": true}}
	assert.NoError(t, db.Create(&adminRole).Error)
	assert.NoError(t, db.Create(&readerRole).Error)

	// Initialize the router and attach the handler
	router := setupTestRouter()
	router.POST("/auth/register", Register)

	// Test valid registration with default role (reader)
	payload := map[string]string{
		"username":  "testreader",
		"password":  "securepassword",
		"email":     "testreader@example.com",
		"role_name": "reader",
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "User registered successfully", response["message"])

	// Test registration with an invalid role
	payload["role_name"] = "invalid_role"
	body, _ = json.Marshal(payload)

	req, _ = http.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "Role 'invalid_role' does not exist")

	// Test registration with duplicate username
	payload["role_name"] = "reader"
	body, _ = json.Marshal(payload)

	req, _ = http.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "duplicate entry for field: username")
}

func TestLogin(t *testing.T) {
	// Setup the test database
	db := testutils.SetupTestDB()
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
	router := setupTestRouter()
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

	var response map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotEmpty(t, response["token"])
}
