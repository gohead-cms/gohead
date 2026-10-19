package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/storage"

	"github.com/gin-gonic/gin"
)

// Content history: every change of an entry is a revision (see
// storage.Revision). Reading history and restoring need the editor or admin
// role, as revisions hold draft content.

// revisionJSON is a revision in API responses. Lists leave the data out.
func revisionJSON(r models.Revision, withData bool) gin.H {
	out := gin.H{
		"id":         r.ID,
		"entityType": r.EntityType,
		"entityId":   r.EntityID,
		"version":    r.Version,
		"typeName":   r.TypeName,
		"action":     r.Action,
		"actorKind":  r.ActorKind,
		"actorName":  r.ActorName,
		"createdAt":  models.APITime(r.CreatedAt),
	}
	if r.Status != "" {
		out["status"] = r.Status
	}
	if r.AgentRunID != nil {
		out["agentRunId"] = *r.AgentRunID
	}
	if r.RestoredFrom != nil {
		out["restoredFrom"] = *r.RestoredFrom
	}
	if withData {
		out["data"] = r.Data
	}
	return out
}

func canEditContent(c *gin.Context) bool {
	if hasPermission(c.GetString("role"), "update") {
		return true
	}
	c.Set("response", "Access denied")
	c.Set("details", "content history needs the editor or admin role")
	c.Set("status", http.StatusForbidden)
	return false
}

func uintParam(c *gin.Context, name string) (uint, bool) {
	v, err := strconv.ParseUint(c.Param(name), 10, 32)
	if err != nil || v == 0 {
		validationError(c, "invalid "+name)
		return 0, false
	}
	return uint(v), true
}

func pageParams(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	return page, size
}

func setRevisionList(c *gin.Context, revs []models.Revision, total, page, size int) {
	list := make([]gin.H, len(revs))
	for i, r := range revs {
		list[i] = revisionJSON(r, false)
	}
	c.Set("response", list)
	c.Set("meta", gin.H{"pagination": gin.H{
		"page": page, "pageSize": size, "total": total,
		"pageCount": (total + size - 1) / size,
	}})
	c.Set("status", http.StatusOK)
}

// historyTarget is the entry a history route is about.
type historyTarget struct {
	entityType string
	entityID   uint
	typeName   string
	collection *models.Collection
	singleton  *models.Singleton
}

// itemTarget resolves /collections/:collection/:id. The item may have been
// deleted: its history stays readable and restorable.
func itemTarget(c *gin.Context) (*historyTarget, bool) {
	ct, err := storage.GetCollectionByName(c.Param("collection"))
	if err != nil {
		c.Set("response", "Collection not found")
		c.Set("status", http.StatusNotFound)
		return nil, false
	}
	id, ok := uintParam(c, "id")
	if !ok {
		return nil, false
	}
	return &historyTarget{entityType: models.RevisionEntityItem, entityID: id, typeName: ct.Name, collection: ct}, true
}

func singletonTarget(c *gin.Context) (*historyTarget, bool) {
	st, err := storage.GetSingletonByName(c.Param("name"))
	if err != nil {
		c.Set("response", "Singleton not found")
		c.Set("status", http.StatusNotFound)
		return nil, false
	}
	return &historyTarget{entityType: models.RevisionEntitySingleton, entityID: st.ID, typeName: st.Name, singleton: st}, true
}

func listHistory(resolve func(*gin.Context) (*historyTarget, bool)) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !canEditContent(c) {
			return
		}
		t, ok := resolve(c)
		if !ok {
			return
		}
		page, size := pageParams(c)
		revs, total, err := storage.ListRevisions(storage.RevisionQuery{
			EntityType: t.entityType, EntityID: t.entityID, TypeName: t.typeName, Page: page, PageSize: size,
		})
		if err != nil {
			c.Set("response", "Failed to load the history")
			c.Set("status", http.StatusInternalServerError)
			return
		}
		if total == 0 && t.collection != nil {
			if _, err := storage.GetItemByID(t.collection.ID, t.entityID); err != nil {
				c.Set("response", "Item not found in this collection")
				c.Set("status", http.StatusNotFound)
				return
			}
		}
		setRevisionList(c, revs, total, page, size)
	}
}

