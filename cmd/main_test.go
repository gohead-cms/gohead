package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/sudo.bngz/gohead/pkg/config"
)

func TestInitializeServer(t *testing.T) {
	// Create a temporary test configuration
	testConfig := &config.Config{
		DatabaseURL:      "sqlite://:memory:",
		LogLevel:         "debug",
		JWTSecret:        "test-secret",
		ServerPort:       "8080",
		TelemetryEnabled: false,
	}

	// Write the test configuration to a temporary file
	cfgPath := "config_test.yaml"
	err := config.SaveConfig(testConfig, cfgPath) // Implement SaveConfig if not available
	assert.NoError(t, err, "Failed to save test config file")
	defer os.Remove(cfgPath)

	// Initialize the server
	router, err := InitializeServer(cfgPath)
	assert.NoError(t, err, "Server initialization should not fail")
	assert.NotNil(t, router, "Router should not be nil")

	// Test a health check endpoint
	req, _ := http.NewRequest("GET", "/_health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Healthcheck should return 200 OK")
	assert.Contains(t, w.Body.String(), "ok", "Healthcheck response should contain 'ok'")
}
