// pkg/validation/uniqueness.go
package validation

import (
	"fmt"

	"gohead/pkg/database"
	"gohead/pkg/logger"
)

// CheckFieldUniqueness checks if a field value is unique within a collection.
func CheckFieldUniqueness(collectionID uint, fieldName string, value interface{}) error {
	var count int64
	query := database.DB.Table("items").
		Where("collection_id = ? AND data ->> ? = ?", collectionID, fieldName, value).
		Count(&count)

	if query.Error != nil {
		logger.Log.WithError(query.Error).Error("Failed to check field uniqueness")
		return fmt.Errorf("could not validate uniqueness for field '%s'", fieldName)
	}

	if count > 0 {
		logger.Log.WithFields(map[string]interface{}{
			"field": fieldName,
			"value": value,
		}).Warn("Uniqueness validation failed")
		return fmt.Errorf("value for field '%s' must be unique", fieldName)
	}

	return nil
}
