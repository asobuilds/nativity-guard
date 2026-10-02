package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"security-solution/config"
	"security-solution/models"
	"security-solution/services"
)

// currentUserID returns the authenticated user's ID. Prefers the "user_id"
// value if the middleware set it (older code path); falls back to the
// "user" object that AuthMiddleware always sets.
func currentUserID(c *gin.Context) (uuid.UUID, bool) {
	if value, exists := c.Get("user_id"); exists {
		switch v := value.(type) {
		case uuid.UUID:
			return v, true
		case string:
			if id, err := uuid.Parse(v); err == nil {
				return id, true
			}
		}
	}
	if value, exists := c.Get("user"); exists {
		if user, ok := value.(*models.User); ok && user != nil {
			return user.ID, true
		}
	}
	return uuid.Nil, false
}

// UploadIdentityDocument accepts a document image or PDF for identity
// verification and stores it under the private `gov_ids/` prefix.
func UploadIdentityDocument(c *gin.Context) {
	value, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	user, ok := value.(*models.User)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authenticated user"})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}

	storageSvc := services.NewFileStorageService()
	stored, err := storageSvc.Save(file, "gov_ids")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"documentPath": stored.RelativePath,
		"hash":         stored.Hash,
		"size":         stored.Size,
		"uploadedBy":   user.ID,
	})
}

