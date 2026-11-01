package models

import (
	"time"

	"github.com/google/uuid"
)

type AgencyTenant struct {
	ID           uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	AgencyType   string    `gorm:"type:varchar(16);not null;index:idx_agency_tenant_type" json:"agencyType"`
	State        string    `gorm:"type:varchar(120);not null;index:idx_agency_tenant_state" json:"state"`
	LGA          string    `gorm:"type:varchar(120);not null;index:idx_agency_tenant_lga" json:"lga"`
	DivisionName string    `gorm:"type:varchar(160);not null" json:"divisionName"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

func (AgencyTenant) TableName() string {
	return "agency_tenants"
}
