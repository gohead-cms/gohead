package models

import (
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// JSONDBDataType implements GormDBDataType for JSON-valued types. MySQL has
// no JSONB, so JSON columns are created as JSON there; other databases keep
// the type given in the field's tag (json or jsonb).
func JSONDBDataType(db *gorm.DB, _ *schema.Field) string {
	if db.Dialector.Name() == "mysql" {
		return "JSON"
	}
	return ""
}

// JSONBytes returns the raw JSON of a scanned column: drivers return JSON
// columns as []byte (PostgreSQL, MySQL) or string (SQLite text columns).
func JSONBytes(value any) ([]byte, error) {
	switch v := value.(type) {
	case []byte:
		return v, nil
	case string:
		return []byte(v), nil
	default:
		return nil, errors.New("unsupported source type for a JSON column")
	}
}
