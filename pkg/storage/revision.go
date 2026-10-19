package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"time"

	"github.com/gohead-cms/gohead/internal/agent/events"
	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/database"
	"github.com/gohead-cms/gohead/pkg/logger"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Actor is who makes a content change. It is recorded on each revision.
type Actor struct {
	Kind       string // models.ActorUser, ActorToken, ActorAgent or ActorSystem
	Name       string // username, token name or agent name
	AgentRunID uint   // the agent run, for agents

	// expectVersion, when set, makes the change fail with errChangedSince
	// unless the entry's latest version is still this one (checked under
	// the row lock, in the change's transaction).
	expectVersion uint
}

// errChangedSince: the entry changed between a check and the change.
var errChangedSince = errors.New("changed in the meantime")

// SystemActor is used for changes nobody in particular made (imports,
// tests, internal jobs).
var SystemActor = Actor{Kind: models.ActorSystem, Name: "system"}

type actorKey struct{}

// WithActor returns a context carrying the actor, for code that writes
// content on someone's behalf (agent functions).
func WithActor(ctx context.Context, a Actor) context.Context {
	return context.WithValue(ctx, actorKey{}, a)
}

// ActorFrom returns the context's actor, or SystemActor.
func ActorFrom(ctx context.Context) Actor {
	if a, ok := ctx.Value(actorKey{}).(Actor); ok && a.Kind != "" {
		return a
	}
	return SystemActor
}

// RevisionLimit is how many revisions are kept per entry; older ones are
// pruned when a new one is recorded. 0 keeps every revision.
var RevisionLimit = 100

var (
	// ErrRevisionNotFound is returned for an unknown revision.
	ErrRevisionNotFound = errors.New("revision not found")
	// ErrRunHasNoChanges is returned when reverting a run that changed no content.
	ErrRunHasNoChanges = errors.New("this run made no content changes")
)

// revisionInput is one revision to record.
type revisionInput struct {
	entityType   string
	entityID     uint
	typeName     string
	action       string
	status       string
	data         models.JSONMap
	translations models.JSONMap
	restoredFrom *uint
	// before is the entry's state before the change. When the entry has no
	// revision yet (it predates history), it is recorded first as a
	// "baseline", so the change can be undone.
	before *entryState
}

// entryState is an entry's content and status at one point.
type entryState struct {
	data         models.JSONMap
	translations models.JSONMap
	status       string
}

// recordRevision stores the next revision of an entry inside tx and prunes
// the oldest ones beyond RevisionLimit. Callers lock the entry's row first
// (see lockRow), so two writers of one entry cannot pick the same version;
// the unique index on (entity, version) catches it if they do.
func recordRevision(tx *gorm.DB, actor Actor, in revisionInput) (*models.Revision, error) {
	var last uint
	if err := tx.Model(&models.Revision{}).
		Where("entity_type = ? AND entity_id = ?", in.entityType, in.entityID).
		Select("COALESCE(MAX(version), 0)").Scan(&last).Error; err != nil {
		return nil, fmt.Errorf("failed to read the last revision: %w", err)
	}
	if actor.expectVersion != 0 && last != actor.expectVersion {
		return nil, errChangedSince
	}
	if actor.Kind == "" {
		actor = SystemActor
	}
	if last == 0 && in.before != nil {
		base := &models.Revision{
			EntityType: in.entityType, EntityID: in.entityID, Version: 1, TypeName: in.typeName,
			Action: models.RevisionBaseline, Status: in.before.status, Data: copyMap(in.before.data),
			Translations: copyTranslations(in.before.translations),
			ActorKind:    SystemActor.Kind, ActorName: SystemActor.Name, CreatedAt: models.APITime(time.Now()),
		}
		if err := tx.Create(base).Error; err != nil {
			return nil, fmt.Errorf("failed to record the baseline revision: %w", err)
		}
		last = 1
	}
	rev := &models.Revision{
		EntityType:   in.entityType,
		EntityID:     in.entityID,
		Version:      last + 1,
		TypeName:     in.typeName,
		Action:       in.action,
		Status:       in.status,
		Data:         copyMap(in.data),
		Translations: copyTranslations(in.translations),
		ActorKind:    actor.Kind,
		ActorName:    actor.Name,
		RestoredFrom: in.restoredFrom,
		CreatedAt:    models.APITime(time.Now()),
	}
	if actor.AgentRunID != 0 {
		id := actor.AgentRunID
		rev.AgentRunID = &id
	}
	if err := tx.Create(rev).Error; err != nil {
		return nil, fmt.Errorf("failed to record the revision: %w", err)
	}
	if RevisionLimit > 0 && rev.Version > uint(RevisionLimit) {
		if err := tx.Where("entity_type = ? AND entity_id = ? AND version <= ?",
			in.entityType, in.entityID, rev.Version-uint(RevisionLimit)).
			Delete(&models.Revision{}).Error; err != nil {
			return nil, fmt.Errorf("failed to prune old revisions: %w", err)
		}
	}
	return rev, nil
}

