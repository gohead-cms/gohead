package storage

import (
	"errors"
	"fmt"
	"slices"

	"gorm.io/gorm"

	models "github.com/gohead-cms/gohead/internal/models/agents"
	"github.com/gohead-cms/gohead/pkg/database"
	"github.com/gohead-cms/gohead/pkg/llm"
	"github.com/gohead-cms/gohead/pkg/logger"
)

// SaveAgent persists an Agent to the database, handling both new and soft-deleted records.
func SaveAgent(agent *models.Agent) error {
	var existing models.Agent

	logger.Log.WithField("agent", agent.Name).Info("Attempting to save agent")

	// Check if a record with the same name already exists, including soft-deleted records.
	err := database.DB.Unscoped().Where("name = ?", agent.Name).First(&existing).Error
	if err == nil {
		// Agent with the same name exists.
		if !existing.DeletedAt.Valid {
			// The agent is not soft-deleted, so it's a conflict.
			logger.Log.WithField("agent", agent.Name).Warn("Agent with the same name already exists")
			return fmt.Errorf("an agent with the name '%s' already exists", agent.Name)
		}

		// The agent is soft-deleted; restore it.
		logger.Log.WithField("agent", agent.Name).Info("Found soft-deleted agent, restoring")
		if err := database.DB.Unscoped().Save(&existing).Error; err != nil {
			logger.Log.WithError(err).WithField("agent", agent.Name).Error("Failed to restore agent")
			return fmt.Errorf("failed to restore agent: %w", err)
		}
		logger.Log.WithField("agent", agent.Name).Info("Agent restored successfully")
		return nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		// An unexpected error occurred.
		logger.Log.WithError(err).WithField("agent", agent.Name).Error("Failed to check for existing agent")
		return fmt.Errorf("failed to check for existing agent: %w", err)
	}

	// No conflict, create a new agent.
	logger.Log.WithField("agent", agent.Name).Info("Creating new agent")

	if err := database.DB.Create(agent).Error; err != nil {
		logger.Log.WithError(err).WithField("agent", agent.Name).Error("Failed to create agent")
		return fmt.Errorf("failed to save agent: %w", err)
	}

	logger.Log.WithField("agent", agent.Name).Info("Agent created successfully")
	return nil
}

// GetAgentByID retrieves an agent by its ID.
func GetAgentByID(id uint) (*models.Agent, error) {
	var agent models.Agent

	if err := database.DB.Where("id = ?", id).First(&agent).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Log.WithField("id", id).Warn("Agent not found")
			return nil, fmt.Errorf("agent with ID '%d' not found", id)
		}
		logger.Log.WithError(err).WithField("id", id).Error("Failed to fetch agent")
		return nil, fmt.Errorf("failed to fetch agent with ID '%d': %w", id, err)
	}

	logger.Log.WithField("agent", agent.Name).Info("Agent fetched successfully")
	return &agent, nil
}

// GetAgentByName retrieves an agent by its name.
func GetAgentByName(name string) (*models.Agent, error) {
	var agent models.Agent
	if err := database.DB.Where("name = ?", name).First(&agent).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Log.WithField("name", name).Warn("Agent not found")
			return nil, fmt.Errorf("agent '%s' not found", name)
		}
		logger.Log.WithError(err).WithField("name", name).Error("Failed to fetch agent")
		return nil, fmt.Errorf("failed to fetch agent '%s': %w", name, err)
	}

	logger.Log.WithField("agent", agent.Name).Info("Agent fetched successfully")
	return &agent, nil
}

// GetAllAgents retrieves agents with optional filtering and pagination.
func GetAllAgents(filters map[string]any, rangeValues []int) ([]models.Agent, int, error) {
	var agents []models.Agent
	query := database.DB.Model(&models.Agent{})

	// Apply filters.
	if len(filters) > 0 {
		for key, value := range filters {
			query = query.Where(fmt.Sprintf("%s = ?", key), value)
		}
	}

	// Count total number of agents.
	var total int64
	if err := query.Count(&total).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to count agents")
		return nil, 0, err
	}

	// Apply pagination.
	if len(rangeValues) == 2 {
		offset := rangeValues[0]
		limit := rangeValues[1] - rangeValues[0] + 1
		query = query.Offset(offset).Limit(limit)
	}

	// Execute query. GORM now automatically handles the JSON conversions for the nested structs.
	if err := query.Find(&agents).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Log.Warn("No agents found")
			return nil, int(total), nil
		}
		logger.Log.WithError(err).Error("Failed to fetch agents")
		return nil, 0, err
	}

	logger.Log.WithField("count", len(agents)).Info("Agents retrieved successfully")
	return agents, int(total), nil
}

// UpdateAgent updates an existing Agent in the database by its ID.
func UpdateAgent(id uint, updated *models.Agent) error {
	var existing models.Agent

	logger.Log.WithField("agent_id", id).Info("Attempting to update agent in database")

	// Find the existing agent by ID.
	if err := database.DB.Where("id = ?", id).First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Log.WithField("agent_id", id).Warn("Agent not found for update")
			return fmt.Errorf("agent with ID '%d' not found", id)
		}
		logger.Log.WithError(err).WithField("agent_id", id).Error("Failed to retrieve agent for update")
		return fmt.Errorf("failed to retrieve agent: %w", err)
	}

	// Update the individual fields.
	existing.Name = updated.Name
	existing.SystemPrompt = updated.SystemPrompt
	existing.MaxTurns = updated.MaxTurns
	existing.LLMConfig = updated.LLMConfig
	existing.Memory = updated.Memory
	existing.Trigger = updated.Trigger
	existing.Functions = updated.Functions
	existing.Config = updated.Config

	if err := database.DB.Save(&existing).Error; err != nil {
		logger.Log.WithError(err).WithField("agent_id", id).Error("Failed to save updated agent")
		return fmt.Errorf("failed to save updated agent: %w", err)
	}

	logger.Log.WithField("agent_id", id).Info("Agent updated successfully")
	return nil
}

