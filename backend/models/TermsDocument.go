package models

import (
    "time"

    "github.com/google/uuid"
)

// TermsDocument is the master copy of a terms or privacy document at a
// specific version. One row per (kind, role, version). The `Content` field
// holds markdown. Never edit an existing row — bump Version and insert a
// new one so any acceptance row still points at the exact text the user saw.
type TermsDocument struct {
    ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
    Kind        string    `gorm:"type:varchar(16);not null;index:idx_terms_kind_role_version,priority:1" json:"kind"` // "terms" | "privacy"
    Role        string    `gorm:"type:varchar(32);not null;index:idx_terms_kind_role_version,priority:2" json:"role"` // "citizen" | "officer" | "unit_admin" | "super_admin" | "all"
    Version     string    `gorm:"type:varchar(32);not null;index:idx_terms_kind_role_version,priority:3" json:"version"`
    Title       string    `gorm:"type:varchar(160);not null" json:"title"`
    Content     string    `gorm:"type:text;not null" json:"content"`
    Summary     string    `gorm:"type:text" json:"summary,omitempty"`
    EffectiveAt time.Time `gorm:"not null" json:"effectiveAt"`
    IsActive    bool      `gorm:"default:true;index:idx_terms_active" json:"isActive"`
    CreatedAt   time.Time `json:"createdAt"`
    UpdatedAt   time.Time `json:"updatedAt"`
}

func (TermsDocument) TableName() string { return "terms_documents" }