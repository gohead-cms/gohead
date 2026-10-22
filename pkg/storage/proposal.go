package storage

import (
	"errors"
	"fmt"
	"time"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/database"
	"github.com/gohead-cms/gohead/pkg/logger"

	"gorm.io/gorm"
)

var (
	// ErrProposalNotFound is returned for an unknown proposal.
	ErrProposalNotFound = errors.New("proposal not found")
	// ErrProposalDecided is returned when approving or rejecting a proposal
	// that is no longer pending.
	ErrProposalDecided = errors.New("this proposal was already decided")
	// ErrProposalStale is returned when approving a proposal whose item
	// changed since the agent proposed it (approve with force to apply it
	// anyway).
	ErrProposalStale = errors.New("the item changed since the agent proposed this")
)

// ProposeItemWrite validates a change an agent wants to make and stores it
// as a pending proposal instead of applying it. action is
// models.ProposalCreate, ProposalUpdate or ProposalDelete; locale (update
// only) targets a translation.
func ProposeItemWrite(actor Actor, ct models.Collection, action string, itemID uint, locale string, data models.JSONMap, reason string) (*models.Proposal, error) {
	if err := CheckItemWrite(ct, action, itemID, locale, data); err != nil {
		return nil, err
	}
	p := &models.Proposal{
		AgentName: actor.Name, Action: action, Collection: ct.Name, Locale: locale,
		Data: copyMap(data), Reason: reason, Status: models.ProposalPending, CreatedAt: models.APITime(time.Now()),
	}
	if actor.AgentRunID != 0 {
		id := actor.AgentRunID
		p.AgentRunID = &id
	}
	if itemID != 0 {
		id := itemID
		p.ItemID = &id
		if rev, err := LatestRevision(models.RevisionEntityItem, itemID); err == nil {
			p.BaseVersion = rev.Version
		}
	}
	if err := database.DB.Create(p).Error; err != nil {
		return nil, err
	}
	return p, nil
}

// errRollback undoes a validation transaction.
var errRollback = errors.New("rollback")

// CheckItemWrite validates a change without applying it: the same checks
// as the real write, in a transaction that is rolled back (nested items a
// relation would create are not kept). It returns ErrItemNotFound or an
// *ItemValidationError.
func CheckItemWrite(ct models.Collection, action string, itemID uint, locale string, data models.JSONMap) error {
	// Locales are resolved first: their lookup does not use the
	// transaction (and would wait for it on single-connection databases).
	var code, def string
	if locale != "" && action == models.ProposalUpdate {
		var err error
		if code, def, err = ResolveLocale(locale); err != nil {
			return &ItemValidationError{Err: fmt.Errorf("unknown locale '%s'", locale)}
		}
	}
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		switch action {
		case models.ProposalCreate:
			if _, err := models.ValidateItemValues(ct, data, tx); err != nil {
				return &ItemValidationError{Err: err}
			}
		case models.ProposalUpdate, models.ProposalDelete:
			var item models.Item
			if err := tx.Where("id = ? AND collection_id = ?", itemID, ct.ID).First(&item).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return ErrItemNotFound
				}
				return err
			}
			if action == models.ProposalDelete {
				break
			}
			if locale != "" {
				if code != def {
					values, err := checkTranslationKeys(ct.Attributes, data, def)
					if err != nil {
						return &ItemValidationError{Err: err}
					}
					translations := withTranslation(nil, code, values)
					if _, err := validateItemTranslations(tx, ct, item.ID, translations, &models.NestedCreated{}); err != nil {
						return &ItemValidationError{Err: err}
					}
					break
				}
			}
			if _, err := models.ValidateItemUpdate(ct, data, tx, item.ID); err != nil {
				return &ItemValidationError{Err: err}
			}
		default:
			return fmt.Errorf("unknown action '%s'", action)
		}
		return errRollback
	})
	if errors.Is(err, errRollback) {
		return nil
	}
	return err
}

// ProposalQuery filters proposals.
type ProposalQuery struct {
	Status     string // "" or "all": every status
	Collection string // one collection
	// Scope, when not nil, limits each action (create, update, delete) to
	// the collections listed for it: a reviewer's permissions.
	Scope      map[string][]string
	AgentName  string
	AgentRunID uint
	// NoCalls leaves out calls to connected apps (reviewed by admins only).
	NoCalls  bool
	Page     int
	PageSize int
}

// ListProposals lists proposals, newest first.
func ListProposals(q ProposalQuery) ([]models.Proposal, int, error) {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 || q.PageSize > 100 {
		q.PageSize = 20
	}
	db := database.DB.Model(&models.Proposal{})
	if q.Status != "" && q.Status != "all" {
		db = db.Where("status = ?", q.Status)
	}
	if q.Collection != "" {
		db = db.Where("collection = ?", q.Collection)
	}
	if q.Scope != nil {
		cond := database.DB.Where("1 = 0")
		for action, cols := range q.Scope {
			if len(cols) > 0 {
				cond = cond.Or("action = ? AND collection IN ?", action, cols)
			}
		}
		db = db.Where(cond)
	}
	if q.NoCalls {
		db = db.Where("action <> ?", models.ProposalCall)
	}
	if q.AgentName != "" {
		db = db.Where("agent_name = ?", q.AgentName)
	}
	if q.AgentRunID != 0 {
		db = db.Where("agent_run_id = ?", q.AgentRunID)
	}
	var total int64
	if err := db.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var out []models.Proposal
	if err := db.Session(&gorm.Session{}).Order("id DESC").Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Find(&out).Error; err != nil {
		return nil, 0, err
	}
	return out, int(total), nil
}

