package services

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"security-solution/config"
	"security-solution/models"
)

type PasswordResetService struct{}

func NewPasswordResetService() *PasswordResetService {
	return &PasswordResetService{}
}

// ResetNotifier abstracts out-of-band delivery so the service does not
// depend on any specific transport. The handlers package assigns a
// concrete implementation at startup; if nil, delivery is skipped.
type ResetNotifier interface {
	NotifyEmail(toEmail, toName, code string)
	NotifySMS(toPhone, code string)
}

var resetNotifier ResetNotifier

// SetResetNotifier installs the out-of-band delivery implementation.
// Call once at startup (e.g. from main or an init in the handlers pkg).
func SetResetNotifier(n ResetNotifier) {
	resetNotifier = n
}

const defaultResetTokenTTL = 1 * time.Hour

// sha256Hex returns the SHA-256 hex digest of raw. Only the hash is stored
// in the database; the raw token is transmitted to the user once and never
// persisted.
func sha256Hex(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// generateResetCode generates a cryptographically secure six-digit code
// in the range 100000–999999.
func generateResetCode() (string, error) {
	const minCode = 100000
	const codeRange = 900000

	n, err := rand.Int(rand.Reader, big.NewInt(codeRange))
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%06d", minCode+int(n.Int64())), nil
}

// RequestReset initiates a password reset for the account identified by
// email or phone. Unknown identifiers are handled silently to prevent
// account enumeration.
//
// The raw six-digit code is returned to the caller for compatibility.
// Only its SHA-256 hash is persisted. Notifications are dispatched
// asynchronously through the configured notifier.
func (s *PasswordResetService) RequestReset(identifier string) (*models.PasswordReset, string, error) {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return nil, "", nil
	}

	var user models.User
	query := config.DB

	if strings.Contains(identifier, "@") {
		query = query.Where("LOWER(email) = ?", strings.ToLower(identifier))
	} else {
		digits := removeNonDigits(identifier)
		if digits == "" {
			return nil, "", nil
		}
		query = query.Where("phone = ?", digits)
	}

	if err := query.First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", nil
		}
		return nil, "", err
	}

	// Do not issue reset codes for deactivated accounts.
	if user.DeletedAt.Valid {
		return nil, "", nil
	}

	rawCode, err := generateResetCode()
	if err != nil {
		return nil, "", err
	}

	record := models.PasswordReset{
		UserID:    user.ID,
		Token:     sha256Hex(rawCode),
		ExpiresAt: time.Now().UTC().Add(defaultResetTokenTTL),
		Used:      false,
	}

	if err := config.DB.Create(&record).Error; err != nil {
		return nil, "", err
	}

	// Keep the existing SMS and email notification behavior.
	// Missing contact details are skipped.
	go func() {
		if resetNotifier == nil {
			return
		}

		if user.Phone != "" {
			resetNotifier.NotifySMS(user.Phone, rawCode)
		}
		if user.Email != "" {
			resetNotifier.NotifyEmail(user.Email, user.FirstName, rawCode)
		}
	}()

	return &record, rawCode, nil
}

// ResetWithToken validates a raw reset code, hashes the new password,
// consumes the reset record once, and revokes existing sessions and
// refresh tokens.
func (s *PasswordResetService) ResetWithToken(rawToken, newPassword string) error {
	if len(newPassword) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	if strings.TrimSpace(rawToken) == "" {
		return errors.New("invalid or expired reset token")
	}

	now := time.Now().UTC()
	hash := sha256Hex(rawToken)

	var record models.PasswordReset
	if err := config.DB.
		Where("token = ? AND used = ? AND expires_at > ?", hash, false, now).
		First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("invalid or expired reset token")
		}
		return err
	}

	var user models.User
	if err := config.DB.First(&user, "id = ?", record.UserID).Error; err != nil {
		return errors.New("invalid or expired reset token")
	}

	if user.DeletedAt.Valid {
		return errors.New("account is deactivated")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(newPassword),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return err
	}

	tx := config.DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Consume the token conditionally. If another request already used it,
	// or it expired while this request was running, the transaction fails.
	result := tx.Model(&models.PasswordReset{}).
		Where(
			"id = ? AND used = ? AND expires_at > ?",
			record.ID,
			false,
			time.Now().UTC(),
		).
		Update("used", true)

	if result.Error != nil {
		tx.Rollback()
		return result.Error
	}
	if result.RowsAffected != 1 {
		tx.Rollback()
		return errors.New("invalid or expired reset token")
	}

	if err := tx.Model(&models.User{}).
		Where("id = ?", user.ID).
		Update("password", string(hashedPassword)).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Revoke existing sessions and refresh tokens so the user must
	// authenticate again on all devices after resetting the password.
	tokenSvc := NewTokenService()
	if err := tokenSvc.RevokeAllForUser(user.ID, "password_reset"); err != nil {
		tx.Rollback()
		return err
	}

	refreshSvc := NewRefreshTokenService()
	if err := refreshSvc.RevokeAllForUser(user.ID); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	return nil
}

// removeNonDigits strips every non-digit character from a phone string.
func removeNonDigits(s string) string {
	b := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= '0' && c <= '9' {
			b = append(b, c)
		}
	}
	return string(b)
}
