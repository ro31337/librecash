package objects

import (
	"time"
)

// TimelineRecord represents a fanout message sent to a user for an exchange
type TimelineRecord struct {
	ID                int64
	ExchangeID        int64
	RecipientUserID   int64
	TelegramMessageID *int       // nullable until message is sent
	Status            string     // 'pending', 'sent', 'failed', 'deleted'
	IsDeleted         bool       // soft delete flag
	DeletedAt         *time.Time // when message was deleted (nullable)
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// Status constants
const (
	TimelineStatusPending = "pending"
	TimelineStatusSent    = "sent"
	TimelineStatusFailed  = "failed"
	TimelineStatusDeleted = "deleted"
)

// NewTimelineRecord creates a new timeline record with initial values
func NewTimelineRecord(exchangeID, recipientUserID int64) *TimelineRecord {
	return &TimelineRecord{
		ExchangeID:      exchangeID,
		RecipientUserID: recipientUserID,
		Status:          TimelineStatusPending,
		IsDeleted:       false,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
}
