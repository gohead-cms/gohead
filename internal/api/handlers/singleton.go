package handlers

import (
	"errors"
	"net/http"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/auth"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/storage"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// GetSingleItem returns a singleton's content in the flat item shape, with
// relations populated down to "level" (default 1).
func GetSingleItem(c *gin.Context) {
	name := c.Param("name")
	st, err := storage.GetSingletonByName(name)
	if err != nil {
		c.Set("response", "Singleton not found")
		c.Set("status", http.StatusNotFound)
		return
	}
	// Checked against the stored name, which databases may match loosely.
	if !allow(c, auth.ActionRead, auth.SingletonResource(st.Name)) {
		return
	}
	rc, ok := writeContext(c) // level and locale
	if !ok {
		return
	}
	// status=all: related drafts too (editors and admins), so a form can
	// send the content back without losing links to drafts.
	publishedOnly := true
	switch c.Query("status") {
	case "", models.StatusPublished:
	case "all":
		if !auth.CanSeeDrafts(c.GetString("role"), auth.SingletonResource(st.Name)) {
			c.Set("response", "Access denied")
			c.Set("details", "status=all needs a role that can edit this singleton")
			c.Set("status", http.StatusForbidden)
			return
		}
		publishedOnly = false
	default:
		validationError(c, "status must be published or all")
		return
	}
	item, err := storage.GetSingleItemByType(st.Name)
	if err != nil {
		c.Set("response", err.Error())
		c.Set("status", http.StatusNotFound)
		return
	}
	rc.publishedOnly = publishedOnly
	out, err := singleItemResponseFor(st, item, rc)
	if err != nil {
		c.Set("response", "Failed to populate relations")
		c.Set("status", http.StatusInternalServerError)
		return
	}
	c.Set("response", out)
	c.Set("status", http.StatusOK)
}

// singleItemResponseFor is singleItemResponse with the draft filter chosen
// by the caller.
func singleItemResponseFor(st *models.Singleton, item *models.SingleItem, rc readContext) (map[string]any, error) {
	data := rc.localize(st.Attributes, item.Data, item.Translations)
	data, err := storage.PopulateRelationsIn(models.Collection{Attributes: st.Attributes}, data, rc.level, rc.publishedOnly, rc.readOptions())
	if err != nil {
		return nil, err
	}
	out := models.SingleItemResponse(*item, data)
	return models.LocalizedResponse(out, st.Attributes, item.Translations, rc.locale, rc.defaultLocale), nil
}

// GetSingleton retrieves a single type by its name.
func GetSingleton(c *gin.Context) {
	name := c.Param("name")
	logger.Log.WithField("name", name).Debug("Handler.GetSingleton")

	// Retrieve single type from storage
	st, err := storage.GetSingletonByName(name)
	if err == nil && !can(c, auth.ActionRead, auth.SingletonResource(st.Name)) {
		err = storage.ErrItemNotFound
	}
	if err != nil {
		logger.Log.WithField("name", name).Warn("GetSingleton: single type not found")
		c.Set("response", "Single type not found")
		c.Set("status", http.StatusNotFound)
		return
	}

	// Prepare response
	response := map[string]any{
		"name":        st.Name,
		"description": st.Description,
		"attributes":  st.Attributes,
	}

	logger.Log.WithField("name", name).Info("GetSingleton: single type retrieved successfully")
	c.Set("response", response)
	c.Set("status", http.StatusOK)
}

// CreateOrUpdateSingleton handles the creation or updating of a single type.
func CreateOrUpdateSingleton(c *gin.Context) {
	name := c.Param("name")

	// Parse the JSON input into a map
	var input map[string]any
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Set("response", "Invalid JSON input")
		c.Set("status", http.StatusBadRequest)
		return
	}
	logger.Log.WithField("payload", input).Info("CreateOrUpdateSingleton payload")

	// Transform the map into a Singleton struct
	Singleton, err := models.ParseSingletonInput(input)
	if err != nil {
		c.Set("response", err.Error())
		c.Set("status", http.StatusBadRequest)
		return
	}

	// Optionally, ensure the name from the route matches the name field
	// or fallback to the route name if not provided in the input.
	if Singleton.Name == "" {
		Singleton.Name = name
	}

	// Validate the single type
	if err := models.ValidateSingletonSchema(Singleton); err != nil {
		logger.Log.WithError(err).Warn("CreateOrUpdateSingleton: validation failed")
		c.Set("response", "Single type validation failed")
		c.Set("details", err.Error())
		c.Set("status", http.StatusBadRequest)
		return
	}

	// Save or update the Singleton in the database
	if err := storage.SaveOrUpdateSingleton(&Singleton); err != nil {
		logger.Log.WithError(err).Error("CreateOrUpdateSingleton: Failed to save single type")
		c.Set("response", "Failed to save single type")
		c.Set("details", err.Error())
		c.Set("status", http.StatusInternalServerError)
		return
	}

	logger.Log.WithField("Singleton", Singleton.Name).Info("single type created or updated successfully")
	c.Set("response", gin.H{"message": "single type created/updated successfully", "single_type": input})
	c.Set("status", http.StatusCreated)
}

