package models

import "time"

// Proposal statuses.
const (
	ProposalPending  = "pending"
	ProposalApproved = "approved"
	ProposalRejected = "rejected"
	// ProposalFailed: approved, but the change could not be applied (the
	// data no longer validates, the item is gone…).
	ProposalFailed = "failed"
)

// Proposal actions.
const (
	ProposalCreate = "create"
	ProposalUpdate = "update"
	ProposalDelete = "delete"
)

// Proposal is a content change an agent asked for, applied only once an
// editor approves it (agents with require_approval).
type Proposal struct {
	ID         uint    `json:"id" gorm:"primaryKey"`
	AgentName  string  `json:"agentName" gorm:"size:191;index"`
	AgentRunID *uint   `json:"agentRunId,omitempty" gorm:"index"`
	Action     string  `json:"action" gorm:"size:16;not null"`
	Collection string  `json:"collection" gorm:"size:191;not null;index"`
	ItemID     *uint   `json:"itemId,omitempty" gorm:"index"`
	Locale     string  `json:"locale,omitempty" gorm:"size:35"`
	Data       JSONMap `json:"data" gorm:"type:json"`
	// Reason is the agent's explanation, when it gave one.
	Reason string `json:"reason,omitempty" gorm:"type:text"`
	// BaseVersion is the item's latest revision when proposed (0: none),
	// to tell whether it changed since.
	BaseVersion uint       `json:"baseVersion"`
	Status      string     `json:"status" gorm:"size:16;not null;index"`
	ReviewedBy  string     `json:"reviewedBy,omitempty" gorm:"size:191"`
	ReviewNote  string     `json:"reviewNote,omitempty" gorm:"type:text"`
	Error       string     `json:"error,omitempty" gorm:"type:text"`
	CreatedAt   time.Time  `json:"createdAt" gorm:"index"`
	DecidedAt   *time.Time `json:"decidedAt,omitempty"`
}