// DeleteAgent deletes an agent from the database by its ID.
func DeleteAgent(agentID uint) error {
	var agent models.Agent

	logger.Log.WithField("agent_id", agentID).Info("Attempting to delete agent")

	if err := database.DB.Where("id = ?", agentID).First(&agent).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Log.WithField("agent_id", agentID).Warn("Agent not found for deletion")
			return fmt.Errorf("agent with ID '%d' not found", agentID)
		}
		logger.Log.WithError(err).WithField("agent_id", agentID).Error("Failed to find agent for deletion")
		return fmt.Errorf("failed to find agent: %w", err)
	}

	if err := database.DB.Delete(&agent).Error; err != nil {
		logger.Log.WithError(err).WithField("agent_id", agentID).Error("Failed to delete agent")
		return fmt.Errorf("failed to delete agent with ID '%d': %w", agentID, err)
	}

	logger.Log.WithField("agent_id", agentID).Info("Agent deleted successfully")
	return nil
}

// GetConversationHistory retrieves the messages for an agent.
func GetConversationHistory(agentID uint) ([]llm.Message, error) {
	var dbMessages []models.AgentMessage

	// Retrieve messages, ordered by their turn sequence.
	err := database.DB.
		Where("agent_id = ?", agentID).
		Order("turn asc").
		Find(&dbMessages).Error

	if err != nil {
		logger.Log.WithError(err).WithField("agent_id", agentID).Error("Failed to retrieve conversation history")
		return nil, fmt.Errorf("could not get conversation history: %w", err)
	}

	// Convert database models to the llm.Message type used by the runner.
	history := make([]llm.Message, len(dbMessages))
	for i, msg := range dbMessages {
		history[i] = llm.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	return history, nil
}

// SaveConversationHistory replaces the old history with the new one for an agent.
func SaveConversationHistory(agentID uint, messages []llm.Message) error {
	// Use a transaction for atomicity.
	tx := database.DB.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}

	// 1. Delete all existing messages for this agent.
	if err := tx.Where("agent_id = ?", agentID).Delete(&models.AgentMessage{}).Error; err != nil {
		tx.Rollback() // Rollback on error
		logger.Log.WithError(err).WithField("agent_id", agentID).Error("Failed to delete old conversation history")
		return fmt.Errorf("failed to clear old history: %w", err)
	}

	// 2. Insert the new messages with their correct turn order.
	for i, msg := range messages {
		dbMessage := models.AgentMessage{
			AgentID: agentID,
			Role:    msg.Role,
			Content: msg.Content,
			Turn:    i + 1, // Store the turn order
		}
		if err := tx.Create(&dbMessage).Error; err != nil {
			tx.Rollback() // Rollback on error
			logger.Log.WithError(err).WithField("agent_id", agentID).Error("Failed to insert new conversation message")
			return fmt.Errorf("failed to save new history: %w", err)
		}
	}

	// 3. Commit the transaction.
	if err := tx.Commit().Error; err != nil {
		logger.Log.WithError(err).WithField("agent_id", agentID).Error("Failed to commit conversation history transaction")
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// FindAgentsByEventTrigger queries for agents subscribed to a specific collection event.
// TOFIX This version uses a query for JSONB that is compatible with PostgreSQL.
func FindAgentsByEventTrigger(collectionName string, eventType string) ([]models.Agent, error) {
	var allEventAgents []models.Agent
	var subscribedAgents []models.Agent

	// This query is specific and reliable for finding agents with a 'collection_event' trigger type
	// by directly querying the 'type' key within the JSONB 'trigger' column.
	err := database.DB.
		Where("trigger ->> 'type' = ?", "collection_event").
		Find(&allEventAgents).Error

	if err != nil {
		logger.Log.WithError(err).
			WithField("collection", collectionName).
			WithField("event", eventType).
			Error("Failed to query for event-triggered agents")
		return nil, fmt.Errorf("failed to find agents: %w", err)
	}

	for _, agent := range allEventAgents {
		// This check is a good practice, though the query should already ensure it.
		if agent.Trigger.Type != "collection_event" {
			continue
		}

		config := agent.Trigger.EventTrigger

		// Check if the agent is subscribed to this collection ("*" is a wildcard for all collections).
		if config.Collection == collectionName || config.Collection == "*" {
			// Check if the agent is subscribed to this specific event type.
			if slices.Contains(config.Events, eventType) {
				subscribedAgents = append(subscribedAgents, agent) // Found a match, no need to check other events for this agent.
			}
		}
	}

	logger.Log.
		WithField("count", len(subscribedAgents)).
		WithField("collection", collectionName).
		WithField("event", eventType).
		Info("Found subscribed agents for event")

	return subscribedAgents, nil
}
