package handlers

import (
	"errors"
	"math"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"security-solution/config"
	"security-solution/models"
)

// GetPlatformDonationInfo returns the platform's own bank account details.
// Public endpoint — no auth required.
func GetPlatformDonationInfo(c *gin.Context) {
	bankName := os.Getenv("PLATFORM_BANK_NAME")
	accountNumber := os.Getenv("PLATFORM_ACCOUNT_NUMBER")
	accountName := os.Getenv("PLATFORM_ACCOUNT_NAME")

	if bankName == "" || accountNumber == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "platform donation account is not configured",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"bankName":      bankName,
		"accountNumber": accountNumber,
		"accountName":   accountName,
		"purpose":       "Maintenance of Nativity Guard platform — servers, storage, development",
	})
}

// CreatePlatformDonation records a pending donation to Nativity Guard.
// Public endpoint — any visitor may submit. A super admin confirms later.
func CreatePlatformDonation(c *gin.Context) {
	var input struct {
		DonorName   string  `json:"donorName" binding:"required"`
		DonorEmail  string  `json:"donorEmail"`
		DonorPhone  string  `json:"donorPhone"`
		Amount      float64 `json:"amount" binding:"required"`
		Method      string  `json:"method"`
		ReferenceID string  `json:"referenceId"`
		Message     string  `json:"message"`
		IsPublic    *bool   `json:"isPublic"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if math.IsNaN(input.Amount) || math.IsInf(input.Amount, 0) || input.Amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "amount must be positive"})
		return
	}

	var donorID *uuid.UUID
	if userValue, exists := c.Get("user"); exists {
		if userObj, ok := userValue.(*models.User); ok && userObj != nil {
			donorID = &userObj.ID
		}
	}

	method := input.Method
	if method == "" {
		method = "bank_transfer"
	}

	isPublic := true
	if input.IsPublic != nil {
		isPublic = *input.IsPublic
	}

	donation := models.PlatformDonation{
		DonorID:     donorID,
		DonorName:   input.DonorName,
		DonorEmail:  input.DonorEmail,
		DonorPhone:  input.DonorPhone,
		Amount:      input.Amount,
		Currency:    "NGN",
		Method:      method,
		ReferenceID: input.ReferenceID,
		Message:     input.Message,
		IsPublic:    isPublic,
		Status:      "pending",
	}

	if err := config.DB.Create(&donation).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record donation"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Thank you. Your donation is recorded and awaiting confirmation.",
		"donation": gin.H{
			"id":     donation.ID,
			"amount": donation.Amount,
			"status": donation.Status,
		},
	})
}

// ListPlatformDonations — super admin only. Full list with donor detail.
func ListPlatformDonations(c *gin.Context) {
	if _, ok := requireSuperAdminUser(c); !ok {
		return
	}

	var donations []models.PlatformDonation
	if err := config.DB.Order("created_at DESC").Limit(500).Find(&donations).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch donations"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"donations": donations})
}

// ConfirmPlatformDonation — super admin only. Marks a donation as confirmed.
func ConfirmPlatformDonation(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid donation ID"})
		return
	}

	userObj, ok := requireSuperAdminUser(c)
	if !ok {
		return
	}

	var donation models.PlatformDonation
	var errStatus int
	var errMsg string
	if err := config.DB.Transaction(func(tx *gorm.DB) error {
		var locked models.PlatformDonation
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&locked, "id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				errStatus = http.StatusNotFound
				errMsg = "Donation not found"
			}
			return err
		}

		if locked.Status == "confirmed" {
			errStatus = http.StatusConflict
			errMsg = "Donation is already confirmed"
			return errors.New(errMsg)
		}
		if locked.Status != "pending" {
			errStatus = http.StatusBadRequest
			errMsg = "Donation is no longer pending"
			return errors.New(errMsg)
		}

		now := time.Now().UTC()
		locked.Status = "confirmed"
		locked.ConfirmedBy = &userObj.ID
		locked.ConfirmedAt = &now
		if err := tx.Save(&locked).Error; err != nil {
			return err
		}
		donation = locked
		return nil
	}); err != nil {
		if errStatus != 0 {
			c.JSON(errStatus, gin.H{"error": errMsg})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to confirm donation"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Donation confirmed",
		"donation": donation,
	})
}

func requireSuperAdminUser(c *gin.Context) (*models.User, bool) {
	userValue, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return nil, false
	}
	userObj, ok := userValue.(*models.User)
	if !ok || userObj == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user"})
		return nil, false
	}
	if !userObj.IsSuperAdmin && userObj.Role != "super_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Super admin access required"})
		return nil, false
	}
	return userObj, true
}

// ListPublicSupporters — public. Returns confirmed donations where IsPublic=true.
// Donor email and phone are never exposed. Names may be shown.
func ListPublicSupporters(c *gin.Context) {
	var donations []models.PlatformDonation
	if err := config.DB.
		Where("status = ? AND is_public = ?", "confirmed", true).
		Order("confirmed_at DESC").
		Limit(200).
		Find(&donations).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch supporters"})
		return
	}

	type Supporter struct {
		DonorName string    `json:"donorName"`
		Amount    float64   `json:"amount"`
		Message   string    `json:"message,omitempty"`
		CreatedAt time.Time `json:"createdAt"`
	}

	out := make([]Supporter, 0, len(donations))
	for _, d := range donations {
		out = append(out, Supporter{
			DonorName: d.DonorName,
			Amount:    d.Amount,
			Message:   d.Message,
			CreatedAt: d.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"supporters": out})
}
