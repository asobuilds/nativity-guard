package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type Notification struct {
	ID        uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID      `gorm:"type:uuid;not null;index:idx_notification_user_created,priority:1;index:idx_notification_user_status,priority:1" json:"userId"`
	Title     string         `gorm:"not null" json:"title"`
	Message   string         `gorm:"type:text;not null" json:"message"`
	Type      string         `json:"type"`
	Status    string         `gorm:"default:unread;index:idx_notification_user_status,priority:2" json:"status"`
	EntityType string        `gorm:"type:varchar(24);index:idx_notification_entity_type" json:"entityType,omitempty"` // "sos" | "case" | "unit"
	EntityID   uuid.UUID     `gorm:"type:uuid;index:idx_notification_entity_id" json:"entityId,omitempty"`
	LinkTo     string        `gorm:"type:varchar(160)" json:"linkTo,omitempty"` // client path, e.g. "/sos/<id>"
	CreatedAt time.Time      `gorm:"index:idx_notification_user_created,priority:2,sort:desc" json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (Notification) TableName() string {
	return "notifications"
}
