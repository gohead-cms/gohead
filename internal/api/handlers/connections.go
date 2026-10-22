package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/gohead-cms/gohead/internal/agent/functions"
	"github.com/gohead-cms/gohead/internal/apps"
	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/secrets"
	"github.com/gohead-cms/gohead/pkg/storage"

	"github.com/gin-gonic/gin"
)

// Connections are other services (remote MCP servers) whose tools agents
// can use. Their tokens are write-only.

// connectionInput is the body of connection create and update requests.
// Fields left out of an update keep their value; "token": "" removes the
// token.
type connectionInput struct {
	Name       *string `json:"name"`
	Label      *string `json:"label"`
	Preset     *string `json:"preset"`
	URL        *string `json:"url"`
	AuthType   *string `json:"authType"`
	HeaderName *string `json:"headerName"`
	Token      *string `json:"token"`
}

func (in connectionInput) apply(c *models.Connection, creating bool) error {
	if creating {
		if in.Name != nil {
			c.Name = strings.TrimSpace(*in.Name)
		}
		if in.Preset != nil && *in.Preset != "" {
			p, ok := apps.PresetByID(*in.Preset)
			if !ok {
				return fmt.Errorf("unknown preset '%s'", *in.Preset)
			}
			c.Preset, c.URL, c.AuthType, c.HeaderName, c.Label = p.ID, p.URL, p.AuthType, p.HeaderName, p.Label
			if c.Name == "" {
				c.Name = p.ID
			}
		}
	} else if in.Name != nil && strings.TrimSpace(*in.Name) != c.Name {
		return errors.New("a connection cannot be renamed: agents use its name")
	}
	before := *c
	if in.Label != nil {
		c.Label = *in.Label
	}
	// A URL sent back as the API shows it (query values hidden) is kept.
	if in.URL != nil && (creating || *in.URL != shownURL(c.URL)) {
		c.URL = *in.URL
	}
	if in.AuthType != nil {
		c.AuthType = *in.AuthType
	}
	if in.HeaderName != nil {
		c.HeaderName = *in.HeaderName
	}
	if err := c.Validate(); err != nil {
		return err
	}
	if creating && functions.ConnectionNameTaken(c.Name) {
		return fmt.Errorf("'%s' is the name of built-in tools: choose another name", c.Name)
	}
	// A stored token is never sent anywhere else than where it was given
	// for: changing where it goes needs it again.
	if !creating && in.Token == nil && c.Secret != "" &&
		(c.URL != before.URL || c.AuthType != before.AuthType || c.HeaderName != before.HeaderName) &&
		c.AuthType != models.ConnectionAuthNone {
		return errors.New("enter the token again to change the URL or how it is sent")
	}
	if in.Token != nil {
		sealed, err := secrets.Seal(strings.TrimSpace(*in.Token))
		if err != nil {
			return err
		}
		c.Secret = sealed
	}
	if c.AuthType == models.ConnectionAuthNone {
		c.Secret = ""
	}
	return nil
}

func connectionJSON(c models.Connection) gin.H {
	tools := c.Tools
	if tools == nil {
		tools = models.ConnectionTools{}
	}
	return gin.H{
		"id": c.ID, "name": c.Name, "label": c.Label, "preset": c.Preset, "url": shownURL(c.URL),
		"authType": c.AuthType, "headerName": c.HeaderName, "hasToken": c.HasSecret(),
		"tools": tools, "toolsUpdatedAt": c.ToolsUpdatedAt, "lastError": c.LastError,
		"createdAt": c.CreatedAt, "updatedAt": c.UpdatedAt,
	}
}

// ListConnectionPresets lists the services the admin UI offers to connect.
func ListConnectionPresets(c *gin.Context) {
	c.Set("response", apps.Presets)
	c.Set("meta", gin.H{"encrypted": secrets.Enabled()})
	c.Set("status", http.StatusOK)
}

