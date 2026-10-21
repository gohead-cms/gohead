// Package apps connects GoHead to other services: remote MCP servers
// (Model Context Protocol, Streamable HTTP transport) whose tools agents can
// use. A connection stores the server's URL and token; its tools are listed
// once (on save or refresh) and called on demand, each call over a short
// session of its own.
package apps

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/internal/webhooks"
	"github.com/gohead-cms/gohead/pkg/secrets"
	"github.com/gohead-cms/gohead/pkg/storage"
)

// Config controls connections to other services.
type Config struct {
	// AllowPrivateNetworks lets connections reach private and loopback
	// addresses (an MCP server on your own network). Link-local addresses
	// (cloud metadata endpoints) are always refused.
	AllowPrivateNetworks bool
	// Timeout of one call (connecting, listing or calling a tool).
	Timeout time.Duration
}

var cfg = Config{Timeout: 30 * time.Second}

// Configure sets the connection settings (call at startup).
func Configure(c Config) {
	if c.Timeout <= 0 {
		c.Timeout = 30 * time.Second
	}
	cfg = c
}

// maxResult caps what a tool call returns to the model.
const maxResult = 64 << 10

// ErrBlockedAddress is returned when a connection's URL resolves to an
// address it may not reach.
var ErrBlockedAddress = errors.New("address not allowed for connections (see connections.allow_private_networks)")

// Endpoint is where and how to reach a server.
type Endpoint struct {
	URL        string
	AuthType   string
	HeaderName string
	Token      string
}

// EndpointOf returns a connection's endpoint, with its token decrypted.
func EndpointOf(c models.Connection) (Endpoint, error) {
	token, err := secrets.Open(c.Secret)
	if err != nil {
		return Endpoint{}, err
	}
	return Endpoint{URL: c.URL, AuthType: c.AuthType, HeaderName: c.HeaderName, Token: token}, nil
}

// authTransport adds the token to each request.
type authTransport struct {
	base http.RoundTripper
	ep   Endpoint
}

func (t authTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if t.ep.Token != "" {
		r = r.Clone(r.Context())
		switch t.ep.AuthType {
		case models.ConnectionAuthBearer:
			r.Header.Set("Authorization", "Bearer "+t.ep.Token)
		case models.ConnectionAuthHeader:
			r.Header.Set(t.ep.HeaderName, t.ep.Token)
		}
	}
	return t.base.RoundTrip(r)
}

