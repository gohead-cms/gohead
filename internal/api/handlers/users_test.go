package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"gitlab.com/sudo.bngz/gohead/internal/models"
	"gitlab.com/sudo.bngz/gohead/pkg/logger"
	"gitlab.com/sudo.bngz/gohead/pkg/storage"
)

// Initialize logger for testing
func init() {
	// Configure logger to write logs to a buffer for testing
	var buffer bytes.Buffer
	logger.InitLogger("debug")
	logger.Log.SetOutput(&buffer)
	logger.Log.SetFormatter(&logrus.TextFormatter{})
}

func setupRouter() *gin.Engine {
	router := gin.Default()
	router.POST("/users", CreateUser)
	router.GET("/users", GetAllUsers)
	router.GET("/users/:id", GetUser)
	router.PUT("/users/:id", UpdateUser)
	router.DELETE("/users/:id", DeleteUser)
	return router
}

func TestCreateUser(t *testing.T) {
	router := setupRouter()

	newUser := models.User{
		Username: "test_user",
		Email:    "test_user@example.com",
		Role:     models.UserRole{Name: "user"},
	}
	payload, _ := json.Marshal(newUser)

	req, _ := http.NewRequest("POST", "/users", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusCreated, resp.Code)
	var response map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "User created successfully", response["message"])
}

func TestGetAllUsers(t *testing.T) {
	router := setupRouter()

	err := storage.CreateUser(&models.User{
		Username: "user1",
		Email:    "user1@example.com",
		Role:     models.UserRole{Name: "user"},
	})
	assert.NoError(t, err)
	err = storage.CreateUser(&models.User{
		Username: "user2",
		Email:    "user2@example.com",
		Role:     models.UserRole{Name: "admin"},
	})
	assert.NoError(t, err)
	req, _ := http.NewRequest("GET", "/users", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	var response map[string]interface{}
	err = json.Unmarshal(resp.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Len(t, response["users"], 2)
}

func TestGetUser(t *testing.T) {
	router := setupRouter()

	mockUser := models.User{
		Username: "user1",
		Email:    "user1@example.com",
		Role:     models.UserRole{Name: "user"},
	}
	err := storage.CreateUser(&mockUser)
	assert.NoError(t, err)
	req, _ := http.NewRequest("GET", "/users/1", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	var response map[string]interface{}
	err = json.Unmarshal(resp.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "user1", response["user"].(map[string]interface{})["username"])
}

func TestUpdateUser(t *testing.T) {
	router := setupRouter()

	mockUser := models.User{
		Username: "user1",
		Email:    "user1@example.com",
		Role:     models.UserRole{Name: "user"},
	}
	err := storage.CreateUser(&mockUser)
	assert.NoError(t, err)
	updates := map[string]interface{}{
		"email": "updated_user@example.com",
	}
	payload, _ := json.Marshal(updates)

	req, _ := http.NewRequest("PUT", "/users/1", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	var response map[string]interface{}
	err = json.Unmarshal(resp.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "User updated successfully", response["message"])

	updatedUser, _ := storage.GetUserByID(1)
	assert.Equal(t, "updated_user@example.com", updatedUser.Email)
}

func TestDeleteUser(t *testing.T) {
	router := setupRouter()

	mockUser := models.User{
		Username: "user1",
		Email:    "user1@example.com",
		Role:     models.UserRole{Name: "user"},
	}
	err := storage.CreateUser(&mockUser)
	assert.NoError(t, err)
	req, _ := http.NewRequest("DELETE", "/users/1", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	var response map[string]interface{}
	err = json.Unmarshal(resp.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "User deleted successfully", response["message"])

	_, err = storage.GetUserByID(1)
	assert.NotNil(t, err)
}
