package storage

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/database"

	"gorm.io/gorm"
)

// DefaultLocaleCode is the default locale used until one is stored (set
// from i18n.default_locale at startup).
var DefaultLocaleCode = "en"

var (
	// ErrLocaleNotFound is returned for a locale that is not configured.
	ErrLocaleNotFound = errors.New("locale not found")
	// ErrLocaleExists is returned when adding a locale twice.
	ErrLocaleExists = errors.New("locale already exists")
	// ErrDefaultLocale is returned when deleting the default locale.
	ErrDefaultLocale = errors.New("the default locale cannot be deleted")
)

// LocaleInUseError reports content in a locale that would be lost or
// mislabelled.
type LocaleInUseError struct {
	Code    string
	Entries int64
}

func (e *LocaleInUseError) Error() string {
	return fmt.Sprintf("%d entries have content in locale '%s'", e.Entries, e.Code)
}

// EnsureDefaultLocale stores the default locale when no locale is stored
// yet (first start, or an upgrade from a version without locales).
func EnsureDefaultLocale(code string) error {
	if !models.ValidLocaleCode(code) {
		return fmt.Errorf("i18n.default_locale '%s' is not a valid locale code (e.g. en, fr, pt-BR)", code)
	}
	DefaultLocaleCode = code
	var n int64
	if err := database.DB.Model(&models.Locale{}).Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	return database.DB.Create(&models.Locale{Code: code, Name: code, IsDefault: true}).Error
}

// ListLocales returns the locales, the default first.
func ListLocales() ([]models.Locale, error) {
	var out []models.Locale
	if err := database.DB.Order("is_default DESC, code ASC").Find(&out).Error; err != nil {
		return nil, err
	}
	if len(out) == 0 {
		out = []models.Locale{{Code: DefaultLocaleCode, Name: DefaultLocaleCode, IsDefault: true}}
	}
	return out, nil
}

// DefaultLocale returns the default locale's code.
func DefaultLocale() (string, error) {
	var l models.Locale
	err := database.DB.Where("is_default = ?", true).First(&l).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return DefaultLocaleCode, nil
	}
	if err != nil {
		return "", err
	}
	return l.Code, nil
}

// ResolveLocale checks a requested locale and returns it with the default
// locale. "" means the default locale.
func ResolveLocale(requested string) (locale, defaultLocale string, err error) {
	defaultLocale, err = DefaultLocale()
	if err != nil {
		return "", "", err
	}
	if requested == "" || requested == defaultLocale {
		return defaultLocale, defaultLocale, nil
	}
	var n int64
	if err := database.DB.Model(&models.Locale{}).Where("code = ?", requested).Count(&n).Error; err != nil {
		return "", "", err
	}
	if n == 0 {
		return "", "", ErrLocaleNotFound
	}
	return requested, defaultLocale, nil
}

// CreateLocale adds a locale.
func CreateLocale(code, name string) (*models.Locale, error) {
	if !models.ValidLocaleCode(code) {
		return nil, &ItemValidationError{Err: fmt.Errorf("'%s' is not a valid locale code (e.g. en, fr, pt-BR)", code)}
	}
	if strings.TrimSpace(name) == "" {
		name = code
	}
	var out *models.Locale
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var n int64
		if err := tx.Model(&models.Locale{}).Where("code = ?", code).Count(&n).Error; err != nil {
			return err
		}
		if n > 0 {
			return ErrLocaleExists
		}
		// The implicit default locale is stored with the first new one.
		var total int64
		if err := tx.Model(&models.Locale{}).Count(&total).Error; err != nil {
			return err
		}
		if total == 0 && code != DefaultLocaleCode {
			if err := tx.Create(&models.Locale{Code: DefaultLocaleCode, Name: DefaultLocaleCode, IsDefault: true}).Error; err != nil {
				return err
			}
		}
		out = &models.Locale{Code: code, Name: name, IsDefault: total == 0 && code == DefaultLocaleCode}
		return tx.Create(out).Error
	})
	return out, err
}

// RenameLocale changes a locale's display name.
func RenameLocale(code, name string) (*models.Locale, error) {
	var l models.Locale
	if err := database.DB.Where("code = ?", code).First(&l).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrLocaleNotFound
		}
		return nil, err
	}
	if strings.TrimSpace(name) == "" {
		return nil, &ItemValidationError{Err: errors.New("name is required")}
	}
	if err := database.DB.Model(&l).Update("name", name).Error; err != nil {
		return nil, err
	}
	l.Name = name
	return &l, nil
}

