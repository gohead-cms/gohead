package models

import (
	"time"

	"gorm.io/gorm"
)

// APITokenPrefix starts every API token, so tokens are recognizable in
// headers, logs and secret scanners, and never confused with JWTs.
const APITokenPrefix = "ght_"

// APIToken is a long-lived key for front-ends, CI and integrations. Only a
// SHA-256 hash of the secret is stored; the secret is shown once, at
// creation. A token acts with the permissions of its role.
type APIToken struct {
	gorm.Model
	ID          uint       `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Role        string     `json:"role"`                         // viewer, editor or admin
	Hash        string     `json:"-" gorm:"uniqueIndex;size:64"` // hex SHA-256 of the secret
	Prefix      string     `json:"prefix"`                       // first characters, to recognize it
	CreatedBy   string     `json:"createdBy"`                    //
	ExpiresAt   *time.Time `json:"expiresAt"`                    // nil: never
	LastUsedAt  *time.Time `json:"lastUsedAt"`                   // updated at most once a minute
	RevokedAt   *time.Time `json:"revokedAt" gorm:"index"`       //
}

// Active reports whether the token can still be used at t.
func (t APIToken) Active(now time.Time) bool {
	if t.RevokedAt != nil {
		return false
	}
	return t.ExpiresAt == nil || now.Before(*t.ExpiresAt)
}

// APITokenResponse is the JSON shape of a token (never its secret).
func APITokenResponse(t APIToken) map[string]any {
	status := "active"
	switch {
	case t.RevokedAt != nil:
		status = "revoked"
	case !t.Active(time.Now()):
		status = "expired"
	}
	opt := func(p *time.Time) any {
		if p == nil {
			return nil
		}
		return APITime(*p)
	}
	return map[string]any{
		"id":          t.ID,
		"name":        t.Name,
		"description": t.Description,
		"role":        t.Role,
		"prefix":      t.Prefix,
		"status":      status,
		"createdBy":   t.CreatedBy,
		"createdAt":   APITime(t.CreatedAt),
		"expiresAt":   opt(t.ExpiresAt),
		"lastUsedAt":  opt(t.LastUsedAt),
		"revokedAt":   opt(t.RevokedAt),
	}
}
