package handlers

import "github.com/gohead-cms/gohead/pkg/auth"

// hasPermission delegates to the shared role/permission table.
func hasPermission(role, action string) bool {
	return auth.HasPermission(role, action)
}
