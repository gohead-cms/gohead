package storage

import (
	"testing"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/database"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func proposalFixture(t *testing.T) models.Collection {
	t.Helper()
	ct := historyFixture(t)
	require.NoError(t, database.DB.AutoMigrate(&models.Proposal{}, &models.Locale{}))
	return ct
}

var bot = Actor{Kind: models.ActorAgent, Name: "bot", AgentRunID: 9}

func TestProposalApproveAppliesAsTheAgent(t *testing.T) {
	ct := proposalFixture(t)
	item, err := SaveItem(Actor{Kind: models.ActorUser, Name: "alice"}, ct, models.JSONMap{"title": "Hello"})
	require.NoError(t, err)

	p, err := ProposeItemWrite(bot, ct, models.ProposalUpdate, item.ID, "", models.JSONMap{"title": "Hi"}, "shorter")
	require.NoError(t, err)
	assert.Equal(t, models.ProposalPending, p.Status)
	assert.EqualValues(t, 1, p.BaseVersion)
	assert.False(t, ProposalIsStale(p))

	got, applied, err := ApproveProposal(p.ID, "editor", nil, false)
	require.NoError(t, err)
	assert.Equal(t, models.ProposalApproved, got.Status)
	assert.Equal(t, "editor", got.ReviewedBy)
	assert.NotNil(t, got.DecidedAt)
	assert.Equal(t, "Hi", applied.Data["title"])

	revs, _, err := ListRevisions(RevisionQuery{AgentRunID: 9})
	require.NoError(t, err)
	require.Len(t, revs, 1, "the change belongs to the agent's run, so a run revert undoes it")
	assert.Equal(t, "bot", revs[0].ActorName)

	_, _, err = ApproveProposal(p.ID, "editor", nil, false)
	assert.ErrorIs(t, err, ErrProposalDecided)
	_, err = RejectProposal(p.ID, "editor", "")
	assert.ErrorIs(t, err, ErrProposalDecided)
}

func TestProposalStaleAndForce(t *testing.T) {
	ct := proposalFixture(t)
	alice := Actor{Kind: models.ActorUser, Name: "alice"}
	item, err := SaveItem(alice, ct, models.JSONMap{"title": "Hello"})
	require.NoError(t, err)
	p, err := ProposeItemWrite(bot, ct, models.ProposalUpdate, item.ID, "", models.JSONMap{"title": "Bot"}, "")
	require.NoError(t, err)
	_, err = UpdateItemContent(alice, ct, item.ID, models.JSONMap{"title": "Edited by hand"}, nil)
	require.NoError(t, err)

	assert.True(t, ProposalIsStale(p))
	got, _, err := ApproveProposal(p.ID, "editor", nil, false)
	assert.ErrorIs(t, err, ErrProposalStale)
	assert.Equal(t, models.ProposalPending, got.Status)
	stored, _ := GetProposal(p.ID)
	assert.Equal(t, models.ProposalPending, stored.Status, "a stale proposal stays pending")

	// Edited values replace the agent's.
	got, applied, err := ApproveProposal(p.ID, "editor", models.JSONMap{"title": "Merged"}, true)
	require.NoError(t, err)
	assert.Equal(t, "Merged", applied.Data["title"])
	assert.Equal(t, "Merged", got.Data["title"])
}

func TestProposalCreateDeleteRejectFail(t *testing.T) {
	ct := proposalFixture(t)
	created, err := ProposeItemWrite(bot, ct, models.ProposalCreate, 0, "", models.JSONMap{"title": "New"}, "")
	require.NoError(t, err)
	var n int64
	database.DB.Model(&models.Item{}).Count(&n)
	assert.Zero(t, n, "nothing is created before approval")
	got, item, err := ApproveProposal(created.ID, "editor", nil, false)
	require.NoError(t, err)
	require.NotNil(t, got.ItemID)
	assert.Equal(t, item.ID, *got.ItemID)

	del, err := ProposeItemWrite(bot, ct, models.ProposalDelete, item.ID, "", nil, "spam")
	require.NoError(t, err)
	rejected, err := RejectProposal(del.ID, "editor", "not spam")
	require.NoError(t, err)
	assert.Equal(t, models.ProposalRejected, rejected.Status)
	assert.Equal(t, "not spam", rejected.ReviewNote)
	_, err = GetItemByID(ct.ID, item.ID)
	require.NoError(t, err)

	del, err = ProposeItemWrite(bot, ct, models.ProposalDelete, item.ID, "", nil, "")
	require.NoError(t, err)
	upd, err := ProposeItemWrite(bot, ct, models.ProposalUpdate, item.ID, "", models.JSONMap{"title": "Late"}, "")
	require.NoError(t, err)
	_, _, err = ApproveProposal(del.ID, "editor", nil, false)
	require.NoError(t, err)
	_, err = GetItemByID(ct.ID, item.ID)
	assert.Error(t, err)

	// The other proposal's item is gone: applying it fails and says why.
	failed, _, err := ApproveProposal(upd.ID, "editor", nil, true)
	assert.ErrorIs(t, err, ErrItemNotFound)
	assert.Equal(t, models.ProposalFailed, failed.Status)
	assert.Equal(t, "the item no longer exists", failed.Error)

	// Invalid or impossible changes are refused when proposed.
	_, err = ProposeItemWrite(bot, ct, models.ProposalUpdate, 999, "", models.JSONMap{"title": "x"}, "")
	assert.ErrorIs(t, err, ErrItemNotFound)
	_, err = ProposeItemWrite(bot, ct, models.ProposalCreate, 0, "", models.JSONMap{"nope": 1}, "")
	var verr *ItemValidationError
	assert.ErrorAs(t, err, &verr)
}

func TestListProposalsFilters(t *testing.T) {
	ct := proposalFixture(t)
	for i := 0; i < 3; i++ {
		_, err := ProposeItemWrite(bot, ct, models.ProposalCreate, 0, "", models.JSONMap{"title": "x"}, "")
		require.NoError(t, err)
	}
	other := Actor{Kind: models.ActorAgent, Name: "other"}
	p, err := ProposeItemWrite(other, ct, models.ProposalCreate, 0, "", models.JSONMap{"title": "y"}, "")
	require.NoError(t, err)
	_, err = RejectProposal(p.ID, "editor", "")
	require.NoError(t, err)

	_, total, err := ListProposals(ProposalQuery{Status: models.ProposalPending})
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	_, total, _ = ListProposals(ProposalQuery{Status: "all", AgentName: "other"})
	assert.Equal(t, 1, total)
	_, total, _ = ListProposals(ProposalQuery{AgentRunID: 9})
	assert.Equal(t, 3, total)
	_, total, _ = ListProposals(ProposalQuery{Collections: []string{}})
	assert.Equal(t, 0, total, "an empty scope lists nothing")
	list, total, _ := ListProposals(ProposalQuery{Collections: []string{"posts"}, PageSize: 2})
	assert.Equal(t, 4, total)
	assert.Len(t, list, 2)
}
