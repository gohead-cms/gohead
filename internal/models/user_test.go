package models

import (
	"bytes"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"gitlab.com/sudo.bngz/gohead/pkg/logger"
)

// Initialize logger for testing
func init() {
	// Configure logger to write logs to a buffer for testing
	var buffer bytes.Buffer
	logger.InitLogger("debug")
	logger.Log.SetOutput(&buffer)
	logger.Log.SetFormatter(&logrus.TextFormatter{})
}

func TestValidateUser(t *testing.T) {
	tests := []struct {
		name    string
		user    User
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid user",
			user: User{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "securepassword",
				Role: UserRole{
					Name: "admin",
					Permissions: JSONMap{
						"read":  true,
						"write": true,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Missing username",
			user: User{
				Email:    "test@example.com",
				Password: "securepassword",
				Role: UserRole{
					Name: "admin",
					Permissions: JSONMap{
						"read":  true,
						"write": true,
					},
				},
			},
			wantErr: true,
			errMsg:  "username is required",
		},
		{
			name: "Invalid email",
			user: User{
				Username: "testuser",
				Email:    "invalid-email",
				Password: "securepassword",
				Role: UserRole{
					Name: "admin",
					Permissions: JSONMap{
						"read":  true,
						"write": true,
					},
				},
			},
			wantErr: true,
			errMsg:  "invalid email address",
		},
		{
			name: "Password too short",
			user: User{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "123",
				Role: UserRole{
					Name: "admin",
					Permissions: JSONMap{
						"read":  true,
						"write": true,
					},
				},
			},
			wantErr: true,
			errMsg:  "password must be at least 6 characters long",
		},
		{
			name: "Missing role",
			user: User{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "securepassword",
			},
			wantErr: true,
			errMsg:  "role is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUser(tt.user)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Equal(t, tt.errMsg, err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateUserRole(t *testing.T) {
	tests := []struct {
		name    string
		role    UserRole
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid role",
			role: UserRole{
				Name: "admin",
				Permissions: JSONMap{
					"read":  true,
					"write": true,
				},
			},
			wantErr: false,
		},
		{
			name: "Missing role name",
			role: UserRole{
				Permissions: JSONMap{
					"read":  true,
					"write": true,
				},
			},
			wantErr: true,
			errMsg:  "role name is required",
		},
		{
			name: "No permissions",
			role: UserRole{
				Name:        "editor",
				Permissions: JSONMap{},
			},
			wantErr: true,
			errMsg:  "at least one permission is required for the role",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUserRole(tt.role)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Equal(t, tt.errMsg, err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateUserUpdates(t *testing.T) {
	tests := []struct {
		name    string
		updates map[string]interface{}
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid updates",
			updates: map[string]interface{}{
				"username": "newusername",
				"email":    "new@example.com",
				"password": "newpassword",
				"role": UserRole{
					Name: "editor",
					Permissions: JSONMap{
						"read": true,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Invalid username",
			updates: map[string]interface{}{
				"username": "",
			},
			wantErr: true,
			errMsg:  "invalid username",
		},
		{
			name: "Invalid email",
			updates: map[string]interface{}{
				"email": "invalid-email",
			},
			wantErr: true,
			errMsg:  "invalid email address",
		},
		{
			name: "Short password",
			updates: map[string]interface{}{
				"password": "123",
			},
			wantErr: true,
			errMsg:  "password must be at least 6 characters long",
		},
		{
			name: "Unsupported field",
			updates: map[string]interface{}{
				"unsupported_field": "value",
			},
			wantErr: true,
			errMsg:  "unsupported field for update: unsupported_field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUserUpdates(tt.updates)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Equal(t, tt.errMsg, err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
