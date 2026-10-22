package handlers

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/gohead-cms/gohead/internal/api/middleware"
	"github.com/gohead-cms/gohead/internal/apps"
	"github.com/gohead-cms/gohead/internal/demo"
	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/auth"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/storage"

	"github.com/gin-gonic/gin"
)

// Proposals are content changes agents with require_approval asked for.
// Editors review them: whoever may make a change may approve or reject the
// proposal of it.

// proposalAction is the permission a proposal's change needs.
func proposalAction(p *models.Proposal) string {
	switch p.Action {
	case models.ProposalCreate:
		return auth.ActionCreate
	case models.ProposalDelete:
		return auth.ActionDelete
	default:
		return auth.ActionUpdate
	}
}

// reviewScope maps each proposal action to the collections where the role
// may make that change (nil: everything).
func reviewScope(role string) (map[string][]string, error) {
	if auth.IsBuiltinRole(role) && auth.HasPermission(role, auth.ActionUpdate) {
		return nil, nil
	}
	collections, _, err := storage.GetAllCollections(nil, nil, nil)
	if err != nil {
		return nil, err
	}
	scope := map[string][]string{}
	for _, ct := range collections {
		for action, perm := range map[string]string{
			models.ProposalCreate: auth.ActionCreate, models.ProposalUpdate: auth.ActionUpdate, models.ProposalDelete: auth.ActionDelete,
		} {
			if auth.Can(role, perm, ct.Name) {
				scope[action] = append(scope[action], ct.Name)
			}
		}
	}
	return scope, nil
}

// bindOptional decodes an optional JSON body: an empty body is fine, a
// malformed one is refused.
func bindOptional(c *gin.Context, body any) bool {
	if err := c.ShouldBindJSON(body); err != nil && !errors.Is(err, io.EOF) {
		validationError(c, "invalid JSON body: "+err.Error())
		return false
	}
	return true
}

// redactProposal hides password values of a proposal's collection.
func redactProposal(p models.Proposal, hidden map[string]bool) models.Proposal {
	if len(hidden) > 0 && p.Data != nil {
		p.Data = redact(p.Data, hidden)
	}
	return p
}

func hiddenFields(ct *models.Collection) map[string]bool {
	if ct == nil {
		return nil
	}
	return (&historyTarget{collection: ct}).passwordFields()
}

// ListProposals lists proposals, newest first: status (pending by default;
// approved, rejected, failed or all), collection, agent, agentRun, page,
// pageSize. Only collections the caller may change are listed.
func ListProposals(c *gin.Context) {
	role := c.GetString("role")
	if !hasPermission(role, auth.ActionCreate) && !hasPermission(role, auth.ActionUpdate) && !hasPermission(role, auth.ActionDelete) {
		c.Set("response", "Access denied")
		c.Set("details", "reviewing agent proposals needs a role that can change content")
		c.Set("status", http.StatusForbidden)
		return
	}
	q := storage.ProposalQuery{Status: c.DefaultQuery("status", models.ProposalPending), Collection: c.Query("collection"), AgentName: c.Query("agent")}
	switch q.Status {
	case "all", models.ProposalPending, models.ProposalApproved, models.ProposalRejected, models.ProposalFailed:
	default:
		validationError(c, "status must be pending, approved, rejected, failed or all")
		return
	}
	if v := c.Query("agentRun"); v != "" {
		n, err := strconv.ParseUint(v, 10, 32)
		if err != nil || n == 0 {
			validationError(c, "agentRun must be a run ID")
			return
		}
		q.AgentRunID = uint(n)
	}
	var ok bool
	if q.Page, q.PageSize, ok = pageParams(c); !ok {
		return
	}
	// Calls to connected apps are reviewed by admins only (and shown to
	// visitors of a demo).
	q.NoCalls = !seesCalls(c)
	scope, err := reviewScope(role)
	if err != nil {
		c.Set("response", "Failed to list proposals")
		c.Set("status", http.StatusInternalServerError)
		return
	}
	q.Scope = scope
	list, total, err := storage.ListProposals(q)
	if err != nil {
		c.Set("response", "Failed to list proposals")
		c.Set("status", http.StatusInternalServerError)
		return
	}
	hidden := map[string]map[string]bool{}
	for i := range list {
		name := list[i].Collection
		if _, ok := hidden[name]; !ok {
			ct, _ := storage.GetCollectionByName(name)
			hidden[name] = hiddenFields(ct)
		}
		list[i] = redactProposal(list[i], hidden[name])
		if list[i].Status == models.ProposalPending && list[i].Action != models.ProposalCreate {
			list[i].Stale = storage.ProposalIsStale(&list[i])
		}
	}
	c.Set("response", list)
	c.Set("meta", gin.H{"pagination": gin.H{
		"page": q.Page, "pageSize": q.PageSize, "total": total, "pageCount": (total + q.PageSize - 1) / q.PageSize,
	}})
	c.Set("status", http.StatusOK)
}

