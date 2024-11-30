package handlers

func hasPermission(role, action string) bool {
	permissions := map[string]map[string]bool{
		"admin": {
			"create": true,
			"read":   true,
			"update": true,
			"delete": true,
		},
		"editor": {
			"create": true,
			"read":   true,
			"update": true,
			"delete": true,
		},
		"viewer": {
			"read": true,
		},
	}

	if rolePermissions, exists := permissions[role]; exists {
		return rolePermissions[action]
	}
	return false
}
