package storage

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/database"
	"github.com/gohead-cms/gohead/pkg/logger"

	"gorm.io/gorm"
)

// ErrTokenNotFound is returned for unknown token IDs.
var ErrTokenNotFound = errors.New("token not found")

// ErrInvalidToken is returned for unknown, revoked or expired secrets.
var ErrInvalidToken = errors.New("invalid API token")

// HashAPIToken returns the stored form of a token secret. Tokens carry 256
// bits of randomness, so a fast hash is enough (unlike passwords).
func HashAPIToken(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

// NewAPITokenInput describes a token to create.
type NewAPITokenInput struct {
	Name        string
	Description string
	Role        string
	ExpiresAt   *time.Time
	CreatedBy   string
}

// CreateAPIToken stores a new token and returns it with its secret, which
// is not kept and cannot be shown again.
func CreateAPIToken(in NewAPITokenInput) (*models.APIToken, string, error) {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return nil, "", fmt.Errorf("name is required")
	}
	if _, err := AssignableRole(in.Role); err != nil {
		if errors.Is(err, ErrPublicRole) {
			return nil, "", err
		}
		return nil, "", fmt.Errorf("unknown role '%s'", in.Role)
	}
	if in.ExpiresAt != nil && !in.ExpiresAt.After(time.Now()) {
		return nil, "", fmt.Errorf("expiry must be in the future")
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return nil, "", err
	}
	secret := models.APITokenPrefix + hex.EncodeToString(raw)
	t := &models.APIToken{
		Name:        in.Name,
		Description: in.Description,
		Role:        in.Role,
		Hash:        HashAPIToken(secret),
		Prefix:      secret[:len(models.APITokenPrefix)+6],
		CreatedBy:   in.CreatedBy,
		ExpiresAt:   in.ExpiresAt,
	}
	if err := database.DB.Create(t).Error; err != nil {
		return nil, "", err
	}
	return t, secret, nil
}

// AuthenticateAPIToken returns the active token matching secret, and
// records its use (at most once a minute, to avoid a write per request).
func AuthenticateAPIToken(secret string) (*models.APIToken, error) {
	if !strings.HasPrefix(secret, models.APITokenPrefix) {
		return nil, ErrInvalidToken
	}
	var t models.APIToken
	if err := database.DB.Where("hash = ?", HashAPIToken(secret)).First(&t).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidToken
		}
		return nil, err
	}
	now := time.Now()
	if !t.Active(now) {
		return nil, ErrInvalidToken
	}
	if t.LastUsedAt == nil || now.Sub(*t.LastUsedAt) > time.Minute {
		if err := database.DB.Model(&models.APIToken{}).Where("id = ?", t.ID).
			Update("last_used_at", now).Error; err != nil {
			logger.Log.WithError(err).Warn("Failed to record API token use")
		}
	}
	return &t, nil
}

// ListAPITokens returns every token, newest first.
func ListAPITokens() ([]models.APIToken, error) {
	var out []models.APIToken
	err := database.DB.Order("id DESC").Find(&out).Error
	return out, err
}

// GetAPIToken returns a token by ID.
func GetAPIToken(id uint) (*models.APIToken, error) {
	var t models.APIToken
	if err := database.DB.First(&t, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTokenNotFound
		}
		return nil, err
	}
	return &t, nil
}

// RevokeAPIToken makes a token unusable immediately. The record stays, so
// the audit trail (name, creator, last use) is kept.
func RevokeAPIToken(id uint) (*models.APIToken, error) {
	t, err := GetAPIToken(id)
	if err != nil {
		return nil, err
	}
	if t.RevokedAt == nil {
		now := time.Now()
		if err := database.DB.Model(t).Update("revoked_at", now).Error; err != nil {
			return nil, err
		}
	}
	return GetAPIToken(id)
}
