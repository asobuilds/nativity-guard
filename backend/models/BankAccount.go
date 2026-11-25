package models

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"security-solution/cryptoutil"
)

// BankAccount represents a unit's bank account for receiving funds.
// AccountNumber is never persisted in plaintext: BeforeSave encrypts it
// into EncryptedAccountNumber, and AfterFind decrypts it back.
type BankAccount struct {
	ID                  uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UnitID              uuid.UUID  `gorm:"type:uuid;not null;index:idx_bank_account_unit" json:"unitId"`
	BankName            string     `gorm:"not null" json:"bankName"`
	AccountNumber       string     `gorm:"-" json:"-"`             // plaintext in-memory only; encrypted via BeforeSave/AfterFind
	EncryptedAccountNumber string   `gorm:"type:text;not null;column:account_number" json:"-"` // encrypted, maps to DB column
	AccountName         string     `gorm:"not null" json:"accountName"`
	BankCode            string     `json:"bankCode"`
	Branch              string     `json:"branch"`
	SwiftCode           string     `json:"swiftCode"`
	IsVerified          bool       `gorm:"default:false" json:"isVerified"`
	IsDefault           bool       `gorm:"default:false" json:"isDefault"`
	IsPublic           bool       `gorm:"default:false;index:idx_bank_account_public" json:"isPublic"`
	DisplayOrder        int        `gorm:"default:0" json:"displayOrder"`
	Status              string     `gorm:"default:active" json:"status"`
	Notes               string     `json:"notes"`
	CreatedBy           uuid.UUID  `gorm:"type:uuid;not null" json:"createdBy"`
	CreatedAt           time.Time  `json:"createdAt"`
	UpdatedAt           time.Time  `json:"updatedAt"`
	DeletedAt           gorm.DeletedAt `gorm:"index" json:"-"`

	Unit    SecurityUnit `gorm:"foreignKey:UnitID" json:"unit,omitempty"`
	Creator User         `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
}

func (BankAccount) TableName() string {
	return "bank_accounts"
}

// BeforeSave encrypts AccountNumber into EncryptedAccountNumber before GORM writes to DB.
func (b *BankAccount) BeforeSave(tx *gorm.DB) error {
	if strings.HasPrefix(b.AccountNumber, cryptoutil.EncryptedPrefix) {
		// Already encrypted (e.g. loaded from DB, re-encrypt if plaintext leaked)
		b.EncryptedAccountNumber = b.AccountNumber
		return nil
	}
	if b.AccountNumber != "" {
		enc, err := cryptoutil.Encrypt(b.AccountNumber)
		if err != nil {
			return err
		}
		b.EncryptedAccountNumber = enc
	}
	return nil
}

// AfterFind decrypts EncryptedAccountNumber into AccountNumber after GORM reads from DB.
func (b *BankAccount) AfterFind(tx *gorm.DB) error {
	if b.EncryptedAccountNumber != "" {
		b.AccountNumber = cryptoutil.Decrypt(b.EncryptedAccountNumber)
	}
	return nil
}