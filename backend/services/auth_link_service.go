package services

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"security-solution/config"
	"security-solution/models"

	"gorm.io/gorm"
)

const (
	authLinkPurposeSignIn = "sign_in"
	authLinkLifetime      = 15 * time.Minute
)

type AuthLinkService struct{}

func NewAuthLinkService() *AuthLinkService {
	return &AuthLinkService{}
}

func hashAuthLinkToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// CreateSignInLink creates a short-lived, single-use sign-in token.
func (s *AuthLinkService) CreateSignInLink(email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return "", errors.New("email is required")
	}

	var user models.User
	err := config.DB.Where("LOWER(email) = ?", email).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}

	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}

	rawToken := hex.EncodeToString(randomBytes)

	record := models.AuthLinkToken{
		UserID:    user.ID,
		TokenHash: hashAuthLinkToken(rawToken),
		Purpose:   authLinkPurposeSignIn,
		ExpiresAt: time.Now().UTC().Add(authLinkLifetime),
	}

	if err := config.DB.Create(&record).Error; err != nil {
		return "", err
	}

	return rawToken, nil
}

// RedeemSignInLink validates and consumes a sign-in token exactly once.
func (s *AuthLinkService) RedeemSignInLink(rawToken string) (*models.User, error) {
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return nil, errors.New("invalid or expired sign-in link")
	}

	tokenHash := hashAuthLinkToken(rawToken)
	now := time.Now().UTC()

	var user models.User

	err := config.DB.Transaction(func(tx *gorm.DB) error {
		var record models.AuthLinkToken

		err := tx.
			Where(
				"token_hash = ? AND purpose = ? AND used_at IS NULL AND expires_at > ?",
				tokenHash,
				authLinkPurposeSignIn,
				now,
			).
			First(&record).Error
		if err != nil {
			return errors.New("invalid or expired sign-in link")
		}

		result := tx.Model(&models.AuthLinkToken{}).
			Where(
				"id = ? AND used_at IS NULL AND expires_at > ?",
				record.ID,
				now,
			).
			Update("used_at", now)

		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.New("invalid or expired sign-in link")
		}

		if err := tx.First(&user, "id = ?", record.UserID).Error; err != nil {
			return errors.New("account not found")
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &user, nil
}
