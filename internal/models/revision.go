package models

import "time"

// Revision entity types.
const (
	RevisionEntityItem      = "item"      // EntityID is the item ID
	RevisionEntitySingleton = "singleton" // EntityID is the singleton (single type) ID
)

// Revision actions.
const (
	RevisionCreate    = "create"
	RevisionUpdate    = "update"
	RevisionPublish   = "publish"
	RevisionUnpublish = "unpublish"
	RevisionDelete    = "delete"
	RevisionRestore   = "restore"
	// RevisionBaseline holds an entry's content from before history was
	// recorded: it is written just before the first change of an entry
	// that has no revisions yet.
	RevisionBaseline = "baseline"
)

// Revision actor kinds.
const (
	ActorUser   = "user"
	ActorToken  = "token"
	ActorAgent  = "agent"
	ActorSystem = "system"
)

// Revision is a snapshot of an entry (a collection item or a singleton's
// content) after one change. Versions count from 1 per entry. Data is the
// stored data after the change; for a delete, the data the entry had.
type Revision struct {
	ID         uint    `json:"id" gorm:"primaryKey"`
	EntityType string  `json:"entityType" gorm:"size:16;not null;uniqueIndex:idx_revisions_entity_version,priority:1"`
	EntityID   uint    `json:"entityId" gorm:"not null;uniqueIndex:idx_revisions_entity_version,priority:2"`
	Version    uint    `json:"version" gorm:"not null;uniqueIndex:idx_revisions_entity_version,priority:3"`
	TypeName   string  `json:"typeName" gorm:"size:191;index"` // collection or singleton name
	Action     string  `json:"action" gorm:"size:16;not null"`
	Status     string  `json:"status,omitempty" gorm:"size:16"` // draft or published (draft-and-publish collections)
	Data       JSONMap `json:"data" gorm:"type:json"`
	// Translations are the entry's localized values in other locales.
	Translations JSONMap `json:"translations,omitempty" gorm:"type:json"`
	ActorKind    string  `json:"actorKind" gorm:"size:16;not null"`
	ActorName    string  `json:"actorName" gorm:"size:191"`
	AgentRunID   *uint   `json:"agentRunId,omitempty" gorm:"index"`
	// RestoredFrom is the version a restore copied.
	RestoredFrom *uint     `json:"restoredFrom,omitempty"`
	CreatedAt    time.Time `json:"createdAt" gorm:"index"`
}
