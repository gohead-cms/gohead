// internal/api/handlers/auth_test.go
package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gohead-cms/gohead/internal/api/middleware"
	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/storage"

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
	// Handler tests register accounts; production defaults to disabled.
	Registration = RegistrationOpen
}

func TestRegister(t *testing.T) {
	r, db := testutils.SetupTestServer()
	defer testutils.CleanupTestDB()
	assert.NoError(t, db.AutoMigrate(&models.User{}))
	r.Use(middleware.ResponseWrapper())
	r.POST("/auth/register", Register)

	register := func(username string, extra map[string]string) *httptest.ResponseRecorder {
		payload := map[string]string{
			"username": username,
			"password": "securepassword",
			"email":    username + "@example.com",
		}
		for k, v := range extra {
			payload[k] = v
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}

	t.Run("disabled registration is refused", func(t *testing.T) {
		Registration = RegistrationDisabled
		defer func() { Registration = RegistrationOpen }()
		w := register("intruder", nil)
		assert.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
		_, err := storage.GetUserByUsername("intruder")
		assert.Error(t, err, "no account was created")
	})

	t.Run("the first account is not an admin", func(t *testing.T) {
		w := register("founder", nil)
		assert.Equal(t, http.StatusCreated, w.Code, w.Body.String())
		u, err := storage.GetUserByUsername("founder")
		assert.NoError(t, err)
		assert.Equal(t, DefaultRegistrationRole, u.Role.Name, "admins come from the configuration or the CLI")
	})

	t.Run("client cannot choose its role", func(t *testing.T) {
		w := register("mallory", map[string]string{"role_name": "admin"})
		assert.Equal(t, http.StatusCreated, w.Code, w.Body.String())
		u, err := storage.GetUserByUsername("mallory")
		assert.NoError(t, err)
		assert.Equal(t, DefaultRegistrationRole, u.Role.Name)
	})

	t.Run("short password rejected", func(t *testing.T) {
		payload, _ := json.Marshal(map[string]string{"username": "shorty", "password": "123", "email": "s@example.com"})
		req, _ := http.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("duplicate username", func(t *testing.T) {
		w := register("founder", nil)
		assert.Equal(t, http.StatusConflict, w.Code, w.Body.String())
	})
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
