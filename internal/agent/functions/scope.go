package functions

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/storage"
)

type scopeKey struct{}

// withCollections returns a context limiting collections.* functions to
// the given collections (nil or empty: all collections).
func withCollections(ctx context.Context, collections []string) context.Context {
	if len(collections) == 0 {
		return ctx
	}
	return context.WithValue(ctx, scopeKey{}, collections)
}

// collectionAllowed reports whether the running function may use the
// collection.
func collectionAllowed(ctx context.Context, name string) bool {
	allowed, ok := ctx.Value(scopeKey{}).([]string)
	return !ok || slices.Contains(allowed, name)
}

// scopeError is the result returned to the model for an out-of-scope
// collection. It is a normal result (not an error) so the model can adapt.
func scopeError(name string) string {
	msg, _ := json.Marshal(fmt.Sprintf("this function is not allowed to access collection '%s'", name))
	return fmt.Sprintf(`{"status": "error", "message": %s}`, msg)
}

// scoped wraps a function so it runs with its spec's collection scope.
func scoped(collections []string, fn ToolFunc) ToolFunc {
	if len(collections) == 0 {
		return fn
	}
	return func(ctx context.Context, args any) (string, error) {
		return fn(withCollections(ctx, collections), args)
	}
}

// nestedScopeViolation returns the first collection outside the scope that
// writing data into ct would look up or create items in ("" if none).
// Relation values given as objects without an "id" are matched against, or
// created in, the target collection, so their target must be in scope too.
// Raw IDs and {"id": …} only connect existing items and are allowed.
func nestedScopeViolation(ctx context.Context, ct *models.Collection, data map[string]any, depth int) (string, error) {
	if _, scopedCtx := ctx.Value(scopeKey{}).([]string); !scopedCtx || depth > 5 {
		return "", nil
	}
	for _, attr := range ct.Attributes {
		if attr.Type != "relation" || attr.Target == "" {
			continue
		}
		var objects []map[string]any
		switch v := data[attr.Name].(type) {
		case map[string]any:
			objects = append(objects, v)
		case []any:
			for _, el := range v {
				if m, ok := el.(map[string]any); ok {
					objects = append(objects, m)
				}
			}
		}
		for _, obj := range objects {
			if _, hasID := obj["id"]; hasID {
				continue
			}
			if !collectionAllowed(ctx, attr.Target) {
				return attr.Target, nil
			}
			target, err := storage.GetCollectionByName(attr.Target)
			if err != nil {
				return "", err
			}
			if name, err := nestedScopeViolation(ctx, target, obj, depth+1); name != "" || err != nil {
				return name, err
			}
		}
	}
	return "", nil
}
