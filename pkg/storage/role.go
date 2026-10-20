package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/auth"
	"github.com/gohead-cms/gohead/pkg/database"
	"github.com/gohead-cms/gohead/pkg/logger"

	"gorm.io/gorm"
)

// CreateRole creates a new role.
func SaveRole(role *models.UserRole) error {
	if err := database.DB.Create(role).Error; err != nil {
		logger.Log.WithError(err).WithField("role", role.Name).Error("Failed to create role")
		return fmt.Errorf("failed to create role: %w", err)
	}
	logger.Log.WithField("role", role.Name).Info("Role created successfully")
	return nil
}

// GetRoleByName retrieves a role by its name.
func GetRoleByName(name string) (*models.UserRole, error) {
	var role models.UserRole
	err := database.DB.Where("name = ?", name).First(&role).Error
	if err != nil {
		return nil, fmt.Errorf("role '%s' not found: %w", name, err)
	}
	return &role, nil
}

// GetRoleByID retrieves a role by its ID.
func GetRoleByID(id uint) (*models.UserRole, error) {
	var role models.UserRole
	if err := database.DB.First(&role, id).Error; err != nil {
		logger.Log.WithError(err).WithField("id", id).Warn("Role not found")
		return nil, fmt.Errorf("role with ID %d not found: %w", id, err)
	}
	return &role, nil
}

// GetAllRoles retrieves all roles from the database.
func GetAllRoles() ([]models.UserRole, error) {
	var roles []models.UserRole
	if err := database.DB.Find(&roles).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to retrieve roles")
		return nil, fmt.Errorf("failed to retrieve roles: %w", err)
	}
	return roles, nil
}

// UpdateRole updates an existing role by ID.
func UpdateRole(id uint, updates map[string]interface{}) error {
	var role models.UserRole
	if err := database.DB.First(&role, id).Error; err != nil {
		logger.Log.WithError(err).WithField("id", id).Warn("Role not found")
		return fmt.Errorf("role with ID %d not found: %w", id, err)
	}

	if err := database.DB.Model(&role).Updates(updates).Error; err != nil {
		logger.Log.WithError(err).WithField("id", id).Error("Failed to update role")
		return fmt.Errorf("failed to update role: %w", err)
	}
	logger.Log.WithField("role", role.Name).Info("Role updated successfully")
	return nil
}

// DeleteRole deletes a role by ID.
func DeleteRole(id uint) error {
	if err := database.DB.Delete(&models.UserRole{}, id).Error; err != nil {
		logger.Log.WithError(err).WithField("id", id).Error("Failed to delete role")
		return fmt.Errorf("failed to delete role: %w", err)
	}
	logger.Log.WithField("id", id).Info("Role deleted successfully")
	return nil
}

// ---- Custom roles ----

var (
	// ErrRoleNotFound is returned for an unknown role.
	ErrRoleNotFound = errors.New("role not found")
	// ErrRoleExists is returned when a role name is taken.
	ErrRoleExists = errors.New("a role with this name already exists")
	// ErrBuiltinRole is returned when changing admin, editor or viewer.
	ErrBuiltinRole = errors.New("built-in roles (admin, editor, viewer) cannot be changed")
	// ErrPublicRole is returned when deleting the public role or giving it
	// to an account or a token.
	ErrPublicRole = errors.New("the public role is for anonymous requests: it cannot be deleted or given to accounts and tokens")
)

// AssignableRole returns a role that accounts and tokens may have.
func AssignableRole(name string) (*models.UserRole, error) {
	if auth.IsPublicRole(name) {
		return nil, ErrPublicRole
	}
	return GetRoleByName(name)
}

// EnsurePublicRole creates the public role (granting nothing) when missing.
func EnsurePublicRole() error {
	defer invalidateRoles()
	role := models.UserRole{Name: auth.RolePublic, Description: "Anonymous visitors (requests without a token)",
		Permissions: permissionsMap(auth.Permissions{Content: map[string][]string{}, Media: []string{}})}
	return database.DB.Where(models.UserRole{Name: auth.RolePublic}).Attrs(role).FirstOrCreate(&models.UserRole{}).Error
}

// PublicAccessEnabled reports whether the public role grants anything, so
// anonymous requests are let in.
func PublicAccessEnabled() bool {
	return !auth.PermissionsOf(auth.RolePublic).Empty()
}

// RoleInUseError is returned when deleting a role that accounts or API
// tokens still use.
type RoleInUseError struct {
	Users, Tokens int64
}

