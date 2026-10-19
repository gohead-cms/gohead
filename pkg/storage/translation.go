package storage

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/gohead-cms/gohead/internal/agent/events"
	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/database"
	"github.com/gohead-cms/gohead/pkg/validation"

	"gorm.io/gorm"
)

// ErrDefaultLocaleWrite is returned when a translation is written in the
// default locale (its content is the entry's main data).
var ErrDefaultLocaleWrite = errors.New("this is the default locale: write the entry itself")

// localizedOnly returns the localized attributes, made optional: a missing
// value falls back to the default locale.
func localizedOnly(attrs []models.Attribute) []models.Attribute {
	var out []models.Attribute
	for _, a := range attrs {
		if a.Localized {
			a.Required = false
			a.Unique = false // checked per locale
			out = append(out, a)
		}
	}
	return out
}

// checkTranslationKeys refuses values for attributes that are not
// localized, and drops nulls (a null removes the value in that locale).
func checkTranslationKeys(attrs []models.Attribute, data map[string]any, defaultLocale string) (map[string]any, error) {
	localized := models.LocalizedNames(attrs)
	known := map[string]bool{}
	for _, a := range attrs {
		known[a.Name] = true
	}
	var shared []string
	out := map[string]any{}
	for k, v := range data {
		switch {
		case !known[k]:
			return nil, fmt.Errorf("unknown attribute: '%s'", k)
		case !localized[k]:
			shared = append(shared, k)
		case v != nil:
			out[k] = v
		}
	}
	if len(shared) > 0 {
		sort.Strings(shared)
		return nil, fmt.Errorf("%s not localized: edit %s in the default locale (%s)",
			plural(shared, "attribute '"+shared[0]+"' is", "attributes '"+strings.Join(shared, "', '")+"' are"),
			plural(shared, "it", "them"), defaultLocale)
	}
	return out, nil
}

func plural(list []string, one, many string) string {
	if len(list) == 1 {
		return one
	}
	return many
}

// withTranslation returns translations with one locale replaced (removed
// when values is empty).
func withTranslation(translations models.JSONMap, locale string, values map[string]any) models.JSONMap {
	out := models.JSONMap{}
	for code, v := range translations {
		if code != locale {
			out[code] = v
		}
	}
	if len(values) > 0 {
		out[locale] = values
	}
	return out
}

// SetItemTranslation replaces an item's values in a locale other than the
// default. data holds localized attributes only; an empty object removes
// the translation. publish changes the publication state like an update.
func SetItemTranslation(actor Actor, ct models.Collection, itemID uint, locale string, data map[string]any, publish *bool) (*models.Item, error) {
	if publish != nil && !ct.DraftAndPublish {
		return nil, ErrDraftAndPublishDisabled
	}
	locale, defaultLocale, err := ResolveLocale(locale)
	if err != nil {
		return nil, err
	}
	if locale == defaultLocale {
		return nil, ErrDefaultLocaleWrite
	}
	values, err := checkTranslationKeys(ct.Attributes, data, defaultLocale)
	if err != nil {
		return nil, &ItemValidationError{Err: err}
	}

	var item models.Item
	var transition events.EventType
	nested := &models.NestedCreated{}
	txErr := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := lockRow(tx).Where("id = ? AND collection_id = ?", itemID, ct.ID).First(&item).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrItemNotFound
			}
			return err
		}
		before := &entryState{data: item.Data, translations: item.Translations, status: itemStatus(ct, &item)}
		pseudo := models.Collection{ID: ct.ID, Name: ct.Name, Attributes: localizedOnly(ct.Attributes)}
		processed, err := models.ValidateItemUpdate(pseudo, values, models.WithNestedCollector(tx, nested), item.ID)
		if err != nil {
			return &ItemValidationError{Err: err}
		}
		for _, a := range ct.Attributes {
			if v, ok := processed[a.Name]; ok && a.Localized && a.Unique {
				if err := validation.CheckLocalizedUniqueness(tx, ct.ID, locale, a.Name, v, item.ID); err != nil {
					return &ItemValidationError{Err: err}
				}
			}
		}
		if err := recordNested(tx, actor, nested); err != nil {
			return err
		}
		translations := withTranslation(item.Translations, locale, processed)
		if err := tx.Model(&item).Updates(map[string]any{"translations": translations}).Error; err != nil {
			return err
		}
		if publish != nil {
			if transition, err = setPublishedTx(tx, &item, *publish); err != nil {
				return err
			}
		}
		if err := reloadItem(tx, &item); err != nil {
			return err
		}
		if sameData(before.translations, item.Translations) && transition == "" {
			return nil
		}
		action := models.RevisionUpdate
		if sameData(before.translations, item.Translations) {
			action = models.RevisionUnpublish
			if transition == events.EventTypeItemPublished {
				action = models.RevisionPublish
			}
		}
		_, err = recordRevision(tx, actor, revisionInput{
			entityType: models.RevisionEntityItem, entityID: item.ID, typeName: ct.Name,
			action: action, status: itemStatus(ct, &item), data: item.Data, translations: item.Translations, before: before,
		})
		return err
	})
	if txErr != nil {
		return nil, txErr
	}
	publishNested(nested)
	publishEvent(database.DB, events.EventTypeItemUpdated, &item)
	if transition != "" {
		publishEvent(database.DB, transition, &item)
	}
	return &item, nil
}

