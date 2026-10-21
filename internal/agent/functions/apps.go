package functions

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/gohead-cms/gohead/internal/agent/tools"
	"github.com/gohead-cms/gohead/internal/apps"
	"github.com/gohead-cms/gohead/internal/models"
	agentModels "github.com/gohead-cms/gohead/internal/models/agents"
	"github.com/gohead-cms/gohead/pkg/storage"
)

// The tools of connected apps are named "<connection>.<tool>"
// ("github.create_issue"). They are listed from the connections' stored
// tool lists, and called over MCP.

func init() {
	tools.Apps = appToolInfo
	tools.AppTools = appToolInfos
}

// splitAppTool splits "github.create_issue" into the connection and the
// tool name.
func splitAppTool(name string) (string, string, bool) {
	conn, tool, ok := strings.Cut(name, ".")
	if !ok || conn == "" || tool == "" {
		return "", "", false
	}
	return conn, tool, true
}

func appToolInfo(name string) (tools.Info, bool) {
	connName, toolName, ok := splitAppTool(name)
	if !ok {
		return tools.Info{}, false
	}
	if _, builtin := StaticFunctionMap[name]; builtin {
		return tools.Info{}, false
	}
	conn, ok := storage.CachedConnection(connName)
	if !ok {
		return tools.Info{}, false
	}
	t, ok := conn.Tool(toolName)
	if !ok {
		return tools.Info{}, false
	}
	return infoOf(conn, t), true
}

const maxDescription = 1024

func infoOf(conn models.Connection, t models.ConnectionTool) tools.Info {
	label := t.Title
	if label == "" {
		label = t.Name
	}
	desc := t.Description
	if len(desc) > maxDescription {
		desc = truncateText(desc, maxDescription)
	}
	if desc == "" {
		desc = label
	}
	key := conn.Name + "." + t.Name
	return tools.Info{
		ImplKey: key, Use: key, Name: tools.ModelName(conn.Name + "_" + t.Name),
		Label: label, Group: tools.GroupApps, Description: desc,
		Parameters: t.InputSchema, Writes: !t.ReadOnly,
		App: conn.Name, AppLabel: conn.Label,
	}
}

func truncateText(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

func appToolInfos() []tools.Info {
	var out []tools.Info
	for _, c := range storage.CachedConnections() {
		for _, t := range c.Tools {
			out = append(out, infoOf(c, t))
		}
	}
	return out
}

// isAppTool reports whether a spec names a connected app's tool.
func isAppTool(implKey string) bool {
	if _, builtin := StaticFunctionMap[implKey]; builtin {
		return false
	}
	_, _, ok := splitAppTool(implKey)
	return ok && !strings.HasPrefix(implKey, "collections.") && !strings.HasPrefix(implKey, "llm.")
}

// needsApproval reports whether calls of the tool wait for an admin.
func needsApproval(spec agentModels.FunctionSpec, info tools.Info, mode WriteMode) bool {
	if spec.Approval != nil && *spec.Approval {
		return true
	}
	if !info.Writes {
		return false
	}
	// An agent that requires approval has every change approved.
	return mode == WritePropose || spec.Approval == nil
}

// appTool calls a connected app's tool, or proposes the call, or (in a
// test run) only says what it would call.
func appTool(spec agentModels.FunctionSpec) ToolFunc {
	connName, toolName, _ := splitAppTool(spec.ImplKey)
	return func(ctx context.Context, args any) (string, error) {
		argMap, err := parseArgs(args)
		if err != nil {
			return `{"status":"error","message":"invalid arguments: give a JSON object"}`, nil
		}
		info, ok := appToolInfo(spec.ImplKey)
		if !ok {
			return errorResult(errString("the tool '" + spec.ImplKey + "' is no longer offered by the connection: refresh it in Settings → Connections")), nil
		}
		mode := writeModeOf(ctx)
		approval := needsApproval(spec, info, mode)
		if mode == WriteDryRun && (approval || info.Writes) {
			b, _ := json.Marshal(map[string]any{
				"status": "success", "dry_run": true, "action": "would_call", "tool": spec.ImplKey,
				"arguments": argMap,
				"message":   "Test run: the call was not made. Continue as if it had been.",
			})
			return string(b), nil
		}
		if approval {
			p, err := storage.ProposeAppCall(storage.ActorFrom(ctx), connName, toolName, argMap, "")
			if err != nil {
				return errorResult(err), nil
			}
			b, _ := json.Marshal(map[string]any{
				"status": "pending_approval", "proposal_id": p.ID, "action": "call", "tool": spec.ImplKey,
				"message": "The call was submitted for approval: an admin will make or reject it. Do not submit it again.",
			})
			return string(b), nil
		}
		res, err := apps.CallTool(ctx, connName, toolName, argMap)
		if err != nil {
			return errorResult(err), nil
		}
		return res.JSON(), nil
	}
}

type errString string

func (e errString) Error() string { return string(e) }

// ConnectionNameTaken reports whether built-in tools already use name as
// their prefix ("content", or "shop" for a fork's "shop.check_stock").
func ConnectionNameTaken(name string) bool {
	for key := range StaticFunctionMap {
		if prefix, _, _ := strings.Cut(key, "."); prefix == name {
			return true
		}
	}
	for _, t := range tools.All() {
		if prefix, _, _ := strings.Cut(t.Use, "."); prefix == name {
			return true
		}
	}
	return false
}
