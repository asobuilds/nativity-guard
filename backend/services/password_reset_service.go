package services

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
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

const defaultResetTokenTTL = 1 * time.Hour

// sha256Hex returns the SHA-256 hex digest of raw.
// Only the hash is stored in the database.
func sha256Hex(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// generateResetToken creates a cryptographically secure, URL-safe token.
func generateResetToken() (string, error) {
	b := make([]byte, 32)

	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(b), nil
}

// RequestReset creates a password-reset token for an email or phone identifier.
// It does not reveal whether an account exists.
// The raw token is returned to the handler for email-link delivery.
// Only its SHA-256 hash is stored in the database.
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
			// Silent no-op: do not reveal whether the account exists.
			return nil, "", nil
		}
		return nil, "", err
	}

	rawToken, err := generateResetToken()
	if err != nil {
		return nil, "", err
	}

	record := models.PasswordReset{
		UserID:    user.ID,
		Token:     sha256Hex(rawToken),
		ExpiresAt: time.Now().UTC().Add(defaultResetTokenTTL),
		Used:      false,
	}

	if err := config.DB.Create(&record).Error; err != nil {
		return nil, "", err
	}

	return &record, rawToken, nil
}

// ResetWithToken validates a raw reset token, hashes the new password,
// marks the token as used, and revokes existing sessions and refresh tokens.
func (s *PasswordResetService) ResetWithToken(rawToken, newPassword string) error {
	if strings.TrimSpace(rawToken) == "" {
		return errors.New("invalid or expired reset token")
	}

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

	if err := tx.Model(&models.User{}).
		Where("id = ?", user.ID).
		Update("password", string(hashedPassword)).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Model(&models.PasswordReset{}).
		Where("id = ? AND used = ?", record.ID, false).
		Update("used", true).Error; err != nil {
		tx.Rollback()
		return err
	}

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
