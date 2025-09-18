package functions

import (
	"context"
	"encoding/json"
	"fmt"

	agentModels "github.com/gohead-cms/gohead/internal/models/agents"
	"github.com/gohead-cms/gohead/pkg/logger"
)

// Name returns the tool name (implements tools.Tool interface)
func (t *RegistryTool) Name() string {
	return t.name
}

// Description returns the tool description (implements tools.Tool interface)
func (t *RegistryTool) Description() string {
	return t.description
}

// Call executes the tool function (implements tools.Tool interface)
func (t *RegistryTool) Call(ctx context.Context, input string) (string, error) {
	// Parse the input JSON to get arguments
	var args map[string]any
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		return "", fmt.Errorf("failed to parse tool arguments: %w", err)
	}

	// Get the function from registry and execute it
	fn, ok := t.registry.Get(t.toolName)
	if !ok {
		return "", fmt.Errorf("tool %s not found in registry", t.toolName)
	}

	return fn(ctx, args)
}

// NewRegistry creates a new tool registry from a list of agent function specs.
func NewRegistry(agentFuncs agentModels.FunctionSpecs) *Registry {
	r := &Registry{
		tools: make(map[string]ToolFunc),
		specs: make([]ToolSpec, 0, len(agentFuncs)),
	}

	logger.Log.WithField("input_specs_count", len(agentFuncs)).Info("Creating function registry")

	// Log all available functions in StaticFunctionMap before processing
	availableFunctions := make([]string, 0, len(StaticFunctionMap))
	for implKey := range StaticFunctionMap {
		availableFunctions = append(availableFunctions, implKey)
	}
	logger.Log.WithFields(map[string]any{
		"available_functions_count": len(StaticFunctionMap),
		"available_functions":       availableFunctions,
	}).Info("Available functions in StaticFunctionMap")

	// Process each function spec
	registeredFunctions := make([]string, 0, len(agentFuncs))
	failedFunctions := make([]string, 0)

	for i, spec := range agentFuncs {
		logger.Log.WithFields(map[string]any{
			"spec_index": i,
			"spec_name":  spec.Name,
			"impl_key":   spec.ImplKey,
		}).Info("Processing function spec")

		// Get the function from the static registry using impl_key
		toolFunc, exists := StaticFunctionMap[spec.ImplKey]
		if !exists {
			logger.Log.WithFields(map[string]any{
				"function_name": spec.Name,
				"impl_key":      spec.ImplKey,
			}).Error("Function implementation not found in StaticFunctionMap")
			failedFunctions = append(failedFunctions, spec.ImplKey)
			continue
		}

		// Register the executable function
		r.tools[spec.Name] = toolFunc

		// Create tool spec for LLM
		toolSpec := ToolSpec{
			Type: "function",
			Function: &FunctionDefinition{
				Name:        spec.Name,
				Description: spec.Description,
				Parameters:  spec.Parameters,
			},
		}
		r.specs = append(r.specs, toolSpec)

		registeredFunctions = append(registeredFunctions, spec.Name)
		logger.Log.WithFields(map[string]any{
			"function_name": spec.Name,
			"impl_key":      spec.ImplKey,
		}).Info("Function registered successfully")
	}

	logger.Log.WithFields(map[string]any{
		"registered_functions_count": len(r.tools),
		"registered_specs_count":     len(r.specs),
	}).Info("Function registry created")

	return r
}

// Get returns the executable function for a given tool name.
func (r *Registry) Get(name string) (ToolFunc, bool) {
	fn, ok := r.tools[name]
	return fn, ok
}

// Specs returns the list of tool specifications to provide to the LLM.
func (r *Registry) Specs() []ToolSpec {
	logger.Log.WithField("specs_count", len(r.specs)).Info("Registry.Specs() called")
	return r.specs
}

// ListFunctions returns a list of registered function names (for debugging)
func (r *Registry) ListFunctions() []string {
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

// ToLangchainTools converts the registry to langchain-compatible tools
func (r *Registry) ToLangchainTools() []*RegistryTool {
	tools := make([]*RegistryTool, len(r.specs))

	for i, spec := range r.specs {
		tools[i] = &RegistryTool{
			name:        spec.Function.Name,
			description: spec.Function.Description,
			registry:    r,
			toolName:    spec.Function.Name,
		}
	}

	logger.Log.WithField("tools_count", len(tools)).Info("Converted registry to langchain tools")
	return tools
}

// RegisterFunction allows runtime registration of functions (optional feature)
func RegisterFunction(implKey string, fn ToolFunc) {
	StaticFunctionMap[implKey] = fn
	logger.Log.WithField("impl_key", implKey).Info("Function registered at runtime")
}

// IsImplementationAvailable checks if a function implementation exists
func IsImplementationAvailable(implKey string) bool {
	_, exists := StaticFunctionMap[implKey]
	return exists
}
