package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// LGAGroupMember records that a security unit has joined the convergence group
// for the local government area it is registered in (RES-012, decision D-015).
//
// Readership of the LGA channel is already open to any verified unit member in
// the LGA; this record makes the group explicit and gives it a roster. Posting
// is a separate right — head admins only — and is deliberately NOT granted by
// this row, so joining can never be a path to posting.
type LGAGroupMember struct {
	ID uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`

	State string `gorm:"type:varchar(120);not null" json:"state"`
	LGA   string `gorm:"type:varchar(120);not null" json:"lga"`

	UnitID       uuid.UUID `gorm:"type:uuid;not null" json:"unitId"`
	UnitName     string    `gorm:"type:varchar(200);not null" json:"unitName"`
	JoinedByUser uuid.UUID `gorm:"type:uuid;not null" json:"joinedByUser"`

	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (LGAGroupMember) TableName() string { return "lga_group_members" }
