package storage

import (
	"testing"
	"time"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/database"
	"github.com/gohead-cms/gohead/pkg/testutils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func proposalFixture(t *testing.T) models.Collection {
	t.Helper()
	ct := historyFixture(t)
	require.NoError(t, database.DB.AutoMigrate(&models.Proposal{}, &models.Locale{}))
	return ct
}

var (
	bot      = Actor{Kind: models.ActorAgent, Name: "bot", AgentRunID: 9}
	reviewer = Actor{Kind: models.ActorUser, Name: "editor"}
)

func TestProposalApproveAppliesAsTheAgent(t *testing.T) {
	ct := proposalFixture(t)
	item, err := SaveItem(Actor{Kind: models.ActorUser, Name: "alice"}, ct, models.JSONMap{"title": "Hello"})
	require.NoError(t, err)

	p, err := ProposeItemWrite(bot, ct, models.ProposalUpdate, item.ID, "", models.JSONMap{"title": "Hi"}, "shorter")
	require.NoError(t, err)
	assert.Equal(t, models.ProposalPending, p.Status)
	assert.EqualValues(t, 1, p.BaseVersion)
	assert.False(t, ProposalIsStale(p))

	got, applied, err := ApproveProposal(p.ID, reviewer, nil, false)
	require.NoError(t, err)
	assert.Equal(t, models.ProposalApproved, got.Status)
	assert.Equal(t, "editor", got.ReviewedBy)
	assert.NotNil(t, got.DecidedAt)
	assert.Equal(t, "Hi", applied.Data["title"])

	revs, _, err := ListRevisions(RevisionQuery{AgentRunID: 9})
	require.NoError(t, err)
	require.Len(t, revs, 1, "the change belongs to the agent's run, so a run revert undoes it")
	assert.Equal(t, "bot", revs[0].ActorName)

	_, _, err = ApproveProposal(p.ID, reviewer, nil, false)
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
	got, _, err := ApproveProposal(p.ID, reviewer, nil, false)
	assert.ErrorIs(t, err, ErrProposalStale)
	assert.Equal(t, models.ProposalPending, got.Status)
	stored, _ := GetProposal(p.ID)
	assert.Equal(t, models.ProposalPending, stored.Status, "a stale proposal stays pending")

	// Edited values replace the agent's.
	got, applied, err := ApproveProposal(p.ID, reviewer, models.JSONMap{"title": "Merged"}, true)
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
	got, item, err := ApproveProposal(created.ID, reviewer, nil, false)
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
	_, _, err = ApproveProposal(del.ID, reviewer, nil, false)
	require.NoError(t, err)
	_, err = GetItemByID(ct.ID, item.ID)
	assert.Error(t, err)

	// The other proposal's item is gone: applying it fails and says why.
	failed, _, err := ApproveProposal(upd.ID, reviewer, nil, true)
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
	_, total, _ = ListProposals(ProposalQuery{Scope: map[string][]string{}})
	assert.Equal(t, 0, total, "an empty scope lists nothing")
	_, total, _ = ListProposals(ProposalQuery{Scope: map[string][]string{models.ProposalUpdate: {"posts"}}})
	assert.Equal(t, 0, total, "the scope is per action")
	list, total, _ := ListProposals(ProposalQuery{Scope: map[string][]string{models.ProposalCreate: {"posts"}}, PageSize: 2})
	assert.Equal(t, 4, total)
	assert.Len(t, list, 2)
}

func TestForcedApprovalKeepsLaterChanges(t *testing.T) {
	ct := historyFixture(t)
	require.NoError(t, database.DB.AutoMigrate(&models.Proposal{}, &models.Locale{}))
	ct.Attributes = append(ct.Attributes, models.Attribute{Name: "summary", Type: "text"})
	cid := ct.ID
	require.NoError(t, database.DB.Create(&models.Attribute{CollectionID: &cid, Name: "summary", Type: "text"}).Error)
	item, err := SaveItem(Actor{Kind: models.ActorUser, Name: "alice"}, ct, models.JSONMap{"title": "T0", "summary": "S0"})
	require.NoError(t, err)

	// Two proposals against the same version, each changing one field of
	// the whole item.
	a, err := ProposeItemWrite(bot, ct, models.ProposalUpdate, item.ID, "", models.JSONMap{"title": "T1", "summary": "S0"}, "")
	require.NoError(t, err)
	b, err := ProposeItemWrite(bot, ct, models.ProposalUpdate, item.ID, "", models.JSONMap{"title": "T0", "summary": "S1"}, "")
	require.NoError(t, err)
	_, _, err = ApproveProposal(a.ID, reviewer, nil, false)
	require.NoError(t, err)

	st, err := ProposalNow(b)
	require.NoError(t, err)
	assert.True(t, st.Stale)
	assert.Equal(t, models.JSONMap{"title": "T1", "summary": "S1"}, st.Proposed, "rebased on the item now")
	_, applied, err := ApproveProposal(b.ID, reviewer, nil, true)
	require.NoError(t, err)
	assert.Equal(t, "T1", applied.Data["title"], "the earlier approved change is kept")
	assert.Equal(t, "S1", applied.Data["summary"])
}

