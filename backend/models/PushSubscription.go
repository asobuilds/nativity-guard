package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PushSubscription struct {
	ID        uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID      `gorm:"type:uuid;not null;index" json:"userId"`
	Endpoint  string         `gorm:"type:text;not null;uniqueIndex" json:"endpoint"`
	P256dh    string         `gorm:"type:text;not null" json:"p256dh"`
	Auth      string         `gorm:"type:text;not null" json:"auth"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (PushSubscription) TableName() string {
	return "push_subscriptions"
}
