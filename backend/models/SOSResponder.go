package models

import (
	"time"

	"github.com/google/uuid"
)

// SOSResponder links a security unit to an SOS alert it has accepted.
//
// For non-critical SOS, at most one row exists (the accepting unit).
// For critical SOS, multiple rows may exist (co-responders) up to a
// cap enforced by the accept handler.
type SOSResponder struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	SOSID      uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_sos_responder_unique,priority:1" json:"sosId"`
	UnitID     uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_sos_responder_unique,priority:2" json:"unitId"`
	Role       string    `gorm:"type:varchar(24);default:'primary'" json:"role"` // primary | support
	AcceptedBy uuid.UUID `gorm:"type:uuid;not null" json:"acceptedBy"`           // user id that performed the accept
	AcceptedAt time.Time `gorm:"not null" json:"acceptedAt"`
	CreatedAt  time.Time `json:"createdAt"`

	SOS  SOSAlert     `gorm:"foreignKey:SOSID" json:"-"`
	Unit SecurityUnit `gorm:"foreignKey:UnitID" json:"unit,omitempty"`
}

func (SOSResponder) TableName() string { return "sos_responders" }
