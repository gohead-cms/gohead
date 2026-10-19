package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/auth"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/storage"

	"github.com/gin-gonic/gin"
)

// GetItems lists a collection's items with pagination, filtering and sorting.
//
// Query parameters:
//
//	page, pageSize                 pagination (pageSize max 100)
//	filters[<field>][<op>]=<value> op: $eq $ne $lt $lte $gt $gte $contains
//	filters[<field>]=<value>       shorthand for $eq
//	sort=<field>[:asc|:desc],...   multiple keys allowed
//	status=published|draft|all     draft-and-publish collections (default published)
//
// <field> is a collection attribute or one of id, createdAt, updatedAt,
// publishedAt (snake_case also accepted).
func GetItems(collection models.Collection, rc readContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		q, err := parseItemQuery(c.Request.URL.Query())
		if err != nil {
			validationError(c, err.Error())
			return
		}
		q.Status = rc.listStatus
		if rc.translated() {
			q.Locale, q.Fallback = rc.locale, rc.fallback
		}

		items, total, err := storage.QueryItems(collection, q)
		if err != nil {
			var qe *storage.QueryFieldError
			if errors.As(err, &qe) {
				validationError(c, qe.Error())
				return
			}
			logger.Log.WithError(err).Error("Failed to fetch items")
			c.Set("response", "Failed to fetch items")
			c.Set("status", http.StatusInternalServerError)
			return
		}

		results := make([]map[string]any, 0, len(items))
		for _, item := range items {
			out, err := itemResponse(collection, item, rc)
			if err != nil {
				logger.Log.WithError(err).Error("Failed to populate relations")
				c.Set("response", "Failed to populate relations")
				c.Set("status", http.StatusInternalServerError)
				return
			}
			results = append(results, out)
		}

		c.Set("response", results)
		c.Set("meta", gin.H{
			"pagination": gin.H{
				"page":      q.Page,
				"pageSize":  q.PageSize,
				"total":     total,
				"pageCount": (total + q.PageSize - 1) / q.PageSize, // Ceiling division
			},
		})
		c.Set("status", http.StatusOK)
	}
}

// itemResponse builds the flat API representation of an item in the
// request's locale, with its relations populated down to rc.level.
func itemResponse(ct models.Collection, item models.Item, rc readContext) (map[string]any, error) {
	data := rc.localize(ct.Attributes, item.Data, item.Translations)
	data, err := storage.PopulateRelationsIn(ct, data, rc.level, rc.publishedOnly, rc.readOptions())
	if err != nil {
		return nil, err
	}
	out := models.ItemResponse(ct, item, data)
	return models.LocalizedResponse(out, ct.Attributes, item.Translations, rc.locale, rc.defaultLocale), nil
}

func validationError(c *gin.Context, msg string) {
	c.Set("response", msg)
	c.Set("status", http.StatusBadRequest)
}

var filterParamRe = regexp.MustCompile(`^filters\[([^\[\]]+)\](?:\[(\$[a-z]+)\])?$`)

// maxFilters bounds the number of conditions a single request can add.
const maxFilters = 20

func parseItemQuery(values url.Values) (storage.ItemQuery, error) {
	q := storage.ItemQuery{Page: 1, PageSize: 10}
	if v := values.Get("page"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil || p < 1 {
			return q, fmt.Errorf("invalid page '%s'", v)
		}
		q.Page = p
	}
	if v := values.Get("pageSize"); v != "" {
		ps, err := strconv.Atoi(v)
		if err != nil || ps < 1 || ps > 100 {
			return q, fmt.Errorf("pageSize must be between 1 and 100")
		}
		q.PageSize = ps
	}

	for key, vals := range values {
		m := filterParamRe.FindStringSubmatch(key)
		if m == nil {
			continue
		}
		op := m[2]
		if op == "" {
			op = storage.OpEq
		}
		for _, v := range vals {
			q.Filters = append(q.Filters, storage.ItemFilter{Field: m[1], Op: op, Value: v})
		}
	}
	if len(q.Filters) > maxFilters {
		return q, fmt.Errorf("too many filters (max %d)", maxFilters)
	}
	// Map iteration order is random; keep the generated SQL stable.
	sort.Slice(q.Filters, func(i, j int) bool {
		if q.Filters[i].Field != q.Filters[j].Field {
			return q.Filters[i].Field < q.Filters[j].Field
		}
		return q.Filters[i].Op < q.Filters[j].Op
	})

	if v := values.Get("sort"); v != "" {
		for _, part := range strings.Split(v, ",") {
			field, dir, _ := strings.Cut(strings.TrimSpace(part), ":")
			if field == "" {
				return q, fmt.Errorf("invalid sort '%s'", v)
			}
			switch strings.ToLower(dir) {
			case "", "asc":
				q.Sort = append(q.Sort, storage.ItemSort{Field: field})
			case "desc":
				q.Sort = append(q.Sort, storage.ItemSort{Field: field, Desc: true})
			default:
				return q, fmt.Errorf("invalid sort direction '%s'", dir)
			}
		}
	}
	return q, nil
}

// GetItemByID retrieves a single item by ID.
func GetItemByID(ct models.Collection, id uint, rc readContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		item, err := storage.GetVisibleItem(ct, id, rc.publishedOnly)
		if err != nil {
			if errors.Is(err, storage.ErrItemNotFound) {
				c.Set("response", "Item not found")
				c.Set("status", http.StatusNotFound)
				return
			}
			logger.Log.WithError(err).Error("Failed to fetch item")
			c.Set("response", "Failed to fetch item")
			c.Set("status", http.StatusInternalServerError)
			return
		}

		out, err := itemResponse(ct, *item, rc)
		if err != nil {
			c.Set("response", "Failed to fetch item relations")
			c.Set("status", http.StatusInternalServerError)
			return
		}
		c.Set("response", out)
		c.Set("status", http.StatusOK)
	}
}