func (e *RoleInUseError) Error() string {
	return fmt.Sprintf("the role is used by %d user(s) and %d API token(s): give them another role first", e.Users, e.Tokens)
}

var roleNameRe = regexp.MustCompile(`^[a-z][a-z0-9_-]{1,31}$`)

// RoleView is a role with its permissions in API form.
type RoleView struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Builtin     bool   `json:"builtin"`
	// Public: the role of anonymous requests (read permissions only).
	Public      bool             `json:"public,omitempty"`
	Permissions auth.Permissions `json:"permissions"`
}

func roleView(r models.UserRole) RoleView {
	return RoleView{ID: r.ID, Name: r.Name, Description: r.Description, Builtin: auth.IsBuiltinRole(r.Name),
		Public: auth.IsPublicRole(r.Name), Permissions: auth.PermissionsOf(r.Name)}
}

// ListRoleViews lists every role: built-in ones, the public role, then
// custom roles.
func ListRoleViews() ([]RoleView, error) {
	if err := EnsurePublicRole(); err != nil {
		return nil, err
	}
	roles, err := GetAllRoles()
	if err != nil {
		return nil, err
	}
	out := make([]RoleView, 0, len(roles))
	for _, r := range roles {
		out = append(out, roleView(r))
	}
	rank := func(v RoleView) int {
		switch {
		case v.Builtin:
			return 0
		case v.Public:
			return 1
		}
		return 2
	}
	sort.SliceStable(out, func(i, j int) bool {
		if rank(out[i]) != rank(out[j]) {
			return rank(out[i]) < rank(out[j])
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

// GetRoleView returns one role.
func GetRoleView(name string) (*RoleView, error) {
	if auth.IsPublicRole(name) {
		if err := EnsurePublicRole(); err != nil {
			return nil, err
		}
	}
	var r models.UserRole
	if err := database.DB.Where("name = ?", name).First(&r).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRoleNotFound
		}
		return nil, err
	}
	v := roleView(r)
	return &v, nil
}

// NormalizePermissions validates permissions against the content model and
// returns them cleaned up: actions sorted and deduplicated, read added to
// any type with another action.
func NormalizePermissions(p auth.Permissions) (auth.Permissions, error) {
	out := auth.Permissions{Content: map[string][]string{}, Media: []string{}}
	collections, _, err := GetAllCollections(nil, nil, nil)
	if err != nil {
		return out, err
	}
	known := map[string]bool{}
	for _, c := range collections {
		known[c.Name] = true
	}
	singletons, err := GetAllSingletons()
	if err != nil {
		return out, err
	}
	for _, s := range singletons {
		known[auth.SingletonResource(s.Name)] = true
	}
	clean := func(what string, actions, allowed []string) ([]string, error) {
		set := map[string]bool{}
		for _, a := range actions {
			if !slices.Contains(allowed, a) {
				return nil, fmt.Errorf("%s: unknown action '%s' (allowed: %s)", what, a, strings.Join(allowed, ", "))
			}
			set[a] = true
		}
		if len(set) > 0 && slices.Contains(allowed, auth.ActionRead) {
			set[auth.ActionRead] = true // every other action needs to see the content
		}
		list := []string{}
		for _, a := range allowed {
			if set[a] {
				list = append(list, a)
			}
		}
		return list, nil
	}
	for key, actions := range p.Content {
		allowed := auth.ContentActions
		if _, ok := auth.IsSingletonResource(key); ok {
			allowed = auth.SingletonActions
		}
		if key != auth.AnyType && !known[key] {
			return out, fmt.Errorf("unknown content type '%s' (a collection name, singleton:<name> or *)", key)
		}
		list, err := clean(key, actions, allowed)
		if err != nil {
			return out, err
		}
		out.Content[key] = list
	}
	if out.Media, err = clean("media", p.Media, auth.MediaActions); err != nil {
		return out, err
	}
	return out, nil
}

// checkPublicPermissions refuses what anonymous visitors may not have:
// anything but read, "*" (types created later would become public), the
// media library (files used by readable content are returned with it), and
// types with password attributes (their values are returned as stored).
func checkPublicPermissions(p auth.Permissions) error {
	secret, err := typesWithPasswords()
	if err != nil {
		return err
	}
	for key, actions := range p.Content {
		if len(actions) == 0 {
			continue
		}
		if key == auth.AnyType {
			return errors.New("the public role lists the types it may read: \"*\" would make every type created later public")
		}
		if secret[key] {
			return fmt.Errorf("%s has password attributes: it cannot be public", key)
		}
		for _, a := range actions {
			if !slices.Contains(auth.PublicActions, a) {
				return fmt.Errorf("the public role can only read: '%s' on %s is not allowed", a, key)
			}
		}
	}
	if len(p.Media) > 0 {
		return errors.New("the public role cannot browse the media library: files used by public content are returned with it")
	}
	return nil
}

// publicView keeps, from the public role's stored permissions, what
// anonymous visitors may have now (a type that gained a password attribute
// since it was opened is closed again).
func publicView(p auth.Permissions) auth.Permissions {
	secret, err := typesWithPasswords()
	out := auth.Permissions{Content: map[string][]string{}, Media: []string{}}
	if err != nil {
		return out // fail closed
	}
	for key, actions := range p.Content {
		if key == auth.AnyType || secret[key] || !slices.Contains(actions, auth.ActionRead) {
			continue
		}
		out.Content[key] = []string{auth.ActionRead}
	}
	return out
}

type attrRef struct{ Type, ComponentRef string }

func attrRefs(attrs []models.Attribute) []attrRef {
	out := make([]attrRef, len(attrs))
	for i, a := range attrs {
		out[i] = attrRef{a.Type, a.ComponentRef}
	}
	return out
}

func componentRefs(c models.Component) []attrRef {
	out := make([]attrRef, len(c.Attributes))
	for i, a := range c.Attributes {
		out[i] = attrRef{a.Type, a.ComponentRef}
	}
	return out
}

// typesWithPasswords lists the content types (permission keys) with a
// password attribute, directly or in a component.
func typesWithPasswords() (map[string]bool, error) {
	components, err := GetAllComponents()
	if err != nil {
		return nil, err
	}
	byName := map[string]models.Component{}
	for _, c := range components {
		byName[c.Name] = c
	}
	var hasPassword func(attrs []attrRef, seen map[string]bool) bool
	hasPassword = func(attrs []attrRef, seen map[string]bool) bool {
		for _, a := range attrs {
			if a.Type == "password" {
				return true
			}
			if a.Type == "component" && !seen[a.ComponentRef] {
				seen[a.ComponentRef] = true
				if c, ok := byName[a.ComponentRef]; ok && hasPassword(componentRefs(c), seen) {
					return true
				}
			}
		}
		return false
	}
	out := map[string]bool{}
	collections, _, err := GetAllCollections(nil, nil, nil)
	if err != nil {
		return nil, err
	}
	for _, c := range collections {
		if hasPassword(attrRefs(c.Attributes), map[string]bool{}) {
			out[c.Name] = true
		}
	}
	singletons, err := GetAllSingletons()
	if err != nil {
		return nil, err
	}
	for _, s := range singletons {
		if hasPassword(attrRefs(s.Attributes), map[string]bool{}) {
			out[auth.SingletonResource(s.Name)] = true
		}
	}
	return out, nil
}

func permissionsMap(p auth.Permissions) models.JSONMap {
	b, _ := json.Marshal(p)
	var m models.JSONMap
	_ = json.Unmarshal(b, &m)
	return m
}

func permissionsFrom(m models.JSONMap) (auth.Permissions, bool) {
	b, err := json.Marshal(m)
	if err != nil {
		return auth.Permissions{}, false
	}
	var p auth.Permissions
	if err := json.Unmarshal(b, &p); err != nil {
		return auth.Permissions{}, false
	}
	return p, true
}

// CreateCustomRole adds a role.
func CreateCustomRole(name, description string, perms auth.Permissions) (*RoleView, error) {
	defer invalidateRoles()
	if auth.IsBuiltinRole(name) || auth.IsPublicRole(name) {
		return nil, ErrRoleExists
	}
	if !roleNameRe.MatchString(name) {
		return nil, &ItemValidationError{Err: errors.New("role names are 2–32 lowercase letters, digits, - or _, starting with a letter")}
	}
	clean, err := NormalizePermissions(perms)
	if err != nil {
		return nil, &ItemValidationError{Err: err}
	}
	role := models.UserRole{Name: name, Description: description, Permissions: permissionsMap(clean)}
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		var n int64
		if err := tx.Model(&models.UserRole{}).Where("name = ?", name).Count(&n).Error; err != nil {
			return err
		}
		if n > 0 {
			return ErrRoleExists
		}
		return tx.Create(&role).Error
	})
	if err != nil {
		return nil, err
	}
	v := RoleView{ID: role.ID, Name: role.Name, Description: role.Description, Permissions: clean}
	return &v, nil
}

