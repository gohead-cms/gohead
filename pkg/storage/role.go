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
	"github.com/gohead-cms/gohead/pkg/database"
	"github.com/gohead-cms/gohead/pkg/auth"
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
)

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
	ID          uint             `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Builtin     bool             `json:"builtin"`
	Permissions auth.Permissions `json:"permissions"`
}

func roleView(r models.UserRole) RoleView {
	return RoleView{ID: r.ID, Name: r.Name, Description: r.Description, Builtin: auth.IsBuiltinRole(r.Name), Permissions: auth.PermissionsOf(r.Name)}
}

// ListRoleViews lists every role, built-in ones first.
func ListRoleViews() ([]RoleView, error) {
	roles, err := GetAllRoles()
	if err != nil {
		return nil, err
	}
	out := make([]RoleView, 0, len(roles))
	for _, r := range roles {
		out = append(out, roleView(r))
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Builtin != out[j].Builtin {
			return out[i].Builtin
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

// GetRoleView returns one role.
func GetRoleView(name string) (*RoleView, error) {
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
	if auth.IsBuiltinRole(name) {
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
