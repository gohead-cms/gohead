// internal/api/handlers/component.go
package handlers

import (
	"net/http"

	"gohead/internal/models"
	"gohead/pkg/logger"
	"gohead/pkg/storage"

	"github.com/gin-gonic/gin"
)

// CreateComponentHandler creates a new component definition.
func CreateComponent(c *gin.Context) {
	var input map[string]interface{}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Set("response", "Invalid JSON input")
		c.Set("status", http.StatusBadRequest)
		return
	}

	// Transform input into a Component struct
	cmp, err := models.ParseComponentInput(input)
	if err != nil {
		c.Set("response", err.Error())
		c.Set("status", http.StatusBadRequest)
		return
	}

	// Create in storage
	if err := storage.CreateComponent(&cmp); err != nil {
		logger.Log.WithError(err).Error("Failed to create component")
		c.Set("response", err.Error())
		c.Set("status", http.StatusBadRequest)
		return
	}

	c.Set("response", gin.H{"message": "component created successfully", "component": cmp.Name})
	c.Set("status", http.StatusCreated)
}

// GetComponentHandler retrieves a component by name.
func GetComponent(c *gin.Context) {
	name := c.Param("name")
	cmp, err := storage.GetComponentByName(name)
	if err != nil {
		c.Set("response", err.Error())
		c.Set("status", http.StatusNotFound)
		return
	}

	c.Set("response", cmp)
	c.Set("status", http.StatusOK)
}

// UpdateComponentHandler updates an existing component definition.
func UpdateComponent(c *gin.Context) {
	name := c.Param("name")

	var input map[string]interface{}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Set("response", "Invalid JSON input")
		c.Set("status", http.StatusBadRequest)
		return
	}

	updatedCmp, err := models.ParseComponentInput(input)
	if err != nil {
		c.Set("response", err.Error())
		c.Set("status", http.StatusBadRequest)
		return
	}

	if err := storage.UpdateComponent(name, &updatedCmp); err != nil {
		c.Set("response", err.Error())
		c.Set("status", http.StatusBadRequest)
		return
	}

	c.Set("response", "component updated successfully")
	c.Set("status", http.StatusOK)
}

// DeleteComponentHandler deletes a component by name.
func DeleteComponent(c *gin.Context) {
	name := c.Param("name")
	if err := storage.DeleteComponent(name); err != nil {
		c.Set("response", err.Error())
		c.Set("status", http.StatusBadRequest)
		return
	}

	c.Set("response", "component deleted successfully")
	c.Set("status", http.StatusOK)
}
