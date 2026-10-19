// internal/models/json_map.go
package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// JSONMap is a custom type that implements driver.Valuer and sql.Scanner
type JSONMap map[string]interface{}

// Value implements the driver.Valuer interface, converting the JSONMap to a JSON string for storage.
// It returns a string: MySQL refuses binary strings in JSON columns.
func (j JSONMap) Value() (driver.Value, error) {
	b, err := json.Marshal(j)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

// GormDBDataType makes JSONMap columns JSON on MySQL.
func (JSONMap) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	return JSONDBDataType(db, field)
}

// Scan implements the sql.Scanner interface, converting a JSON string from the database into a JSONMap.
// internal/models/json_map.go
func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = JSONMap{}
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string: // some drivers (e.g. SQLite for TEXT columns) return strings
		bytes = []byte(v)
	default:
		return errors.New("failed to scan JSONMap: unsupported source type")
	}

	var result map[string]interface{}
	if err := json.Unmarshal(bytes, &result); err != nil {
		return err
	}

	// Normalize types for consistent behavior
	for key, val := range result {
		if v, ok := val.(float64); ok && v == float64(int(v)) {
			result[key] = int(v)
		}
	}

	*j = result
	return nil
}
