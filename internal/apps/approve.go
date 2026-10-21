package apps

import (
	"context"
	"errors"
	"strings"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/storage"
)

// ProposedCall is the call a proposal asks for.
func ProposedCall(p *models.Proposal) (connection, tool string, args map[string]any) {
	connection = strings.TrimPrefix(p.Collection, models.ConnectionProposalPrefix)
	tool, _ = p.Data["tool"].(string)
	args, _ = p.Data["arguments"].(map[string]any)
	if args == nil {
		if m, ok := p.Data["arguments"].(models.JSONMap); ok {
			args = m
		}
	}
	return connection, tool, args
}

// ErrCallFailed is returned when an approved call failed; the proposal is
// marked failed with the error.
var ErrCallFailed = errors.New("the call failed")

// ApproveCall makes the call a proposal asks for, with arguments edited by
// the reviewer if given, and records the result in the proposal.
func ApproveCall(ctx context.Context, id uint, reviewer string, args map[string]any) (*models.Proposal, Result, error) {
	p, err := storage.ClaimProposal(id)
	if err != nil {
		return p, Result{}, err
	}
	if p.Action != models.ProposalCall {
		storage.ReleaseProposal(id)
		p.Status = models.ProposalPending
		return p, Result{}, errors.New("not a call proposal")
	}
	connection, tool, proposed := ProposedCall(p)
	if args == nil {
		args = proposed
	} else {
		p.Data = models.JSONMap{"tool": tool, "arguments": args}
	}
	res, err := CallTool(ctx, connection, tool, args)
	if err == nil && res.IsError {
		err = errors.New(truncate(res.Text, 2000))
		if res.Text == "" {
			err = errors.New("the tool returned an error")
		}
	}
	if err != nil {
		msg := err.Error()
		if errors.Is(err, storage.ErrConnectionNotFound) {
			msg = "the connection no longer exists"
		}
		if ferr := storage.FinishProposal(p, models.ProposalFailed, reviewer, "", msg); ferr != nil {
			return p, res, ferr
		}
		return p, res, ErrCallFailed
	}
	if p.Data == nil {
		p.Data = models.JSONMap{"tool": tool, "arguments": args}
	}
	p.Data["result"] = truncate(res.Text, 4000)
	for i := 0; i < 3; i++ {
		if err = storage.FinishProposal(p, models.ProposalApproved, reviewer, "", ""); err == nil {
			break
		}
	}
	return p, res, err
}
