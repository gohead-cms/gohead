// internal/models/user.go
package models

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Username string `json:"username" gorm:"uniqueIndex"`
	Password string `json:"password"`
	Role     string `json:"role"` // "admin", "editor", "viewer"
}
