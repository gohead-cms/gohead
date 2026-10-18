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
	"github.com/gohead-cms/gohead/pkg/database"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/storage"
	"github.com/gohead-cms/gohead/pkg/utils"
	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
)

// CreateItem handles nested creations
func CreateItem(collection models.Collection) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !hasPermission(c.GetString("role"), "create") {
			c.Set("response", "Access denied")
			c.Set("status", http.StatusForbidden)
			return
		}

		var input struct {
			Data models.JSONMap `json:"data"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			c.Set("response", "Invalid request body")
			c.Set("status", http.StatusBadRequest)
			return
		}

		newItem, err := storage.SaveItem(collection, input.Data)
		if err != nil {
			c.Set("response", err.Error())
			c.Set("status", http.StatusBadRequest) // Validation errors are Bad Request
			return
		}

		// Format the response
		newItem.Data["id"] = newItem.ID
		c.Set("response", gin.H{
			"id":         newItem.ID,
			"attributes": newItem.Data,
		})
		c.Set("status", http.StatusCreated)
	}
}

// GetItems lists a collection's items with pagination, filtering and sorting.
//
// Query parameters:
//
//	page, pageSize                 pagination (pageSize max 100)
//	filters[<field>][<op>]=<value> op: $eq $ne $lt $lte $gt $gte $contains
//	filters[<field>]=<value>       shorthand for $eq
//	sort=<field>[:asc|:desc],...   multiple keys allowed
//
// <field> is a collection attribute or one of id, created_at, updated_at.
func GetItems(collection models.Collection, level uint) gin.HandlerFunc {
	return func(c *gin.Context) {
		q, err := parseItemQuery(c.Request.URL.Query())
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"status": http.StatusBadRequest, "name": "ValidationError", "message": err.Error(), "details": nil}})
			return
		}

		items, total, err := storage.QueryItems(collection, q)
		if err != nil {
			var qe *storage.QueryFieldError
			if errors.As(err, &qe) {
				c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"status": http.StatusBadRequest, "name": "ValidationError", "message": qe.Error(), "details": nil}})
				return
			}
			logger.Log.WithError(err).Error("Failed to fetch items")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch items"})
			return
		}

		results := make([]models.JSONMap, 0, len(items))
		for _, item := range items {
			item.Data["id"] = item.ID
			hydratedData, err := storage.FetchNestedRelations(collection, item.Data, level)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to populate relations"})
				return
			}
			results = append(results, hydratedData)
		}

		c.JSON(http.StatusOK, gin.H{
			"data": results,
			"meta": gin.H{
				"pagination": gin.H{
					"page":      q.Page,
					"pageSize":  q.PageSize,
					"total":     total,
					"pageCount": (total + q.PageSize - 1) / q.PageSize, // Ceiling division
				},
			},
		})
	}
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
func GetItemByID(ct models.Collection, id uint, level uint) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.Set("response", "Invalid item ID format")
			c.Set("status", http.StatusBadRequest)
			return
		}

		item, err := storage.GetItemByID(uint(ct.ID), uint(id))
		if err != nil {
			c.Set("response", "Item not found")
			c.Set("status", http.StatusNotFound)
			return
		}

		data, err := storage.FetchNestedRelations(ct, item.Data, level)
		if err != nil {
			c.Set("response", "Failed to fetch item relations")
			c.Set("status", http.StatusInternalServerError)
			return
		}

		c.Set("response", utils.FormatNestedItems(item.ID, data, &ct))
		c.Set("status", http.StatusOK)
		c.Set("meta", gin.H{})
	}
}

// UpdateItem now handles nested creations and uses the c.Set response pattern.
func UpdateItem(collection models.Collection) gin.HandlerFunc {
	return func(c *gin.Context) {
		itemID, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.Set("response", "Invalid ID format")
			c.Set("status", http.StatusBadRequest)
			return
		}

		var input struct {
			Data models.JSONMap `json:"data"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			c.Set("response", "Invalid request body")
			c.Set("status", http.StatusBadRequest)
			return
		}

		var updatedItem models.Item
		txErr := database.DB.Transaction(func(tx *gorm.DB) error {
			processedData, err := models.ValidateItemValues(collection, input.Data, tx)
			if err != nil {
				return err
			}
			var itemToUpdate models.Item
			if err := tx.Where("id = ? AND collection_id = ?", itemID, collection.ID).First(&itemToUpdate).Error; err != nil {
				return gorm.ErrRecordNotFound
			}
			itemToUpdate.Data = processedData
			if err := tx.Save(&itemToUpdate).Error; err != nil {
				return err
			}
			updatedItem = itemToUpdate
			return nil
		})

		if txErr != nil {
			if txErr == gorm.ErrRecordNotFound {
				c.Set("response", "Item not found")
				c.Set("status", http.StatusNotFound)
				return
			}
			c.Set("response", txErr.Error())
			c.Set("status", http.StatusBadRequest)
			return
		}

		level, _ := strconv.Atoi(c.DefaultQuery("level", "2"))
		hydratedData, err := storage.FetchNestedRelations(collection, updatedItem.Data, uint(level))
		if err != nil {
			c.Set("response", "Failed to populate relations for response")
			c.Set("status", http.StatusInternalServerError)
			return
		}

		c.Set("response", gin.H{
			"id":         updatedItem.ID,
			"attributes": hydratedData,
		})
		c.Set("status", http.StatusOK)
	}
}

func DeleteItem(ct models.Collection) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.Set("response", "Invalid ID")
			c.Set("details", err.Error())
			c.Set("status", http.StatusBadRequest)
			return
		}

		// Make sure the item belongs to this collection!
		_, err = storage.GetItemByID(ct.ID, uint(id))
		if err != nil {
			c.Set("response", "Item not found in this collection")
			c.Set("details", err.Error())
			c.Set("status", http.StatusNotFound)
			return
		}

		// Now it is safe to delete!
		if err := storage.DeleteItem(uint(id)); err != nil {
			c.Set("response", "Failed to delete item")
			c.Set("details", err.Error())
			c.Set("status", http.StatusInternalServerError)
			return
		}

		logger.Log.WithField("iem of ", ct.Name).Info("Collection deleted successfully")
		c.Set("response", nil)
		c.Set("meta", gin.H{})
		c.Set("status", http.StatusOK)
	}
}
