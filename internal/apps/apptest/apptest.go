// Package apptest runs a small MCP server for tests of connected apps.
package apptest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Token is the bearer token the server accepts.
const Token = "test-token"

// Call is a call the server received.
type Call struct {
	Tool      string
	Arguments map[string]any
}

// Server is a test MCP server with two tools: "search" (read-only) and
// "create_issue" (it changes something).
type Server struct {
	*httptest.Server
	mu    sync.Mutex
	calls []Call
}

// Calls returns the calls received so far.
func (s *Server) Calls() []Call {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Call(nil), s.calls...)
}

func (s *Server) record(name string, req *mcp.CallToolRequest) map[string]any {
	var args map[string]any
	_ = json.Unmarshal(req.Params.Arguments, &args)
	s.mu.Lock()
	s.calls = append(s.calls, Call{Tool: name, Arguments: args})
	s.mu.Unlock()
	return args
}

func text(s string) *mcp.CallToolResult {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: s}}}
}

// New starts the server; close it with Close.
func New() *Server {
	s := &Server{}
	server := mcp.NewServer(&mcp.Implementation{Name: "apptest"}, nil)
	schema := map[string]any{"type": "object", "properties": map[string]any{
		"query": map[string]any{"type": "string", "description": "What to search"},
	}, "required": []string{"query"}}
	server.AddTool(&mcp.Tool{
		Name: "search", Title: "Search issues", Description: "Search the issues",
		InputSchema: schema, Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := s.record("search", req)
		b, _ := json.Marshal(map[string]any{"query": args["query"], "issues": []any{map[string]any{"id": 1, "title": "First"}}})
		return text(string(b)), nil
	})
	server.AddTool(&mcp.Tool{
		Name: "create_issue", Description: "Create an issue",
		InputSchema: map[string]any{"type": "object", "properties": map[string]any{
			"title": map[string]any{"type": "string"},
		}, "required": []string{"title"}},
	}, func(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := s.record("create_issue", req)
		if args["title"] == "fail" {
			r := text("title refused")
			r.IsError = true
			return r, nil
		}
		return text("created issue #7"), nil
	})
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, nil)
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+Token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		handler.ServeHTTP(w, r)
	}))
	return s
}
