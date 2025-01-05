// internal/models/json_map.go
package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// JSONMap is a custom type that implements driver.Valuer and sql.Scanner
type JSONMap map[string]interface{}

// Value implements the driver.Valuer interface, converting the JSONMap to a JSON string for storage.
func (j JSONMap) Value() (driver.Value, error) {
	return json.Marshal(j)
}

// Scan implements the sql.Scanner interface, converting a JSON string from the database into a JSONMap.
// internal/models/json_map.go
func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = JSONMap{}
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to scan JSONMap: type assertion to []byte failed")
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
