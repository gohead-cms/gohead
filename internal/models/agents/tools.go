package models

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/gohead-cms/gohead/internal/agent/tools"
	"github.com/gohead-cms/gohead/internal/models"
)

// Tool is the short way to give an agent a built-in tool: which one, and
// on which collections. Everything else comes from the catalog unless the
// agent overrides it.
//
//	{"use": "content.update", "on": ["articles"]}
type Tool struct {
	// Use names the tool: "content.update", or its implementation key
	// "collections.patch_item".
	Use string `json:"use"`
	// On restricts a content tool to these collections.
	On []string `json:"on,omitempty"`
	// Name, Description and Parameters override the catalog's (the name
	// the model sees, what it is told, the arguments' JSON schema).
	Name        string         `json:"name,omitempty"`
	Description string         `json:"description,omitempty"`
	Parameters  models.JSONMap `json:"parameters,omitempty"`
}

// Spec is the tool as an agent stores it.
func (t Tool) Spec() FunctionSpec {
	return FunctionSpec{
		Name:        strings.TrimSpace(t.Name),
		Description: strings.TrimSpace(t.Description),
		Parameters:  t.Parameters,
		ImplKey:     tools.ImplKey(strings.TrimSpace(t.Use)),
		Collections: t.On,
	}
}

// ToolOf returns a stored function in the short form: only what differs
// from the catalog.
func ToolOf(f FunctionSpec) Tool {
	t := Tool{Use: tools.Use(f.ImplKey), On: f.Collections}
	info, known := tools.Lookup(f.ImplKey)
	if f.Name != "" && !(known && f.Name == info.Name) {
		t.Name = f.Name
	}
	if f.Description != "" && !(known && f.Description == info.Description) {
		t.Description = f.Description
	}
	if !tools.EmptySchema(f.Parameters) && !(known && tools.SameSchema(f.Parameters, info.Parameters)) {
		t.Parameters = f.Parameters
	}
	return t
}

// Tools returns an agent's functions in the short form.
func (f FunctionSpecs) Tools() []Tool {
	out := make([]Tool, 0, len(f))
	for _, s := range f {
		out = append(out, ToolOf(s))
	}
	return out
}

// Resolved returns the function as the model gets it: the name, the
// description and the parameters it leaves out come from the catalog.
func (f FunctionSpec) Resolved() FunctionSpec {
	out := f
	info, known := tools.Lookup(f.ImplKey)
	if known {
		out.ImplKey = info.ImplKey
	}
	if out.Name == "" {
		if known {
			out.Name = info.Name
		} else {
			out.Name = strings.ReplaceAll(out.ImplKey, ".", "_")
		}
	}
	if out.Description == "" && known {
		out.Description = info.Description
	}
	if tools.EmptySchema(out.Parameters) {
		if known {
			out.Parameters = clone(info.Parameters)
		} else if out.Parameters == nil {
			out.Parameters = models.JSONMap{"type": "object", "properties": map[string]any{}}
		}
	}
	return out
}

// ForModel is the function as the model gets it: Resolved, and a default
// description says which collections a scoped tool may use. (The scope
// note stays out of Resolved, which the API returns: a client sending it
// back would store it as a custom description.)
func (f FunctionSpec) ForModel() FunctionSpec {
	out := f.Resolved()
	if f.Description == "" && len(f.Collections) > 0 && out.Description != "" {
		out.Description += ". Only in: " + strings.Join(f.Collections, ", ")
	}
	return out
}

// Resolved returns every function as the model gets it.
func (f FunctionSpecs) Resolved() FunctionSpecs {
	out := make(FunctionSpecs, 0, len(f))
	for _, s := range f {
		out = append(out, s.Resolved())
	}
	return out
}

func clone(m models.JSONMap) models.JSONMap {
	b, err := json.Marshal(m)
	if err != nil {
		return m
	}
	var out models.JSONMap
	if err := json.Unmarshal(b, &out); err != nil {
		return m
	}
	return out
}

// toolsInput reads the "tools" list of an agent input, if there is one.
func toolsInput(input map[string]any) (FunctionSpecs, bool, error) {
	raw, ok := input["tools"]
	if !ok || raw == nil {
		return nil, false, nil
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return nil, true, err
	}
	var list []Tool
	if err := json.Unmarshal(b, &list); err != nil {
		return nil, true, fmt.Errorf("'tools' must be a list of {\"use\": …, \"on\": […]}: %w", err)
	}
	specs := make(FunctionSpecs, 0, len(list))
	for _, t := range list {
		specs = append(specs, t.Spec())
	}
	return specs, true, nil
}

// sameTools reports whether two lists give the model the same tools.
func sameTools(a, b FunctionSpecs) bool {
	ja, err1 := json.Marshal(a.Resolved())
	jb, err2 := json.Marshal(b.Resolved())
	return err1 == nil && err2 == nil && string(ja) == string(jb)
}

// Model APIs accept tool names of letters, digits, _ and - only.
var toolName = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)