// httpClient checks every address it connects to, once resolved, so DNS
// tricks cannot reach forbidden ones, and follows no redirect (it would
// carry the token to another host).
func httpClient(ep Endpoint) *http.Client {
	allowPrivate := cfg.AllowPrivateNetworks
	dialer := &net.Dialer{
		Timeout: 10 * time.Second,
		Control: func(_, address string, _ syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return err
			}
			ip := net.ParseIP(host)
			if ip == nil || !webhooks.AddressAllowed(ip, allowPrivate) {
				return ErrBlockedAddress
			}
			return nil
		},
	}
	transport := &http.Transport{
		Proxy:               nil, // a proxy would bypass the address check
		DialContext:         dialer.DialContext,
		TLSHandshakeTimeout: 10 * time.Second,
		MaxIdleConns:        4,
		IdleConnTimeout:     30 * time.Second,
	}
	return &http.Client{
		Transport: authTransport{base: transport, ep: ep},
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// connect opens a session with the server.
func connect(ctx context.Context, ep Endpoint) (*mcp.ClientSession, func(), error) {
	hc := httpClient(ep)
	client := mcp.NewClient(&mcp.Implementation{Name: "gohead", Title: "GoHead"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint: ep.URL, HTTPClient: hc, DisableStandaloneSSE: true,
	}, nil)
	if err != nil {
		hc.CloseIdleConnections()
		return nil, nil, friendlyError(err)
	}
	return session, func() {
		_ = session.Close()
		hc.CloseIdleConnections()
	}, nil
}

// friendlyError shortens the errors people see when a connection fails.
func friendlyError(err error) error {
	msg := err.Error()
	if errors.Is(err, ErrBlockedAddress) || strings.Contains(msg, ErrBlockedAddress.Error()) {
		return ErrBlockedAddress
	}
	switch {
	case strings.Contains(msg, "401"), strings.Contains(strings.ToLower(msg), "unauthorized"):
		return fmt.Errorf("the server refused the token (401 Unauthorized): %w", err)
	case strings.Contains(msg, "403"), strings.Contains(strings.ToLower(msg), "forbidden"):
		return fmt.Errorf("the server refused access (403 Forbidden): %w", err)
	case strings.Contains(msg, "404"):
		return fmt.Errorf("no MCP server at this URL (404): %w", err)
	}
	return err
}

// maxTools caps how many tools of a server are kept.
const maxTools = 500

// ListTools returns the tools the server offers.
func ListTools(ctx context.Context, ep Endpoint) (models.ConnectionTools, error) {
	ctx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()
	session, closeFn, err := connect(ctx, ep)
	if err != nil {
		return nil, err
	}
	defer closeFn()
	var out models.ConnectionTools
	for t, err := range session.Tools(ctx, nil) {
		if err != nil {
			return nil, friendlyError(err)
		}
		if t == nil || t.Name == "" {
			continue
		}
		out = append(out, toolOf(t))
		if len(out) >= maxTools {
			break
		}
	}
	return out, nil
}

func toolOf(t *mcp.Tool) models.ConnectionTool {
	ct := models.ConnectionTool{Name: t.Name, Title: t.Title, Description: t.Description}
	if t.Annotations != nil {
		ct.ReadOnly = t.Annotations.ReadOnlyHint
		ct.Destructive = t.Annotations.DestructiveHint == nil || *t.Annotations.DestructiveHint
		if ct.ReadOnly {
			ct.Destructive = false
		}
		if ct.Title == "" {
			ct.Title = t.Annotations.Title
		}
	} else {
		ct.Destructive = true // the MCP default for tools that say nothing
	}
	if t.InputSchema != nil {
		if b, err := json.Marshal(t.InputSchema); err == nil {
			var schema models.JSONMap
			if json.Unmarshal(b, &schema) == nil {
				ct.InputSchema = schema
			}
		}
	}
	if ct.InputSchema == nil {
		ct.InputSchema = models.JSONMap{"type": "object", "properties": map[string]any{}}
	}
	return ct
}

// Result is what a tool call returned.
type Result struct {
	IsError    bool
	Text       string
	Structured any
}

// JSON is the result as the model gets it.
func (r Result) JSON() string {
	out := map[string]any{"status": "success"}
	if r.IsError {
		out["status"] = "error"
	}
	if r.Structured != nil {
		out["result"] = r.Structured
	} else {
		out["result"] = r.Text
	}
	b, err := json.Marshal(out)
	if err != nil || len(b) > maxResult {
		text := r.Text
		if len(text) > maxResult {
			text = truncate(text, maxResult) + "… (truncated)"
		}
		b, _ = json.Marshal(map[string]any{"status": out["status"], "result": text})
	}
	return string(b)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	s = s[:n]
	for len(s) > 0 && !utf8.ValidString(s) {
		s = s[:len(s)-1]
	}
	return s
}

// Call calls a tool of the server.
func Call(ctx context.Context, ep Endpoint, tool string, args map[string]any) (Result, error) {
	ctx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()
	session, closeFn, err := connect(ctx, ep)
	if err != nil {
		return Result{}, err
	}
	defer closeFn()
	if args == nil {
		args = map[string]any{}
	}
	res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: tool, Arguments: args})
	if err != nil {
		return Result{}, friendlyError(err)
	}
	out := Result{IsError: res.IsError, Structured: res.StructuredContent}
	var texts []string
	for _, c := range res.Content {
		switch v := c.(type) {
		case *mcp.TextContent:
			texts = append(texts, v.Text)
		case *mcp.ImageContent:
			texts = append(texts, "[image "+v.MIMEType+"]")
		case *mcp.AudioContent:
			texts = append(texts, "[audio "+v.MIMEType+"]")
		case *mcp.ResourceLink:
			texts = append(texts, "[resource "+v.URI+"]")
		case *mcp.EmbeddedResource:
			if v.Resource != nil {
				if v.Resource.Text != "" {
					texts = append(texts, v.Resource.Text)
				} else {
					texts = append(texts, "[resource "+v.Resource.URI+"]")
				}
			}
		}
	}
	out.Text = strings.Join(texts, "\n")
	if out.Structured == nil && len(texts) == 1 {
		// Many servers return JSON as text: give it to the model as JSON.
		var v any
		if json.Unmarshal([]byte(texts[0]), &v) == nil {
			if _, isObj := v.(map[string]any); isObj {
				out.Structured = v
			} else if _, isList := v.([]any); isList {
				out.Structured = v
			}
		}
	}
	return out, nil
}

// Refresh lists a connection's tools again and stores them, or the error.
func Refresh(ctx context.Context, c *models.Connection) error {
	ep, err := EndpointOf(*c)
	if err == nil {
		var list models.ConnectionTools
		if list, err = ListTools(ctx, ep); err == nil {
			now := models.APITime(time.Now())
			c.Tools, c.ToolsUpdatedAt, c.LastError = list, &now, ""
		}
	}
	if err != nil {
		c.LastError = err.Error()
	}
	if serr := storage.SaveConnection(c); serr != nil {
		return serr
	}
	return err
}

// CallTool calls a tool of a stored connection.
func CallTool(ctx context.Context, connection, tool string, args map[string]any) (Result, error) {
	c, err := storage.GetConnection(connection)
	if err != nil {
		return Result{}, err
	}
	ep, err := EndpointOf(*c)
	if err != nil {
		return Result{}, err
	}
	return Call(ctx, ep, tool, args)
}
