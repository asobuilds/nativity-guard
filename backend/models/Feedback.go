package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Feedback is a user-submitted bug report, feature request, complaint, or
// support question. Admins read them from a single queue and reply once.
type Feedback struct {
	ID     uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID uuid.UUID `gorm:"type:uuid;not null;index:idx_feedback_user" json:"userId"`

	// Category: bug | feature | complaint | support | other
	Category string `gorm:"type:varchar(24);not null;index:idx_feedback_category" json:"category"`

	// Priority: low | normal | high. Auto-derived on submit from the category.
	Priority string `gorm:"type:varchar(16);default:'normal';index:idx_feedback_priority" json:"priority"`

	// Status: open | in_review | resolved | closed
	Status string `gorm:"type:varchar(16);default:'open';index:idx_feedback_status" json:"status"`

	Subject string `gorm:"type:varchar(200);not null" json:"subject"`
	Body    string `gorm:"type:text;not null" json:"body"`

	// ContactEmail is optional — someone submitting anonymously may want a
	// reply channel that is not their account email.
	ContactEmail string `gorm:"type:varchar(200)" json:"contactEmail,omitempty"`

	// AdminReply is the single written response. Null until an admin replies.
	AdminReply  string     `gorm:"type:text" json:"adminReply,omitempty"`
	RepliedBy   *uuid.UUID `gorm:"type:uuid" json:"repliedBy,omitempty"`
	RepliedAt   *time.Time `json:"repliedAt,omitempty"`
	ClosedAt    *time.Time `json:"closedAt,omitempty"`

	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (Feedback) TableName() string { return "feedback" }