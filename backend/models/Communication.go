package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

// CommunicationRoom represents a communication room for a unit
type CommunicationRoom struct {
	ID          uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UnitID      uuid.UUID      `gorm:"type:uuid;not null" json:"unitId"`
	Name        string         `gorm:"not null" json:"name"`
	Type        string         `gorm:"default:unit" json:"type"` // unit, emergency, private, command
	Status      string         `gorm:"default:active" json:"status"`
	IsEncrypted bool           `gorm:"default:true" json:"isEncrypted"`
	Priority    string         `gorm:"default:normal" json:"priority"` // normal, high, emergency
	CreatedBy   uuid.UUID      `gorm:"type:uuid;not null" json:"createdBy"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	Unit    SecurityUnit `gorm:"foreignKey:UnitID" json:"unit,omitempty"`
	Creator User         `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
}

func (CommunicationRoom) TableName() string {
	return "communication_rooms"
}

// CommunicationMessage represents a message in a room
type CommunicationMessage struct {
	ID          uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	RoomID      uuid.UUID      `gorm:"type:uuid;not null" json:"roomId"`
	SenderID    uuid.UUID      `gorm:"type:uuid;not null" json:"senderId"`
	Message     string         `gorm:"type:text;not null" json:"message"`
	MessageType string         `gorm:"default:text" json:"messageType"` // text, voice, image, location, emergency, file
	Status      string         `gorm:"default:sent" json:"status"`      // sent, delivered, read
	Priority    string         `gorm:"default:normal" json:"priority"`  // normal, high, emergency
	IsBroadcast bool           `gorm:"default:false" json:"isBroadcast"`
	IsEmergency bool           `gorm:"default:false" json:"isEmergency"`
	Reactions   string         `gorm:"type:text" json:"reactions"` // JSON array of reactions
	FileURL     string         `json:"fileUrl"`
	FileName    string         `json:"fileName"`
	FileSize    int64          `json:"fileSize"`
	ReadAt      *time.Time     `json:"readAt"`
	DeliveredAt *time.Time     `json:"deliveredAt"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	Room   CommunicationRoom `gorm:"foreignKey:RoomID" json:"room,omitempty"`
	Sender User              `gorm:"foreignKey:SenderID" json:"sender,omitempty"`
}

func (CommunicationMessage) TableName() string {
	return "communication_messages"
}

// VoiceCall represents a voice call between officers
type VoiceCall struct {
	ID          uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	RoomID      uuid.UUID      `gorm:"type:uuid;not null" json:"roomId"`
	CallerID    uuid.UUID      `gorm:"type:uuid;not null" json:"callerId"`
	ReceiverID  *uuid.UUID     `gorm:"type:uuid" json:"receiverId,omitempty"`
	CallType    string         `gorm:"default:group" json:"callType"`   // group, private, emergency
	Status      string         `gorm:"default:initiated" json:"status"` // initiated, ringing, connected, ended, missed
	StartedAt   *time.Time     `json:"startedAt"`
	EndedAt     *time.Time     `json:"endedAt"`
	Duration    int            `json:"duration"` // in seconds
	IsEncrypted bool           `gorm:"default:true" json:"isEncrypted"`
	Quality     string         `gorm:"default:good" json:"quality"` // excellent, good, fair, poor
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	Room     CommunicationRoom `gorm:"foreignKey:RoomID" json:"room,omitempty"`
	Caller   User              `gorm:"foreignKey:CallerID" json:"caller,omitempty"`
	Receiver User              `gorm:"foreignKey:ReceiverID" json:"receiver,omitempty"`
}

func (VoiceCall) TableName() string {
	return "voice_calls"
}

// CommunicationSync tracks offline message sync for rural areas
type CommunicationSync struct {
	ID           uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID       uuid.UUID      `gorm:"type:uuid;not null" json:"userId"`
	RoomID       uuid.UUID      `gorm:"type:uuid;not null" json:"roomId"`
	LastSyncAt   time.Time      `json:"lastSyncAt"`
	SyncStatus   string         `gorm:"default:pending" json:"syncStatus"` // pending, syncing, completed
	PendingCount int            `gorm:"default:0" json:"pendingCount"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`

	User User              `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Room CommunicationRoom `gorm:"foreignKey:RoomID" json:"room,omitempty"`
}

func (CommunicationSync) TableName() string {
	return "communication_syncs"
}