// recordNested records the first revision of the items validation created
// for relations given as objects (see models.WithNestedCollector).
func recordNested(tx *gorm.DB, actor Actor, nested *models.NestedCreated) error {
	actor.expectVersion = 0 // new entries
	for _, n := range nested.Items {
		if _, err := recordRevision(tx, actor, revisionInput{
			entityType: models.RevisionEntityItem, entityID: n.Item.ID, typeName: n.Collection.Name,
			action: models.RevisionCreate, status: itemStatus(n.Collection, n.Item), data: n.Item.Data,
		}); err != nil {
			return err
		}
	}
	return nil
}

// publishNested sends item:created for nested items, after the commit.
func publishNested(nested *models.NestedCreated) {
	for _, n := range nested.Items {
		publishEvent(database.DB, events.EventTypeItemCreated, n.Item)
	}
}

// lockRow adds SELECT … FOR UPDATE where the database supports it. SQLite
// has a single writer, so it needs none.
func lockRow(tx *gorm.DB) *gorm.DB {
	if tx.Dialector.Name() == "sqlite" {
		return tx
	}
	return tx.Clauses(clause.Locking{Strength: "UPDATE"})
}

// copyTranslations copies translations, nil when there are none.
func copyTranslations(t models.JSONMap) models.JSONMap {
	if len(models.TranslatedLocales(t)) == 0 {
		return nil
	}
	out := models.JSONMap{}
	for code, v := range t {
		if m, ok := v.(map[string]any); ok && len(m) > 0 {
			out[code] = copyMap(m)
		}
	}
	return out
}

func copyMap(m models.JSONMap) models.JSONMap {
	out := make(models.JSONMap, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// sameData reports whether two data maps hold the same JSON.
func sameData(a, b models.JSONMap) bool {
	ja, errA := json.Marshal(a)
	jb, errB := json.Marshal(b)
	return errA == nil && errB == nil && string(ja) == string(jb)
}

func itemStatus(ct models.Collection, item *models.Item) string {
	if !ct.DraftAndPublish {
		return ""
	}
	return item.Status()
}

// RevisionQuery filters a revision list.
type RevisionQuery struct {
	EntityType string
	EntityID   uint
	TypeName   string
	Action     string
	ActorKind  string
	ActorName  string
	AgentRunID uint
	Page       int
	PageSize   int
}

// ListRevisions lists revisions, newest first, without their data.
func ListRevisions(q RevisionQuery) ([]models.Revision, int, error) {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 || q.PageSize > 100 {
		q.PageSize = 20
	}
	db := database.DB.Model(&models.Revision{})
	if q.EntityType != "" {
		db = db.Where("entity_type = ?", q.EntityType)
	}
	if q.EntityID != 0 {
		db = db.Where("entity_id = ?", q.EntityID)
	}
	if q.TypeName != "" {
		db = db.Where("type_name = ?", q.TypeName)
	}
	if q.Action != "" {
		db = db.Where("action = ?", q.Action)
	}
	if q.ActorKind != "" {
		db = db.Where("actor_kind = ?", q.ActorKind)
	}
	if q.ActorName != "" {
		db = db.Where("actor_name = ?", q.ActorName)
	}
	if q.AgentRunID != 0 {
		db = db.Where("agent_run_id = ?", q.AgentRunID)
	}
	var total int64
	if err := db.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count revisions: %w", err)
	}
	var out []models.Revision
	if err := db.Session(&gorm.Session{}).Omit("data").Order("id DESC").
		Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Find(&out).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list revisions: %w", err)
	}
	return out, int(total), nil
}

// GetRevision returns one revision of an entry, with its data.
func GetRevision(entityType string, entityID, version uint) (*models.Revision, error) {
	var rev models.Revision
	err := database.DB.Where("entity_type = ? AND entity_id = ? AND version = ?", entityType, entityID, version).First(&rev).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrRevisionNotFound
	}
	if err != nil {
		return nil, err
	}
	return &rev, nil
}