// UpdateCustomRole replaces a custom role's description and permissions.
// Its name cannot change (accounts and tokens refer to it).
func UpdateCustomRole(name, description string, perms auth.Permissions) (*RoleView, error) {
	defer invalidateRoles()
	if auth.IsBuiltinRole(name) {
		return nil, ErrBuiltinRole
	}
	clean, err := NormalizePermissions(perms)
	if err != nil {
		return nil, &ItemValidationError{Err: err}
	}
	if auth.IsPublicRole(name) {
		if err := checkPublicPermissions(clean); err != nil {
			return nil, &ItemValidationError{Err: err}
		}
		if err := EnsurePublicRole(); err != nil {
			return nil, err
		}
	}
	var role models.UserRole
	if err := database.DB.Where("name = ?", name).First(&role).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRoleNotFound
		}
		return nil, err
	}
	if err := database.DB.Model(&role).Updates(map[string]any{"description": description, "permissions": permissionsMap(clean)}).Error; err != nil {
		return nil, err
	}
	v := RoleView{ID: role.ID, Name: role.Name, Description: description, Permissions: clean}
	return &v, nil
}

// DeleteCustomRole removes a custom role nobody uses.
func DeleteCustomRole(name string) error {
	defer invalidateRoles()
	if auth.IsBuiltinRole(name) {
		return ErrBuiltinRole
	}
	if auth.IsPublicRole(name) {
		return ErrPublicRole
	}
	return database.DB.Transaction(func(tx *gorm.DB) error {
		var role models.UserRole
		if err := lockRow(tx).Where("name = ?", name).First(&role).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrRoleNotFound
			}
			return err
		}
		var users, tokens int64
		if err := tx.Model(&models.User{}).Where("user_role_id = ?", role.ID).Count(&users).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.APIToken{}).Where("role = ? AND revoked_at IS NULL", name).Count(&tokens).Error; err != nil {
			return err
		}
		if users > 0 || tokens > 0 {
			return &RoleInUseError{Users: users, Tokens: tokens}
		}
		return tx.Delete(&role).Error
	})
}

