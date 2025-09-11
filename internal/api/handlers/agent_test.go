package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gohead-cms/gohead/internal/api/middleware"
	agents "github.com/gohead-cms/gohead/internal/models/agents"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/testutils"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// Initialize logger for testing
func init() {
	// Configure logger to write logs to a buffer for testing
	var buffer bytes.Buffer
	logger.InitLogger("debug")
	logger.Log.SetOutput(&buffer)
	logger.Log.SetFormatter(&logrus.TextFormatter{})
}

func TestGetAgentsHandler(t *testing.T) {
	// Setup the test database and router
	router, db := testutils.SetupTestServer()
	router.Use(middleware.ResponseWrapper())
	defer testutils.CleanupTestDB()
	assert.NoError(t, db.AutoMigrate(&agents.Agent{}))

	// Seed data
	agent1 := agents.Agent{
		Name:         "agent_one",
		SystemPrompt: "...",
		MaxTurns:     4,
		LLMConfig:    agents.LLMConfig{Provider: "test_provider", Model: "test_model"},
		Memory:       agents.MemoryConfig{Type: "in-memory", SessionScope: "user"},
		Trigger:      agents.TriggerConfig{Type: "manual"},
	}
	agent2 := agents.Agent{
		Name:         "agent_two",
		SystemPrompt: "...",
		MaxTurns:     4,
		LLMConfig:    agents.LLMConfig{Provider: "test_provider", Model: "test_model"},
		Memory:       agents.MemoryConfig{Type: "in-memory", SessionScope: "user"},
		Trigger:      agents.TriggerConfig{Type: "manual"},
	}
	db.Create(&agent1)
	db.Create(&agent2)

	// Register the handler
	router.GET("/agents", GetAgents)

	testCases := []struct {
		name           string
		queryParams    string
		expectedStatus int
		expectedBody   string
		expectedHeader string
	}{
		{
			name:           "Success without pagination",
			queryParams:    "",
			expectedStatus: http.StatusOK,
			expectedBody:   `"name":"agent_one"`,
			expectedHeader: "items 0-1/2",
		},
		{
			name:           "Success with pagination",
			queryParams:    "?page=1&pageSize=1",
			expectedStatus: http.StatusOK,
			expectedBody:   `"name":"agent_one"`,
			expectedHeader: "items 0-1/2",
		},
		{
			name:           "Invalid range format",
			queryParams:    "?range=invalid",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid range format",
			expectedHeader: "",
		},
		{
			name:           "Invalid filter format",
			queryParams:    `?filter={"name"`,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid filter format",
			expectedHeader: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, "/agents"+tc.queryParams, nil)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			assert.Equal(t, tc.expectedStatus, rr.Code)
			if tc.expectedBody != "" {
				assert.Contains(t, rr.Body.String(), tc.expectedBody)
			}
			if tc.expectedHeader != "" {
				assert.Equal(t, tc.expectedHeader, rr.Header().Get("Content-Range"))
			}
		})
	}
}

func TestGetAgentHandler(t *testing.T) {
	router, db := testutils.SetupTestServer()
	defer testutils.CleanupTestDB()
	assert.NoError(t, db.AutoMigrate(&agents.Agent{}))

	// Seed data
	testAgent := agents.Agent{
		Name:         "test_get_agent",
		SystemPrompt: "...",
		MaxTurns:     4,
		LLMConfig:    agents.LLMConfig{Provider: "test_provider", Model: "test_model"},
		Memory:       agents.MemoryConfig{Type: "in-memory", SessionScope: "user"},
		Trigger:      agents.TriggerConfig{Type: "manual"},
	}
	db.Create(&testAgent)

	router.GET("/agents/:name", GetAgent)

	testCases := []struct {
		name           string
		agentName      string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Success",
			agentName:      "test_get_agent",
			expectedStatus: http.StatusOK,
			expectedBody:   `"name":"test_get_agent"`,
		},
		{
			name:           "Agent Not Found",
			agentName:      "nonexistent",
			expectedStatus: http.StatusNotFound,
			expectedBody:   "Agent not found",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, "/agents/"+tc.agentName, nil)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			assert.Equal(t, tc.expectedStatus, rr.Code)
			assert.Contains(t, rr.Body.String(), tc.expectedBody)
		})
	}
}

