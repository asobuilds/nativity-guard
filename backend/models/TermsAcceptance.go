package models

import (
    "time"

    "github.com/google/uuid"
)

// TermsAcceptance is proof that a user agreed to a specific document.
// This row is the legal artifact — who, when, which version, from what
// IP and user agent. Never delete or update these rows.
type TermsAcceptance struct {
    ID              uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
    UserID          uuid.UUID `gorm:"type:uuid;not null;index:idx_terms_acc_user" json:"userId"`
    DocumentID      uuid.UUID `gorm:"type:uuid;not null" json:"documentId"`
    DocumentKind    string    `gorm:"type:varchar(16);not null" json:"documentKind"` // "terms" | "privacy"
    DocumentRole    string    `gorm:"type:varchar(32);not null" json:"documentRole"`
    DocumentVersion string    `gorm:"type:varchar(32);not null;index:idx_terms_acc_version" json:"documentVersion"`
    AcceptedAt      time.Time `gorm:"not null;index:idx_terms_acc_at" json:"acceptedAt"`
    IPAddress       string    `gorm:"type:varchar(64)" json:"ipAddress,omitempty"`
    UserAgent       string    `gorm:"type:varchar(400)" json:"userAgent,omitempty"`
    CreatedAt       time.Time `json:"createdAt"`

    User     User          `gorm:"foreignKey:UserID" json:"-"`
    Document TermsDocument `gorm:"foreignKey:DocumentID" json:"-"`
}

func (TermsAcceptance) TableName() string { return "terms_acceptances" }