func TestApprovalChecksTheItemsCollection(t *testing.T) {
	ct := proposalFixture(t)
	other := &models.Collection{Name: "vault", Attributes: []models.Attribute{{Name: "title", Type: "string"}}}
	require.NoError(t, database.DB.Create(other).Error)
	secret, err := SaveItem(SystemActor, *other, models.JSONMap{"title": "secret"})
	require.NoError(t, err)
	p := &models.Proposal{AgentName: "bot", Action: models.ProposalDelete, Collection: ct.Name, ItemID: &secret.ID, Status: models.ProposalPending}
	require.NoError(t, database.DB.Create(p).Error)

	got, _, err := ApproveProposal(p.ID, reviewer, nil, true)
	assert.ErrorIs(t, err, ErrProposalUnappliable)
	assert.Equal(t, models.ProposalFailed, got.Status)
	_, err = GetItemByID(other.ID, secret.ID)
	assert.NoError(t, err, "an item of another collection is never touched")
}

func TestStuckClaimIsReleased(t *testing.T) {
	ct := proposalFixture(t)
	p, err := ProposeItemWrite(bot, ct, models.ProposalCreate, 0, "", models.JSONMap{"title": "x"}, "")
	require.NoError(t, err)
	_, err = claimProposal(p.ID)
	require.NoError(t, err)
	_, err = RejectProposal(p.ID, "editor", "")
	assert.ErrorIs(t, err, ErrProposalDecided, "someone is deciding it")

	// A server that died mid-approval leaves an old claim, which expires.
	require.NoError(t, database.DB.Model(&models.Proposal{}).Where("id = ?", p.ID).
		Update("decided_at", models.APITime(time.Now().Add(-time.Hour))).Error)
	got, err := RejectProposal(p.ID, "editor", "")
	require.NoError(t, err)
	assert.Equal(t, models.ProposalRejected, got.Status)
}

func TestTranslationProposals(t *testing.T) {
	_, db := testutils.SetupTestServer()
	t.Cleanup(testutils.CleanupTestDB)
	require.NoError(t, db.AutoMigrate(&models.Item{}, &models.Collection{}, &models.Attribute{}, &models.Revision{}, &models.Proposal{}, &models.Locale{}))
	ct := &models.Collection{Name: "posts", Attributes: []models.Attribute{{Name: "title", Type: "string", Localized: true}, {Name: "views", Type: "int"}}}
	require.NoError(t, db.Create(ct).Error)
	require.NoError(t, db.Create(&models.Locale{Code: "en", Name: "English", IsDefault: true}).Error)
	require.NoError(t, db.Create(&models.Locale{Code: "fr", Name: "Français"}).Error)
	invalidateLocales() // a cold cache, as after a restart
	item, err := SaveItem(SystemActor, *ct, models.JSONMap{"title": "Hello", "views": 1})
	require.NoError(t, err)
	invalidateLocales()

	_, err = ProposeItemWrite(bot, *ct, models.ProposalUpdate, item.ID, "fr", models.JSONMap{"views": 2}, "")
	assert.ErrorContains(t, err, "not localized")
	require.NoError(t, CheckItemWrite(*ct, models.ProposalUpdate, item.ID, "fr", models.JSONMap{"title": "Bonjour"}))
	p, err := ProposeItemWrite(bot, *ct, models.ProposalUpdate, item.ID, "fr", models.JSONMap{"title": "Bonjour"}, "")
	require.NoError(t, err)
	st, err := ProposalNow(p)
	require.NoError(t, err)
	assert.False(t, st.Stale)
	assert.Empty(t, st.Current, "no French values yet")

	_, applied, err := ApproveProposal(p.ID, reviewer, nil, false)
	require.NoError(t, err)
	assert.Equal(t, "Bonjour", applied.Translations["fr"].(map[string]any)["title"])
	assert.Equal(t, "Hello", applied.Data["title"], "the default locale is untouched")
}
