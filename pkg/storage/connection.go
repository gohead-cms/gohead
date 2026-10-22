package storage

import (
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/database"

	"gorm.io/gorm"
)

var (
	// ErrConnectionNotFound is returned for an unknown connection.
	ErrConnectionNotFound = errors.New("connection not found")
	// ErrConnectionExists is returned when creating a connection whose name
	// is taken.
	ErrConnectionExists = errors.New("a connection with this name already exists")
)

// CreateConnection stores a new connection.
func CreateConnection(c *models.Connection) error {
	var n int64
	if err := database.DB.Model(&models.Connection{}).Where("name = ?", c.Name).Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return ErrConnectionExists
	}
	err := database.DB.Create(c).Error
	forgetConnections()
	return err
}

// SaveConnection updates every field of a connection.
func SaveConnection(c *models.Connection) error {
	err := database.DB.Save(c).Error
	forgetConnections()
	return err
}

// GetConnection returns a connection by name.
func GetConnection(name string) (*models.Connection, error) {
	var c models.Connection
	if err := database.DB.Where("name = ?", name).First(&c).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrConnectionNotFound
		}
		return nil, err
	}
	return &c, nil
}

// ListConnections returns every connection, by name.
func ListConnections() ([]models.Connection, error) {
	var out []models.Connection
	err := database.DB.Order("name ASC").Find(&out).Error
	return out, err
}

// DeleteConnection removes a connection. Pending calls proposed to it are
// rejected: they can no longer be made.
func DeleteConnection(name string) error {
	c, err := GetConnection(name)
	if err != nil {
		return err
	}
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		now := models.APITime(time.Now())
		if err := tx.Model(&models.Proposal{}).
			Where("collection = ? AND status IN ?", models.ConnectionProposalPrefix+c.Name, []string{models.ProposalPending, ProposalApplying}).
			Updates(map[string]any{"status": models.ProposalRejected, "review_note": "the connection was deleted", "decided_at": now}).Error; err != nil {
			return err
		}
		return tx.Delete(&models.Connection{}, c.ID).Error
	})
	forgetConnections()
	return err
}

// Connections are read for every tool lookup (the catalog, each agent
// save): they are cached briefly, and the cache is dropped on every change
// made by this process.
var (
	connMu      sync.Mutex
	connCache   map[string]models.Connection
	connCacheAt time.Time
)

const connCacheTTL = 15 * time.Second

func forgetConnections() {
	connMu.Lock()
	connCache = nil
	connMu.Unlock()
}

// CachedConnection returns a connection from a short-lived cache.
func CachedConnection(name string) (models.Connection, bool) {
	connMu.Lock()
	defer connMu.Unlock()
	if connCache == nil || time.Since(connCacheAt) > connCacheTTL {
		if database.DB == nil || !database.DB.Migrator().HasTable(&models.Connection{}) {
			return models.Connection{}, false
		}
		list, err := ListConnections()
		if err != nil {
			return models.Connection{}, false
		}
		connCache = make(map[string]models.Connection, len(list))
		for _, c := range list {
			connCache[c.Name] = c
		}
		connCacheAt = time.Now()
	}
	c, ok := connCache[name]
	return c, ok
}

// CachedConnections returns every connection from the cache, by name.
func CachedConnections() []models.Connection {
	CachedConnection("") // fills the cache
	connMu.Lock()
	defer connMu.Unlock()
	out := make([]models.Connection, 0, len(connCache))
	for _, c := range connCache {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// ProposeAppCall stores a call to a connected app's tool as a pending
// proposal: an admin approves it before it is made.
func ProposeAppCall(actor Actor, connection, tool string, args map[string]any, reason string) (*models.Proposal, error) {
	p := &models.Proposal{
		AgentName: actor.Name, Action: models.ProposalCall, Collection: models.ConnectionProposalPrefix + connection,
		Data:   models.JSONMap{"tool": tool, "arguments": copyMap(args)},
		Reason: reason, Status: models.ProposalPending, CreatedAt: models.APITime(time.Now()),
	}
	if actor.AgentRunID != 0 {
		id := actor.AgentRunID
		p.AgentRunID = &id
	}
	if err := database.DB.Create(p).Error; err != nil {
		return nil, err
	}
	return p, nil
}

// ClaimProposal marks a pending proposal as being applied, so nobody else
// decides it meanwhile. ReleaseProposal undoes it; FinishProposal records
// the decision.
func ClaimProposal(id uint) (*models.Proposal, error) { return claimProposal(id) }

// ReleaseProposal returns a claimed proposal to pending.
func ReleaseProposal(id uint) { releaseProposal(id) }

// FinishProposal records the decision of a claimed proposal.
func FinishProposal(p *models.Proposal, status, reviewer, note, errText string) error {
	return finishProposal(p, status, reviewer, note, errText)
}

// ForgetCaches drops every in-process cache of stored definitions (locales,
// roles, components, connections), after the database changed under them:
// a demo reset.
func ForgetCaches() {
	invalidateLocales()
	invalidateRoles()
	InvalidateComponentCache()
	forgetConnections()
}