// SubmitIdentityVerification creates or resubmits an identity verification.
func SubmitIdentityVerification(c *gin.Context) {
	value, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	user, ok := value.(*models.User)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authenticated user"})
		return
	}

	var input struct {
		DocumentType   string `json:"documentType" binding:"required"`
		DocumentNumber string `json:"documentNumber" binding:"required"`
		DocumentURL    string `json:"documentUrl" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "documentType, documentNumber and documentUrl are required",
		})
		return
	}

	input.DocumentType = strings.TrimSpace(input.DocumentType)
	input.DocumentNumber = strings.TrimSpace(input.DocumentNumber)
	input.DocumentURL = strings.TrimSpace(input.DocumentURL)

	allowed := map[string]bool{
		"nin":             true,
		"voters_card":     true,
		"drivers_license": true,
		"passport":        true,
	}
	if !allowed[input.DocumentType] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "documentType must be one of: nin, voters_card, drivers_license, passport",
		})
		return
	}

	if !strings.HasPrefix(input.DocumentURL, "gov_ids/") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid documentUrl"})
		return
	}

	if input.DocumentType == "nin" {
		if len(input.DocumentNumber) != 11 || strings.ContainsFunc(input.DocumentNumber, func(r rune) bool { return r < '0' || r > '9' }) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "A NIN must be exactly 11 digits"})
			return
		}
	} else {
		if len(input.DocumentNumber) < 5 || len(input.DocumentNumber) > 40 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "documentNumber must be 5-40 characters"})
			return
		}
	}

	hash := sha256.Sum256([]byte(input.DocumentNumber))
	hashHex := hex.EncodeToString(hash[:])

	var existing models.IdentityVerification
	err := config.DB.Where("user_id = ?", user.ID).First(&existing).Error

	if err == nil {
		switch existing.Status {
		case "verified":
			c.JSON(http.StatusConflict, gin.H{"error": "identity is already verified"})
			return
		case "pending":
			c.JSON(http.StatusConflict, gin.H{"error": "identity verification is already pending review"})
			return
		}
		existing.DocumentType = input.DocumentType
		existing.DocumentNumberHash = hashHex
		existing.DocumentURL = input.DocumentURL
		existing.Status = "pending"
		existing.RejectionReason = ""
		existing.VerifiedBy = nil
		existing.VerifiedAt = nil
		existing.SubmittedAt = time.Now().UTC()
		if err := config.DB.Save(&existing).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save verification"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message": "identity verification resubmitted",
			"verification": gin.H{
				"id":           existing.ID,
				"status":       existing.Status,
				"documentType": existing.DocumentType,
				"submittedAt":  existing.SubmittedAt,
			},
		})
		return
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check existing verification"})
		return
	}

	verification := models.IdentityVerification{
		ID:                 uuid.New(),
		UserID:             user.ID,
		DocumentType:       input.DocumentType,
		DocumentNumberHash: hashHex,
		DocumentURL:        input.DocumentURL,
		Status:             "pending",
		SubmittedAt:        time.Now().UTC(),
	}
	if err := config.DB.Create(&verification).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to submit verification"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "identity verification submitted",
		"verification": gin.H{
			"id":           verification.ID,
			"status":       verification.Status,
			"documentType": verification.DocumentType,
			"submittedAt":  verification.SubmittedAt,
		},
	})
}

// GetMyIdentityVerification returns the authenticated user's own state.
func GetMyIdentityVerification(c *gin.Context) {
	value, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	user, ok := value.(*models.User)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authenticated user"})
		return
	}

	var v models.IdentityVerification
	if err := config.DB.Where("user_id = ?", user.ID).First(&v).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusOK, gin.H{"status": "not_submitted"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load verification"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":          v.Status,
		"documentType":    v.DocumentType,
		"submittedAt":     v.SubmittedAt,
		"verifiedAt":      v.VerifiedAt,
		"rejectionReason": v.RejectionReason,
	})
}

// ListIdentityVerifications — super-admin view. Document paths are turned
// into short-lived presigned URLs.
func ListIdentityVerifications(c *gin.Context) {
	if _, ok := callerIsSuperAdmin(c); !ok {
		return
	}

	status := strings.TrimSpace(c.DefaultQuery("status", "pending"))
	q := config.DB.Model(&models.IdentityVerification{}).Preload("User")
	if status != "" && status != "all" {
		q = q.Where("status = ?", status)
	}

	var items []models.IdentityVerification
	if err := q.Order("submitted_at desc").Limit(200).Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load verifications"})
		return
	}

	ctx := c.Request.Context()
	out := make([]gin.H, 0, len(items))
	for _, it := range items {
		documentUrl := ""
		if it.DocumentURL != "" && services.R2Configured() {
			if r2, err := services.NewR2Service(); err == nil {
				if u, err := r2.PresignGet(ctx, it.DocumentURL, 10*time.Minute); err == nil {
					documentUrl = u
				}
			}
		}
		out = append(out, gin.H{
			"id":              it.ID,
			"userId":          it.UserID,
			"userEmail":       it.User.Email,
			"userFirstName":   it.User.FirstName,
			"userLastName":    it.User.LastName,
			"documentType":    it.DocumentType,
			"documentUrl":     documentUrl,
			"status":          it.Status,
			"submittedAt":     it.SubmittedAt,
			"verifiedAt":      it.VerifiedAt,
			"rejectionReason": it.RejectionReason,
		})
	}

	c.JSON(http.StatusOK, gin.H{"verifications": out})
}

// ApproveIdentityVerification marks a submission as verified.
func ApproveIdentityVerification(c *gin.Context) {
	admin, ok := callerIsSuperAdmin(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid verification id"})
		return
	}

	var v models.IdentityVerification
	if err := config.DB.First(&v, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "verification not found"})
		return
	}
	if v.Status == "verified" {
		c.JSON(http.StatusConflict, gin.H{"error": "identity is already verified"})
		return
	}

	now := time.Now().UTC()
	v.Status = "verified"
	v.VerifiedBy = &admin.ID
	v.VerifiedAt = &now
	v.RejectionReason = ""

	if err := config.DB.Save(&v).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to approve verification"})
		return
	}

	_ = services.NewAuditService().LogAction(
		admin.ID,
		"identity.approve",
		"identity_verification",
		v.ID.String(),
		nil,
		map[string]string{"status": "verified", "userId": v.UserID.String()},
		c.ClientIP(),
		c.Request.UserAgent(),
	)

	c.JSON(http.StatusOK, gin.H{
		"message": "identity verified",
		"verification": gin.H{
			"id":         v.ID,
			"status":     v.Status,
			"verifiedAt": v.VerifiedAt,
		},
	})
}

// RejectIdentityVerification marks a submission as rejected with a reason.
func RejectIdentityVerification(c *gin.Context) {
	admin, ok := callerIsSuperAdmin(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid verification id"})
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
	if len(input.Reason) > 500 {
		input.Reason = input.Reason[:500]
	}

	var v models.IdentityVerification
	if err := config.DB.First(&v, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "verification not found"})
		return
	}

	now := time.Now().UTC()
	v.Status = "rejected"
	v.VerifiedBy = &admin.ID
	v.VerifiedAt = &now
	v.RejectionReason = input.Reason

	if err := config.DB.Save(&v).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reject verification"})
		return
	}

	_ = services.NewAuditService().LogAction(
		admin.ID,
		"identity.reject",
		"identity_verification",
		v.ID.String(),
		nil,
		map[string]string{"status": "rejected", "reason": input.Reason, "userId": v.UserID.String()},
		c.ClientIP(),
		c.Request.UserAgent(),
	)

	c.JSON(http.StatusOK, gin.H{
		"message": "identity rejected",
		"verification": gin.H{
			"id":              v.ID,
			"status":          v.Status,
			"rejectionReason": v.RejectionReason,
		},
	})
}