package functions

import (
	"fmt"
	"strings"

	"github.com/gohead-cms/gohead/internal/agent/tools"
	agentModels "github.com/gohead-cms/gohead/internal/models/agents"
	"github.com/gohead-cms/gohead/pkg/storage"
)

// Info describes a built-in tool for people configuring agents (see
// internal/agent/tools).
type Info = tools.Info

// Catalog lists the built-in tools: the catalog's, in display order, then
// functions registered without a catalog entry.
func Catalog() []Info {
	extra := make([]string, 0, len(StaticFunctionMap))
	for key := range StaticFunctionMap {
		extra = append(extra, key)
	}
	return tools.Known(extra)
}

// CheckTools refuses tools whose implementation doesn't exist.
func CheckTools(specs agentModels.FunctionSpecs) error {
	for _, s := range specs {
		if _, ok := StaticFunctionMap[s.ImplKey]; !ok {
			if isAppTool(s.ImplKey) {
				if _, known := appToolInfo(s.ImplKey); known {
					continue
				}
				conn, _, _ := splitAppTool(s.ImplKey)
				if _, ok := storage.CachedConnection(conn); ok {
					return fmt.Errorf("unknown tool '%s': connection '%s' has no such tool (refresh its tools in Settings → Connections)", s.ImplKey, conn)
				}
			}
			uses := make([]string, 0, len(tools.All()))
			for _, t := range tools.All() {
				uses = append(uses, t.Use)
			}
			return fmt.Errorf("unknown tool '%s' (built-in tools: %s)", s.ImplKey, strings.Join(uses, ", "))
		}
	}
	return nil
}
