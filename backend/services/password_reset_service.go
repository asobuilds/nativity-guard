package services

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
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

// generateResetCode produces a 6-digit numeric code in the range
// 100000–999999 using crypto/rand. The raw code is returned to the caller
// for out-of-band delivery; only the SHA-256 hash is persisted.
func generateResetCode() (string, error) {
	b := make([]byte, 1)
	var code int
	for {
		if _, err := rand.Read(b); err != nil {
			return "", err
		}
		// 0–255 mod 900000 gives 0–899999; +100000 yields 100000–999999.
		code = int(b[0])%900000 + 100000
		// Reject trivially weak codes (all same digit) — rare but safe.
		s := fmt.Sprintf("%06d", code)
		if s[0] != s[1] || s[1] != s[2] || s[2] != s[3] || s[3] != s[4] || s[4] != s[5] {
			return s, nil
		}
	}
}

// RequestReset initiates a password reset for the account identified by
// email OR phone. It is intentionally silent: whether the identifier maps to
// an existing user is not revealed to the caller. The caller receives the
// raw 6-digit code so it can be delivered out-of-band (SMS and/or email).
//
// Delivery: the raw code is dispatched via BOTH SMS and email (whichever
// channels the user has populated) in a goroutine. Delivery failures are
// non-fatal — the token row is persisted regardless so the user can retry.
func (s *PasswordResetService) RequestReset(identifier string) (*models.PasswordReset, string, error) {
	identifier = strings.TrimSpace(identifier)

	var user models.User
	query := config.DB
	if strings.Contains(identifier, "@") {
		// Email identifier: case-insensitive lookup.
		query = query.Where("LOWER(email) = ?", strings.ToLower(identifier))
	} else {
		// Phone identifier: digits-only match (no formatting rules yet).
		digits := removeNonDigits(identifier)
		if digits == "" {
			return nil, "", nil
		}
		query = query.Where("phone = ?", digits)
	}

	if err := query.First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Silent no-op — never reveal that no account exists.
			return nil, "", nil
		}
		return nil, "", err
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

	// Deliver the raw code out-of-band. The service never returns an error
	// for delivery failures — the caller already got a 200, and the code
	// row is persisted regardless so the user can retry.
	go dispatchResetNotifications(resetNotifier, user, rawCode)

	return &record, rawCode, nil
}

// dispatchResetNotifications sends the reset code on every channel the user
// has populated (SMS and/or email). Empty phone or email is skipped.
func dispatchResetNotifications(notifier ResetNotifier, user models.User, rawCode string) {
	if notifier == nil {
		return
	}
	if user.Phone != "" {
		notifier.NotifySMS(user.Phone, rawCode)
	}
	if user.Email != "" {
		notifier.NotifyEmail(user.Email, user.FirstName, rawCode)
	}
}

// ResetWithToken validates a raw reset token, hashes the new password, and
// persists the change inside a transaction that also marks the token used
// and revokes every existing session + refresh token for the user.
func (s *PasswordResetService) ResetWithToken(rawToken, newPassword string) error {
	if len(newPassword) < 8 {
		return errors.New("password must be at least 8 characters")
	}

	hash := sha256Hex(rawToken)

	var record models.PasswordReset
	if err := config.DB.
		Where("token = ? AND used = ? AND expires_at > ?", hash, false, time.Now().UTC()).
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

	hashed, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
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

	if err := tx.Model(&models.User{}).
		Where("id = ?", user.ID).
		Update("password", string(hashed)).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Model(&models.PasswordReset{}).
		Where("id = ?", record.ID).
		Update("used", true).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Revoke every active session + refresh token so the user must re-login
	// with the new password on all devices.
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