package storage

import (
	"errors"
	"fmt"
	"time"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/database"

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
				code, def, err := ResolveLocale(locale)
				if err != nil {
					return &ItemValidationError{Err: fmt.Errorf("unknown locale '%s'", locale)}
				}
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
	Status      string   // "" or "all": every status
	Collection  string   // one collection
	Collections []string // nil: all; otherwise only these (a role's scope)
	AgentName   string
	AgentRunID  uint
	Page        int
	PageSize    int
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
	if q.Collections != nil {
		cols := q.Collections
		if len(cols) == 0 {
			cols = []string{""}
		}
		db = db.Where("collection IN ?", cols)
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

// ProposalIsStale reports whether the proposal's item changed since it was
// proposed.
func ProposalIsStale(p *models.Proposal) bool {
	if p.ItemID == nil || p.Action == models.ProposalCreate {
		return false
	}
	rev, err := LatestRevision(models.RevisionEntityItem, *p.ItemID)
	if err != nil {
		return p.BaseVersion != 0
	}
	return rev.Version != p.BaseVersion
}

// ApproveProposal applies a pending proposal, with data (when not nil)
// replacing what the agent proposed. The change is made as the agent, in
// its run, so its history and a run revert include it; the proposal
// records who approved it. A failed change marks the proposal failed and
// returns the error.
func ApproveProposal(id uint, reviewer string, data models.JSONMap, force bool) (*models.Proposal, *models.Item, error) {
	p, err := claimProposal(id)
	if err != nil {
		return nil, nil, err
	}
	if !force && ProposalIsStale(p) {
		releaseProposal(p.ID)
		p.Status = models.ProposalPending
		return p, nil, ErrProposalStale
	}
	ct, err := GetCollectionByName(p.Collection)
	if err != nil {
		return p, nil, finishProposal(p, models.ProposalFailed, reviewer, "", "the collection no longer exists")
	}
	if data == nil {
		data = p.Data
	} else {
		p.Data = data
	}
	actor := Actor{Kind: models.ActorAgent, Name: p.AgentName}
	if p.AgentRunID != nil {
		actor.AgentRunID = *p.AgentRunID
	}
	var item *models.Item
	switch p.Action {
	case models.ProposalCreate:
		item, err = SaveItem(actor, *ct, data)
	case models.ProposalDelete:
		err = DeleteItem(actor, *p.ItemID)
	default:
		def, _ := DefaultLocale()
		if p.Locale != "" && p.Locale != def {
			item, err = SetItemTranslation(actor, *ct, *p.ItemID, p.Locale, data, nil)
		} else {
			item, err = UpdateItemContent(actor, *ct, *p.ItemID, data, nil)
		}
	}
	if err != nil {
		msg := "could not apply the change"
		var verr *ItemValidationError
		switch {
		case errors.As(err, &verr):
			msg = verr.Error()
		case errors.Is(err, ErrItemNotFound):
			msg = "the item no longer exists"
		}
		if ferr := finishProposal(p, models.ProposalFailed, reviewer, "", msg); ferr != nil {
			return p, nil, ferr
		}
		return p, nil, err
	}
	if item != nil && p.ItemID == nil {
		id := item.ID
		p.ItemID = &id
	}
	return p, item, finishProposal(p, models.ProposalApproved, reviewer, "", "")
}

// RejectProposal discards a pending proposal.
func RejectProposal(id uint, reviewer, note string) (*models.Proposal, error) {
	p, err := claimProposal(id)
	if err != nil {
		return nil, err
	}
	return p, finishProposal(p, models.ProposalRejected, reviewer, note, "")
}

// proposalApplying marks a proposal being decided, so two reviewers
// cannot apply it twice.
const proposalApplying = "applying"

// claimProposal moves a pending proposal to "applying" atomically.
func claimProposal(id uint) (*models.Proposal, error) {
	res := database.DB.Model(&models.Proposal{}).Where("id = ? AND status = ?", id, models.ProposalPending).Update("status", proposalApplying)
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
	database.DB.Model(&models.Proposal{}).Where("id = ? AND status = ?", id, proposalApplying).Update("status", models.ProposalPending)
}

func finishProposal(p *models.Proposal, status, reviewer, note, errText string) error {
	now := models.APITime(time.Now())
	p.Status, p.ReviewedBy, p.ReviewNote, p.Error, p.DecidedAt = status, reviewer, note, errText, &now
	return database.DB.Model(&models.Proposal{}).Where("id = ?", p.ID).Updates(map[string]any{
		"status": status, "reviewed_by": reviewer, "review_note": note, "error": errText,
		"decided_at": now, "data": p.Data, "item_id": p.ItemID,
	}).Error
}
