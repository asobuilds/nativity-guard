package models

import (
	"time"

	"github.com/google/uuid"
)

type AgencyObserver struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_agency_observer_user" json:"userId"`
	TenantID  uuid.UUID `gorm:"type:uuid;not null;index:idx_agency_observer_tenant" json:"tenantId"`
	Rank      string    `gorm:"type:varchar(80);not null;default:'Observer'" json:"rank"`
	CreatedAt time.Time `json:"createdAt"`
}

func (AgencyObserver) TableName() string {
	return "agency_observers"
}
