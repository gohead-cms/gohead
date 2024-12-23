package models

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"gitlab.com/sudo.bngz/gohead/pkg/logger"
	"gorm.io/gorm"
)

type Relationship struct {
	gorm.Model
	Attribute        string     `json:"name"`                                  // Field defining the relationship
	RelationType     string     `json:"relation_type"`                         // e.g., one-to-one, one-to-many, many-to-many
	CollectionTarget string     `json:"collection_target"`                     // e.g., collection target name
	Collection       Collection `json:"-" gorm:"foreignKey:CollectionID"`      // ID of the collection owning this relationship
	CollectionID     uint       `json:"-" gorm:"constraint:OnDelete:CASCADE;"` // Collection ID
	SourceItemID     *uint      `json:"-" gorm:"constraint:OnDelete:SET NULL"` // Nullable source item ID
	SourceItem       Item       `json:"-" gorm:"foreignKey:SourceItemID"`      // Source item
}

/*
// GetRelationshipByField retrieves a relationship by its field name.
func (c *Collection) GetRelationshipByField(fieldName string) (*Relationship, error) {
	if len(c.Relationships) == 0 {
		logger.Log.WithField("collection", c).Info("No relationship in this collection")
		// No relationships are defined in the collection
		return nil, nil
	}

	for _, rel := range c.Relationships {
		if rel.Field == fieldName {
			return &rel, nil
		}
	}

	// Field is not a relationship
	return nil, fmt.Errorf("unknown relationship field '%s'", fieldName)
} */

// func ValidateRelationships(ct *Collection, relationships map[string]interface{}) error {
// 	for field, value := range relationships {
// 		// Check if the field exists as a relationship in the collection schema
// 		rel, err := ct.GetRelationshipByField(field)
// 		if rel == nil && err == nil {
// 			logger.Log.WithField("collection_id", ct.ID).Info("No relationships")
// 			return nil
// 		}
// 		if err != nil {
// 			logger.Log.WithError(err).Error("Failed to retrieve relationship")
// 			// Handle error (e.g., return an HTTP 400/404 response)
// 		} else {
// 			logger.Log.WithField("relationship", rel).Info("Relationship retrieved successfully")
// 			// Proceed with the relationship logic
// 			if rel == nil {
// 				logger.Log.WithField("field", field).Warn("Validation failed: Unknown relationship field")
// 				return fmt.Errorf("unknown relationship field '%s'", field)
// 			}

// 			// Validate the relationship type
// 			switch rel.RelationType {
// 			case "one-to-one", "one-to-many":
// 				if err := validateOneToOneOrOneToMany(field, value); err != nil {
// 					return err
// 				}
// 			case "many-to-many":
// 				if err := validateManyToMany(field, value); err != nil {
// 					return err
// 				}
// 			default:
// 				logger.Log.WithFields(logrus.Fields{
// 					"field":         field,
// 					"relation_type": rel.RelationType,
// 				}).Error("Validation failed: Unsupported relationship type")
// 				return fmt.Errorf("unsupported relationship type '%s' for field '%s'", rel.RelationType, field)
// 			}
// 		}
// 	}

// 	logger.Log.WithField("collection_id", ct.ID).Info("Relationships validated successfully")
// 	return nil
// }

// Helper function to validate one-to-one and one-to-many relationships
func validateOneToOneOrOneToMany(field string, value interface{}) error {
	switch v := value.(type) {
	case float64: // ID
		// Valid ID
		return nil
	case map[string]interface{}: // Nested object
		// Valid nested object
		return nil
	default:
		logger.Log.WithFields(logrus.Fields{
			"field": field,
			"value": v,
		}).Warn("Validation failed: Invalid type for one-to-one/one-to-many")
		return fmt.Errorf("invalid type for relationship field '%s'; expected ID or object", field)
	}
}

// Helper function to validate many-to-many relationships
func validateManyToMany(field string, value interface{}) error {
	arrayValue, ok := value.([]interface{})
	if !ok {
		logger.Log.WithFields(logrus.Fields{
			"field": field,
			"value": value,
		}).Warn("Validation failed: Invalid type for many-to-many")
		return fmt.Errorf("invalid type for relationship field '%s'; expected array", field)
	}

	for _, element := range arrayValue {
		switch element.(type) {
		case float64: // ID
			// Valid ID
			continue
		case map[string]interface{}: // Nested object
			// Valid nested object
			continue
		default:
			logger.Log.WithFields(logrus.Fields{
				"field":   field,
				"element": element,
			}).Warn("Validation failed: Invalid element type in many-to-many array")
			return fmt.Errorf("invalid element type in relationship array for field '%s'; expected ID or object", field)
		}
	}

	return nil
}