// itemWriteBody is the body of item create and update requests.
type itemWriteBody struct {
	Data   map[string]any `json:"data"`
	Status *string        `json:"status"`
}

// publishFlag converts the optional "status" of a write request.
func publishFlag(ct models.Collection, status *string) (*bool, error) {
	if status == nil {
		return nil, nil
	}
	if !ct.DraftAndPublish {
		return nil, storage.ErrDraftAndPublishDisabled
	}
	switch *status {
	case models.StatusPublished:
		t := true
		return &t, nil
	case models.StatusDraft:
		f := false
		return &f, nil
	}
	return nil, fmt.Errorf("invalid status '%s' (expected 'draft' or 'published')", *status)
}

// UpdateItem replaces an item's data. The body's optional "status" publishes
// or unpublishes it in draft-and-publish collections.
func UpdateItem(collection models.Collection) gin.HandlerFunc {
	return func(c *gin.Context) {
		itemID, err := strconv.Atoi(c.Param("id"))
		if err != nil || itemID < 1 {
			validationError(c, "Invalid ID format")
			return
		}
		rc, ok := writeContext(c)
		if !ok {
			return
		}

		var input itemWriteBody
		if err := c.ShouldBindJSON(&input); err != nil || input.Data == nil {
			validationError(c, "Invalid request body. Expecting an object with a 'data' key.")
			return
		}
		publish, err := publishFlag(collection, input.Status)
		if err != nil {
			validationError(c, err.Error())
			return
		}
		if publish != nil && !allow(c, auth.ActionPublish, collection.Name) {
			return
		}
		if !allowNested(c, collection, input.Data) {
			return
		}

		var item *models.Item
		if rc.translated() {
			// Another locale: replaces that locale's localized values.
			item, err = storage.SetItemTranslation(actorFrom(c), collection, uint(itemID), rc.locale, input.Data, publish)
		} else {
			item, err = storage.UpdateItemContent(actorFrom(c), collection, uint(itemID), input.Data, publish)
		}
		if err != nil {
			writeItemError(c, collection, err)
			return
		}

		out, err := itemResponse(collection, *item, rc)
		if err != nil {
			c.Set("response", "Failed to populate relations for response")
			c.Set("status", http.StatusInternalServerError)
			return
		}
		c.Set("response", out)
		c.Set("status", http.StatusOK)
	}
}

// PublishItem publishes (publish=true) or unpublishes an item of a
// draft-and-publish collection: POST /api/collections/:collection/:id/publish
// and /unpublish. Requires the update permission.
func PublishItem(publish bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		ct, err := storage.GetCollectionByName(c.Param("collection"))
		if err != nil {
			c.Set("response", "Collection not found")
			c.Set("status", http.StatusNotFound)
			return
		}
		if !allow(c, auth.ActionPublish, ct.Name) {
			return
		}
		itemID, err := strconv.Atoi(c.Param("id"))
		if err != nil || itemID < 1 {
			validationError(c, "Invalid ID format")
			return
		}
		rc, ok := writeContext(c)
		if !ok {
			return
		}
		item, err := storage.SetItemPublished(actorFrom(c), *ct, uint(itemID), publish)
		if err != nil {
			writeItemError(c, *ct, err)
			return
		}
		out, err := itemResponse(*ct, *item, rc)
		if err != nil {
			c.Set("response", "Failed to populate relations for response")
			c.Set("status", http.StatusInternalServerError)
			return
		}
		c.Set("response", out)
		c.Set("status", http.StatusOK)
	}
}

// writeItemError maps storage write errors to HTTP responses.
func writeItemError(c *gin.Context, ct models.Collection, err error) {
	var verr *storage.ItemValidationError
	switch {
	case errors.Is(err, storage.ErrItemNotFound):
		c.Set("response", "Item not found")
		c.Set("status", http.StatusNotFound)
	case errors.Is(err, storage.ErrLocaleNotFound):
		validationError(c, "unknown locale (see GET /api/locales)")
	case errors.Is(err, storage.ErrDefaultLocaleWrite):
		validationError(c, err.Error())
	case errors.Is(err, storage.ErrDraftAndPublishDisabled):
		validationError(c, fmt.Sprintf("collection '%s' does not use draft and publish", ct.Name))
	case errors.As(err, &verr):
		validationError(c, verr.Error())
	default:
		logger.Log.WithError(err).WithField("collection", ct.Name).Error("Failed to write item")
		c.Set("response", "Failed to save item")
		c.Set("status", http.StatusInternalServerError)
	}
}

func DeleteItem(ct models.Collection) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil || id < 1 {
			validationError(c, "Invalid ID")
			return
		}

		// Make sure the item belongs to this collection.
		if _, err := storage.GetItemByID(ct.ID, uint(id)); err != nil {
			c.Set("response", "Item not found in this collection")
			c.Set("status", http.StatusNotFound)
			return
		}

		if err := storage.DeleteItem(actorFrom(c), uint(id)); err != nil {
			c.Set("response", "Failed to delete item")
			c.Set("details", err.Error())
			c.Set("status", http.StatusInternalServerError)
			return
		}

		logger.Log.WithField("collection", ct.Name).WithField("item_id", id).Info("Item deleted")
		c.Set("response", nil)
		c.Set("meta", gin.H{})
		c.Set("status", http.StatusOK)
	}
}