// translationTables are the tables whose rows hold translations.
var translationTables = []any{&models.Item{}, &models.SingleItem{}}

// countTranslated counts the entries with content in locale.
func countTranslated(tx *gorm.DB, code string) (int64, error) {
	var total int64
	for _, model := range translationTables {
		var n int64
		q := tx.Model(model).Where(translationExistsSQL(tx.Dialector.Name()), translationPathArgs(tx.Dialector.Name(), code)...)
		if err := q.Count(&n).Error; err != nil {
			return 0, err
		}
		total += n
	}
	return total, nil
}

// translationExistsSQL is a condition true when the row has a translation
// in a locale (bound with translationPathArgs).
func translationExistsSQL(dialect string) string {
	switch dialect {
	case "postgres":
		return "translations IS NOT NULL AND (CAST(translations AS JSONB) -> CAST(? AS TEXT)) IS NOT NULL AND (CAST(translations AS JSONB) -> CAST(? AS TEXT)) <> '{}'::jsonb"
	case "mysql":
		return "translations IS NOT NULL AND JSON_EXTRACT(translations, ?) IS NOT NULL AND JSON_LENGTH(JSON_EXTRACT(translations, ?)) > 0"
	default:
		return "translations IS NOT NULL AND json_extract(translations, ?) IS NOT NULL AND json_extract(translations, ?) <> '{}'"
	}
}

func translationPathArgs(dialect, code string) []any {
	if dialect == "postgres" {
		return []any{code, code}
	}
	p := jsonPath(code)
	return []any{p, p}
}

// SetDefaultLocale makes another locale the default. Entries' main data is
// in the default locale, so this only relabels it: it is refused while any
// entry has a translation in the new default (that content would be
// hidden), and the old default becomes a locale without translations.
func SetDefaultLocale(code string) (*models.Locale, error) {
	var out models.Locale
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("code = ?", code).First(&out).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrLocaleNotFound
			}
			return err
		}
		if out.IsDefault {
			return nil
		}
		n, err := countTranslated(tx, code)
		if err != nil {
			return err
		}
		if n > 0 {
			return &LocaleInUseError{Code: code, Entries: n}
		}
		if err := tx.Model(&models.Locale{}).Where("is_default = ?", true).Update("is_default", false).Error; err != nil {
			return err
		}
		out.IsDefault = true
		return tx.Model(&out).Update("is_default", true).Error
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteLocale removes a locale. Entries with content in it are counted
// and the delete is refused (LocaleInUseError) unless purge is set, which
// deletes that content too.
func DeleteLocale(code string, purge bool) (int64, error) {
	var purged int64
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var l models.Locale
		if err := tx.Where("code = ?", code).First(&l).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrLocaleNotFound
			}
			return err
		}
		if l.IsDefault {
			return ErrDefaultLocale
		}
		n, err := countTranslated(tx, code)
		if err != nil {
			return err
		}
		if n > 0 && !purge {
			return &LocaleInUseError{Code: code, Entries: n}
		}
		// Deleted entries keep translations too: purge them all.
		if err := purgeTranslations(tx, code); err != nil {
			return err
		}
		purged = n
		return tx.Delete(&l).Error
	})
	return purged, err
}

// purgeTranslations removes one locale from every entry's translations.
func purgeTranslations(tx *gorm.DB, code string) error {
	dialect := tx.Dialector.Name()
	var items []models.Item
	if err := tx.Unscoped().Where(translationExistsSQL(dialect), translationPathArgs(dialect, code)...).Find(&items).Error; err != nil {
		return err
	}
	for _, it := range items {
		delete(it.Translations, code)
		if err := tx.Unscoped().Model(&models.Item{}).Where("id = ?", it.ID).Update("translations", it.Translations).Error; err != nil {
			return err
		}
	}
	var singles []models.SingleItem
	if err := tx.Where(translationExistsSQL(dialect), translationPathArgs(dialect, code)...).Find(&singles).Error; err != nil {
		return err
	}
	for _, it := range singles {
		delete(it.Translations, code)
		if err := tx.Model(&models.SingleItem{}).Where("id = ?", it.ID).Update("translations", it.Translations).Error; err != nil {
			return err
		}
	}
	return nil
}

// jsonPath is the SQLite/MySQL JSON path of nested keys, each quoted.
func jsonPath(keys ...string) string {
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`)
	p := "$"
	for _, k := range keys {
		p += `."` + r.Replace(k) + `"`
	}
	return p
}
