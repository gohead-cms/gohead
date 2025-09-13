package functions

import (
	"context"

	agentModels "github.com/gohead-cms/gohead/internal/models/agents"
	"github.com/tmc/langchaingo/tools"
)

// ToolFunc is a function that executes a tool.
type ToolFunc func(ctx context.Context, args any) (string, error)

// Registry maps a tool's name to its executable function.
type Registry struct {
	tools map[string]ToolFunc
	specs []tools.Tool
}

// NewRegistry creates a new tool registry from a list of agent function specs.
func NewRegistry(agentFuncs agentModels.FunctionSpecs) *Registry {
	r := &Registry{
		tools: make(map[string]ToolFunc),
		specs: make([]tools.Tool, 0),
	}

	return r
}

// Get returns the executable function for a given tool name.
func (r *Registry) Get(name string) (ToolFunc, bool) {
	fn, ok := r.tools[name]
	return fn, ok
}

// Specs returns the list of tool specifications to provide to the LLM.
func (r *Registry) Specs() []tools.Tool {
	return r.specs
}
