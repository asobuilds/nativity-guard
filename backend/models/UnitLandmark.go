package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UnitLandmark struct {
	ID        uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UnitID    uuid.UUID      `gorm:"type:uuid;not null;index" json:"unitId"`
	Name      string         `gorm:"not null" json:"name"`
	Type      string         `gorm:"not null;default:'landmark'" json:"type"` // "Safe House", "Known Hotspot", "Checkpoint", "Post"
	Latitude  float64        `gorm:"not null" json:"latitude"`
	Longitude float64        `gorm:"not null" json:"longitude"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Unit SecurityUnit `gorm:"foreignKey:UnitID" json:"unit,omitempty"`
}

func (UnitLandmark) TableName() string {
	return "unit_landmarks"
}
