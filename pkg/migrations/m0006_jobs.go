package migrations

import (
	"time"

	"gorm.io/gorm"
)

// Migration 0006 adds the jobs table of the database job queue
// (queue.driver: database), which runs GoHead without Redis.

type job0006 struct {
	ID             uint64  `gorm:"primaryKey"`
	TaskID         *string `gorm:"size:191;uniqueIndex"`
	Queue          string  `gorm:"size:64;not null;index:idx_jobs_claim,priority:2"`
	Type           string  `gorm:"size:128;not null"`
	Payload        []byte
	Status         string     `gorm:"size:16;not null;index:idx_jobs_claim,priority:1"`
	RunAt          time.Time  `gorm:"not null;index:idx_jobs_claim,priority:3"`
	Retried        int        `gorm:"not null;default:0"`
	MaxRetry       int        `gorm:"not null;default:0"`
	TimeoutMS      int64      `gorm:"column:timeout_ms;not null;default:0"`
	RetentionMS    int64      `gorm:"column:retention_ms;not null;default:0"`
	LockedBy       string     `gorm:"size:64"`
	LockedUntil    *time.Time `gorm:"index"`
	RetentionUntil *time.Time `gorm:"index"`
	LastError      string     `gorm:"type:text"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (job0006) TableName() string { return "jobs" }

func jobsUp(tx *gorm.DB) error   { return tx.AutoMigrate(&job0006{}) }
func jobsDown(tx *gorm.DB) error { return tx.Migrator().DropTable("jobs") }