// ---- Permission lookup for pkg/auth ----

const roleCacheTTL = 5 * time.Second

type roleSnapshot struct {
	db    *gorm.DB
	at    time.Time
	roles map[string]auth.Permissions
}

var (
	roleMu    sync.Mutex
	roleCache *roleSnapshot
)

func invalidateRoles() {
	roleMu.Lock()
	roleCache = nil
	roleMu.Unlock()
}

// lookupRole returns a custom role's stored permissions, cached briefly.
func lookupRole(name string) (auth.Permissions, bool) {
	roleMu.Lock()
	defer roleMu.Unlock()
	if database.DB == nil {
		return auth.Permissions{}, false
	}
	if c := roleCache; c == nil || c.db != database.DB || time.Since(c.at) >= roleCacheTTL {
		var roles []models.UserRole
		if err := database.DB.Find(&roles).Error; err != nil {
			return auth.Permissions{}, false
		}
		snap := &roleSnapshot{db: database.DB, at: time.Now(), roles: map[string]auth.Permissions{}}
		for _, r := range roles {
			if p, ok := permissionsFrom(r.Permissions); ok {
				if auth.IsPublicRole(r.Name) {
					p = publicView(p)
				}
				snap.roles[r.Name] = p
			}
		}
		roleCache = snap
	}
	p, ok := roleCache.roles[name]
	return p, ok
}

func init() {
	auth.SetRoleLookup(lookupRole)
}

// moveRoleKeys renames (to != "") or removes (to == "") a content type's
// key in every custom role, so a renamed type keeps its permissions and a
// deleted type's grants do not pass to a new type of the same name.
func moveRoleKeys(tx *gorm.DB, from, to string) error {
	var roles []models.UserRole
	if err := tx.Find(&roles).Error; err != nil {
		return err
	}
	for _, r := range roles {
		if auth.IsBuiltinRole(r.Name) {
			continue
		}
		p, ok := permissionsFrom(r.Permissions)
		if !ok {
			continue
		}
		actions, has := p.Content[from]
		if !has {
			continue
		}
		delete(p.Content, from)
		if to != "" {
			p.Content[to] = actions
		}
		if err := tx.Model(&models.UserRole{}).Where("id = ?", r.ID).Update("permissions", permissionsMap(p)).Error; err != nil {
			return err
		}
	}
	invalidateRoles()
	return nil
}
