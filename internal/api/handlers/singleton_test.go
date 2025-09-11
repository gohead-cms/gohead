package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gohead-cms/gohead/internal/api/middleware"
	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/storage"
	"github.com/gohead-cms/gohead/pkg/testutils"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// Initialize logger for testing
func init() {
	var buffer bytes.Buffer
	logger.InitLogger("debug")
	logger.Log.SetOutput(&buffer)
	logger.Log.SetFormatter(&logrus.TextFormatter{})
}

func TestSingletonHandlers(t *testing.T) {
	// Setup the test database and router
	router, db := testutils.SetupTestServer()
	router.Use(middleware.ResponseWrapper())
	defer testutils.CleanupTestDB()

	// Apply migrations for Singleton and its attributes
	assert.NoError(t, db.AutoMigrate(&models.Singleton{}, &models.Attribute{}))

	// Register the single type routes
	// Assuming CreateOrUpdateSingleton is the handler for PUT
	router.PUT("/singletons/:name", CreateOrUpdateSingleton)
	router.GET("/singletons/:name", GetSingleton)
	router.DELETE("/singletons/:name", DeleteSingleton)

	// --- Test Data ---
	SingletonName := "homepage"
	validSingletonPayload := map[string]any{
		"name": "homepage",
		"attributes": map[string]any{
			"title": map[string]any{
				"type":     "string",
				"required": true,
			},
			"hero_text": map[string]any{
				"type": "richtext",
			},
		},
	}

	// --- Sub-test for Create/Update ---
	t.Run("CreateOrUpdate Singleton", func(t *testing.T) {
		body, _ := json.Marshal(validSingletonPayload)
		req, _ := http.NewRequest(http.MethodPut, "/singletons/"+SingletonName, bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		// Assuming the first time it's created, the status is 201
		assert.Equal(t, http.StatusCreated, rr.Code)

		// Verify in DB
		_, err := storage.GetSingletonByName(SingletonName)
		assert.NoError(t, err, "Singleton should exist in the database after creation")
	})

	// --- Sub-test for Get ---
	t.Run("Get Singleton", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/singletons/"+SingletonName, nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response models.Singleton
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, SingletonName, response.Name)
		assert.Len(t, response.Attributes, 2, "Expected single type to have 2 attributes")
	})

	// --- Sub-test for Get Not Found ---
	t.Run("Get Singleton Not Found", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/singletons/nonexistent", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	// --- Sub-test for Delete ---
	t.Run("Delete Singleton", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodDelete, "/singletons/"+SingletonName, nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		// Verify deletion in DB
		_, err := storage.GetSingletonByName(SingletonName)
		assert.Error(t, err, "Singleton should not exist in the database after deletion")
	})

	// --- Sub-test for Delete Not Found ---
	t.Run("Delete Singleton Not Found", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodDelete, "/singletons/nonexistent", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		// This status code might depend on your handler's implementation (e.g., 404 or 400)
		assert.Equal(t, http.StatusBadRequest, rr.Code, "Expected bad request for deleting non-existent single type")
	})
}
