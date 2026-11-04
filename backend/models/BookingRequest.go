package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	BookingStatusPending   = "pending"
	BookingStatusAccepted  = "accepted"
	BookingStatusDeclined  = "declined"
	BookingStatusPaid      = "paid"
	BookingStatusCompleted = "completed"
	BookingStatusCancelled = "cancelled"
)

// BookingRequest represents a citizen or unit booking a security unit for an event.
// Payments are bank-transfer based; the platform never touches the money. The
// unit admin confirms receipt manually via mark-paid, and the platform fee is
// tracked separately for invoicing.
type BookingRequest struct {
	ID            uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID        uuid.UUID      `gorm:"type:uuid;not null;index" json:"userId"`
	UnitID        uuid.UUID      `gorm:"type:uuid;not null;index" json:"unitId"`
	EventType     string         `gorm:"not null" json:"eventType"`
	EventDate     time.Time      `gorm:"not null" json:"eventDate"`
	EventLocation string        `gorm:"not null" json:"eventLocation"`
	DurationHours float64        `gorm:"not null;default:1" json:"durationHours"`
	GuardsNeeded  int            `gorm:"not null;default:0" json:"guardsNeeded"`
	Status        string         `gorm:"not null;default:pending;index" json:"status"`
	Notes         string         `gorm:"type:text" json:"notes,omitempty"`
	QuotationAmount float64       `gorm:"not null;default:0" json:"quotationAmount"`
	PlatformFee      float64       `gorm:"not null;default:0" json:"platformFee"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`

	User  User          `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Unit  SecurityUnit `gorm:"foreignKey:UnitID" json:"unit,omitempty"`
}

func (BookingRequest) TableName() string {
	return "booking_requests"
}
