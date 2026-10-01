package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserPreferences holds display and notification preferences for a user.
// One row per user, created lazily on first read.
//
// The theme list and text-size list are validated in the handler; the
// frontend ships the same constants in lib/preferences.ts.
type UserPreferences struct {
	ID     uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex" json:"userId"`

	// Theme: forest | midnight | daylight | ocean | stone
	Theme string `gorm:"type:varchar(32);default:'forest'" json:"theme"`

	// TextSize: standard | large | xlarge | xxlarge
	TextSize string `gorm:"type:varchar(16);default:'standard'" json:"textSize"`

	NotifyAlerts           bool `gorm:"default:true" json:"notifyAlerts"`
	NotifyCaseUpdates      bool `gorm:"default:true" json:"notifyCaseUpdates"`
	NotifyCommunityReplies bool `gorm:"default:true" json:"notifyCommunityReplies"`

	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	User User `gorm:"foreignKey:UserID" json:"-"`
}

func (UserPreferences) TableName() string { return "user_preferences" }