// LatestRevision returns the newest revision of an entry.
func LatestRevision(entityType string, entityID uint) (*models.Revision, error) {
	var rev models.Revision
	err := database.DB.Where("entity_type = ? AND entity_id = ?", entityType, entityID).Order("version DESC").First(&rev).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrRevisionNotFound
	}
	if err != nil {
		return nil, err
	}
	return &rev, nil
}

// FieldChange is one attribute that differs between two versions.
type FieldChange struct {
	Field  string `json:"field"`
	Change string `json:"change"` // added, removed or changed
	Before any    `json:"before"`
	After  any    `json:"after"`
}

// DiffData compares two data maps attribute by attribute, sorted by name.
func DiffData(before, after models.JSONMap) []FieldChange {
	keys := map[string]bool{}
	for k := range before {
		keys[k] = true
	}
	for k := range after {
		keys[k] = true
	}
	names := make([]string, 0, len(keys))
	for k := range keys {
		names = append(names, k)
	}
	sort.Strings(names)
	changes := []FieldChange{}
	for _, k := range names {
		b, inBefore := before[k]
		a, inAfter := after[k]
		switch {
		case !inBefore:
			changes = append(changes, FieldChange{Field: k, Change: "added", After: a})
		case !inAfter:
			changes = append(changes, FieldChange{Field: k, Change: "removed", Before: b})
		case !sameJSON(a, b):
			changes = append(changes, FieldChange{Field: k, Change: "changed", Before: b, After: a})
		}
	}
	return changes
}

func sameJSON(a, b any) bool {
	ja, errA := json.Marshal(a)
	jb, errB := json.Marshal(b)
	if errA != nil || errB != nil {
		return reflect.DeepEqual(a, b)
	}
	return string(ja) == string(jb)
}

// restorableData keeps the attributes the type still has, and returns the
// names of the ones it dropped.
func restorableData(attrs []models.Attribute, data models.JSONMap) (models.JSONMap, []string) {
	known := map[string]bool{}
	for _, a := range attrs {
		known[a.Name] = true
	}
	out := models.JSONMap{}
	var dropped []string
	for k, v := range data {
		if known[k] {
			out[k] = v
		} else {
			dropped = append(dropped, k)
		}
	}
	sort.Strings(dropped)
	return out, dropped
}

// RestoreResult describes a restore.
type RestoreResult struct {
	Revision *models.Revision // the new revision
	// Dropped lists attributes of the old version the type no longer has.
	Dropped []string
	// Undeleted is true when the restore brought back a deleted entry.
	Undeleted bool
	// Unchanged is true when the entry already had that content: no
	// revision was recorded, and Revision is the latest one.
	Unchanged bool
}

// latestRevisionTx is LatestRevision inside a transaction.
func latestRevisionTx(tx *gorm.DB, entityType string, entityID uint) (*models.Revision, error) {
	var rev models.Revision
	if err := tx.Where("entity_type = ? AND entity_id = ?", entityType, entityID).Order("version DESC").First(&rev).Error; err != nil {
		return nil, err
	}
	return &rev, nil
}