// getHistoryRevision returns one revision with its data and the changes
// from another version: by default the previous one; ?compare=<version>
// or ?compare=current (the entry's latest revision).
func getHistoryRevision(resolve func(*gin.Context) (*historyTarget, bool)) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !canEditContent(c) {
			return
		}
		t, ok := resolve(c)
		if !ok {
			return
		}
		version, ok := uintParam(c, "version")
		if !ok {
			return
		}
		rev, err := storage.GetRevision(t.entityType, t.entityID, version)
		if err != nil || rev.TypeName != t.typeName {
			revisionLookupError(c, err)
			return
		}

		var base *models.Revision
		switch cmp := c.Query("compare"); cmp {
		case "":
			if version > 1 {
				base, err = storage.GetRevision(t.entityType, t.entityID, version-1)
				if errors.Is(err, storage.ErrRevisionNotFound) {
					base, err = nil, nil // pruned
				}
			}
		case "current":
			base, err = storage.LatestRevision(t.entityType, t.entityID)
		default:
			n, perr := strconv.ParseUint(cmp, 10, 32)
			if perr != nil || n == 0 {
				validationError(c, "compare must be a version number or current")
				return
			}
			base, err = storage.GetRevision(t.entityType, t.entityID, uint(n))
		}
		if err != nil {
			revisionLookupError(c, err)
			return
		}

		out := revisionJSON(*rev, true)
		var baseData models.JSONMap
		if base != nil {
			out["comparedTo"] = base.Version
			baseData = base.Data
		} else {
			out["comparedTo"] = nil
		}
		// Changes read from the compared version to this one.
		out["changes"] = storage.DiffData(baseData, rev.Data)
		c.Set("response", out)
		c.Set("status", http.StatusOK)
	}
}

func revisionLookupError(c *gin.Context, err error) {
	if err == nil || errors.Is(err, storage.ErrRevisionNotFound) {
		c.Set("response", "Revision not found")
		c.Set("status", http.StatusNotFound)
		return
	}
	c.Set("response", "Failed to load the revision")
	c.Set("status", http.StatusInternalServerError)
}

// restoreHistoryRevision makes a version the entry's current content.
func restoreHistoryRevision(resolve func(*gin.Context) (*historyTarget, bool)) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !canEditContent(c) {
			return
		}
		t, ok := resolve(c)
		if !ok {
			return
		}
		version, ok := uintParam(c, "version")
		if !ok {
			return
		}
		actor := actorFrom(c)
		var current map[string]any
		var res *storage.RestoreResult
		var err error
		if t.collection != nil {
			var item *models.Item
			item, res, err = storage.RestoreItemRevision(actor, *t.collection, t.entityID, version)
			if err == nil {
				current = models.ItemResponse(*t.collection, *item, nil)
			}
		} else {
			var item *models.SingleItem
			item, res, err = storage.RestoreSingletonRevision(actor, t.singleton, version)
			if err == nil {
				current = models.SingleItemResponse(*item, nil)
			}
		}
		if err != nil {
			var verr *storage.ItemValidationError
			switch {
			case errors.As(err, &verr):
				c.Set("response", verr.Error())
				c.Set("status", http.StatusBadRequest)
			case errors.Is(err, storage.ErrRevisionNotFound):
				revisionLookupError(c, err)
			case errors.Is(err, storage.ErrItemNotFound):
				c.Set("response", "Item not found in this collection")
				c.Set("status", http.StatusNotFound)
			default:
				c.Set("response", "Failed to restore the revision")
				c.Set("details", err.Error())
				c.Set("status", http.StatusInternalServerError)
			}
			return
		}
		dropped := res.Dropped
		if dropped == nil {
			dropped = []string{}
		}
		c.Set("response", current)
		c.Set("meta", gin.H{
			"revision":  revisionJSON(*res.Revision, false),
			"dropped":   dropped,
			"undeleted": res.Undeleted,
		})
		c.Set("status", http.StatusOK)
	}
}

