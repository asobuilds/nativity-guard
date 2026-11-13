package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CitizenChatRoom is a geographic discussion room. Its place is resolved by
// the server; clients never submit state/LGA/locality names as authority.
type CitizenChatRoom struct {
	ID uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	GeoPlace `gorm:"embedded"`
	Status string `gorm:"type:varchar(16);not null;default:'active';index" json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (CitizenChatRoom) TableName() string { return "citizen_chat_rooms" }

type CitizenChatRoomMember struct {
	ID uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	RoomID uuid.UUID `gorm:"type:uuid;not null;index;uniqueIndex:idx_citizen_chat_member_active,where:deleted_at IS NULL" json:"roomId"`
	UserID uuid.UUID `gorm:"type:uuid;not null;index;uniqueIndex:idx_citizen_chat_member_active,where:deleted_at IS NULL" json:"userId"`
	Status string `gorm:"type:varchar(16);not null;default:'active';index" json:"status"`
	JoinedAt time.Time `json:"joinedAt"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (CitizenChatRoomMember) TableName() string { return "citizen_chat_room_members" }

type CitizenChatMessage struct {
	ID uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	RoomID uuid.UUID `gorm:"type:uuid;not null;index:idx_citizen_chat_message_order,priority:1" json:"roomId"`
	SenderID uuid.UUID `gorm:"type:uuid;not null;index" json:"senderId"`
	Content string `gorm:"type:text;not null" json:"content"`
	Status string `gorm:"type:varchar(16);not null;default:'published';index" json:"status"`
	CreatedAt time.Time `gorm:"index:idx_citizen_chat_message_order,priority:2,sort:desc" json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Sender User `gorm:"foreignKey:SenderID" json:"-"`
}

func (CitizenChatMessage) TableName() string { return "citizen_chat_messages" }

type CitizenChatBlock struct {
	ID uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	BlockerID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_citizen_chat_block_active,where:deleted_at IS NULL" json:"blockerId"`
	BlockedID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_citizen_chat_block_active,where:deleted_at IS NULL" json:"blockedId"`
	CreatedAt time.Time `json:"createdAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (CitizenChatBlock) TableName() string { return "citizen_chat_blocks" }

type CitizenChatReport struct {
	ID uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	MessageID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_citizen_chat_report_active,where:deleted_at IS NULL" json:"messageId"`
	ReporterID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_citizen_chat_report_active,where:deleted_at IS NULL" json:"reporterId"`
	Reason string `gorm:"type:varchar(2000);not null" json:"reason"`
	Status string `gorm:"type:varchar(16);not null;default:'pending';index" json:"status"`
	Resolution string `gorm:"type:varchar(16)" json:"resolution,omitempty"`
	ModeratorID *uuid.UUID `gorm:"type:uuid" json:"moderatorId,omitempty"`
	ModeratorNote string `gorm:"type:varchar(2000)" json:"moderatorNote,omitempty"`
	ResolvedAt *time.Time `json:"resolvedAt,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Message CitizenChatMessage `gorm:"foreignKey:MessageID" json:"-"`
}

func (CitizenChatReport) TableName() string { return "citizen_chat_reports" }