// GetProposal returns one proposal.
func GetProposal(id uint) (*models.Proposal, error) {
	var p models.Proposal
	if err := database.DB.First(&p, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProposalNotFound
		}
		return nil, err
	}
	return &p, nil
}

// proposalBase returns the item's values (or the proposal's locale's)
// as they were when the agent proposed the change; false when that version
// is no longer kept.
func proposalBase(p *models.Proposal) (models.JSONMap, bool) {
	version := p.BaseVersion
	if version == 0 {
		version = 1 // the baseline written before the first change
	}
	rev, err := GetRevision(models.RevisionEntityItem, *p.ItemID, version)
	if err != nil || (p.BaseVersion == 0 && rev.Action != models.RevisionBaseline) {
		return nil, false
	}
	if p.Locale != "" {
		m, _ := rev.Translations[p.Locale].(map[string]any)
		return models.JSONMap(m), true
	}
	return rev.Data, true
}

// ProposalState is a proposal against the item as it is now.
type ProposalState struct {
	// Stale: the item changed since the agent proposed the change.
	Stale bool
	// Current holds the item's values now (or the locale's), nil for a
	// creation or a deleted item.
	Current models.JSONMap
	// Proposed holds the values approving would write: for a stale update,
	// the item as it is now with the attributes the agent changed.
	Proposed models.JSONMap
	// Version is the item's latest version, which the change expects.
	Version uint
	item    *models.Item
}

// ProposalNow compares a pending proposal with its item as it is now.
func ProposalNow(p *models.Proposal) (*ProposalState, error) {
	st := &ProposalState{Proposed: p.Data}
	if p.Action == models.ProposalCreate || p.ItemID == nil {
		return st, nil
	}
	if p.Action == models.ProposalDelete {
		st.Proposed = nil
	}
	var item models.Item
	if err := database.DB.First(&item, *p.ItemID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			st.Stale = true
			return st, nil
		}
		return nil, err
	}
	st.item = &item
	st.Current = item.Data
	if p.Locale != "" {
		m, _ := item.Translations[p.Locale].(map[string]any)
		st.Current = models.JSONMap(m)
	}
	if rev, err := LatestRevision(models.RevisionEntityItem, item.ID); err == nil {
		st.Version = rev.Version
	} else if !errors.Is(err, ErrRevisionNotFound) {
		return nil, err
	}
	st.Stale = st.Version != p.BaseVersion
	if !st.Stale || p.Action != models.ProposalUpdate {
		return st, nil
	}
	// Rebase: keep what changed since, apply what the agent changed.
	base, ok := proposalBase(p)
	if !ok {
		return st, nil
	}
	proposed := copyMap(st.Current)
	for k := range keysOf(base, p.Data) {
		after, inProposal := p.Data[k]
		if sameData(models.JSONMap{"v": after}, models.JSONMap{"v": base[k]}) {
			continue
		}
		if inProposal {
			proposed[k] = after
		} else {
			delete(proposed, k)
		}
	}
	st.Proposed = proposed
	return st, nil
}

func keysOf(maps ...models.JSONMap) map[string]bool {
	out := map[string]bool{}
	for _, m := range maps {
		for k := range m {
			out[k] = true
		}
	}
	return out
}

// ProposalIsStale reports whether the proposal's item changed since it was
// proposed.
func ProposalIsStale(p *models.Proposal) bool {
	st, err := ProposalNow(p)
	return err == nil && st.Stale
}