// RestoreItemRevision makes a version of an item its current content, as a
// new revision. A deleted item is brought back with the publication state
// the restored version had; otherwise the publication state is left as it
// is. The data is validated against the current schema.
func RestoreItemRevision(actor Actor, ct models.Collection, itemID, version uint) (*models.Item, *RestoreResult, error) {
	rev, err := GetRevision(models.RevisionEntityItem, itemID, version)
	if err != nil {
		return nil, nil, err
	}
	if rev.TypeName != ct.Name {
		return nil, nil, ErrRevisionNotFound
	}
	data, dropped := restorableData(ct.Attributes, rev.Data)
	translations := restorableTranslations(ct.Attributes, rev.Translations)
	res := &RestoreResult{Dropped: dropped}
	nested := &models.NestedCreated{}
	var item models.Item
	txErr := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := lockRow(tx.Unscoped()).Where("id = ? AND collection_id = ?", itemID, ct.ID).First(&item).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrItemNotFound
			}
			return err
		}
		processed, err := models.ValidateItemUpdate(ct, data, models.WithNestedCollector(tx, nested), item.ID)
		if err != nil {
			return &ItemValidationError{Err: fmt.Errorf("version %d no longer fits the collection: %w", version, err)}
		}
		if !item.DeletedAt.Valid && len(nested.Items) == 0 && sameData(item.Data, processed) &&
			sameData(copyTranslations(item.Translations), copyTranslations(translations)) {
			res.Unchanged = true
			res.Revision, err = latestRevisionTx(tx, models.RevisionEntityItem, item.ID)
			return err
		}
		if err := recordNested(tx, actor, nested); err != nil {
			return err
		}
		updates := map[string]any{"data": processed, "translations": translations}
		if item.DeletedAt.Valid {
			res.Undeleted = true
			updates["deleted_at"] = nil
			updates["published_at"] = nil
			if ct.DraftAndPublish && rev.Status == models.StatusPublished {
				updates["published_at"] = models.APITime(time.Now())
			}
		}
		if err := tx.Unscoped().Model(&item).Updates(updates).Error; err != nil {
			return err
		}
		if err := reloadItem(tx, &item); err != nil {
			return err
		}
		from := version
		res.Revision, err = recordRevision(tx, actor, revisionInput{
			entityType: models.RevisionEntityItem, entityID: item.ID, typeName: ct.Name,
			action: models.RevisionRestore, status: itemStatus(ct, &item), data: item.Data, translations: item.Translations, restoredFrom: &from,
		})
		return err
	})
	if txErr != nil {
		return nil, nil, txErr
	}
	publishNested(nested)
	switch {
	case res.Unchanged:
	case res.Undeleted:
		publishEvent(database.DB, events.EventTypeItemCreated, &item)
		if item.PublishedAt != nil {
			publishEvent(database.DB, events.EventTypeItemPublished, &item)
		}
	default:
		publishEvent(database.DB, events.EventTypeItemUpdated, &item)
	}
	return &item, res, nil
}

// RestoreSingletonRevision makes a version of a singleton's content its
// current content, as a new revision (re-creating deleted content).
func RestoreSingletonRevision(actor Actor, st *models.Singleton, version uint) (*models.SingleItem, *RestoreResult, error) {
	rev, err := GetRevision(models.RevisionEntitySingleton, st.ID, version)
	if err != nil {
		return nil, nil, err
	}
	data, dropped := restorableData(st.Attributes, rev.Data)
	if err := models.ValidateSingleItemValues(*st, data); err != nil {
		return nil, nil, &ItemValidationError{Err: fmt.Errorf("version %d no longer fits the singleton: %w", version, err)}
	}
	translations := restorableTranslations(st.Attributes, rev.Translations)
	res := &RestoreResult{Dropped: dropped}
	var item models.SingleItem
	txErr := database.DB.Transaction(func(tx *gorm.DB) error {
		err := lockRow(tx).Where("single_type_id = ?", st.ID).First(&item).Error
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			res.Undeleted = true
			item = models.SingleItem{SingleTypeID: st.ID, Data: data, Translations: translations}
			if err := tx.Create(&item).Error; err != nil {
				return err
			}
		case err != nil:
			return err
		default:
			if sameData(item.Data, data) && sameData(copyTranslations(item.Translations), copyTranslations(translations)) {
				res.Unchanged = true
				res.Revision, err = latestRevisionTx(tx, models.RevisionEntitySingleton, st.ID)
				return err
			}
			if err := tx.Model(&item).Updates(map[string]any{"data": data, "translations": translations}).Error; err != nil {
				return err
			}
			item.Data, item.Translations = data, translations
		}
		from := version
		res.Revision, err = recordRevision(tx, actor, revisionInput{
			entityType: models.RevisionEntitySingleton, entityID: st.ID, typeName: st.Name,
			action: models.RevisionRestore, data: data, translations: translations, restoredFrom: &from,
		})
		return err
	})
	if txErr != nil {
		return nil, nil, txErr
	}
	return &item, res, nil
}

// RevertOutcome is what reverting a run did to one entry.
type RevertOutcome struct {
	EntityType string `json:"entityType"`
	EntityID   uint   `json:"entityId"`
	TypeName   string `json:"typeName"`
	// Result: reverted (restored the version before the run), deleted
	// (the run created the entry), skipped or failed.
	Result string `json:"result"`
	// Version restored, for "reverted".
	Version uint   `json:"version,omitempty"`
	Reason  string `json:"reason,omitempty"`
}