// loadProposal resolves :id and checks the caller may decide it.
func loadProposal(c *gin.Context) (*models.Proposal, *models.Collection, bool) {
	id, ok := uintParam(c, "id")
	if !ok {
		return nil, nil, false
	}
	p, err := storage.GetProposal(id)
	if err != nil {
		c.Set("response", "Proposal not found")
		c.Set("status", http.StatusNotFound)
		return nil, nil, false
	}
	if p.Action == models.ProposalCall {
		// Calls act with a connected app's token: like the connections
		// themselves, only admins in a user session (not API tokens) see them.
		if !seesCalls(c) {
			c.Set("response", "Proposal not found")
			c.Set("status", http.StatusNotFound)
			return nil, nil, false
		}
		return p, nil, true
	}
	if !can(c, proposalAction(p), p.Collection) {
		c.Set("response", "Proposal not found")
		c.Set("status", http.StatusNotFound)
		return nil, nil, false
	}
	ct, err := storage.GetCollectionByName(p.Collection)
	if err != nil {
		ct = nil // the collection was deleted: the proposal can only be rejected
	}
	return p, ct, true
}

// GetProposal returns a proposal with the item as it is now, the values
// approving it would write ("proposed": for a stale update, the item now
// with the agent's changes) and the changes between the two.
func GetProposal(c *gin.Context) {
	p, ct, ok := loadProposal(c)
	if !ok {
		return
	}
	if p.Action == models.ProposalCall {
		c.Set("response", callProposalJSON(p))
		c.Set("status", http.StatusOK)
		return
	}
	hidden := hiddenFields(ct)
	out := gin.H{"proposal": redactProposal(*p, hidden), "stale": false}
	if p.Status == models.ProposalPending {
		st, err := storage.ProposalNow(p)
		if err != nil {
			c.Set("response", "Failed to load the proposal")
			c.Set("status", http.StatusInternalServerError)
			return
		}
		out["stale"] = st.Stale
		if st.Current != nil {
			out["current"] = redact(st.Current, hidden)
		}
		if st.Proposed != nil {
			out["proposed"] = redact(st.Proposed, hidden)
		}
		if ct != nil && p.ItemID != nil && p.Action != models.ProposalCreate {
			if item, err := storage.GetItemByID(ct.ID, *p.ItemID); err == nil {
				out["currentStatus"] = item.Status()
			}
		}
		changes := storage.DiffData(st.Current, st.Proposed)
		for i := range changes {
			if hidden[changes[i].Field] {
				changes[i].Before, changes[i].After = maskValue(changes[i].Before), maskValue(changes[i].After)
			}
		}
		out["changes"] = changes
	}
	c.Set("response", out)
	c.Set("status", http.StatusOK)
}

// ApproveProposal applies a proposal. Body (optional): {"data": {…}} to
// apply edited values instead of the agent's, {"force": true} to apply it
// although the item changed since. The change needs the caller's own
// permissions (publish to change a published item, create in the
// collections a relation object would create items in).
func ApproveProposal(c *gin.Context) {
	p, ct, ok := loadProposal(c)
	if !ok {
		return
	}
	if p.Action == models.ProposalCall {
		if demoDenied(c) {
			return
		}
		approveCall(c, p)
		return
	}
	if ct == nil {
		c.Set("response", "The collection no longer exists: reject this proposal")
		c.Set("status", http.StatusConflict)
		return
	}
	var body struct {
		Data  map[string]any `json:"data"`
		Force bool           `json:"force"`
	}
	if !bindOptional(c, &body) {
		return
	}
	data := models.JSONMap(body.Data)
	effective := p.Data
	if data != nil {
		effective = data
		// Password values are masked in what reviewers see: a masked value
		// sent back keeps the proposed one.
		if hidden := hiddenFields(ct); len(hidden) > 0 {
			st, err := storage.ProposalNow(p)
			if err != nil {
				c.Set("response", "Failed to load the proposal")
				c.Set("status", http.StatusInternalServerError)
				return
			}
			for k := range hidden {
				if data[k] == maskedValue {
					if v, ok := st.Proposed[k]; ok {
						data[k] = v
					} else {
						delete(data, k)
					}
				}
			}
		}
	}
	if p.Action != models.ProposalDelete && !allowNested(c, *ct, effective) {
		return
	}
	if p.Action == models.ProposalUpdate && p.ItemID != nil && !allowLiveEdit(c, *ct, *p.ItemID) {
		return
	}
	proposal, item, err := storage.ApproveProposal(p.ID, actorFrom(c), data, body.Force)
	if err != nil {
		var verr *storage.ItemValidationError
		switch {
		case errors.Is(err, storage.ErrProposalDecided):
			c.Set("response", decidedMessage(proposal))
			c.Set("details", gin.H{"code": "decided"})
			c.Set("status", http.StatusConflict)
		case errors.Is(err, storage.ErrProposalStale):
			c.Set("response", err.Error())
			c.Set("details", gin.H{"code": "stale", "hint": "review the item as it is now; approve with force to apply the proposal anyway"})
			c.Set("status", http.StatusConflict)
		case errors.As(err, &verr), errors.Is(err, storage.ErrItemNotFound), errors.Is(err, storage.ErrProposalUnappliable):
			c.Set("response", "The change could not be applied: "+proposal.Error)
			c.Set("details", gin.H{"proposal": proposal})
			c.Set("status", http.StatusUnprocessableEntity)
		default:
			logger.Log.WithError(err).WithField("proposal", p.ID).Error("Failed to apply a proposal")
			c.Set("response", "Failed to apply the proposal")
			c.Set("status", http.StatusInternalServerError)
		}
		return
	}
	out := gin.H{"proposal": proposal}
	if item != nil {
		out["item"] = models.ItemResponse(*ct, *item, nil)
	}
	c.Set("response", out)
	c.Set("status", http.StatusOK)
}

