package auth

import (
	"context"
	"slices"
	"strings"
)

// Content actions checked against a role.
const (
	ActionCreate  = "create"
	ActionRead    = "read"
	ActionUpdate  = "update"
	ActionDelete  = "delete"
	ActionPublish = "publish"
)

// Built-in roles. Their permissions are fixed; custom roles are stored.
const (
	RoleAdmin  = "admin"
	RoleEditor = "editor"
	RoleViewer = "viewer"
)

// Resources other than content types.
const (
	// ResourceMedia is the media library.
	ResourceMedia = "media"
	// AnyType stands for every content type a role has no specific entry for.
	AnyType = "*"
	// singletonPrefix prefixes singleton names in permission keys.
	singletonPrefix = "singleton:"
)

// ContentActions are the actions on collection items; singletons use read
// and update; the media library read, create, update and delete.
var (
	ContentActions   = []string{ActionRead, ActionCreate, ActionUpdate, ActionDelete, ActionPublish}
	SingletonActions = []string{ActionRead, ActionUpdate}
	MediaActions     = []string{ActionRead, ActionCreate, ActionUpdate, ActionDelete}
)

// SingletonResource is the permission key of a singleton.
func SingletonResource(name string) string { return singletonPrefix + name }

// IsSingletonResource reports whether a key names a singleton, and which.
func IsSingletonResource(key string) (string, bool) { return strings.CutPrefix(key, singletonPrefix) }

// Permissions is what a role may do. Content maps a collection name,
// "singleton:<name>" or "*" (every other type) to actions; Media lists the
// actions on the media library.
type Permissions struct {
	Content map[string][]string `json:"content"`
	Media   []string            `json:"media"`
}

// Allows reports whether the permissions grant action on resource.
func (p Permissions) Allows(action, resource string) bool {
	if resource == ResourceMedia {
		return slices.Contains(p.Media, action)
	}
	actions, ok := p.Content[resource]
	if !ok {
		actions = p.Content[AnyType]
	}
	return slices.Contains(actions, action)
}

// AllowsAny reports whether action is granted on at least one content type.
func (p Permissions) AllowsAny(action string) bool {
	for _, actions := range p.Content {
		if slices.Contains(actions, action) {
			return true
		}
	}
	return false
}

var builtin = map[string]Permissions{
	RoleAdmin:  {Content: map[string][]string{AnyType: ContentActions}, Media: MediaActions},
	RoleEditor: {Content: map[string][]string{AnyType: ContentActions}, Media: MediaActions},
	RoleViewer: {Content: map[string][]string{AnyType: {ActionRead}}, Media: []string{}},
}

// IsBuiltinRole reports whether name is admin, editor or viewer.
func IsBuiltinRole(name string) bool {
	_, ok := builtin[name]
	return ok
}

// RoleLookup returns a custom role's permissions (false: no such role).
type RoleLookup func(role string) (Permissions, bool)

var lookup RoleLookup

// SetRoleLookup installs how custom roles are found (the storage layer
// does it at startup).
func SetRoleLookup(f RoleLookup) { lookup = f }

// PermissionsOf returns a role's permissions: fixed for built-in roles,
// stored for custom ones, none for unknown roles.
func PermissionsOf(role string) Permissions {
	if p, ok := builtin[role]; ok {
		return p
	}
	if lookup != nil {
		if p, ok := lookup(role); ok {
			return p
		}
	}
	return Permissions{}
}

// Can reports whether role may perform action on resource: a collection
// name, SingletonResource(name) or ResourceMedia.
func Can(role, action, resource string) bool {
	return PermissionsOf(role).Allows(action, resource)
}

// CanSeeDrafts reports whether role sees draft content of a type: roles
// that can edit or publish it.
func CanSeeDrafts(role, resource string) bool {
	p := PermissionsOf(role)
	return p.Allows(ActionUpdate, resource) || p.Allows(ActionPublish, resource)
}

// HasPermission reports whether role may perform action on at least one
// content type. Prefer Can, which checks one type.
func HasPermission(role, action string) bool {
	return PermissionsOf(role).AllowsAny(action)
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
