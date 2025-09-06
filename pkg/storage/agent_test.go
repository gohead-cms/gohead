package storage_test

import (
	"bytes"
	"testing"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/storage"
	"github.com/gohead-cms/gohead/pkg/testutils"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func init() {
	// Configure logger to write logs to a buffer for testing
	var buffer bytes.Buffer
	logger.InitLogger("debug")
	logger.Log.SetOutput(&buffer)
	logger.Log.SetFormatter(&logrus.TextFormatter{})
}

func TestAgentStorage(t *testing.T) {
	// Set up the test database
	_, db := testutils.SetupTestServer()
	defer testutils.CleanupTestDB()

	// Apply migrations for the Agent model
	err := db.AutoMigrate(&models.Agent{})
	assert.NoError(t, err, "Failed to apply migrations for Agent model")

	// Seed initial data
	testAgent := &models.Agent{
		Name: "CustomerServiceAgent",
		// The Config field must contain a valid JSON object matching the Go struct.
		Config: `{
            "ID": 1,
            "Name": "CustomerServiceAgent",
            "SystemPrompt": "You are a helpful assistant.",
            "MaxTurns": 5
        }`,
	}
	err = db.Create(testAgent).Error
	assert.NoError(t, err, "Failed to seed initial 'CustomerServiceAgent'")

	t.Run("SaveAgent_CreateNew", func(t *testing.T) {
		newAgent := &models.Agent{
			Name: "MarketingAgent",
			Config: `{
                "Name": "MarketingAgent",
                "SystemPrompt": "You generate creative marketing copy."
            }`,
		}
		err := storage.SaveAgent(newAgent)
		assert.NoError(t, err, "Expected no error saving new agent")
		assert.Greater(t, newAgent.ID, uint(0), "Expected a new ID to be assigned")
	})

	t.Run("SaveAgent_Conflict", func(t *testing.T) {
		// Attempt to save an agent with a name that already exists
		conflictingAgent := &models.Agent{
			Name:   "CustomerServiceAgent",
			Config: `{"Name": "CustomerServiceAgent"}`,
		}
		err := storage.SaveAgent(conflictingAgent)
		assert.Error(t, err, "Expected an error when saving an agent with a conflicting name")
		assert.Contains(t, err.Error(), "already exists", "Expected a conflict error message")
	})

	t.Run("GetAgentByID_Success", func(t *testing.T) {
		retrievedAgent, err := storage.GetAgentByID(testAgent.ID)
		assert.NoError(t, err, "Expected no error retrieving agent by ID")
		assert.NotNil(t, retrievedAgent, "Expected agent to be retrieved")
		assert.Equal(t, testAgent.Name, retrievedAgent.Name, "Agent name mismatch")
	})

	t.Run("GetAgentByID_NotFound", func(t *testing.T) {
		_, err := storage.GetAgentByID(9999) // Non-existent ID
		assert.Error(t, err, "Expected an error for non-existent agent")
		assert.Contains(t, err.Error(), "not found", "Expected 'not found' error message")
	})

	t.Run("GetAgentByName_Success", func(t *testing.T) {
		retrievedAgent, err := storage.GetAgentByName("MarketingAgent")
		assert.NoError(t, err, "Expected no error retrieving agent by name")
		assert.NotNil(t, retrievedAgent, "Expected agent to be retrieved")
		assert.Equal(t, "MarketingAgent", retrievedAgent.Name, "Agent name mismatch")
	})

	t.Run("GetAgentByName_NotFound", func(t *testing.T) {
		_, err := storage.GetAgentByName("NonExistentAgent")
		assert.Error(t, err, "Expected an error for non-existent agent name")
		assert.Contains(t, err.Error(), "not found", "Expected 'not found' error message")
	})

	t.Run("GetAllAgents_NoFilters", func(t *testing.T) {
		agents, total, err := storage.GetAllAgents(nil, nil)
		assert.NoError(t, err, "Expected no error retrieving all agents")
		assert.GreaterOrEqual(t, len(agents), 2, "Expected at least two agents")
		assert.GreaterOrEqual(t, total, 2, "Expected total count to be >= 2")
	})

	t.Run("GetAllAgents_WithFilter", func(t *testing.T) {
		filters := map[string]interface{}{"name": "CustomerServiceAgent"}
		agents, total, err := storage.GetAllAgents(filters, nil)
		assert.NoError(t, err, "Expected no error retrieving filtered agents")
		assert.Equal(t, 1, len(agents), "Expected exactly one agent with name = 'CustomerServiceAgent'")
		assert.Equal(t, 1, total, "Expected total count = 1 for name='CustomerServiceAgent'")
		assert.Equal(t, "CustomerServiceAgent", agents[0].Name, "Agent name mismatch")
	})

	t.Run("GetAllAgents_WithRange", func(t *testing.T) {
		// Range as [0, 0] => first item only
		rangeValues := []int{0, 0}
		agents, total, err := storage.GetAllAgents(nil, rangeValues)
		assert.NoError(t, err, "Expected no error retrieving paginated agents")
		assert.Equal(t, 1, len(agents), "Expected 1 agent in this range")
		assert.GreaterOrEqual(t, total, 2)
	})

	t.Run("UpdateAgent_Success", func(t *testing.T) {
		// Retrieve the agent we want to update
		agentToUpdate, err := storage.GetAgentByName("CustomerServiceAgent")
		assert.NoError(t, err, "Failed to retrieve agent for update")

		// Modify the agent's properties
		agentToUpdate.Name = "UpdatedCustomerAgent"
		agentToUpdate.Config = `{"Name": "UpdatedCustomerAgent"}`

		// Call the update function
		err = storage.UpdateAgent(agentToUpdate.ID, agentToUpdate)
		assert.NoError(t, err, "Expected no error when updating agent")

		// Retrieve and verify the changes
		retrievedAgent, err := storage.GetAgentByID(agentToUpdate.ID)
		assert.NoError(t, err)
		assert.Equal(t, "UpdatedCustomerAgent", retrievedAgent.Name, "Name was not updated")
	})

	t.Run("DeleteAgent_Success", func(t *testing.T) {
		// Create a new agent to be deleted
		agentToDelete := &models.Agent{
			Name:   "TemporaryAgent",
			Config: `{"Name": "TemporaryAgent"}`,
		}
		err := db.Create(agentToDelete).Error
		assert.NoError(t, err, "Failed to create agent for deletion test")

		// Call the delete function
		err = storage.DeleteAgent(agentToDelete.ID)
		assert.NoError(t, err, "Expected no error when deleting agent")

		// Verify it's gone
		_, err = storage.GetAgentByID(agentToDelete.ID)
		assert.Error(t, err, "Expected an error for the deleted agent")
		assert.Contains(t, err.Error(), "not found", "Expected 'not found' error after deletion")
	})

	t.Run("DeleteAgent_NotFound", func(t *testing.T) {
		err := storage.DeleteAgent(9999) // Non-existent ID
		assert.Error(t, err, "Expected an error when deleting a non-existent agent")
		assert.Contains(t, err.Error(), "not found", "Expected 'not found' error message")
	})
}