// ApproveProposal applies a pending proposal. Without data, the agent's
// change is applied as the agent, in its run, so its history and a run
// revert include it; with data (values a reviewer edited), the reviewer
// makes the change. A stale proposal is refused unless force is set; then
// only the attributes the agent changed are applied over the item as it is
// now. A failed change marks the proposal failed and returns the error.
func ApproveProposal(id uint, reviewer Actor, data models.JSONMap, force bool) (p *models.Proposal, item *models.Item, err error) {
	p, err = claimProposal(id)
	if err != nil {
		return p, nil, err
	}
	done := false
	defer func() {
		if !done {
			releaseProposal(id) // a panic or an unexpected error
			p.Status = models.ProposalPending
		}
	}()
	fail := func(msg string, cause error) (*models.Proposal, *models.Item, error) {
		done = true
		if ferr := finishProposal(p, models.ProposalFailed, reviewer.Name, "", msg); ferr != nil {
			return p, nil, ferr
		}
		return p, nil, cause
	}
	ct, err := GetCollectionByName(p.Collection)
	if err != nil {
		return fail("the collection no longer exists", ErrProposalUnappliable)
	}
	st, err := ProposalNow(p)
	if err != nil {
		return p, nil, err
	}
	if st.Stale && !force {
		return p, nil, ErrProposalStale
	}
	if p.Action != models.ProposalCreate {
		if st.item == nil {
			return fail("the item no longer exists", ErrItemNotFound)
		}
		if st.item.CollectionID != ct.ID {
			return fail("the item is no longer in this collection", ErrProposalUnappliable)
		}
	}
	var def string
	if p.Locale != "" {
		if def, err = DefaultLocale(); err != nil {
			return p, nil, err
		}
		if p.Locale == def {
			return fail("the locale is now the default locale", ErrProposalUnappliable)
		}
		if !LocaleExists(p.Locale) {
			return fail(fmt.Sprintf("locale '%s' no longer exists", p.Locale), ErrProposalUnappliable)
		}
	}

	actor := Actor{Kind: models.ActorAgent, Name: p.AgentName}
	if p.AgentRunID != nil {
		actor.AgentRunID = *p.AgentRunID
	}
	if data == nil {
		data = st.Proposed
	} else {
		actor = reviewer
	}
	p.Data = data
	// Nobody may change the item between this check and the change.
	actor.expectVersion, actor.expectSet = st.Version, p.Action != models.ProposalCreate

	switch p.Action {
	case models.ProposalCreate:
		item, err = SaveItem(actor, *ct, data)
	case models.ProposalDelete:
		err = DeleteItem(actor, st.item.ID)
	default:
		if p.Locale != "" {
			item, err = SetItemTranslation(actor, *ct, *p.ItemID, p.Locale, data, nil)
		} else {
			item, err = UpdateItemContent(actor, *ct, *p.ItemID, data, nil)
		}
	}
	if errors.Is(err, errChangedSince) {
		return p, nil, ErrProposalStale
	}
	if err != nil {
		var verr *ItemValidationError
		switch {
		case errors.As(err, &verr):
			return fail(verr.Error(), err)
		case errors.Is(err, ErrItemNotFound):
			return fail("the item no longer exists", err)
		case errors.Is(err, ErrLocaleNotFound):
			return fail(err.Error(), ErrProposalUnappliable)
		}
		return fail("could not apply the change", err)
	}
	if item != nil && p.ItemID == nil {
		id := item.ID
		p.ItemID = &id
	}
	done = true
	// The change is made: record it, retrying a failed write.
	for i := 0; i < 3; i++ {
		if err = finishProposal(p, models.ProposalApproved, reviewer.Name, "", ""); err == nil {
			break
		}
	}
	return p, item, err
}

// ErrProposalUnappliable is returned when a proposal can no longer be
// applied (its collection, item or locale changed); the proposal is
// marked failed.
var ErrProposalUnappliable = errors.New("the proposal can no longer be applied")

// RejectProposal discards a pending proposal.
func RejectProposal(id uint, reviewer, note string) (*models.Proposal, error) {
	p, err := claimProposal(id)
	if err != nil {
		return p, err
	}
	return p, finishProposal(p, models.ProposalRejected, reviewer, note, "")
}

// ProposalApplying marks a proposal being decided, so two reviewers
// cannot apply it twice. A claim older than claimTimeout (a crashed
// server) can be taken over.
const ProposalApplying = "applying"

const claimTimeout = 2 * time.Minute

// claimProposal moves a pending proposal to "applying" atomically.
func claimProposal(id uint) (*models.Proposal, error) {
	now := models.APITime(time.Now())
	res := database.DB.Model(&models.Proposal{}).
		Where("id = ? AND (status = ? OR (status = ? AND decided_at < ?))", id, models.ProposalPending, ProposalApplying, now.Add(-claimTimeout)).
		Updates(map[string]any{"status": ProposalApplying, "decided_at": now})
	if res.Error != nil {
		return nil, res.Error
	}
	p, err := GetProposal(id)
	if err != nil {
		return nil, err
	}
	if res.RowsAffected == 0 {
		return p, ErrProposalDecided
	}
	return p, nil
}

func releaseProposal(id uint) {
	if err := database.DB.Model(&models.Proposal{}).Where("id = ? AND status = ?", id, ProposalApplying).
		Updates(map[string]any{"status": models.ProposalPending, "decided_at": nil}).Error; err != nil {
		logger.Log.WithError(err).WithField("proposal", id).Error("Failed to release a proposal")
	}
}

func finishProposal(p *models.Proposal, status, reviewer, note, errText string) error {
	now := models.APITime(time.Now())
	p.Status, p.ReviewedBy, p.ReviewNote, p.Error, p.DecidedAt = status, reviewer, note, errText, &now
	return database.DB.Model(&models.Proposal{}).Where("id = ?", p.ID).Updates(map[string]any{
		"status": status, "reviewed_by": reviewer, "review_note": note, "error": errText,
		"decided_at": now, "data": p.Data, "item_id": p.ItemID,
	}).Error
}
