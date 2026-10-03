package models

import (
    "time"

    "github.com/google/uuid"
    "gorm.io/gorm"
)

// LGAMessage is a private notice posted to every verified security unit
// operating in the same LGA. It is a bulletin board, not a chat — one
// message per topic, no threading, no DMs.
//
// Access is decided entirely at read/write time from the caller's verified
// membership. The LGA and state stored on the row are copied from the
// caller's unit at the moment of posting — never accepted from the client.
type LGAMessage struct {
    ID    uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
    State string    `gorm:"type:varchar(120);not null;index:idx_lga_msg_state,priority:1" json:"state"`
    LGA   string    `gorm:"type:varchar(120);not null;index:idx_lga_msg_state,priority:2" json:"lga"`

    Category string `gorm:"type:varchar(24);not null;index:idx_lga_msg_category" json:"category"`
    Priority string `gorm:"type:varchar(16);default:'normal';index:idx_lga_msg_priority" json:"priority"`

    Title string `gorm:"type:varchar(200);not null" json:"title"`
    Body  string `gorm:"type:text;not null" json:"body"`

    AuthorUserID uuid.UUID `gorm:"type:uuid;not null;index:idx_lga_msg_author" json:"authorUserId"`
    AuthorUnitID uuid.UUID `gorm:"type:uuid;not null;index:idx_lga_msg_unit" json:"authorUnitId"`
    AuthorName   string    `gorm:"type:varchar(160);not null" json:"authorName"`
    AuthorUnit   string    `gorm:"type:varchar(200);not null" json:"authorUnit"`

    ExpiresAt *time.Time `gorm:"index:idx_lga_msg_expires" json:"expiresAt,omitempty"`

    CreatedAt time.Time      `json:"createdAt"`
    UpdatedAt time.Time      `json:"updatedAt"`
    DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (LGAMessage) TableName() string { return "lga_messages" }
