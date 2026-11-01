package models

import (
	"time"

	"github.com/google/uuid"
)

// OfficerPanic records an officer-initiated distress event. Created when an
// officer triggers the panic endpoint; the audit trail keeps one row per event.
type OfficerPanic struct {
	ID        uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	OfficerID uuid.UUID  `gorm:"type:uuid;not null;index:idx_officer_panic_officer" json:"officerId"` // user id of the officer account
	CaseID    *uuid.UUID `gorm:"type:uuid;index:idx_officer_panic_case" json:"caseId,omitempty"`
	Latitude  float64    `gorm:"not null" json:"latitude"`
	Longitude float64    `gorm:"not null" json:"longitude"`
	Status    string     `gorm:"default:active;index:idx_officer_panic_status" json:"status"`
	CreatedAt time.Time  `json:"createdAt"`
}

func (OfficerPanic) TableName() string {
	return "officer_panics"
}