func TestCreateAgentHandler(t *testing.T) {
	router, db := testutils.SetupTestServer()
	defer testutils.CleanupTestDB()
	assert.NoError(t, db.AutoMigrate(&agents.Agent{}))

	router.POST("/agents", CreateAgent)

	testCases := []struct {
		name           string
		inputData      map[string]any
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Valid Agent",
			inputData: map[string]any{
				"name":          "new_agent",
				"system_prompt": "You are a new agent.",
				"max_turns":     4,
				"llm_config":    map[string]any{"provider": "openai", "model": "gpt-4"},
				"memory":        map[string]any{"type": "in-memory", "session_scope": "user"},
				"trigger":       map[string]any{"type": "manual"},
			},
			expectedStatus: http.StatusCreated,
			expectedBody:   `"message":"Agent created successfully"`,
		},
		{
			name:           "Invalid JSON",
			inputData:      nil, // Simulates an empty body which results in JSON parsing error
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid JSON input",
		},
		{
			name: "Missing Name",
			inputData: map[string]any{
				"system_prompt": "...",
				"llm_config":    map[string]any{"provider": "openai", "model": "gpt-4"},
				"memory":        map[string]any{"type": "in-memory", "session_scope": "user"},
				"trigger":       map[string]any{"type": "manual"},
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "agent must have a 'name'",
		},
		{
			name: "Agent Already Exists",
			inputData: map[string]any{
				"name":          "existing_agent",
				"system_prompt": "...",
				"max_turns":     4,
				"llm_config":    map[string]any{"provider": "openai", "model": "gpt-4"},
				"memory":        map[string]any{"type": "in-memory", "session_scope": "user"},
				"trigger":       map[string]any{"type": "manual"},
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "This agent already exists",
		},
	}

	// Pre-create an agent for the "Agent Already Exists" case
	db.Create(&agents.Agent{
		Name:         "existing_agent",
		SystemPrompt: "...",
		MaxTurns:     4,
		LLMConfig:    agents.LLMConfig{Provider: "openai", Model: "gpt-4"},
		Memory:       agents.MemoryConfig{Type: "in-memory", SessionScope: "user"},
		Trigger:      agents.TriggerConfig{Type: "manual"},
	})

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var body *bytes.Buffer
			if tc.inputData != nil {
				jsonBody, _ := json.Marshal(tc.inputData)
				body = bytes.NewBuffer(jsonBody)
			} else {
				body = bytes.NewBufferString("")
			}

			req, _ := http.NewRequest(http.MethodPost, "/agents", body)
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			assert.Equal(t, tc.expectedStatus, rr.Code)
			assert.Contains(t, rr.Body.String(), tc.expectedBody)
		})
	}
}

func TestUpdateAgentHandler(t *testing.T) {
	router, db := testutils.SetupTestServer()
	defer testutils.CleanupTestDB()
	assert.NoError(t, db.AutoMigrate(&agents.Agent{}))

	// Seed agent to be updated
	testAgent := agents.Agent{
		Name:         "to_be_updated",
		SystemPrompt: "initial prompt",
		MaxTurns:     4,
		LLMConfig:    agents.LLMConfig{Provider: "openai", Model: "gpt-4"},
		Memory:       agents.MemoryConfig{Type: "in-memory", SessionScope: "user"},
		Trigger:      agents.TriggerConfig{Type: "manual"},
	}
	db.Create(&testAgent)

	router.PUT("/agents/:name", UpdateAgent)

	testCases := []struct {
		name           string
		agentName      string
		inputData      map[string]any
		expectedStatus int
		expectedBody   string
	}{
		{
			name:      "Success",
			agentName: "to_be_updated",
			inputData: map[string]any{
				"name":          "updated_agent",
				"system_prompt": "updated prompt",
				"max_turns":     5,
				"llm_config":    map[string]any{"provider": "google", "model": "gemini"},
				"memory":        map[string]any{"type": "in-memory", "session_scope": "user"},
				"trigger":       map[string]any{"type": "manual"},
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"name":"updated_agent"`,
		},
		{
			name:      "Agent Not Found",
			agentName: "nonexistent",
			inputData: map[string]any{
				"name":          "nonexistent",
				"system_prompt": "...",
				"llm_config":    map[string]any{"provider": "openai"},
				"memory":        map[string]any{"type": "in-memory", "session_scope": "user"},
				"trigger":       map[string]any{"type": "manual"},
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   "Agent not found",
		},
		{
			name:      "Validation Fails",
			agentName: "to_be_updated",
			inputData: map[string]any{
				"name":          "invalid_update",
				"system_prompt": "", // Invalid field
				"llm_config":    map[string]any{"provider": "openai"},
				"memory":        map[string]any{"type": "in-memory", "session_scope": "user"},
				"trigger":       map[string]any{"type": "manual"},
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "agent must have a 'system_prompt'",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			jsonBody, _ := json.Marshal(tc.inputData)
			req, _ := http.NewRequest(http.MethodPut, "/agents/"+tc.agentName, bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			assert.Equal(t, tc.expectedStatus, rr.Code)
			assert.Contains(t, rr.Body.String(), tc.expectedBody)
		})
	}
}

func TestDeleteAgentHandler(t *testing.T) {
	router, db := testutils.SetupTestServer()
	defer testutils.CleanupTestDB()
	assert.NoError(t, db.AutoMigrate(&agents.Agent{}))

	// Seed agent to be deleted
	testAgent := agents.Agent{
		Name:         "to_be_deleted",
		SystemPrompt: "...",
		MaxTurns:     4,
		LLMConfig:    agents.LLMConfig{Provider: "openai", Model: "gpt-4"},
		Memory:       agents.MemoryConfig{Type: "in-memory", SessionScope: "user"},
		Trigger:      agents.TriggerConfig{Type: "manual"},
	}
	db.Create(&testAgent)

	router.DELETE("/agents/:name", DeleteAgent)

	testCases := []struct {
		name           string
		agentName      string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Success",
			agentName:      "to_be_deleted",
			expectedStatus: http.StatusOK,
			expectedBody:   `"message":"Agent deleted successfully"`,
		},
		{
			name:           "Agent Not Found",
			agentName:      "nonexistent",
			expectedStatus: http.StatusNotFound,
			expectedBody:   "Agent not found",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodDelete, "/agents/"+tc.agentName, nil)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			assert.Equal(t, tc.expectedStatus, rr.Code)
			assert.Contains(t, rr.Body.String(), tc.expectedBody)

			if tc.name == "Success" {
				// Verify the agent was deleted from the database
				var agent agents.Agent
				result := db.Where("name = ?", tc.agentName).First(&agent)
				assert.ErrorIs(t, result.Error, gorm.ErrRecordNotFound)
			}
		})
	}
}
