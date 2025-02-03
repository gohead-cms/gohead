package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"gohead/pkg/logger"
)

// TestSingleTypeHandlers demonstrates minimal tests for SingleType endpoints
func TestSingleTypeHandlers(t *testing.T) {
	// Initialize gin in test mode
	gin.SetMode(gin.TestMode)
	logger.InitLogger("debug")

	// Create a router with the routes you want to test
	router := gin.New()

	// Here we assume a ResponseWrapper middleware is used in your real server
	// For simplicity, we manually write JSON in tests or can mock the wrapper.
	// Example routes:
	router.GET("/single-types/:name", GetSingleType)
	router.PUT("/single-types/:name", CreateOrUpdateSingleType)
	router.DELETE("/single-types/:name", DeleteSingleType)

	// ============ TEST 1: GET Single Type (not found) ============
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/single-types/homepage", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	// Optionally, parse response if needed
	// ...

	// ============ TEST 2: CREATE Single Type ============
	// Sample payload for creating a single type (like a "homepage")
	singleTypePayload := map[string]interface{}{
		"name": "homepage",
		"attributes": map[string]interface{}{
			"title": map[string]interface{}{
				"type":     "string",
				"required": true,
			},
			"heroText": map[string]interface{}{
				"type": "richtext",
			},
		},
	}
	payloadBytes, _ := json.Marshal(singleTypePayload)
	req, _ = http.NewRequest("PUT", "/single-types/homepage", bytes.NewBuffer(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	// ============ TEST 3: GET Single Type (now found) ============
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/single-types/homepage", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	// Optionally, parse and verify JSON fields in the response

	// ============ TEST 4: DELETE Single Type (optional) ============
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("DELETE", "/single-types/homepage", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// GET again should be 404
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/single-types/homepage", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}
