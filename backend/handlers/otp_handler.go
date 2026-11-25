package handlers

import (
	cryptorand "crypto/rand"
	"crypto/subtle"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"security-solution/config"
	"security-solution/models"
	"security-solution/services"
)

const (
	otpTTL            = 10 * time.Minute
	maxOTPAttempts    = 5
	otpQueuedMessage  = "If this account can receive an OTP, one has been sent."
	otpInvalidMessage = "Invalid or expired OTP"
)

type otpRequest struct {
	Type string `json:"type"`
}

type otpVerifyRequest struct {
	Code string `json:"code" binding:"required"`
}

func generateOTP() (string, error) {
	n, err := cryptorand.Int(cryptorand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func currentOTPUser(c *gin.Context) (*models.User, bool) {
	value, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return nil, false
	}

	user, ok := value.(*models.User)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authenticated user"})
		return nil, false
	}
	return user, true
}

func issueOTP(user *models.User, otpType string) error {
	code, err := generateOTP()
	if err != nil {
		return err
	}

	expiresAt := time.Now().UTC().Add(otpTTL)
	if err := config.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ? AND verified = ?", user.ID, false).
			Delete(&models.OTP{}).Error; err != nil {
			return err
		}
		return tx.Create(&models.OTP{
			UserID:    user.ID,
			Code:      code,
			Type:      otpType,
			Attempts:  0,
			ExpiresAt: expiresAt,
			Verified:  false,
		}).Error
	}); err != nil {
		return err
	}

	if user.Phone != "" {
		phone := user.Phone
		go func() {
			if err := services.SendOTPSMS(phone, code); err != nil {
				log.Printf("otp: SMS delivery failed: %v", err)
			}
		}()
	}

	return nil
}

func respondOTPQueued(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": otpQueuedMessage})
}

// SendOTP sends an OTP to the authenticated user's phone number.
func SendOTP(c *gin.Context) {
	user, ok := currentOTPUser(c)
	if !ok {
		return
	}

	var input otpRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := issueOTP(user, input.Type); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send OTP"})
		return
	}
	respondOTPQueued(c)
}

// VerifyOTP verifies an OTP for the authenticated user.
func VerifyOTP(c *gin.Context) {
	user, ok := currentOTPUser(c)
	if !ok {
		return
	}

	var input otpVerifyRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	invalid := false
	err := config.DB.Transaction(func(tx *gorm.DB) error {
		var otp models.OTP
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND verified = ?", user.ID, false).
			Order("created_at DESC").
			First(&otp).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			invalid = true
			return nil
		}
		if err != nil {
			return err
		}

		if time.Now().UTC().After(otp.ExpiresAt) {
			invalid = true
			return nil
		}
		if otp.Attempts >= maxOTPAttempts {
			invalid = true
			return nil
		}
		if subtle.ConstantTimeCompare([]byte(otp.Code), []byte(input.Code)) != 1 {
			otp.Attempts++
			if err := tx.Model(&otp).Update("attempts", otp.Attempts).Error; err != nil {
				return err
			}
			invalid = true
			return nil
		}

		return tx.Model(&otp).Update("verified", true).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify OTP"})
		return
	}
	if invalid {
		c.JSON(http.StatusBadRequest, gin.H{"error": otpInvalidMessage})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "OTP verified successfully"})
}

// ResendOTP replaces the authenticated user's pending OTP and sends a new one.
func ResendOTP(c *gin.Context) {
	user, ok := currentOTPUser(c)
	if !ok {
		return
	}

	var input otpRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := issueOTP(user, input.Type); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send OTP"})
		return
	}
	respondOTPQueued(c)
}
