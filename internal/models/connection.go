package models

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// Connection auth types.
const (
	ConnectionAuthNone   = "none"
	ConnectionAuthBearer = "bearer" // Authorization: Bearer <secret>
	ConnectionAuthHeader = "header" // <HeaderName>: <secret>
)

// Connection is another service agents can use: a remote MCP server (GitHub,
// Zapier, your own…) whose tools become agent tools named
// "<connection>.<tool>".
type Connection struct {
	ID    uint   `json:"id" gorm:"primaryKey"`
	Name  string `json:"name" gorm:"size:64;uniqueIndex;not null"`
	Label string `json:"label" gorm:"size:191"`
	// Preset is the catalog entry it was made from ("github", "zapier"),
	// empty for a custom server.
	Preset string `json:"preset,omitempty" gorm:"size:64"`
	// URL of the MCP server (Streamable HTTP transport).
	URL        string `json:"url" gorm:"type:text;not null"`
	AuthType   string `json:"authType" gorm:"size:16;not null"`
	HeaderName string `json:"headerName,omitempty" gorm:"size:191"`
	// Secret is the token, encrypted (see pkg/secrets). Never returned.
	Secret string `json:"-" gorm:"type:text"`
	// Tools is the server's tool list, as of ToolsUpdatedAt.
	Tools          ConnectionTools `json:"tools" gorm:"type:json;serializer:json"`
	ToolsUpdatedAt *time.Time      `json:"toolsUpdatedAt,omitempty"`
	// LastError is the error of the last refresh, empty when it worked.
	LastError string    `json:"lastError,omitempty" gorm:"type:text"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// ConnectionTools is a connection's tool list.
type ConnectionTools []ConnectionTool

// ConnectionTool is a tool a connected server offers.
type ConnectionTool struct {
	Name        string  `json:"name"`
	Title       string  `json:"title,omitempty"`
	Description string  `json:"description,omitempty"`
	InputSchema JSONMap `json:"inputSchema,omitempty"`
	// ReadOnly: the server says the tool changes nothing. Other tools
	// "write": they need approval unless the agent says otherwise.
	ReadOnly    bool `json:"readOnly"`
	Destructive bool `json:"destructive,omitempty"`
}

// Tool returns the connection's tool of that name.
func (c Connection) Tool(name string) (ConnectionTool, bool) {
	for _, t := range c.Tools {
		if t.Name == name {
			return t, true
		}
	}
	return ConnectionTool{}, false
}

// HasSecret reports whether a token is stored.
func (c Connection) HasSecret() bool { return c.Secret != "" }

var connectionName = regexp.MustCompile(`^[a-z][a-z0-9-]{1,30}$`)

// reservedConnectionNames are the prefixes of built-in tools.
var reservedConnectionNames = map[string]bool{
	"content": true, "ai": true, "system": true, "collections": true, "llm": true, "app": true, "apps": true,
}

// ProposalCall is the action of a proposal to call a connected app's tool.
const ProposalCall = "call"

// ConnectionProposalPrefix starts the Collection of call proposals
// ("app:github"): no collection name has a colon.
const ConnectionProposalPrefix = "app:"

// Validate checks the connection's settings (not its secret).
func (c *Connection) Validate() error {
	c.Name = strings.TrimSpace(c.Name)
	c.URL = strings.TrimSpace(c.URL)
	c.Label = strings.TrimSpace(c.Label)
	c.HeaderName = strings.TrimSpace(c.HeaderName)
	if !connectionName.MatchString(c.Name) {
		return fmt.Errorf("name must be 2 to 31 lowercase letters, digits or dashes, starting with a letter")
	}
	if reservedConnectionNames[c.Name] {
		return fmt.Errorf("'%s' is the name of built-in tools: choose another name", c.Name)
	}
	u, err := url.Parse(c.URL)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
		return fmt.Errorf("url must be an http(s) URL")
	}
	if u.User != nil {
		return fmt.Errorf("url must not contain credentials: use the token instead")
	}
	switch c.AuthType {
	case "":
		c.AuthType = ConnectionAuthNone
	case ConnectionAuthNone, ConnectionAuthBearer:
		c.HeaderName = ""
	case ConnectionAuthHeader:
		if !headerName.MatchString(c.HeaderName) {
			return fmt.Errorf("headerName must be an HTTP header name")
		}
		switch strings.ToLower(c.HeaderName) {
		case "host", "content-type", "content-length", "accept", "mcp-session-id", "mcp-protocol-version", "connection", "transfer-encoding":
			return fmt.Errorf("header '%s' cannot be used for the token", c.HeaderName)
		}
	default:
		return fmt.Errorf("authType must be none, bearer or header")
	}
	if c.Label == "" {
		c.Label = c.Name
	}
	return nil
}

var headerName = regexp.MustCompile(`^[A-Za-z0-9-]{1,100}$`)
