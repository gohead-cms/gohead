// Package validation holds checks that need the database.
package validation

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gohead-cms/gohead/pkg/database"
	"github.com/gohead-cms/gohead/pkg/logger"

	"gorm.io/gorm"
)

// JSONPath returns the SQLite/MySQL JSON path of a top-level key. The key is
// quoted, so names with dots, dashes or spaces address the right member.
func JSONPath(key string) string {
	return `$."` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(key) + `"`
}

// JSONTextExpr returns a SQL expression for the text value of a top-level
// key of the JSON "data" column, and its argument. The key is always bound,
// never interpolated.
func JSONTextExpr(dialect, key string) (string, any) {
	switch dialect {
	case "mysql":
		return "JSON_UNQUOTE(JSON_EXTRACT(data, ?))", JSONPath(key)
	case "postgres":
		return "(CAST(data AS JSONB) ->> CAST(? AS TEXT))", key
	default: // sqlite
		return "CAST(json_extract(data, ?) AS TEXT)", JSONPath(key)
	}
}

// CheckFieldUniqueness checks that no other live item of the collection has
// the same value for fieldName. excludeID (the item being updated, or 0) is
// ignored, and so are soft-deleted items. tx may be nil.
func CheckFieldUniqueness(tx *gorm.DB, collectionID uint, fieldName string, value any, excludeID uint) error {
	if tx == nil {
		tx = database.DB
	}
	// Compare typed JSON values, so numbers match by value (1e6 = 1000000)
	// and strings never match numbers.
	var expr string
	var args []any
	switch tx.Dialector.Name() {
	case "mysql":
		expr, args = "JSON_EXTRACT(data, ?) = CAST(? AS JSON)", []any{JSONPath(fieldName), jsonText(value)}
	case "postgres":
		expr, args = "(CAST(data AS JSONB) -> CAST(? AS TEXT)) = CAST(? AS JSONB)", []any{fieldName, jsonText(value)}
	default: // sqlite: json_extract returns SQL values (true/false as 1/0)
		v := value
		if b, ok := value.(bool); ok {
			v = 0
			if b {
				v = 1
			}
		}
		expr, args = "json_extract(data, ?) = ?", []any{JSONPath(fieldName), v}
	}

	q := tx.Table("items").
		Where("collection_id = ? AND deleted_at IS NULL", collectionID).
		Where(expr, args...)
	if excludeID != 0 {
		q = q.Where("id <> ?", excludeID)
	}
	var count int64
	if err := q.Count(&count).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to check field uniqueness")
		return fmt.Errorf("could not validate uniqueness for field '%s'", fieldName)
	}
	if count > 0 {
		logger.Log.WithField("field", fieldName).Warn("Uniqueness validation failed")
		return fmt.Errorf("value for field '%s' must be unique", fieldName)
	}
	return nil
}

// jsonText encodes value as a JSON literal.
func jsonText(value any) string {
	b, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf("%q", fmt.Sprint(value))
	}
	return string(b)
}