// RejectProposal discards a proposal. Body (optional): {"note": "…"}.
func RejectProposal(c *gin.Context) {
	p, _, ok := loadProposal(c)
	if !ok {
		return
	}
	if p.Action == models.ProposalCall && demoDenied(c) {
		return
	}
	var body struct {
		Note string `json:"note"`
	}
	if !bindOptional(c, &body) {
		return
	}
	proposal, err := storage.RejectProposal(p.ID, actorFrom(c).Name, body.Note)
	if err != nil {
		if errors.Is(err, storage.ErrProposalDecided) {
			c.Set("response", decidedMessage(proposal))
			c.Set("status", http.StatusConflict)
			return
		}
		c.Set("response", "Failed to reject the proposal")
		c.Set("status", http.StatusInternalServerError)
		return
	}
	c.Set("response", gin.H{"proposal": proposal})
	c.Set("status", http.StatusOK)
}

func decidedMessage(p *models.Proposal) string {
	if p == nil || p.Status == "applying" {
		return "This proposal is being decided by someone else"
	}
	return "This proposal was already " + p.Status
}

// callProposalJSON is a proposed call to a connected app's tool, with the
// tool as the connection lists it now.
func callProposalJSON(p *models.Proposal) gin.H {
	connection, tool, args := apps.ProposedCall(p)
	out := gin.H{"proposal": p, "stale": false, "call": gin.H{
		"connection": connection, "tool": tool, "arguments": args,
	}}
	if conn, err := storage.GetConnection(connection); err == nil {
		call := out["call"].(gin.H)
		call["connectionLabel"] = conn.Label
		if t, ok := conn.Tool(tool); ok {
			call["title"], call["description"], call["readOnly"], call["destructive"] = t.Title, t.Description, t.ReadOnly, t.Destructive
		}
	} else {
		out["missing"] = true
	}
	return out
}

// approveCall makes a proposed call. Body (optional): {"arguments": {…}}
// to call the tool with edited arguments.
func approveCall(c *gin.Context, p *models.Proposal) {
	var body struct {
		Arguments map[string]any `json:"arguments"`
	}
	if !bindOptional(c, &body) {
		return
	}
	// The call goes on if the reviewer leaves: once sent, the app may act on
	// it, and the proposal must say how it ended.
	proposal, res, err := apps.ApproveCall(context.WithoutCancel(c.Request.Context()), p.ID, actorFrom(c).Name, body.Arguments)
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrProposalDecided):
			c.Set("response", decidedMessage(proposal))
			c.Set("details", gin.H{"code": "decided"})
			c.Set("status", http.StatusConflict)
		case errors.Is(err, apps.ErrCallFailed):
			c.Set("response", "The call failed: "+proposal.Error)
			c.Set("details", gin.H{"proposal": proposal})
			c.Set("status", http.StatusUnprocessableEntity)
		default:
			logger.Log.WithError(err).WithField("proposal", p.ID).Error("Failed to make a proposed call")
			c.Set("response", "Failed to make the call")
			c.Set("status", http.StatusInternalServerError)
		}
		return
	}
	c.Set("response", gin.H{"proposal": proposal, "result": res.Text})
	c.Set("status", http.StatusOK)
}

// seesCalls reports whether the caller sees calls to connected apps: admins
// in a user session, and visitors of a demo (who cannot decide them).
func seesCalls(c *gin.Context) bool {
	if _, isToken := c.Get(middleware.APITokenIDKey); isToken {
		return false
	}
	role := c.GetString("role")
	return role == auth.RoleAdmin || demo.IsDemo(role)
}

// demoDenied refuses decisions on calls to visitors of a demo.
func demoDenied(c *gin.Context) bool {
	if !demo.IsDemo(c.GetString("role")) {
		return false
	}
	c.Set("response", "This is a demo: calls to connected apps are approved by the demo's owner only")
	c.Set("status", http.StatusForbidden)
	return true
}
