package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gohead-cms/gohead/internal/models"
	agentModels "github.com/gohead-cms/gohead/internal/models/agents"
	"github.com/gohead-cms/gohead/pkg/auth"
	"github.com/gohead-cms/gohead/pkg/logger"
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

// revisionScope is the types whose history role may see: those it can
// edit (nil: all of them).
func revisionScope(role string) (*storage.RevisionScope, error) {
	if auth.PermissionsOf(role).Allows(auth.ActionUpdate, auth.AnyType) && auth.IsBuiltinRole(role) {
		return nil, nil
	}
	collections, _, err := storage.GetAllCollections(nil, nil, nil)
	if err != nil {
		return nil, err
	}
	singletons, err := storage.GetAllSingletons()
	if err != nil {
		return nil, err
	}
	scope := &storage.RevisionScope{}
	for _, ct := range collections {
		if auth.Can(role, auth.ActionUpdate, ct.Name) {
			scope.Collections = append(scope.Collections, ct.Name)
		}
	}
	for _, st := range singletons {
		if auth.Can(role, auth.ActionUpdate, auth.SingletonResource(st.Name)) {
			scope.Singletons = append(scope.Singletons, st.ID)
		}
	}
	return scope, nil
}

func canEditContent(c *gin.Context) bool {
	if hasPermission(c.GetString("role"), "update") {
		return true
	}
	c.Set("response", "Access denied")
	c.Set("details", "content history needs a role that can edit content")
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

// pageParams reads page and pageSize (1–100, default 20), answering 400
// for invalid values like the other list endpoints.
func pageParams(c *gin.Context) (int, int, bool) {
	page, size := 1, 20
	if v := c.Query("page"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil || p < 1 {
			validationError(c, "invalid page")
			return 0, 0, false
		}
		page = p
	}
	if v := c.Query("pageSize"); v != "" {
		ps, err := strconv.Atoi(v)
		if err != nil || ps < 1 || ps > 100 {
			validationError(c, "pageSize must be between 1 and 100")
			return 0, 0, false
		}
		size = ps
	}
	return page, size, true
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
	if !allow(c, auth.ActionUpdate, ct.Name) {
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
	if !allow(c, auth.ActionUpdate, auth.SingletonResource(st.Name)) {
		return nil, false
	}
	return &historyTarget{entityType: models.RevisionEntitySingleton, entityID: st.ID, typeName: st.Name, singleton: st}, true
}

// itemTypeName filters item history by collection name; singletons are
// found by ID alone.
func (t *historyTarget) itemTypeName() string {
	if t.collection != nil {
		return t.typeName
	}
	return ""
}

// owns reports whether a revision belongs to this entry's type.
func (t *historyTarget) owns(rev *models.Revision) bool {
	return t.collection == nil || rev.TypeName == t.typeName
}

// passwordFields lists the type's password attributes, which revision
// responses mask.
func (t *historyTarget) passwordFields() map[string]bool {
	attrs := []models.Attribute{}
	if t.collection != nil {
		attrs = t.collection.Attributes
	} else if t.singleton != nil {
		attrs = t.singleton.Attributes
	}
	out := map[string]bool{}
	for _, a := range attrs {
		if a.Type == "password" {
			out[a.Name] = true
		}
	}
	return out
}

const maskedValue = "********"

func maskValue(v any) any {
	if v == nil {
		return nil
	}
	return maskedValue
}

func redactTranslations(translations models.JSONMap, hidden map[string]bool) models.JSONMap {
	out := models.JSONMap{}
	for code, v := range translations {
		if m, ok := v.(map[string]any); ok && len(m) > 0 {
			out[code] = redact(m, hidden)
		}
	}
	return out
}

func redact(data models.JSONMap, hidden map[string]bool) models.JSONMap {
	out := make(models.JSONMap, len(data))
	for k, v := range data {
		if hidden[k] {
			v = maskValue(v)
		}
		out[k] = v
	}
	return out
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
		page, size, ok := pageParams(c)
		if !ok {
			return
		}
		revs, total, err := storage.ListRevisions(storage.RevisionQuery{
			EntityType: t.entityType, EntityID: t.entityID, TypeName: t.itemTypeName(), Page: page, PageSize: size,
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
		if err != nil || !t.owns(rev) {
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

		hidden := t.passwordFields()
		out := revisionJSON(*rev, false)
		out["data"] = redact(rev.Data, hidden)
		if tr := redactTranslations(rev.Translations, hidden); len(tr) > 0 {
			out["translations"] = tr
		}
		var baseData models.JSONMap
		if base != nil {
			out["comparedTo"] = base.Version
			baseData = storage.FlattenEntry(base.Data, base.Translations)
		} else {
			out["comparedTo"] = nil
		}
		// Changes read from the compared version to this one. Values in
		// other locales are named "<attribute>@<locale>".
		changes := storage.DiffData(baseData, storage.FlattenEntry(rev.Data, rev.Translations))
		for i := range changes {
			name, _, _ := strings.Cut(changes[i].Field, "@")
			if hidden[name] {
				changes[i].Before, changes[i].After = maskValue(changes[i].Before), maskValue(changes[i].After)
			}
		}
		out["changes"] = changes
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
				def, _ := storage.DefaultLocale()
				current = models.LocalizedResponse(models.ItemResponse(*t.collection, *item, nil), t.collection.Attributes, item.Translations, "", def)
			}
		} else {
			var item *models.SingleItem
			item, res, err = storage.RestoreSingletonRevision(actor, t.singleton, version)
			if err == nil {
				def, _ := storage.DefaultLocale()
				current = models.LocalizedResponse(models.SingleItemResponse(*item, nil), t.singleton.Attributes, item.Translations, "", def)
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
				logger.Log.WithError(err).WithField("type", t.typeName).WithField("version", version).Error("Restore failed")
				c.Set("response", "Failed to restore the revision")
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
			"unchanged": res.Unchanged,
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
		// By ID: a singleton's history follows it when it is renamed.
		st, err := storage.GetSingletonByName(c.Query("singleton"))
		if err != nil {
			c.Set("response", "Singleton not found")
			c.Set("status", http.StatusNotFound)
			return
		}
		q.EntityType, q.EntityID = models.RevisionEntitySingleton, st.ID
	}
	scope, err := revisionScope(c.GetString("role"))
	if err != nil {
		c.Set("response", "Failed to load the history")
		c.Set("status", http.StatusInternalServerError)
		return
	}
	q.Scope = scope
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
		models.RevisionUnpublish, models.RevisionDelete, models.RevisionRestore, models.RevisionBaseline:
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
	var ok bool
	if q.Page, q.PageSize, ok = pageParams(c); !ok {
		return
	}
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
	run, err := storage.GetAgentRun(agent.ID, id)
	if err != nil {
		if errors.Is(err, storage.ErrRunNotFound) {
			c.Set("response", "Run not found")
			c.Set("status", http.StatusNotFound)
			return
		}
		c.Set("response", "Failed to load run")
		c.Set("status", http.StatusInternalServerError)
		return
	}
	if run.Status == agentModels.RunRunning {
		c.Set("response", "the run is still running; revert it once it has finished")
		c.Set("status", http.StatusConflict)
		return
	}
	outcomes, err := storage.RevertAgentRun(actorFrom(c), id, c.Query("force") == "true")
	if err != nil {
		if errors.Is(err, storage.ErrRunHasNoChanges) {
			c.Set("response", err.Error())
			c.Set("status", http.StatusConflict)
			return
		}
		logger.Log.WithError(err).WithField("run_id", id).Error("Revert failed")
		c.Set("response", "Failed to revert the run")
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
