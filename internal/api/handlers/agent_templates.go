package handlers

import (
	"errors"
	"net/http"

	"github.com/gohead-cms/gohead/internal/agent/templates"

	"github.com/gin-gonic/gin"
)

// ListAgentTemplates lists the ready-made agents and their parameters.
func ListAgentTemplates(c *gin.Context) {
	c.Set("response", templates.List())
	c.Set("status", http.StatusOK)
}

// RenderAgentTemplate builds an agent definition from a template. Body:
// {"params": {…}}. The definition is not saved: review it, then create it
// with POST /admin/agents.
func RenderAgentTemplate(c *gin.Context) {
	var body struct {
		Params map[string]any `json:"params"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		validationError(c, "body must be {\"params\": {...}}")
		return
	}
	def, err := templates.Render(c.Param("id"), body.Params)
	if err != nil {
		if errors.Is(err, templates.ErrUnknown) {
			c.Set("response", "Template not found")
			c.Set("status", http.StatusNotFound)
			return
		}
		validationError(c, err.Error())
		return
	}
	c.Set("response", def)
	c.Set("status", http.StatusOK)
}
