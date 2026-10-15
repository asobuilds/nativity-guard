package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AuthLinkToken struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID      `gorm:"type:uuid;not null;index:idx_auth_link_user" json:"userId"`
	TokenHash string         `gorm:"type:varchar(64);uniqueIndex;not null" json:"-"`
	Purpose   string         `gorm:"type:varchar(32);not null;index:idx_auth_link_purpose" json:"-"`
	ExpiresAt time.Time      `gorm:"not null;index:idx_auth_link_expires" json:"expiresAt"`
	UsedAt    *time.Time     `json:"-"`
	CreatedAt time.Time      `json:"createdAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (AuthLinkToken) TableName() string {
	return "auth_link_tokens"
}