// DeleteSingleton handles deleting a single type by its name (optional).
func DeleteSingleton(c *gin.Context) {
	name := c.Param("name")

	// Fetch the single type by its name
	st, err := storage.GetSingletonByName(name)
	if err != nil {
		logger.Log.WithError(err).Warn("DeleteSingleton: single type not found")
		c.Set("response", "Single type not found")
		c.Set("details", err.Error())
		c.Set("status", http.StatusNotFound)
		return
	}

	// Call the storage function to delete the single type
	if err := storage.DeleteSingleton(st.ID); err != nil {
		logger.Log.WithError(err).Error("DeleteSingleton: Failed to delete single type")
		c.Set("response", "Failed to delete single type")
		c.Set("details", err.Error())
		c.Set("status", http.StatusInternalServerError)
		return
	}

	logger.Log.WithFields(logrus.Fields{
		"Singleton": name,
	}).Info("single type deleted successfully")

	c.Set("response", gin.H{"message": "single type delete successfully", "single_type": name})
	c.Set("status", http.StatusOK)
}

// CreateOrUpdateSingletonItem creates or replaces a singleton's content.
// Requires the update permission.
func CreateOrUpdateSingletonItem(c *gin.Context) {
	if st, err := storage.GetSingletonByName(c.Param("name")); err == nil &&
		!allow(c, auth.ActionUpdate, auth.SingletonResource(st.Name)) {
		return
	}
	rc, ok := writeContext(c)
	if !ok {
		return
	}
	rc.publishedOnly = true
	name := c.Param("name")
	st, err := storage.GetSingletonByName(name)
	if err != nil {
		c.Set("response", "Singleton not found")
		c.Set("status", http.StatusNotFound)
		return
	}

	var input struct {
		Data map[string]any `json:"data"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || input.Data == nil {
		c.Set("response", "Invalid request body. Expecting an object with a 'data' key.")
		c.Set("status", http.StatusBadRequest)
		return
	}
	if rc.translated() {
		putSingletonTranslation(c, st, rc, input.Data)
		return
	}
	if err := models.ValidateSingleItemValues(*st, input.Data); err != nil {
		c.Set("response", err.Error())
		c.Set("status", http.StatusBadRequest)
		return
	}

	var item *models.SingleItem
	status := http.StatusOK
	if _, getErr := storage.GetSingleItemByType(name); getErr == nil {
		item, err = storage.UpdateSingleItem(actorFrom(c), name, input.Data)
	} else {
		item, err = storage.CreateSingleItem(actorFrom(c), st, input.Data)
		status = http.StatusCreated
	}
	if err != nil {
		logger.Log.WithError(err).WithField("singleton", name).Error("Failed to save singleton content")
		c.Set("response", "Failed to save singleton content")
		c.Set("status", http.StatusInternalServerError)
		return
	}

	out, err := singleItemResponseFor(st, item, rc)
	if err != nil {
		c.Set("response", "Failed to populate relations")
		c.Set("status", http.StatusInternalServerError)
		return
	}
	c.Set("response", out)
	c.Set("status", status)
}

// putSingletonTranslation writes a singleton's values in another locale.
func putSingletonTranslation(c *gin.Context, st *models.Singleton, rc readContext, data map[string]any) {
	item, err := storage.SetSingletonTranslation(actorFrom(c), st, rc.locale, data)
	if err != nil {
		var verr *storage.ItemValidationError
		switch {
		case errors.As(err, &verr):
			validationError(c, verr.Error())
		case errors.Is(err, storage.ErrItemNotFound):
			c.Set("response", "This singleton has no content yet: create it in the default locale ("+rc.defaultLocale+") first")
			c.Set("status", http.StatusNotFound)
		default:
			logger.Log.WithError(err).WithField("singleton", st.Name).Error("Failed to save singleton translation")
			c.Set("response", "Failed to save singleton content")
			c.Set("status", http.StatusInternalServerError)
		}
		return
	}
	out, err := singleItemResponseFor(st, item, rc)
	if err != nil {
		c.Set("response", "Failed to populate relations")
		c.Set("status", http.StatusInternalServerError)
		return
	}
	c.Set("response", out)
	c.Set("status", http.StatusOK)
}

// GetSingletons lists all singleton definitions.
func GetSingletons(c *gin.Context) {
	sts, err := storage.GetAllSingletons()
	if err != nil {
		logger.Log.WithError(err).Error("Failed to list singletons")
		c.Set("response", "Failed to fetch singletons")
		c.Set("status", http.StatusInternalServerError)
		return
	}
	visible := sts[:0:0]
	for _, st := range sts {
		if can(c, auth.ActionRead, auth.SingletonResource(st.Name)) {
			visible = append(visible, st)
		}
	}
	c.Set("response", visible)
	c.Set("meta", gin.H{"total": len(visible)})
	c.Set("status", http.StatusOK)
}
