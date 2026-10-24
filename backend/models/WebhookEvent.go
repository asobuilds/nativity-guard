package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// WebhookEvent records an incoming payment-provider webhook so that
// duplicate deliveries (Paystack/Flutterwave retry on timeouts) cannot
// cause double-credits or double-notifications.
//
// The idempotency key is the provider's event ID (e.g. "evt_123abc").
// Unique on idempotency_key alone — the same event never processed twice.
type WebhookEvent struct {
	ID             uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	IdempotencyKey string     `gorm:"type:varchar(120);not null;uniqueIndex:idx_webhook_idempotency" json:"idempotencyKey"`
	Provider       string     `gorm:"type:varchar(32);not null;index:idx_webhook_provider" json:"provider"` // paystack | flutterwave
	EventType      string     `gorm:"type:varchar(64);not null;index:idx_webhook_event_type" json:"eventType"`
	Payload        string     `gorm:"type:text;not null" json:"payload"` // raw JSON body
	Status         string     `gorm:"type:varchar(16);not null;default:'pending'" json:"status"` // pending | processed | failed
	ProcessedAt    *time.Time `json:"processedAt,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

func (WebhookEvent) TableName() string {
	return "webhook_events"
}