// SetSingletonTranslation replaces a singleton's values in a locale other
// than the default (see SetItemTranslation). The singleton must have
// content already.
func SetSingletonTranslation(actor Actor, st *models.Singleton, locale string, data map[string]any) (*models.SingleItem, error) {
	locale, defaultLocale, err := ResolveLocale(locale)
	if err != nil {
		return nil, err
	}
	if locale == defaultLocale {
		return nil, ErrDefaultLocaleWrite
	}
	values, err := checkTranslationKeys(st.Attributes, data, defaultLocale)
	if err != nil {
		return nil, &ItemValidationError{Err: err}
	}
	pseudo := models.Singleton{Name: st.Name, Attributes: localizedOnly(st.Attributes)}
	if err := models.ValidateSingleItemValues(pseudo, values); err != nil {
		return nil, &ItemValidationError{Err: err}
	}

	var item models.SingleItem
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		if err := lockRow(tx).Where("single_type_id = ?", st.ID).First(&item).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrItemNotFound
			}
			return err
		}
		before := &entryState{data: item.Data, translations: item.Translations}
		translations := withTranslation(item.Translations, locale, values)
		if sameData(item.Translations, translations) {
			return nil
		}
		if err := tx.Model(&item).Updates(map[string]any{"translations": translations}).Error; err != nil {
			return err
		}
		item.Translations = translations
		_, err := recordRevision(tx, actor, revisionInput{
			entityType: models.RevisionEntitySingleton, entityID: st.ID, typeName: st.Name,
			action: models.RevisionUpdate, data: item.Data, translations: item.Translations, before: before,
		})
		return err
	})
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// restorableTranslations keeps the localized attributes of a revision's
// translations.
func restorableTranslations(attrs []models.Attribute, translations models.JSONMap) models.JSONMap {
	localized := models.LocalizedNames(attrs)
	out := models.JSONMap{}
	for code, v := range translations {
		m, ok := v.(map[string]any)
		if !ok {
			continue
		}
		kept := map[string]any{}
		for k, val := range m {
			if localized[k] {
				kept[k] = val
			}
		}
		if len(kept) > 0 {
			out[code] = kept
		}
	}
	return out
}

// FlattenEntry merges an entry's data and translations into one map for
// diffs: translated values are keyed "<attribute>@<locale>".
func FlattenEntry(data, translations models.JSONMap) models.JSONMap {
	out := models.JSONMap{}
	for k, v := range data {
		out[k] = v
	}
	for code, v := range translations {
		if m, ok := v.(map[string]any); ok {
			for k, val := range m {
				out[k+"@"+code] = val
			}
		}
	}
	return out
}
