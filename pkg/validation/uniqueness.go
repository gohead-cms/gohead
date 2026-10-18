// Package validation holds checks that need the database.
package validation

import (
	"fmt"

	"github.com/gohead-cms/gohead/pkg/database"
	"github.com/gohead-cms/gohead/pkg/logger"

	"gorm.io/gorm"
)

// CheckFieldUniqueness checks that no other live item of the collection has
// the same value for fieldName. excludeID (the item being updated, or 0) is
// ignored, and so are soft-deleted items. tx may be nil.
func CheckFieldUniqueness(tx *gorm.DB, collectionID uint, fieldName string, value any, excludeID uint) error {
	if tx == nil {
		tx = database.DB
	}
	var expr string
	var path any
	switch tx.Dialector.Name() {
	case "mysql":
		expr, path = "JSON_UNQUOTE(JSON_EXTRACT(data, ?)) = ?", "$."+fieldName
	case "postgres":
		expr, path = "data->>CAST(? AS TEXT) = ?", fieldName
	default: // sqlite
		expr, path = "CAST(json_extract(data, ?) AS TEXT) = ?", "$."+fieldName
	}

	q := tx.Table("items").
		Where("collection_id = ? AND deleted_at IS NULL", collectionID).
		Where(expr, path, fmt.Sprintf("%v", value))
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
