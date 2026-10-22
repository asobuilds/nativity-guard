package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CommunicationRoomMember struct {
	ID        uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	RoomID    uuid.UUID      `gorm:"type:uuid;not null;index;uniqueIndex:idx_room_user" json:"roomId"`
	UserID    uuid.UUID      `gorm:"type:uuid;not null;index;uniqueIndex:idx_room_user" json:"userId"`
	Role      string         `gorm:"default:member" json:"role"`   // admin, member, observer
	Status    string         `gorm:"default:active" json:"status"` // active, invited, left, removed
	JoinedAt  *time.Time     `json:"joinedAt,omitempty"`
	LeftAt    *time.Time     `json:"leftAt,omitempty"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Room CommunicationRoom `gorm:"foreignKey:RoomID" json:"room,omitempty"`
	User User              `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (CommunicationRoomMember) TableName() string {
	return "communication_room_members"
}