// RevertAgentRun undoes the content changes of an agent run, entry by
// entry: entries the run created are deleted, the others go back to the
// version they had before the run. An entry changed by someone else after
// the run is skipped unless force is set.
func RevertAgentRun(actor Actor, runID uint, force bool) ([]RevertOutcome, error) {
	var revs []models.Revision
	if err := database.DB.Where("agent_run_id = ?", runID).Order("id ASC").Find(&revs).Error; err != nil {
		return nil, err
	}
	if len(revs) == 0 {
		return nil, ErrRunHasNoChanges
	}
	type entry struct {
		entityType string
		entityID   uint
		typeName   string
		first      models.Revision // the run's first revision of the entry
		last       models.Revision // the run's last revision of the entry
	}
	var order []string
	entries := map[string]*entry{}
	for _, r := range revs {
		key := fmt.Sprintf("%s/%d", r.EntityType, r.EntityID)
		e, ok := entries[key]
		if !ok {
			e = &entry{entityType: r.EntityType, entityID: r.EntityID, typeName: r.TypeName, first: r}
			entries[key] = e
			order = append(order, key)
		}
		e.last = r
	}

	out := make([]RevertOutcome, 0, len(order))
	for _, key := range order {
		e := entries[key]
		o := RevertOutcome{EntityType: e.entityType, EntityID: e.entityID, TypeName: e.typeName}
		latest, err := LatestRevision(e.entityType, e.entityID)
		if err != nil {
			logger.Log.WithError(err).WithField("run_id", runID).Warn("Revert: failed to read the latest revision")
			o.Result, o.Reason = "failed", "could not read the entry's history"
			out = append(out, o)
			continue
		}
		// Without force, the change only applies if nobody changed the
		// entry between this check and the change itself.
		guarded := actor
		if !force {
			guarded.expectVersion = latest.Version
		}
		if latest.ID != e.last.ID && !force {
			o.Result = "skipped"
			o.Reason = fmt.Sprintf("changed after the run (version %d by %s)", latest.Version, describeActor(latest))
			out = append(out, o)
			continue
		}
		created := e.first.Action == models.RevisionCreate
		switch {
		case created && latest.Action == models.RevisionDelete:
			o.Result, o.Reason = "skipped", "already deleted"
		case created:
			err = revertDelete(guarded, e.entityType, e.entityID, e.typeName)
			o.Result = "deleted"
		case e.first.Version <= 1:
			o.Result, o.Reason = "skipped", "no earlier version is kept"
		default:
			o.Version = e.first.Version - 1
			if _, err = GetRevision(e.entityType, e.entityID, o.Version); errors.Is(err, ErrRevisionNotFound) {
				o.Result, o.Reason, o.Version, err = "skipped", "the version before the run was pruned", 0, nil
				break
			}
			if err == nil {
				err = revertRestore(guarded, e.entityType, e.entityID, e.typeName, o.Version)
			}
			o.Result = "reverted"
		}
		var verr *ItemValidationError
		switch {
		case err == nil:
		case errors.Is(err, errChangedSince):
			o.Result, o.Reason, o.Version = "skipped", "changed while reverting", 0
		case errors.As(err, &verr):
			o.Result, o.Reason, o.Version = "failed", verr.Error(), 0
		default:
			logger.Log.WithError(err).WithField("run_id", runID).WithField("entity_id", e.entityID).Warn("Revert failed")
			o.Result, o.Reason, o.Version = "failed", "could not revert (see the server log)", 0
		}
		out = append(out, o)
	}
	return out, nil
}

func describeActor(r *models.Revision) string {
	if r.ActorName == "" {
		return r.ActorKind
	}
	return fmt.Sprintf("%s %s", r.ActorKind, r.ActorName)
}

func revertDelete(actor Actor, entityType string, id uint, typeName string) error {
	if entityType == models.RevisionEntitySingleton {
		return DeleteSingleItem(actor, typeName)
	}
	return DeleteItem(actor, id)
}

func revertRestore(actor Actor, entityType string, id uint, typeName string, version uint) error {
	if entityType == models.RevisionEntitySingleton {
		st, err := GetSingletonByName(typeName)
		if err != nil {
			return err
		}
		_, _, err = RestoreSingletonRevision(actor, st, version)
		return err
	}
	ct, err := GetCollectionByName(typeName)
	if err != nil {
		return err
	}
	_, _, err = RestoreItemRevision(actor, *ct, id, version)
	return err
}
