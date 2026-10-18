package functions

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
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
