// internal/api/handlers/agent_test.go
package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/testutils"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// Initialize logger for testing (same style as type_test.go)
func init() {
	var buffer bytes.Buffer
	logger.InitLogger("debug")
	logger.Log.SetOutput(&buffer)
	logger.Log.SetFormatter(&logrus.TextFormatter{})
}

func TestAgentHandlers(t *testing.T) {
	// Setup test server and DB
	router, db := testutils.SetupTestServer()
	defer testutils.CleanupTestDB()

	// Migrate Agent (and anything it needs)
	assert.NoError(t, db.AutoMigrate(&models.Agent{}))

	gin.SetMode(gin.TestMode)

	// Mount routes for agents
	router.GET("/agents", GetAgents)
	router.GET("/agents/:name", GetAgent)
	router.POST("/agents", CreateAgent)
	router.PUT("/agents/:name", UpdateAgent)
	router.DELETE("/agents/:name", DeleteAgent)

	// ---- Helpers ----
	validAgentInput := func(name string) map[string]any {
		return map[string]any{
			"name":          name,
			"system_prompt": "You are a test agent.",
			"max_turns":     5,
			"llm_config": map[string]any{
				"provider": "openai",
				"model":    "gpt-3.5-turbo",
			},
			"memory": map[string]any{
				"type":          "postgres",
				"session_scope": "conversation",
			},
			"trigger": map[string]any{
				"type": "manual",
			},
			"functions": []any{
				map[string]any{
					"impl_key":    "tool.test",
					"name":        "TestTool",
					"description": "A test tool.",
					"parameters":  `{"type":"object","properties":{}}`,
				},
			},
		}
	}

	// ---------- CreateAgent ----------
	t.Run("CreateAgent - Valid", func(t *testing.T) {
		body, _ := json.Marshal(validAgentInput("agent-alpha"))
		req, _ := http.NewRequest(http.MethodPost, "/agents", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)
		assert.Contains(t, rr.Body.String(), "Agent created successfully")
	})

	t.Run("CreateAgent - Missing Name", func(t *testing.T) {
		payload := validAgentInput("")
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPost, "/agents", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "agent must have a 'name'")
	})

	t.Run("CreateAgent - Duplicate Name", func(t *testing.T) {
		// Create initial
		bodyA, _ := json.Marshal(validAgentInput("agent-dup"))
		reqA, _ := http.NewRequest(http.MethodPost, "/agents", bytes.NewBuffer(bodyA))
		reqA.Header.Set("Content-Type", "application/json")
		rrA := httptest.NewRecorder()
		router.ServeHTTP(rrA, reqA)
		assert.Equal(t, http.StatusCreated, rrA.Code)

		// Create duplicate
		bodyB, _ := json.Marshal(validAgentInput("agent-dup"))
		reqB, _ := http.NewRequest(http.MethodPost, "/agents", bytes.NewBuffer(bodyB))
		reqB.Header.Set("Content-Type", "application/json")
		rrB := httptest.NewRecorder()
		router.ServeHTTP(rrB, reqB)

		assert.Equal(t, http.StatusBadRequest, rrB.Code)
		assert.Contains(t, rrB.Body.String(), "This agent already exists")
	})

	// ---------- GetAgent ----------
	t.Run("GetAgent - Exists", func(t *testing.T) {
		// Ensure exists
		body, _ := json.Marshal(validAgentInput("agent-get"))
		reqC, _ := http.NewRequest(http.MethodPost, "/agents", bytes.NewBuffer(body))
		reqC.Header.Set("Content-Type", "application/json")
		rrC := httptest.NewRecorder()
		router.ServeHTTP(rrC, reqC)
		assert.Equal(t, http.StatusCreated, rrC.Code)

		req, _ := http.NewRequest(http.MethodGet, "/agents/agent-get", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Body.String(), "agent-get")
	})

	t.Run("GetAgent - Not Found", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/agents/does-not-exist", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
		assert.Contains(t, rr.Body.String(), "Agent not found")
	})

	// ---------- GetAgents (list) ----------
	t.Run("GetAgents - List", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/agents", nil)
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		// Should include Content-Range header and at least one agent's name
		assert.NotEmpty(t, rr.Header().Get("Content-Range"))
		assert.Contains(t, rr.Body.String(), "agent-alpha")
	})

	// ---------- UpdateAgent ----------
	t.Run("UpdateAgent - Valid", func(t *testing.T) {
		// Seed a record to update
		bodySeed, _ := json.Marshal(validAgentInput("agent-update"))
		reqSeed, _ := http.NewRequest(http.MethodPost, "/agents", bytes.NewBuffer(bodySeed))
		reqSeed.Header.Set("Content-Type", "application/json")
		rrSeed := httptest.NewRecorder()
		router.ServeHTTP(rrSeed, reqSeed)
		assert.Equal(t, http.StatusCreated, rrSeed.Code)

		// Update payload
		update := validAgentInput("agent-update")
		update["system_prompt"] = "Updated system prompt."
		body, _ := json.Marshal(update)

		req, _ := http.NewRequest(http.MethodPut, "/agents/agent-update", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Body.String(), "Updated system prompt.")
	})

	t.Run("UpdateAgent - Not Found", func(t *testing.T) {
		update := validAgentInput("nope")
		update["system_prompt"] = "X"
		body, _ := json.Marshal(update)

		req, _ := http.NewRequest(http.MethodPut, "/agents/nope", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
		assert.Contains(t, rr.Body.String(), "Agent not found")
	})

	// ---------- DeleteAgent ----------
	t.Run("DeleteAgent - Success", func(t *testing.T) {
		// Seed
		bodySeed, _ := json.Marshal(validAgentInput("agent-delete"))
		reqSeed, _ := http.NewRequest(http.MethodPost, "/agents", bytes.NewBuffer(bodySeed))
		reqSeed.Header.Set("Content-Type", "application/json")
		rrSeed := httptest.NewRecorder()
		router.ServeHTTP(rrSeed, reqSeed)
		assert.Equal(t, http.StatusCreated, rrSeed.Code)

		req, _ := http.NewRequest(http.MethodDelete, "/agents/agent-delete", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Body.String(), "Agent deleted successfully")
	})

	t.Run("DeleteAgent - Not Found", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodDelete, "/agents/no-such-agent", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
		assert.Contains(t, rr.Body.String(), "Agent not found")
	})
}
