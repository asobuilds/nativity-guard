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
	MinorStatus           string         `gorm:"type:varchar(16);default:'adult';index:idx_user_minor_status" json:"minorStatus"` // minor | adult
	DOBVerified           bool           `gorm:"default:false;index:idx_user_dob_verified" json:"dobVerified"`
	MinorExceptionGranted bool           `gorm:"default:false;index:idx_user_minor_exception" json:"minorExceptionGranted"`
	GuardianID            *uuid.UUID     `gorm:"type:uuid;index:idx_user_guardian" json:"guardianId,omitempty"`
	Impersonating *uuid.UUID     `gorm:"type:uuid" json:"impersonating,omitempty"`
	MedicalInfo   string         `gorm:"type:text" json:"medicalInfo,omitempty"` // encrypted at rest via BeforeSave/AfterFind
	AvatarPath    string         `gorm:"type:varchar(255)" json:"avatarPath,omitempty"`
	CoverPath     string         `gorm:"type:varchar(255)" json:"coverPath,omitempty"`
	// CoverPosition is the object-position anchor for the cover photo:
	// "top" (default), "center", or "bottom". Stored server-side so it
	// follows the account across devices.
	CoverPosition string         `gorm:"type:varchar(16);default:'top'" json:"coverPosition"`
	LastLogin     *time.Time     `json:"lastLogin,omitempty"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
	DeletionRequestedAt *time.Time `gorm:"index:idx_user_deletion_requested" json:"deletionRequestedAt,omitempty"`
}

func (User) TableName() string {
	return "users"
}

// BeforeSave encrypts MedicalInfo at rest. If ENCRYPTION_KEY is missing, the save
// fails rather than persisting plaintext. Values already carrying the encryption
// prefix are skipped, so this hook is idempotent.
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
// plaintext directly — on failure it returns the input unchanged, which
// leaves the encrypted value in place rather than failing the row load.
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