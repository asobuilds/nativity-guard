package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

// LocalIntelligence represents extracted factual intelligence from community reports and alerts.
type LocalIntelligence struct {
	ID            uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	State         string         `gorm:"type:varchar(120);index:idx_local_intel_state" json:"state,omitempty"`
	LGA           string         `gorm:"type:varchar(120);index:idx_local_intel_lga" json:"lga,omitempty"`
	Fact          string         `gorm:"type:text;not null" json:"fact"`
	Category      string         `gorm:"type:varchar(32);default:'general'" json:"category"`
	Confidence    string         `gorm:"type:varchar(16);default:'medium'" json:"confidence"` // high, medium, low
	SourceType    string         `gorm:"type:varchar(32);not null" json:"sourceType"`         // report, sos, post
	SourceID      string         `gorm:"type:varchar(64);not null" json:"sourceId"`
	Upvotes       int            `gorm:"not null;default:0" json:"upvotes"`
	LastUpvotedAt *time.Time     `json:"lastUpvotedAt,omitempty"`
	ExpiresAt     *time.Time     `gorm:"index:idx_local_intel_expires" json:"expiresAt,omitempty"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

func (LocalIntelligence) TableName() string {
	return "local_intel"
}
