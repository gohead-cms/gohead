package models

import (
	"errors"
	"fmt"

	"github.com/gohead-cms/gohead/pkg/database"

	"gorm.io/gorm"
)

// maxComponentDepth bounds how deeply components may nest inside each other
// in item values (it also stops a component that references itself).
const maxComponentDepth = 5

// validateComponentReference checks that a component attribute names an
// existing component.
func validateComponentReference(attrName, ref string) error {
	if ref == "" {
		return fmt.Errorf("attribute '%s' has type 'component' but no 'component' name", attrName)
	}
	var cmp Component
	if err := database.DB.Where("name = ?", ref).First(&cmp).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("component '%s' for attribute '%s' does not exist", ref, attrName)
		}
		return fmt.Errorf("failed to check component '%s': %w", ref, err)
	}
	return nil
}

// validateComponentValue validates the value of a component attribute and
// returns it normalized: an object, or a list of objects when the attribute
// is repeatable. Unknown keys and missing required fields are rejected, and
// every field goes through the same checks as collection attributes.
func validateComponentValue(attr Attribute, value any, tx *gorm.DB, depth int) (any, error) {
	if depth > maxComponentDepth {
		return nil, fmt.Errorf("components cannot be nested more than %d levels deep", maxComponentDepth)
	}
	if tx == nil {
		tx = database.DB
	}
	var cmp Component
	if err := tx.Preload("Attributes").Where("name = ?", attr.ComponentRef).First(&cmp).Error; err != nil {
		return nil, fmt.Errorf("component '%s' does not exist", attr.ComponentRef)
	}

	if !attr.Repeatable {
		obj, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("expected an object for component '%s'", cmp.Name)
		}
		return validateComponentObject(cmp, obj, tx, depth)
	}

	list, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("expected an array of '%s' components", cmp.Name)
	}
	if attr.Required && len(list) == 0 {
		return nil, fmt.Errorf("at least one '%s' component is required", cmp.Name)
	}
	if attr.Min != nil && len(list) < *attr.Min {
		return nil, fmt.Errorf("at least %d '%s' components are required", *attr.Min, cmp.Name)
	}
	if attr.Max != nil && len(list) > *attr.Max {
		return nil, fmt.Errorf("at most %d '%s' components are allowed", *attr.Max, cmp.Name)
	}
	out := make([]any, 0, len(list))
	for i, el := range list {
		obj, ok := el.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("element %d: expected an object for component '%s'", i, cmp.Name)
		}
		v, err := validateComponentObject(cmp, obj, tx, depth)
		if err != nil {
			return nil, fmt.Errorf("element %d: %w", i, err)
		}
		out = append(out, v)
	}
	return out, nil
}

func validateComponentObject(cmp Component, obj map[string]any, tx *gorm.DB, depth int) (map[string]any, error) {
	known := make(map[string]bool, len(cmp.Attributes))
	for _, a := range cmp.Attributes {
		known[a.Name] = true
	}
	for k := range obj {
		if !known[k] {
			return nil, fmt.Errorf("unknown attribute '%s' in component '%s'", k, cmp.Name)
		}
	}

	out := make(map[string]any, len(obj))
	for _, ca := range cmp.Attributes {
		v, exists := obj[ca.Name]
		if !exists || v == nil {
			if ca.Required {
				return nil, fmt.Errorf("missing required attribute '%s' in component '%s'", ca.Name, cmp.Name)
			}
			continue
		}
		attr := ca.ToAttribute()
		switch ca.Type {
		case "component":
			nv, err := validateComponentValue(attr, v, tx, depth+1)
			if err != nil {
				return nil, fmt.Errorf("'%s': %w", ca.Name, err)
			}
			out[ca.Name] = nv
		case "media":
			nv, err := validateMediaValue(attr, v, tx)
			if err != nil {
				return nil, fmt.Errorf("'%s' in component '%s': %w", ca.Name, cmp.Name, err)
			}
			out[ca.Name] = nv
		case "relation", "dynamiczone":
			return nil, fmt.Errorf("attribute '%s' in component '%s': type '%s' is not supported inside components", ca.Name, cmp.Name, ca.Type)
		default:
			if err := validateAttributeValue(attr, v); err != nil {
				return nil, fmt.Errorf("'%s' in component '%s': %w", ca.Name, cmp.Name, err)
			}
			out[ca.Name] = v
		}
	}
	return out, nil
}
