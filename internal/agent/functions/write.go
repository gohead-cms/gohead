package functions

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/storage"
)

// WriteMode is how content functions handle writes in a run.
type WriteMode int

const (
	// WriteLive applies writes.
	WriteLive WriteMode = iota
	// WriteDryRun validates writes and reports what they would do, without
	// writing anything (test runs).
	WriteDryRun
	// WritePropose validates writes and stores them as proposals an editor
	// approves (agents with require_approval).
	WritePropose
)

type writeModeKey struct{}

// WithWriteMode returns a context in which content functions use mode.
func WithWriteMode(ctx context.Context, mode WriteMode) context.Context {
	return context.WithValue(ctx, writeModeKey{}, mode)
}

func writeModeOf(ctx context.Context) WriteMode {
	m, _ := ctx.Value(writeModeKey{}).(WriteMode)
	return m
}

// writeItem applies, simulates or proposes a content change, depending on
// the run's write mode, and returns the result for the model.
func writeItem(ctx context.Context, ct *models.Collection, action string, itemID uint, locale string, data map[string]any, reason string) string {
	actor := storage.ActorFrom(ctx)
	result := map[string]any{"collection": ct.Name}
	if itemID != 0 {
		result["item_id"] = itemID
	}
	if locale != "" {
		result["locale"] = locale
	}
	switch writeModeOf(ctx) {
	case WriteDryRun:
		if err := storage.CheckItemWrite(*ct, action, itemID, locale, data); err != nil {
			return errorResult(err)
		}
		result["status"] = "success"
		result["dry_run"] = true
		result["action"] = "would_" + action
		result["message"] = "Test run: the change is valid but nothing was written. Continue as if it had been applied."
	case WritePropose:
		p, err := storage.ProposeItemWrite(actor, *ct, action, itemID, locale, data, reason)
		if err != nil {
			return errorResult(err)
		}
		result["status"] = "pending_approval"
		result["proposal_id"] = p.ID
		result["action"] = action
		result["message"] = "The change was submitted for approval: an editor will apply or reject it. Do not submit it again."
	default:
		var err error
		switch action {
		case models.ProposalCreate:
			var item *models.Item
			if item, err = storage.SaveItem(actor, *ct, data); err == nil {
				result["item_id"] = item.ID
			}
			result["action"] = "created"
		case models.ProposalDelete:
			err = storage.DeleteItem(actor, itemID)
			result["action"] = "deleted"
		default:
			if locale != "" {
				_, err = storage.SetItemTranslation(actor, *ct, itemID, locale, data, nil)
				result["action"] = "translated"
			} else {
				_, err = storage.UpdateItemContent(actor, *ct, itemID, data, nil)
				result["action"] = "updated"
			}
		}
		if err != nil {
			return errorResult(err)
		}
		result["status"] = "success"
	}
	b, _ := json.Marshal(result)
	return string(b)
}

// errorResult is a function result reporting err to the model.
func errorResult(err error) string {
	msg := err.Error()
	if errors.Is(err, storage.ErrItemNotFound) {
		msg = "item not found in this collection"
	}
	b, _ := json.Marshal(map[string]any{"status": "error", "message": msg})
	return string(b)
}
