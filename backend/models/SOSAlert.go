package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type SOSAlert struct {
	ID          uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID      uuid.UUID      `gorm:"type:uuid;not null" json:"userId"`
	UnitID      *uuid.UUID     `gorm:"type:uuid" json:"unitId,omitempty"`
	Latitude    float64        `json:"latitude"`
	Longitude   float64        `json:"longitude"`
	LocationGeohash string      `gorm:"type:varchar(12);index:idx_sos_geohash" json:"locationGeohash,omitempty"`
	IsAnonymous bool        `gorm:"default:false;index:idx_sos_anonymous" json:"isAnonymous"`
	Description string         `json:"description"`
	Status      string         `gorm:"default:pending" json:"status"` // pending, dispatched, resolved, cancelled
	Priority    string         `gorm:"default:high" json:"priority"`  // high, medium, low
	// Severity is distinct from Priority. Priority is the reporter's
	// self-rated urgency (high/medium/low). Severity is the dispatch
	// engine's own classification: "critical" allows multiple units to
	// respond as co-responders; "normal" locks to the first responder.
	Severity string `gorm:"type:varchar(16);default:'normal';index:idx_sos_severity" json:"severity"`

	// DispatchState tracks the operational state of the alert from the
	// dispatch engine's perspective (not the reporter-facing Status).
	// Values: open | locked | multi_responder | resolved
	DispatchState string `gorm:"type:varchar(24);default:'open';index:idx_sos_dispatch_state" json:"dispatchState"`

	// AcceptedByUnitID is the primary responder. For critical SOS it is
	// the first accepting unit; additional co-responders live in the
	// sos_responders table.
	AcceptedByUnitID *uuid.UUID `gorm:"type:uuid;index:idx_sos_accepted_by" json:"acceptedByUnitId,omitempty"`
	AcceptedAt       *time.Time `json:"acceptedAt,omitempty"`

	// Jurisdiction captured at write time from reverse geocoding so the
	// escalation engine can notify head admins in the right state/LGA
	// without re-geocoding on every query.
	JurisdictionState string `gorm:"type:varchar(120);index:idx_sos_jurisdiction_state" json:"jurisdictionState,omitempty"`
	JurisdictionLGA   string `gorm:"type:varchar(120);index:idx_sos_jurisdiction_lga"   json:"jurisdictionLga,omitempty"`

	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	User User          `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Unit *SecurityUnit `gorm:"foreignKey:UnitID" json:"unit,omitempty"`
}

func (SOSAlert) TableName() string {
	return "sos_alerts"
}
