package models

import (
  "time"
  "github.com/google/uuid"
)

// SocialIdentity is keyed by the provider's immutable subject, not its email.
// Never silently attach an external identity to an existing password account.
type SocialIdentity struct {
  ID uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
  UserID uuid.UUID `gorm:"type:uuid;not null;index"`
  Provider string `gorm:"type:varchar(16);not null;uniqueIndex:idx_social_subject"`
  Subject string `gorm:"type:varchar(256);not null;uniqueIndex:idx_social_subject"`
  CreatedAt time.Time
}

func (SocialIdentity) TableName() string { return "social_identities" }

// LoginChallenge is short lived and one-use; it prevents replay of externally
// issued ID tokens and binds the signed token to a fresh auth attempt.
type LoginChallenge struct {
  ID uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
  NonceHash string `gorm:"type:char(64);not null;uniqueIndex"`
  Provider string `gorm:"type:varchar(16);not null"`
  ExpiresAt time.Time `gorm:"not null;index"`
  UsedAt *time.Time
  CreatedAt time.Time
}
func (LoginChallenge) TableName() string { return "login_challenges" }