// Item and singleton history routes.
var (
	ListItemRevisions        = listHistory(itemTarget)
	GetItemRevision          = getHistoryRevision(itemTarget)
	RestoreItemRevision      = restoreHistoryRevision(itemTarget)
	ListSingletonRevisions   = listHistory(singletonTarget)
	GetSingletonRevision     = getHistoryRevision(singletonTarget)
	RestoreSingletonRevision = restoreHistoryRevision(singletonTarget)
)

// ListRevisions is the change feed across entries, newest first. Filters:
// collection, singleton, action, actorKind, actor, agentRun.
func ListRevisions(c *gin.Context) {
	if !canEditContent(c) {
		return
	}
	q := storage.RevisionQuery{
		Action:    c.Query("action"),
		ActorKind: c.Query("actorKind"),
		ActorName: c.Query("actor"),
	}
	switch {
	case c.Query("collection") != "" && c.Query("singleton") != "":
		validationError(c, "use collection or singleton, not both")
		return
	case c.Query("collection") != "":
		q.EntityType, q.TypeName = models.RevisionEntityItem, c.Query("collection")
	case c.Query("singleton") != "":
		q.EntityType, q.TypeName = models.RevisionEntitySingleton, c.Query("singleton")
	}
	if v := c.Query("agentRun"); v != "" {
		n, err := strconv.ParseUint(v, 10, 32)
		if err != nil || n == 0 {
			validationError(c, "agentRun must be a run ID")
			return
		}
		q.AgentRunID = uint(n)
	}
	switch q.Action {
	case "", models.RevisionCreate, models.RevisionUpdate, models.RevisionPublish,
		models.RevisionUnpublish, models.RevisionDelete, models.RevisionRestore:
	default:
		validationError(c, "unknown action '"+q.Action+"'")
		return
	}
	switch q.ActorKind {
	case "", models.ActorUser, models.ActorToken, models.ActorAgent, models.ActorSystem:
	default:
		validationError(c, "actorKind must be user, token, agent or system")
		return
	}
	q.Page, q.PageSize = pageParams(c)
	revs, total, err := storage.ListRevisions(q)
	if err != nil {
		c.Set("response", "Failed to load the history")
		c.Set("status", http.StatusInternalServerError)
		return
	}
	setRevisionList(c, revs, total, q.Page, q.PageSize)
}

// RevertAgentRun undoes the content changes of one of an agent's runs.
// Entries changed by someone else since are skipped unless ?force=true.
func RevertAgentRun(c *gin.Context) {
	agent, ok := runAgent(c)
	if !ok {
		return
	}
	id, ok := uintParam(c, "run")
	if !ok {
		return
	}
	if _, err := storage.GetAgentRun(agent.ID, id); err != nil {
		if errors.Is(err, storage.ErrRunNotFound) {
			c.Set("response", "Run not found")
			c.Set("status", http.StatusNotFound)
			return
		}
		c.Set("response", "Failed to load run")
		c.Set("status", http.StatusInternalServerError)
		return
	}
	outcomes, err := storage.RevertAgentRun(actorFrom(c), id, c.Query("force") == "true")
	if err != nil {
		if errors.Is(err, storage.ErrRunHasNoChanges) {
			c.Set("response", err.Error())
			c.Set("status", http.StatusConflict)
			return
		}
		c.Set("response", "Failed to revert the run")
		c.Set("details", err.Error())
		c.Set("status", http.StatusInternalServerError)
		return
	}
	counts := map[string]int{}
	for _, o := range outcomes {
		counts[o.Result]++
	}
	c.Set("response", outcomes)
	c.Set("meta", gin.H{"counts": counts})
	c.Set("status", http.StatusOK)
}
