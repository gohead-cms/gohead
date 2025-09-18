package functions

import "context"

// FunctionDefinition defines a function that can be called by the LLM
type FunctionDefinition struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  any    `json:"parameters"`
}

// ToolFunc is a function that executes a tool.
type ToolFunc func(ctx context.Context, args any) (string, error)

// ToolSpec represents a function specification for the LLM
type ToolSpec struct {
	Type     string              `json:"type"`
	Function *FunctionDefinition `json:"function"`
}

// Registry maps a tool's name to its executable function.
type Registry struct {
	tools map[string]ToolFunc
	specs []ToolSpec
}

// RegistryTool implements langchain's tools.Tool interface
type RegistryTool struct {
	name        string
	description string
	registry    *Registry
	toolName    string
}
