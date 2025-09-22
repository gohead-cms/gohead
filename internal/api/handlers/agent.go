package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	agents "github.com/gohead-cms/gohead/internal/models/agents"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/storage"
	"github.com/gohead-cms/gohead/pkg/utils"
)

// GetAgents retrieves a list of agents with optional filtering and pagination.
func GetAgents(c *gin.Context) {
	logger.Log.Debug("handler.get_agents")

	// Pagination management
	filterParam := c.Query("filter")
	rangeParam := c.Query("range")
	pageParam := c.DefaultQuery("page", "1")
	pageSizeParam := c.DefaultQuery("pageSize", "10")
	page, _ := strconv.Atoi(pageParam)
	pageSize, _ := strconv.Atoi(pageSizeParam)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	var filters map[string]any
	var rangeValues []int

	// Parse filter (JSON object)
	if filterParam != "" {
		if err := json.Unmarshal([]byte(filterParam), &filters); err != nil {
			logger.Log.WithError(err).Warn("Invalid filter format")
			c.Set("response", "Invalid filter format")
			c.Set("status", http.StatusBadRequest)
			return
		}
	}

	// Parse range (JSON array [start, end])
	if rangeParam != "" {
		if err := json.Unmarshal([]byte(rangeParam), &rangeValues); err != nil || len(rangeValues) != 2 {
			logger.Log.WithError(err).Warn("Invalid range format")
			c.Set("response", "Invalid range format")
			c.Set("status", http.StatusBadRequest)
			return
		}
	}

	// Retrieve agents from storage with optional filters and pagination.
	// The call is updated to match the new storage function signature.
	agents, total, err := storage.GetAllAgents(filters, rangeValues)
	if err != nil {
		logger.Log.WithError(err).Warn("GetAgents: failed to retrieve agents")
		c.Set("response", "Failed to fetch agents")
		c.Set("status", http.StatusInternalServerError)
		return
	}

	pageCount := (total + pageSize - 1) / pageSize

	// Format response
	c.Header("Content-Range", formatContentRange(len(agents), total))
	c.Set("response", utils.FormatAgentsSchema(agents))
	c.Set("status", http.StatusOK)
	c.Set("meta", gin.H{
		"pagination": gin.H{
			"page":      page,
			"pageSize":  pageSize,
			"pageCount": pageCount,
			"total":     total,
		},
	})
}

func GetAgent(c *gin.Context) {
	// The `name` parameter is now optional.
	name := c.Param("name")

	if name == "" {
		agents, _, err := storage.GetAllAgents(nil, nil)
		if err != nil {
			c.Set("response", "Failed to retrieve agents")
			c.Set("status", http.StatusInternalServerError)
			return
		}

		c.Set("response", agents)
		c.Set("status", http.StatusOK)
		return
	}

	agent, err := storage.GetAgentByName(name)
	if err != nil {
		c.Set("response", "Agent not found")
		c.Set("status", http.StatusNotFound)
		return
	}

	c.Set("response", utils.FormatAgentSchema(agent))
	c.Set("status", http.StatusOK)
}

// CreateAgent handles creating a new agent.
func CreateAgent(c *gin.Context) {
	var input map[string]any
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Set("response", "Invalid JSON input")
		c.Set("status", http.StatusBadRequest)
		return
	}

	logger.Log.WithField("input", input).Info("CreateAgent")

	agent, err := agents.ParseAgentInput(input)
	if err != nil {
		c.Set("response", err.Error())
		c.Set("status", http.StatusBadRequest)
		return
	}

	if err := agents.ValidateAgentSchema(agent); err != nil {
		logger.Log.WithError(err).Warn("CreateAgent: Validation failed")
		c.Set("response", err.Error())
		c.Set("status", http.StatusBadRequest)
		return
	}

	// Check if agent with same name already exists
	existing, err := storage.GetAgentByName(agent.Name)
	if err == nil && existing != nil {
		c.Set("response", "This agent already exists")
		c.Set("status", http.StatusBadRequest)
		return
	}

	if err := storage.SaveAgent(&agent); err != nil {
		logger.Log.WithError(err).Error("CreateAgent: Failed to save agent")
		c.Set("response", "Failed to save agent")
		c.Set("status", http.StatusInternalServerError)
		return
	}

	logger.Log.WithField("agent", agent.Name).Info("Agent created successfully")
	c.Set("response", utils.FormatAgentSchema(&agent))
	c.Set("meta", gin.H{"message": "Agent created successfully"})
	c.Set("status", http.StatusCreated)
}

// UpdateAgent handles updating an existing agent.
func UpdateAgent(c *gin.Context) {
	name := c.Param("name")

	var input map[string]any
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Set("response", "Invalid JSON input")
		c.Set("status", http.StatusBadRequest)
		return
	}

	agent, err := agents.ParseAgentInput(input)
	if err != nil {
		c.Set("response", err.Error())
		c.Set("status", http.StatusBadRequest)
		return
	}

	if err := agents.ValidateAgentSchema(agent); err != nil {
		logger.Log.WithError(err).Warn("UpdateAgent: Validation failed")
		c.Set("response", err.Error())
		c.Set("status", http.StatusBadRequest)
		return
	}

	// Fetch the existing agent by name
	existing, err := storage.GetAgentByName(name)
	if err != nil {
		logger.Log.WithError(err).Warn("UpdateAgent: Agent not found")
		c.Set("response", "Agent not found")
		c.Set("status", http.StatusNotFound)
		return
	}

	// Update in DB
	if err := storage.UpdateAgent(existing.ID, &agent); err != nil {
		logger.Log.WithError(err).Error("UpdateAgent: Failed to update agent")
		c.Set("response", "Failed to update agent")
		c.Set("status", http.StatusInternalServerError)
		return
	}

	// Fetch updated agent
	updated, err := storage.GetAgentByID(existing.ID)
	if err != nil {
		logger.Log.WithError(err).Error("UpdateAgent: Failed to fetch updated agent")
		c.Set("response", "Failed to fetch updated agent")
		c.Set("status", http.StatusInternalServerError)
		return
	}

	c.Set("response",
		utils.FormatAgentSchema(updated),
	)
	c.Set("status", http.StatusOK)
}

// DeleteAgent handles deleting an agent by its name.
func DeleteAgent(c *gin.Context) {
	name := c.Param("name")

	logger.Log.WithField("agent_name", name).Debug("Handler:DeleteAgent")

	// Fetch the agent by name
	agent, err := storage.GetAgentByName(name)
	if err != nil {
		logger.Log.WithError(err).WithField("agent_name", name).Warn("DeleteAgent: Agent not found")
		c.Set("response", "Agent not found")
		c.Set("status", http.StatusNotFound)
		return
	}

	// Now delete using the agent's ID
	if err := storage.DeleteAgent(agent.ID); err != nil {
		logger.Log.WithError(err).WithField("agent_name", name).Error("DeleteAgent: Failed to delete agent")
		c.Set("response", err.Error())
		c.Set("status", http.StatusBadRequest)
		return
	}

	logger.Log.WithField("agent_name", name).Info("Agent deleted successfully")
	c.Set("response", nil)
	c.Set("meta", gin.H{
		"message": "Agent deleted successfully",
	})
	c.Set("status", http.StatusOK)
}
