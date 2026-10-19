package models

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"sort"
	"time"
)

// Locale is a content locale. Exactly one is the default: entries' main
// data is in it, and other locales are stored as translations.
type Locale struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Code      string    `json:"code" gorm:"size:35;not null;uniqueIndex"`
	Name      string    `json:"name" gorm:"size:191"`
	IsDefault bool      `json:"isDefault" gorm:"not null;default:false"`
	CreatedAt time.Time `json:"createdAt"`
}

// localeCodeRe accepts BCP 47 style codes: "en", "fr", "pt-BR", "zh-Hant".
var localeCodeRe = regexp.MustCompile(`^[a-z]{2,3}(-[A-Za-z0-9]{2,8})*$`)

// ValidLocaleCode reports whether code looks like a BCP 47 language tag.
func ValidLocaleCode(code string) bool {
	return len(code) <= 35 && localeCodeRe.MatchString(code)
}

// localizableTypes are the attribute types that can be localized: every
// type except relations (links are shared by all locales).
func localizable(attr Attribute) bool { return attr.Type != "relation" }

// validateLocalized checks one attribute's localized setting and the names
// it reserves.
func validateLocalized(all []Attribute, attr Attribute) error {
	if attr.Localized && !localizable(attr) {
		return fmt.Errorf("attribute '%s': relations cannot be localized (they are shared by all locales)", attr.Name)
	}
	if HasLocalized(all) && slices.Contains(ReservedLocaleAttributeNames, attr.Name) {
		return fmt.Errorf("attribute name '%s' is reserved in types with localized attributes", attr.Name)
	}
	return nil
}

// HasLocalized reports whether any attribute is localized.
func HasLocalized(attrs []Attribute) bool {
	return slices.ContainsFunc(attrs, func(a Attribute) bool { return a.Localized })
}

// LocalizedNames returns the names of the localized attributes.
func LocalizedNames(attrs []Attribute) map[string]bool {
	out := map[string]bool{}
	for _, a := range attrs {
		if a.Localized {
			out[a.Name] = true
		}
	}
	return out
}

// Translation returns the stored values of one locale (nil if none).
func Translation(translations JSONMap, locale string) map[string]any {
	m, _ := translations[locale].(map[string]any)
	return m
}

// TranslatedLocales lists the locales with at least one value, sorted.
func TranslatedLocales(translations JSONMap) []string {
	var out []string
	for code, v := range translations {
		if m, ok := v.(map[string]any); ok && len(m) > 0 {
			out = append(out, code)
		}
	}
	sort.Strings(out)
	return out
}

// TranslatedLocalesOf lists the locales with a value for at least one of
// the localized attributes (values kept for attributes that are no longer
// localized do not count), sorted.
func TranslatedLocalesOf(attrs []Attribute, translations JSONMap) []string {
	localized := LocalizedNames(attrs)
	var out []string
	for code, v := range translations {
		m, _ := v.(map[string]any)
		for k := range m {
			if localized[k] {
				out = append(out, code)
				break
			}
		}
	}
	sort.Strings(out)
	return out
}

// LocalizeData returns data as seen in locale: localized attributes take
// their value in that locale; with fallback, a missing value falls back to
// the default locale's, otherwise it is left out. The default locale (or
// "") returns data unchanged.
func LocalizeData(attrs []Attribute, data, translations JSONMap, locale, defaultLocale string, fallback bool) JSONMap {
	if locale == "" || locale == defaultLocale {
		return data
	}
	out := make(JSONMap, len(data))
	maps.Copy(out, data)
	tr := Translation(translations, locale)
	for _, a := range attrs {
		if !a.Localized {
			continue
		}
		if v, ok := tr[a.Name]; ok {
			out[a.Name] = v
		} else if !fallback {
			delete(out, a.Name)
		}
	}
	return out
}

// LocalizedResponse adds "locale" and "locales" to an entry's response when
// its type has localized attributes.
func LocalizedResponse(out map[string]any, attrs []Attribute, translations JSONMap, locale, defaultLocale string) map[string]any {
	if !HasLocalized(attrs) {
		return out
	}
	if locale == "" {
		locale = defaultLocale
	}
	locales := []string{defaultLocale}
	for _, code := range TranslatedLocalesOf(attrs, translations) {
		if code != defaultLocale {
			locales = append(locales, code)
		}
	}
	out["locale"] = locale
	out["locales"] = locales
	return out
}
