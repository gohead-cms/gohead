package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/storage"

	"github.com/gin-gonic/gin"
)

// localeContext is the locale a request reads or writes.
type localeContext struct {
	locale        string // resolved; equals defaultLocale when not given
	defaultLocale string
	fallback      bool // missing localized values fall back to the default locale
}

// translated reports whether the request targets a non-default locale.
func (l localeContext) translated() bool { return l.locale != l.defaultLocale }

// parseLocale reads ?locale= (a configured locale; default: the default
// locale) and ?fallback= (default true). It answers 400 and returns false
// for invalid values.
func parseLocale(c *gin.Context) (localeContext, bool) {
	lc := localeContext{fallback: true}
	switch c.Query("fallback") {
	case "", "true":
	case "false":
		lc.fallback = false
	default:
		validationError(c, "fallback must be true or false")
		return lc, false
	}
	locale, def, err := storage.ResolveLocale(c.Query("locale"))
	if err != nil {
		if errors.Is(err, storage.ErrLocaleNotFound) {
			validationError(c, "unknown locale '"+c.Query("locale")+"' (see GET /api/locales)")
			return lc, false
		}
		logger.Log.WithError(err).Error("Failed to resolve the locale")
		c.Set("response", "Failed to resolve the locale")
		c.Set("status", http.StatusInternalServerError)
		return lc, false
	}
	lc.locale, lc.defaultLocale = locale, def
	return lc, true
}

// readLocale is the locale related entries are populated in.
func (l localeContext) readLocale() storage.ReadLocale {
	return storage.ReadLocale{Locale: l.locale, Default: l.defaultLocale, Fallback: l.fallback}
}

// localize returns an entry's data in the request's locale.
func (l localeContext) localize(attrs []models.Attribute, data, translations models.JSONMap) models.JSONMap {
	return models.LocalizeData(attrs, data, translations, l.locale, l.defaultLocale, l.fallback)
}

// ListLocales lists the content locales, the default first. Any signed-in
// user can read them.
func ListLocales(c *gin.Context) {
	locales, err := storage.ListLocales()
	if err != nil {
		c.Set("response", "Failed to list locales")
		c.Set("status", http.StatusInternalServerError)
		return
	}
	c.Set("response", locales)
	c.Set("status", http.StatusOK)
}

type localeBody struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// CreateLocale adds a locale (admin).
func CreateLocale(c *gin.Context) {
	var body localeBody
	if err := c.ShouldBindJSON(&body); err != nil {
		validationError(c, "expected {\"code\": \"fr\", \"name\": \"Français\"}")
		return
	}
	l, err := storage.CreateLocale(body.Code, body.Name)
	if err != nil {
		localeError(c, err)
		return
	}
	c.Set("response", l)
	c.Set("status", http.StatusCreated)
}

// UpdateLocale renames a locale, and makes it the default with
// {"isDefault": true} (admin).
func UpdateLocale(c *gin.Context) {
	var body struct {
		Name      *string `json:"name"`
		IsDefault *bool   `json:"isDefault"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		validationError(c, "expected {\"name\": …} and/or {\"isDefault\": true}")
		return
	}
	code := c.Param("code")
	if body.IsDefault != nil && !*body.IsDefault {
		validationError(c, "make another locale the default instead")
		return
	}
	if body.Name != nil && strings.TrimSpace(*body.Name) == "" {
		validationError(c, "name is required")
		return
	}
	var l *models.Locale
	var err error
	// The default first: it is the change that can be refused.
	if body.IsDefault != nil {
		if l, err = storage.SetDefaultLocale(code); err != nil {
			localeError(c, err)
			return
		}
	}
	if body.Name != nil {
		if l, err = storage.RenameLocale(code, *body.Name); err != nil {
			localeError(c, err)
			return
		}
	}
	if l == nil {
		validationError(c, "nothing to change")
		return
	}
	c.Set("response", l)
	c.Set("status", http.StatusOK)
}

// DeleteLocale removes a locale (admin). Content in it is deleted with
// ?purge=true; without it the request fails with 409 when there is some.
func DeleteLocale(c *gin.Context) {
	purged, err := storage.DeleteLocale(c.Param("code"), c.Query("purge") == "true")
	if err != nil {
		localeError(c, err)
		return
	}
	c.Set("response", nil)
	c.Set("meta", gin.H{"purgedEntries": purged})
	c.Set("status", http.StatusOK)
}

func localeError(c *gin.Context, err error) {
	var verr *storage.ItemValidationError
	var inUse *storage.LocaleInUseError
	switch {
	case errors.As(err, &verr):
		validationError(c, verr.Error())
	case errors.Is(err, storage.ErrLocaleNotFound):
		c.Set("response", "Locale not found")
		c.Set("status", http.StatusNotFound)
	case errors.Is(err, storage.ErrLocaleExists):
		c.Set("response", "This locale already exists")
		c.Set("status", http.StatusConflict)
	case errors.Is(err, storage.ErrDefaultLocale):
		c.Set("response", "The default locale cannot be deleted: make another locale the default first")
		c.Set("status", http.StatusConflict)
	case errors.As(err, &inUse):
		c.Set("response", inUse.Error())
		c.Set("details", gin.H{"entries": inUse.Entries})
		c.Set("status", http.StatusConflict)
	default:
		logger.Log.WithError(err).Error("Locale operation failed")
		c.Set("response", "Failed to change locales")
		c.Set("status", http.StatusInternalServerError)
	}
}
