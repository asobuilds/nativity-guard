package services

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"security-solution/config"
	"security-solution/models"
)

const defaultResetTokenTTL = 15 * time.Minute

type PasswordResetService struct{}

func NewPasswordResetService() *PasswordResetService {
	return &PasswordResetService{}
}

// ResetNotifier is retained for compatibility with existing startup code.
type ResetNotifier interface {
	NotifyEmail(toEmail, toName, code string)
	NotifySMS(toPhone, code string)
}

var resetNotifier ResetNotifier

func SetResetNotifier(n ResetNotifier) {
	resetNotifier = n
}

// sha256Hex returns the SHA-256 hash of a string as hexadecimal.
func sha256Hex(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// removeNonDigits preserves compatibility with existing tests and utilities.
// It does not enable phone-number-based password resets.
func removeNonDigits(value string) string {
	var result strings.Builder

	for _, c := range value {
		if c >= '0' && c <= '9' {
			result.WriteRune(c)
		}
	}

	return result.String()
}

// RequestReset accepts an email address only.
//
// For unknown accounts, it returns no token and no error so the handler
// can return a generic response without revealing whether an account exists.
func (s *PasswordResetService) RequestReset(
	email string,
) (*models.PasswordReset, string, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	if email == "" || !strings.Contains(email, "@") {
		return nil, "", nil
	}

	var user models.User

	err := config.DB.
		Where("LOWER(email) = ?", email).
		First(&user).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, "", nil
	}

	if err != nil {
		return nil, "", err
	}

	if user.DeletedAt.Valid {
		return nil, "", nil
	}

	// Generate a cryptographically secure, 32-byte random token.
	randomBytes := make([]byte, 32)

	if _, err := rand.Read(randomBytes); err != nil {
		return nil, "", err
	}

	rawToken := hex.EncodeToString(randomBytes)

	// Store only the SHA-256 hash, never the raw token.
	record := models.PasswordReset{
		UserID:    user.ID,
		Token:     sha256Hex(rawToken),
		ExpiresAt: time.Now().UTC().Add(defaultResetTokenTTL),
		Used:      false,
	}

	if err := config.DB.Create(&record).Error; err != nil {
		return nil, "", err
	}

	// The raw token is returned to the handler for email delivery.
	// It must never be stored in the database or application logs.
	return &record, rawToken, nil
}

// ResetWithToken validates and consumes a reset token, changes the password,
// and revokes the user's existing access and refresh tokens.
func (s *PasswordResetService) ResetWithToken(
	rawToken string,
	newPassword string,
) error {
	if len(newPassword) < 8 {
		return errors.New("password must be at least 8 characters")
	}

	rawToken = strings.TrimSpace(rawToken)

	if rawToken == "" {
		return errors.New("invalid or expired reset token")
	}

	tokenHash := sha256Hex(rawToken)
	now := time.Now().UTC()

	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(newPassword),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return err
	}

	var user models.User

	err = config.DB.Transaction(func(tx *gorm.DB) error {
		var record models.PasswordReset

		err := tx.
			Where(
				"token = ? AND used = ? AND expires_at > ?",
				tokenHash,
				false,
				now,
			).
			First(&record).Error

		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("invalid or expired reset token")
			}

			return err
		}

		// Atomically mark the token as used. Only one request can
		// successfully consume this token.
		result := tx.
			Model(&models.PasswordReset{}).
			Where(
				"id = ? AND used = ? AND expires_at > ?",
				record.ID,
				false,
				time.Now().UTC(),
			).
			Update("used", true)

		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected != 1 {
			return errors.New("invalid or expired reset token")
		}

		if err := tx.
			Where("id = ?", record.UserID).
			First(&user).Error; err != nil {
			return errors.New("invalid or expired reset token")
		}

		if user.DeletedAt.Valid {
			return errors.New("invalid or expired reset token")
		}

		// Update the password using the bcrypt hash.
		result = tx.
			Model(&models.User{}).
			Where("id = ?", user.ID).
			Update("password", string(hashedPassword))

		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected != 1 {
			return errors.New("failed to update password")
		}

		return nil
	})

	if err != nil {
		return err
	}

	// Revoke existing access tokens so the user must authenticate again.
	if err := NewTokenService().RevokeAllForUser(
		user.ID,
		"password_reset",
	); err != nil {
		return err
	}

	// Revoke existing refresh tokens.
	if err := NewRefreshTokenService().RevokeAllForUser(user.ID); err != nil {
		return err
	}

	return nil
}
