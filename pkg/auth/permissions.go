package auth

import "context"

// Content actions checked against a role.
const (
	ActionCreate = "create"
	ActionRead   = "read"
	ActionUpdate = "update"
	ActionDelete = "delete"
)

// rolePermissions maps the seeded roles to the content actions they may
// perform. It is shared by the REST and GraphQL APIs so both enforce the
// same rules.
var rolePermissions = map[string]map[string]bool{
	"admin":  {ActionCreate: true, ActionRead: true, ActionUpdate: true, ActionDelete: true},
	"editor": {ActionCreate: true, ActionRead: true, ActionUpdate: true, ActionDelete: true},
	"viewer": {ActionRead: true},
}

// HasPermission reports whether role may perform action on content.
func HasPermission(role, action string) bool {
	return rolePermissions[role][action]
}

type roleCtxKey struct{}

// WithRole returns a context carrying the authenticated user's role.
func WithRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, roleCtxKey{}, role)
}

// RoleFromContext returns the role stored by WithRole, or "".
func RoleFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	role, _ := ctx.Value(roleCtxKey{}).(string)
	return role
}
