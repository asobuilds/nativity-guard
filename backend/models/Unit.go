package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type SecurityUnit struct {
	ID                      uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Name                    string     `gorm:"not null;unique" json:"name"`
	Type                    string     `gorm:"not null" json:"type"`
	Latitude                float64    `json:"latitude"`
	Longitude               float64    `json:"longitude"`
	OperationalRadius       float64    `gorm:"default:10" json:"operationalRadius"` // in km
	State                   string     `gorm:"type:varchar(120);index:idx_security_unit_state" json:"state"`
	LGA                     string     `gorm:"type:varchar(120);index:idx_security_unit_lga" json:"lga"` // Local Government Area
	Ward                    string     `gorm:"type:varchar(120);index" json:"ward,omitempty"`
	FormationDate           *time.Time `gorm:"type:date" json:"formationDate,omitempty"`
	TotalMembers            int        `gorm:"default:0" json:"totalMembers"`
	City                    string     `json:"city"`
	CoverageArea            string     `json:"coverageArea"`
	ContactPerson           string     `json:"contactPerson"`
	ContactPhone            string     `json:"contactPhone"`
	ContactEmail            string     `json:"contactEmail"`
	RegistrationNumber      string     `gorm:"unique" json:"registrationNumber"`
	Status                  string     `gorm:"default:active" json:"status"`
	IsVerified              bool       `gorm:"default:false;index" json:"isVerified"`
	VerificationStatus      string     `gorm:"default:pending;index" json:"verificationStatus"` // pending, under_review, verified, rejected
	VerificationSubmittedAt *time.Time `json:"verificationSubmittedAt,omitempty"`
	VerifiedAt              *time.Time `json:"verifiedAt,omitempty"`
	VerifiedBy              *uuid.UUID `gorm:"type:uuid" json:"verifiedBy,omitempty"`
	VerificationNotes       string     `gorm:"type:text" json:"verificationNotes,omitempty"`
	AdminCount              int        `gorm:"default:0" json:"adminCount"`
	HostUserID              *uuid.UUID `gorm:"type:uuid" json:"hostUserId,omitempty"`
	FormationStatus         string     `gorm:"not null;default:forming;index:idx_security_unit_formation_status" json:"formationStatus"`
	MemberCount             int        `gorm:"default:0" json:"memberCount"`
	BrandName               string     `gorm:"type:varchar(120)" json:"brandName,omitempty"`
	BrandLogoURL            string     `gorm:"type:text" json:"brandLogoUrl,omitempty"`
	BrandCoverURL           string     `gorm:"type:text" json:"brandCoverUrl,omitempty"`

	// Registration form, Section B — commander.
	CommanderName            string `gorm:"type:varchar(160)" json:"commanderName,omitempty"`
	CommanderNIN             string `gorm:"type:varchar(20);index" json:"commanderNin,omitempty"`
	CommanderPhoneAlt        string `gorm:"type:varchar(20)" json:"commanderPhoneAlt,omitempty"`
	CommanderOccupation      string `gorm:"type:varchar(120)" json:"commanderOccupation,omitempty"`
	CommanderPriorExperience string `gorm:"type:text" json:"commanderPriorExperience,omitempty"`

	// Registration form, Section C — operational profile.
	HasUniform         bool   `gorm:"default:false" json:"hasUniform"`
	UniformDescription string `gorm:"type:text" json:"uniformDescription,omitempty"`
	ShiftPattern       string `gorm:"type:varchar(32)" json:"shiftPattern,omitempty"` // "day" | "night" | "24h"
	PermittedTools     string `gorm:"type:text" json:"permittedTools,omitempty"`      // JSON array of strings
	WeaponsRegistered  bool   `gorm:"default:false" json:"weaponsRegistered"`

	// Registration form, Section D — traditional endorsement.
	KindredHeadName  string `gorm:"type:varchar(160)" json:"kindredHeadName,omitempty"`
	KindredHeadPhone string `gorm:"type:varchar(20)" json:"kindredHeadPhone,omitempty"`
	WardHeadName     string `gorm:"type:varchar(160)" json:"wardHeadName,omitempty"`
	WardHeadPhone    string `gorm:"type:varchar(20)" json:"wardHeadPhone,omitempty"`

	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Officers []Officer `gorm:"foreignKey:UnitID" json:"officers,omitempty"`
	Cases    []Case    `gorm:"foreignKey:UnitID" json:"cases,omitempty"`
}

func (SecurityUnit) TableName() string {
	return "security_units"
}
