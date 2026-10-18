package models

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/gohead-cms/gohead/pkg/database"
	"github.com/gohead-cms/gohead/pkg/media"

	"gorm.io/gorm"
)

// Media is an uploaded file. Its content lives in the media store (local
// directory or S3 bucket) under Key; this row holds its metadata.
type Media struct {
	gorm.Model
	ID         uint   `json:"id"`
	Name       string `json:"name"`                          // original file name
	Key        string `json:"-" gorm:"uniqueIndex;size:255"` // path in the store
	Driver     string `json:"-"`                             // store that holds it
	MimeType   string `json:"mime" gorm:"index"`             // detected from the content
	Size       int64  `json:"size"`                          // bytes
	Width      *int   `json:"width"`                         // images only
	Height     *int   `json:"height"`                        // images only
	Hash       string `json:"hash" gorm:"index"`             // SHA-256 of the content
	Alt        string `json:"alt"`                           // alternative text
	Caption    string `json:"caption"`                       //
	UploadedBy string `json:"uploadedBy" gorm:"index"`       // username
}

// MediaResponse is the JSON shape of a media file in API responses.
func MediaResponse(m Media, url string) map[string]any {
	return map[string]any{
		"id":         m.ID,
		"url":        url,
		"name":       m.Name,
		"mime":       m.MimeType,
		"size":       m.Size,
		"width":      m.Width,
		"height":     m.Height,
		"hash":       m.Hash,
		"alt":        m.Alt,
		"caption":    m.Caption,
		"uploadedBy": m.UploadedBy,
		"createdAt":  APITime(m.CreatedAt),
		"updatedAt":  APITime(m.UpdatedAt),
	}
}

var digitsOnly = regexp.MustCompile(`^[0-9]+$`)

// validateMediaValue validates the value of a media attribute and returns it
// normalized. A value references an uploaded file by ID (42, "42" or
// {"id": 42}); any other non-empty string is kept as an external URL, as
// media attributes held URLs before uploads existed. With "multiple" the
// value is a list of those.
func validateMediaValue(attr Attribute, value any, tx *gorm.DB) (any, error) {
	if tx == nil {
		tx = database.DB
	}
	if !attr.Multiple {
		return resolveMediaRef(attr, value, tx)
	}
	list, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("expected a list of media for '%s'", attr.Name)
	}
	if attr.Required && len(list) == 0 {
		return nil, fmt.Errorf("at least one media file is required for '%s'", attr.Name)
	}
	if attr.Min != nil && len(list) < *attr.Min {
		return nil, fmt.Errorf("at least %d media files are required for '%s'", *attr.Min, attr.Name)
	}
	if attr.Max != nil && len(list) > *attr.Max {
		return nil, fmt.Errorf("at most %d media files are allowed for '%s'", *attr.Max, attr.Name)
	}
	out := make([]any, 0, len(list))
	for i, el := range list {
		v, err := resolveMediaRef(attr, el, tx)
		if err != nil {
			return nil, fmt.Errorf("element %d: %w", i, err)
		}
		out = append(out, v)
	}
	return out, nil
}

func resolveMediaRef(attr Attribute, value any, tx *gorm.DB) (any, error) {
	var id uint
	switch v := value.(type) {
	case string:
		if v == "" {
			return nil, fmt.Errorf("empty media value for '%s'", attr.Name)
		}
		if !digitsOnly.MatchString(v) {
			if len(attr.AllowedTypes) > 0 {
				return nil, fmt.Errorf("'%s' only accepts uploaded files (restricted to %v)", attr.Name, attr.AllowedTypes)
			}
			return v, nil // external URL or path
		}
		id = ToUint(v)
	case map[string]any:
		id = ToUint(v["id"])
	default:
		id = ToUint(v)
	}
	if id == 0 {
		return nil, fmt.Errorf("expected a media ID, {\"id\": …} or a URL for '%s'", attr.Name)
	}
	var m Media
	if err := tx.Select("id", "mime_type").First(&m, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("media %d does not exist", id)
		}
		return nil, err
	}
	if len(attr.AllowedTypes) > 0 && !media.Allowed(m.MimeType, attr.AllowedTypes) {
		return nil, fmt.Errorf("media %d is %s; '%s' accepts %v", id, m.MimeType, attr.Name, attr.AllowedTypes)
	}
	return id, nil
}
