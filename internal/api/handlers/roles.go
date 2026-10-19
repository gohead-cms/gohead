package handlers

import (
	"errors"
	"net/http"

	"github.com/gohead-cms/gohead/pkg/auth"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/storage"

	"github.com/gin-gonic/gin"
)

// ListRoles lists the roles with their permissions (admin).
func ListRoles(c *gin.Context) {
	roles, err := storage.ListRoleViews()
	if err != nil {
		c.Set("response", "Failed to list roles")
		c.Set("status", http.StatusInternalServerError)
		return
	}
	c.Set("response", roles)
	c.Set("status", http.StatusOK)
}

// GetRole returns one role (admin).
func GetRole(c *gin.Context) {
	role, err := storage.GetRoleView(c.Param("name"))
	if err != nil {
		roleError(c, err)
		return
	}
	c.Set("response", role)
	c.Set("status", http.StatusOK)
}

type roleBody struct {
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Permissions auth.Permissions `json:"permissions"`
}

// CreateRole adds a custom role (admin).
func CreateRole(c *gin.Context) {
	var body roleBody
	if err := c.ShouldBindJSON(&body); err != nil {
		validationError(c, "expected {\"name\": …, \"description\": …, \"permissions\": {\"content\": {…}, \"media\": […]}}")
		return
	}
	role, err := storage.CreateCustomRole(body.Name, body.Description, body.Permissions)
	if err != nil {
		roleError(c, err)
		return
	}
	c.Set("response", role)
	c.Set("status", http.StatusCreated)
}

// UpdateRole replaces a custom role's description and permissions (admin).
func UpdateRole(c *gin.Context) {
	var body roleBody
	if err := c.ShouldBindJSON(&body); err != nil {
		validationError(c, "expected {\"description\": …, \"permissions\": {…}}")
		return
	}
	if body.Name != "" && body.Name != c.Param("name") {
		validationError(c, "roles cannot be renamed: accounts and API tokens refer to them by name")
		return
	}
	role, err := storage.UpdateCustomRole(c.Param("name"), body.Description, body.Permissions)
	if err != nil {
		roleError(c, err)
		return
	}
	c.Set("response", role)
	c.Set("status", http.StatusOK)
}

// DeleteRole removes a custom role no account or token uses (admin).
func DeleteRole(c *gin.Context) {
	if err := storage.DeleteCustomRole(c.Param("name")); err != nil {
		roleError(c, err)
		return
	}
	c.Set("response", nil)
	c.Set("status", http.StatusOK)
}

func roleError(c *gin.Context, err error) {
	var verr *storage.ItemValidationError
	var inUse *storage.RoleInUseError
	switch {
	case errors.As(err, &verr):
		validationError(c, verr.Error())
	case errors.Is(err, storage.ErrRoleNotFound):
		c.Set("response", "Role not found")
		c.Set("status", http.StatusNotFound)
	case errors.Is(err, storage.ErrRoleExists):
		c.Set("response", err.Error())
		c.Set("status", http.StatusConflict)
	case errors.Is(err, storage.ErrBuiltinRole):
		c.Set("response", err.Error())
		c.Set("status", http.StatusConflict)
	case errors.As(err, &inUse):
		c.Set("response", inUse.Error())
		c.Set("details", gin.H{"users": inUse.Users, "tokens": inUse.Tokens})
		c.Set("status", http.StatusConflict)
	default:
		logger.Log.WithError(err).Error("Role operation failed")
		c.Set("response", "Failed to change roles")
		c.Set("status", http.StatusInternalServerError)
	}
}
