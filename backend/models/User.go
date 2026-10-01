package models

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"security-solution/cryptoutil"
)

type User struct {
	ID            uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Email         string         `gorm:"unique;not null" json:"email"`
	Phone         string         `gorm:"unique" json:"phone"`
	FirstName     string         `gorm:"not null" json:"firstName"`
	LastName      string         `gorm:"not null" json:"lastName"`
	Password      string         `gorm:"not null" json:"-"`
	Role          string         `gorm:"default:citizen" json:"role"`
	UnitID        *uuid.UUID     `gorm:"type:uuid" json:"unitId,omitempty"`
	Status        string         `gorm:"default:pending" json:"status"`
	IsSuperAdmin          bool           `gorm:"default:false" json:"isSuperAdmin"`
	LocationSharingEnabled bool          `gorm:"default:false;index:idx_user_location_sharing" json:"locationSharingEnabled"`
	DateOfBirth           *time.Time     `gorm:"type:date;index:idx_user_dob" json:"dateOfBirth,omitempty"`
	MinorStatus           string         `gorm:"type:varchar(16);default:'adult';index:idx_user_minor_status" json:"minorStatus"`
	DOBVerified           bool           `gorm:"default:false;index:idx_user_dob_verified" json:"dobVerified"`
	MinorExceptionGranted bool           `gorm:"default:false;index:idx_user_minor_exception" json:"minorExceptionGranted"`
	GuardianID            *uuid.UUID     `gorm:"type:uuid;index:idx_user_guardian" json:"guardianId,omitempty"`
	Impersonating *uuid.UUID     `gorm:"type:uuid" json:"impersonating,omitempty"`
	MedicalInfo   string         `gorm:"type:text" json:"medicalInfo,omitempty"`
	AvatarPath    string         `gorm:"type:varchar(255)" json:"avatarPath,omitempty"`
	CoverPath     string         `gorm:"type:varchar(255)" json:"coverPath,omitempty"`

	// Cover photo focus — three server-side values so the adjustment
	// follows the account across devices.
	//   X, Y: object-position percentage 0–100. Y=50 is centered, which is
	//         the correct default for a typical portrait.
	//   Zoom: integer 100–300 (100 = 1.0×, 300 = 3.0×)
	CoverPositionX int `gorm:"default:50" json:"coverPositionX"`
	CoverPositionY int `gorm:"default:50" json:"coverPositionY"`
	CoverZoom      int `gorm:"default:100" json:"coverZoom"`

	LastLogin     *time.Time     `json:"lastLogin,omitempty"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
	DeletionRequestedAt *time.Time `gorm:"index:idx_user_deletion_requested" json:"deletionRequestedAt,omitempty"`
}

func (User) TableName() string {
	return "users"
}

func (u *User) BeforeSave(tx *gorm.DB) error {
	if u.MedicalInfo == "" {
		return nil
	}
	if strings.HasPrefix(u.MedicalInfo, cryptoutil.EncryptedPrefix) {
		return nil
	}
	enc, err := cryptoutil.Encrypt(u.MedicalInfo)
	if err != nil {
		return err
	}
	u.MedicalInfo = enc
	return nil
}

// AfterFind decrypts MedicalInfo on read. cryptoutil.Decrypt returns the
// plaintext directly — on failure it returns the input unchanged, leaving
// the encrypted value in place rather than failing the row load.
func (u *User) AfterFind(tx *gorm.DB) error {
	if u.MedicalInfo == "" {
		return nil
	}
	if !strings.HasPrefix(u.MedicalInfo, cryptoutil.EncryptedPrefix) {
		return nil
	}
	u.MedicalInfo = cryptoutil.Decrypt(u.MedicalInfo)
	return nil
}