package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"security-solution/config"
	"security-solution/models"
	"security-solution/services"
)

// ListUnitVerifications — super-admin view of unit registrations awaiting
// verification.
func ListUnitVerifications(c *gin.Context) {
	if _, ok := callerIsSuperAdmin(c); !ok {
		return
	}

	status := strings.TrimSpace(c.DefaultQuery("status", "pending"))
	q := config.DB.Model(&models.SecurityUnit{})
	if status != "" && status != "all" {
		q = q.Where("verification_status = ?", status)
	}

	var items []models.SecurityUnit
	if err := q.Order("created_at desc").Limit(200).Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load units"})
		return
	}

	out := make([]gin.H, 0, len(items))
	for _, u := range items {
		out = append(out, gin.H{
			"id":                      u.ID,
			"name":                    u.Name,
			"type":                    u.Type,
			"state":                   u.State,
			"lga":                     u.LGA,
			"ward":                    u.Ward,
			"city":                    u.City,
			"registrationNumber":      u.RegistrationNumber,
			"contactPerson":           u.ContactPerson,
			"contactPhone":            u.ContactPhone,
			"contactEmail":            u.ContactEmail,
			"commanderName":           u.CommanderName,
			"commanderNin":            u.CommanderNIN,
			"commanderPhoneAlt":       u.CommanderPhoneAlt,
			"kindredHeadName":         u.KindredHeadName,
			"kindredHeadPhone":        u.KindredHeadPhone,
			"wardHeadName":            u.WardHeadName,
			"wardHeadPhone":           u.WardHeadPhone,
			"isVerified":              u.IsVerified,
			"verificationStatus":      u.VerificationStatus,
			"verificationSubmittedAt": u.VerificationSubmittedAt,
			"verifiedAt":              u.VerifiedAt,
			"verificationNotes":       u.VerificationNotes,
			"createdAt":               u.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"units": out})
}

// ApproveUnit marks a unit registration as verified.
func ApproveUnit(c *gin.Context) {
	admin, ok := callerIsSuperAdmin(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid unit id"})
		return
	}

	var unit models.SecurityUnit
	if err := config.DB.First(&unit, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "unit not found"})
		return
	}
	if unit.IsVerified && unit.VerificationStatus == "verified" {
		c.JSON(http.StatusConflict, gin.H{"error": "unit is already verified"})
		return
	}

	now := time.Now().UTC()
	unit.IsVerified = true
	unit.VerificationStatus = "verified"
	unit.VerifiedAt = &now
	unit.VerifiedBy = &admin.ID
	unit.VerificationNotes = ""

	if err := config.DB.Save(&unit).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to approve unit"})
		return
	}

	_ = services.NewAuditService().LogAction(
		admin.ID,
		"unit.approve",
		"security_unit",
		unit.ID.String(),
		nil,
		map[string]string{"status": "verified", "unitName": unit.Name},
		c.ClientIP(),
		c.Request.UserAgent(),
	)

	c.JSON(http.StatusOK, gin.H{
		"message": "unit verified",
		"unit": gin.H{
			"id":                 unit.ID,
			"isVerified":         unit.IsVerified,
			"verificationStatus": unit.VerificationStatus,
			"verifiedAt":         unit.VerifiedAt,
		},
	})
}

// RejectUnit marks a unit registration as rejected with a reason.
func RejectUnit(c *gin.Context) {
	admin, ok := callerIsSuperAdmin(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid unit id"})
		return
	}

	var input struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "reason is required"})
		return
	}
	input.Reason = strings.TrimSpace(input.Reason)
	if input.Reason == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "reason cannot be empty"})
		return
	}
	if len(input.Reason) > 1000 {
		input.Reason = input.Reason[:1000]
	}

	var unit models.SecurityUnit
	if err := config.DB.First(&unit, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "unit not found"})
		return
	}

	now := time.Now().UTC()
	unit.IsVerified = false
	unit.VerificationStatus = "rejected"
	unit.VerifiedAt = &now
	unit.VerifiedBy = &admin.ID
	unit.VerificationNotes = input.Reason

	if err := config.DB.Save(&unit).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reject unit"})
		return
	}

	_ = services.NewAuditService().LogAction(
		admin.ID,
		"unit.reject",
		"security_unit",
		unit.ID.String(),
		nil,
		map[string]string{"status": "rejected", "reason": input.Reason, "unitName": unit.Name},
		c.ClientIP(),
		c.Request.UserAgent(),
	)

	c.JSON(http.StatusOK, gin.H{
		"message": "unit rejected",
		"unit": gin.H{
			"id":                 unit.ID,
			"isVerified":         unit.IsVerified,
			"verificationStatus": unit.VerificationStatus,
			"verificationNotes":  unit.VerificationNotes,
			"verifiedAt":         unit.VerifiedAt,
		},
	})
}

// MarkUnitUnderReview sets a unit's verification_status to under_review.
func MarkUnitUnderReview(c *gin.Context) {
	admin, ok := callerIsSuperAdmin(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid unit id"})
		return
	}

	var unit models.SecurityUnit
	if err := config.DB.First(&unit, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "unit not found"})
		return
	}
	if unit.VerificationStatus == "verified" {
		c.JSON(http.StatusConflict, gin.H{"error": "unit is already verified"})
		return
	}

	unit.VerificationStatus = "under_review"
	if err := config.DB.Save(&unit).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to mark under review"})
		return
	}

	_ = services.NewAuditService().LogAction(
		admin.ID,
		"unit.under_review",
		"security_unit",
		unit.ID.String(),
		nil,
		map[string]string{"status": "under_review", "unitName": unit.Name},
		c.ClientIP(),
		c.Request.UserAgent(),
	)

	c.JSON(http.StatusOK, gin.H{
		"message":            "unit marked under review",
		"verificationStatus": unit.VerificationStatus,
	})
}