package models

import (
	"fmt"

	"gitlab.com/sudo.bngz/gohead/pkg/logger"
	"gorm.io/gorm"
)

type Relationship struct {
	gorm.Model
	Field        string     `json:"field"`                                 // Field defining the relationship
	RelationType string     `json:"relation_type"`                         // e.g., one-to-one, one-to-many, many-to-many
	Collection   Collection `json:"-" gorm:"foreignKey:CollectionID"`      // ID of the collection owning this relationship
	CollectionID uint       `json:"-" gorm:"constraint:OnDelete:CASCADE;"` // Collection ID
	SourceItemID *uint      `json:"-" gorm:"constraint:OnDelete:SET NULL"` // Nullable source item ID
	SourceItem   Item       `json:"-" gorm:"foreignKey:SourceItemID"`      // Source item
}

// GetRelationshipByField retrieves a relationship by its field name.
func (c *Collection) GetRelationshipByField(fieldName string) *Relationship {
	for _, rel := range c.Relationships {
		if rel.Field == fieldName {
			return &rel
		}
	}
	return nil
}

func ValidateRelationships(ct *Collection, relationships map[string]interface{}) error {
	for field, value := range relationships {
		// Check if the field exists as a relationship in the collection schema
		rel := ct.GetRelationshipByField(field)
		if rel == nil {
			logger.Log.WithField("field", field).Warn("Validation failed: Unknown relationship field")
			return fmt.Errorf("unknown relationship field '%s'", field)
		}

		// Validate the relationship type
		switch rel.RelationType {
		case "one-to-one", "one-to-many":
			if _, ok := value.(float64); !ok {
				if _, isMap := value.(map[string]interface{}); !isMap {
					logger.Log.WithField("field", field).Warn("Validation failed: Invalid type for one-to-one/one-to-many")
					return fmt.Errorf("invalid type for relationship field '%s'; expected ID or object", field)
				}
			}
		case "many-to-many":
			arrayValue, ok := value.([]interface{})
			if !ok {
				logger.Log.WithField("field", field).Debug("Validation failed: Invalid type for many-to-many")
				return fmt.Errorf("invalid type for relationship field '%s'; expected array", field)
			}
			for _, element := range arrayValue {
				if _, isID := element.(float64); !isID {
					if _, isMap := element.(map[string]interface{}); !isMap {
						logger.Log.WithField("field", field).Warn("Validation failed: Invalid element type in many-to-many array")
						return fmt.Errorf("invalid element type in relationship array for field '%s'; expected ID or object", field)
					}
				}
			}
		default:
			logger.Log.WithField("relation_type", rel.RelationType).Error("Validation failed: Unsupported relationship type")
			return fmt.Errorf("unsupported relationship type '%s' for field '%s'", rel.RelationType, field)
		}
	}
	logger.Log.WithField("collection_id", ct.ID).Info("Relationships validated successfully")
	return nil
}
