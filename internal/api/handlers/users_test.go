package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gohead-cms/gohead/internal/api/middleware"
	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/storage"
	"github.com/gohead-cms/gohead/pkg/testutils"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// setupUsersRouter wires the admin user handlers behind the response wrapper,
// with a fake authenticated admin named "root" in the context.
func setupUsersRouter(t *testing.T) *gin.Engine {
	t.Helper()
	router, db := testutils.SetupTestServer()
	t.Cleanup(testutils.CleanupTestDB)
	require.NoError(t, db.AutoMigrate(&models.User{}))

	router.Use(func(c *gin.Context) {
		c.Set("username", "root")
		c.Set("role", "admin")
		c.Next()
	})
	router.Use(middleware.ResponseWrapper())
	router.POST("/users", CreateUser)
	router.GET("/users", GetAllUsers)
	router.GET("/users/:id", GetUser)
	router.PUT("/users/:id", UpdateUser)
	router.DELETE("/users/:id", DeleteUser)
	return router
}

func doJSON(router *gin.Engine, method, path string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req, _ := http.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func seedUser(t *testing.T, username, roleName string) *models.User {
	t.Helper()
	role, err := storage.GetRoleByName(roleName)
	require.NoError(t, err)
	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.MinCost)
	u := &models.User{
		Username: username,
		Email:    username + "@example.com",
		Password: string(hash),
		Role:     *role,
		Slug:     username,
	}
	require.NoError(t, storage.CreateUser(u))
	return u
}

func TestCreateUser_HashesPasswordAndAssignsRole(t *testing.T) {
	router := setupUsersRouter(t)

	rec := doJSON(router, http.MethodPost, "/users", map[string]string{
		"username":  "alice",
		"email":     "alice@example.com",
		"password":  "s3cretpass",
		"role_name": "editor",
	})
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	assert.NotContains(t, rec.Body.String(), "s3cretpass")
	assert.NotContains(t, rec.Body.String(), `"password"`)

	stored, err := storage.GetUserByUsername("alice")
	require.NoError(t, err)
	assert.Equal(t, "editor", stored.Role.Name)
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(stored.Password), []byte("s3cretpass")))
}

func TestCreateUser_UnknownRole(t *testing.T) {
	router := setupUsersRouter(t)
	rec := doJSON(router, http.MethodPost, "/users", map[string]string{
		"username": "bob", "email": "bob@example.com", "password": "s3cretpass", "role_name": "superuser",
	})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreateUser_Duplicate(t *testing.T) {
	router := setupUsersRouter(t)
	seedUser(t, "carol", "viewer")
	rec := doJSON(router, http.MethodPost, "/users", map[string]string{
		"username": "carol", "email": "other@example.com", "password": "s3cretpass",
	})
	assert.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
}

func TestGetAllUsers_DoesNotLeakHashes(t *testing.T) {
	router := setupUsersRouter(t)
	seedUser(t, "user1", "viewer")
	seedUser(t, "user2", "admin")

	rec := doJSON(router, http.MethodGet, "/users", nil)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.NotContains(t, rec.Body.String(), "$2a$")

	var resp struct {
		Data struct {
			Users []map[string]any `json:"users"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Len(t, resp.Data.Users, 2)
}

func TestGetUser(t *testing.T) {
	router := setupUsersRouter(t)
	u := seedUser(t, "user1", "viewer")

	rec := doJSON(router, http.MethodGet, fmt.Sprintf("/users/%d", u.ID), nil)
	require.Equal(t, http.StatusOK, rec.Code)
	var resp struct {
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "user1", resp.Data["username"])

	rec = doJSON(router, http.MethodGet, "/users/9999", nil)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestUpdateUser(t *testing.T) {
	router := setupUsersRouter(t)
	u := seedUser(t, "user1", "viewer")
	path := fmt.Sprintf("/users/%d", u.ID)

	rec := doJSON(router, http.MethodPut, path, map[string]any{
		"email":     "updated@example.com",
		"password":  "brandnewpass",
		"role_name": "editor",
	})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	updated, err := storage.GetUserByID(u.ID)
	require.NoError(t, err)
	assert.Equal(t, "updated@example.com", updated.Email)
	assert.Equal(t, "editor", updated.Role.Name)
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(updated.Password), []byte("brandnewpass")))
}

func TestUpdateUser_RejectsArbitraryFields(t *testing.T) {
	router := setupUsersRouter(t)
	u := seedUser(t, "user1", "viewer")
	rec := doJSON(router, http.MethodPut, fmt.Sprintf("/users/%d", u.ID), map[string]any{
		"user_role_id": 1,
	})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestDeleteUser(t *testing.T) {
	router := setupUsersRouter(t)
	u := seedUser(t, "user1", "viewer")

	rec := doJSON(router, http.MethodDelete, fmt.Sprintf("/users/%d", u.ID), nil)
	require.Equal(t, http.StatusOK, rec.Code)
	_, err := storage.GetUserByID(u.ID)
	assert.Error(t, err)
}

func TestDeleteUser_CannotDeleteSelf(t *testing.T) {
	router := setupUsersRouter(t)
	root := seedUser(t, "root", "admin")
	rec := doJSON(router, http.MethodDelete, fmt.Sprintf("/users/%d", root.ID), nil)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