// ListConnections lists the connections.
func ListConnections(c *gin.Context) {
	list, err := storage.ListConnections()
	if err != nil {
		c.Set("response", "Failed to list connections")
		c.Set("status", http.StatusInternalServerError)
		return
	}
	out := make([]gin.H, 0, len(list))
	for _, conn := range list {
		out = append(out, demoURL(c, connectionJSON(conn)))
	}
	c.Set("response", out)
	c.Set("meta", gin.H{"total": len(out), "encrypted": secrets.Enabled()})
	c.Set("status", http.StatusOK)
}

func loadConnection(c *gin.Context) (*models.Connection, bool) {
	conn, err := storage.GetConnection(c.Param("name"))
	if err != nil {
		if errors.Is(err, storage.ErrConnectionNotFound) {
			c.Set("response", "Connection not found")
			c.Set("status", http.StatusNotFound)
		} else {
			c.Set("response", "Failed to load the connection")
			c.Set("status", http.StatusInternalServerError)
		}
		return nil, false
	}
	return conn, true
}

// GetConnection returns a connection with its tools.
func GetConnection(c *gin.Context) {
	conn, ok := loadConnection(c)
	if !ok {
		return
	}
	c.Set("response", demoURL(c, connectionJSON(*conn)))
	c.Set("status", http.StatusOK)
}

// refreshed lists a connection's tools and returns it, with the refresh
// error in meta: a connection is saved even when its server can't be
// reached yet.
func refreshed(c *gin.Context, conn *models.Connection, status int) {
	err := apps.Refresh(c.Request.Context(), conn)
	c.Set("response", connectionJSON(*conn))
	if err != nil {
		logger.Log.WithError(err).WithField("connection", conn.Name).Warn("Could not list a connection's tools")
		c.Set("meta", gin.H{"warning": "Could not list the tools: " + err.Error()})
	}
	c.Set("status", status)
}

// CreateConnection adds a connection and lists its tools.
func CreateConnection(c *gin.Context) {
	var in connectionInput
	if err := c.ShouldBindJSON(&in); err != nil {
		validationError(c, "Invalid JSON body")
		return
	}
	var conn models.Connection
	if err := in.apply(&conn, true); err != nil {
		validationError(c, err.Error())
		return
	}
	if err := storage.CreateConnection(&conn); err != nil {
		if errors.Is(err, storage.ErrConnectionExists) {
			c.Set("response", err.Error())
			c.Set("status", http.StatusConflict)
			return
		}
		c.Set("response", "Failed to save the connection")
		c.Set("status", http.StatusInternalServerError)
		return
	}
	refreshed(c, &conn, http.StatusCreated)
}

// UpdateConnection changes a connection and lists its tools again.
func UpdateConnection(c *gin.Context) {
	conn, ok := loadConnection(c)
	if !ok {
		return
	}
	var in connectionInput
	if err := c.ShouldBindJSON(&in); err != nil {
		validationError(c, "Invalid JSON body")
		return
	}
	if err := in.apply(conn, false); err != nil {
		validationError(c, err.Error())
		return
	}
	if err := storage.SaveConnection(conn); err != nil {
		c.Set("response", "Failed to save the connection")
		c.Set("status", http.StatusInternalServerError)
		return
	}
	refreshed(c, conn, http.StatusOK)
}

// RefreshConnection lists a connection's tools again.
func RefreshConnection(c *gin.Context) {
	conn, ok := loadConnection(c)
	if !ok {
		return
	}
	refreshed(c, conn, http.StatusOK)
}

// DeleteConnection removes a connection. Agents that use its tools get an
// error when they call them.
func DeleteConnection(c *gin.Context) {
	if err := storage.DeleteConnection(c.Param("name")); err != nil {
		if errors.Is(err, storage.ErrConnectionNotFound) {
			c.Set("response", "Connection not found")
			c.Set("status", http.StatusNotFound)
			return
		}
		c.Set("response", "Failed to delete the connection")
		c.Set("status", http.StatusInternalServerError)
		return
	}
	c.Set("response", gin.H{"deleted": c.Param("name")})
	c.Set("status", http.StatusOK)
}

// shownURL hides the values of a URL's query string, where some servers
// take a secret.
func shownURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.RawQuery == "" {
		return raw
	}
	q := u.Query()
	for k := range q {
		q[k] = []string{"hidden"}
	}
	u.RawQuery = q.Encode()
	return u.String()
}
