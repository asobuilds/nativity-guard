package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"security-solution/config"
	"security-solution/models"
)

func transferSummary(request models.TransferRequest) gin.H {
	return gin.H{
		"id": request.ID, "targetId": request.TargetID, "targetType": request.TargetType,
		"fromUnitId": request.FromUnitID, "toUnitId": request.ToUnitID,
		"reason": request.Reason, "status": request.Status,
		"approvalCount": request.ApprovalCount, "requiredApprovals": request.RequiredApprovals,
		"createdAt": request.CreatedAt, "completedAt": request.CompletedAt,
	}
}

// RequestTransfer initiates a multi-signature transfer request
func RequestTransfer(c *gin.Context) {
	var input struct {
		TargetID   string `json:"targetId" binding:"required"`
		TargetType string `json:"targetType" binding:"required"` // suspect or case
		ToUnitID   string `json:"toUnitId" binding:"required"`
		Reason     string `json:"reason" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj := user.(*models.User)

	// Only admins can request transfers
	if userObj.Role != "super_admin" && userObj.Role != "unit_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only admins can request transfers"})
		return
	}

	if userObj.UnitID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "You are not assigned to a unit"})
		return
	}

	targetID, err := uuid.Parse(input.TargetID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid target ID"})
		return
	}

	toUnitID, err := uuid.Parse(input.ToUnitID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid target unit ID"})
		return
	}

	// Verify target exists and user has access
	var fromUnitID uuid.UUID
	if input.TargetType == "suspect" {
		var suspect models.Suspect
		if err := config.DB.First(&suspect, "id = ?", targetID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Suspect not found"})
			return
		}
		if suspect.UnitID == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Suspect has no assigned unit"})
			return
		}
		fromUnitID = *suspect.UnitID
	} else if input.TargetType == "case" {
		var caseObj models.Case
		if err := config.DB.First(&caseObj, "id = ?", targetID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Case not found"})
			return
		}
		fromUnitID = caseObj.UnitID
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid target type"})
		return
	}

	// Check if fromUnit matches user's unit
	if fromUnitID != *userObj.UnitID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only transfer from your unit"})
		return
	}

	// Check if toUnit exists
	var toUnit models.SecurityUnit
	if err := config.DB.First(&toUnit, "id = ?", toUnitID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Target unit not found"})
		return
	}

	if fromUnitID == toUnitID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot transfer to the same unit"})
		return
	}

	// Create transfer request with required approvals (3 by default)
	requiredApprovals := 3
	if userObj.Role == "super_admin" {
		requiredApprovals = 1 // Super admin can approve alone
	}

	transferRequest := models.TransferRequest{
		TargetID:          targetID,
		TargetType:        input.TargetType,
		FromUnitID:        fromUnitID,
		ToUnitID:          toUnitID,
		RequestedBy:       userObj.ID,
		Reason:            input.Reason,
		Status:            "pending",
		RequiredApprovals: requiredApprovals,
		ApprovalCount:     0,
	}

	if err := config.DB.Create(&transferRequest).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create transfer request"})
		return
	}

	// Get admins from the unit to notify
	var admins []models.User
	config.DB.Where("unit_id = ? AND role IN (?)", fromUnitID, []string{"unit_admin", "super_admin"}).Find(&admins)

	c.JSON(http.StatusCreated, gin.H{
		"message":           "Transfer request created successfully",
		"transferRequest":   transferRequest,
		"requiredApprovals": requiredApprovals,
		"admins":            len(admins),
	})
}

// ApproveTransfer allows an admin to approve a transfer (multi-signature)
func ApproveTransfer(c *gin.Context) {
	requestID := c.Param("id")
	id, err := uuid.Parse(requestID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request ID"})
		return
	}

	var input struct {
		Comment string `json:"comment"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj := user.(*models.User)

	// Only admins can approve
	if userObj.Role != "super_admin" && userObj.Role != "unit_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only admins can approve transfers"})
		return
	}

	var transferRequest models.TransferRequest
	if err := config.DB.First(&transferRequest, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Transfer request not found"})
		return
	}

	if transferRequest.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Transfer request is no longer pending"})
		return
	}

	// Check if user is from the same unit
	if userObj.UnitID == nil || *userObj.UnitID != transferRequest.FromUnitID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only approve transfers from your unit"})
		return
	}

	// Check if user already approved
	var existingApproval models.TransferApproval
	if err := config.DB.Where("transfer_id = ? AND approver_id = ?", transferRequest.ID, userObj.ID).First(&existingApproval).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "You have already approved this transfer"})
		return
	}

	// Create approval record
	approval := models.TransferApproval{
		TransferID:   transferRequest.ID,
		TransferType: transferRequest.TargetType,
		ApproverID:   userObj.ID,
		UnitID:       transferRequest.FromUnitID,
		Status:       "approved",
		Comment:      input.Comment,
	}

	if err := config.DB.Create(&approval).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record approval"})
		return
	}

	// Increment approval count
	transferRequest.ApprovalCount++
	config.DB.Save(&transferRequest)

	// Check if we have enough approvals
	if transferRequest.ApprovalCount >= transferRequest.RequiredApprovals {
		// Transfer is approved!
		transferRequest.Status = "approved"
		now := time.Now()
		transferRequest.ApprovedAt = &now
		config.DB.Save(&transferRequest)

		// Execute the actual transfer based on type
		if transferRequest.TargetType == "suspect" {
			var suspect models.Suspect
			if err := config.DB.First(&suspect, "id = ?", transferRequest.TargetID).Error; err == nil {
				suspect.UnitID = &transferRequest.ToUnitID
				suspect.TransferStatus = "approved"
				config.DB.Save(&suspect)
			}
		} else if transferRequest.TargetType == "case" {
			var caseObj models.Case
			if err := config.DB.First(&caseObj, "id = ?", transferRequest.TargetID).Error; err == nil {
				caseObj.UnitID = transferRequest.ToUnitID
				caseObj.Status = "transferred"
				config.DB.Save(&caseObj)
			}
		}

		// Mark as completed
		transferRequest.Status = "completed"
		now2 := time.Now()
		transferRequest.CompletedAt = &now2
		config.DB.Save(&transferRequest)

		c.JSON(http.StatusOK, gin.H{
			"message":           "Transfer fully approved and executed!",
			"transferRequest":   transferRequest,
			"approvalCount":     transferRequest.ApprovalCount,
			"requiredApprovals": transferRequest.RequiredApprovals,
			"transferCompleted": true,
		})
		return
	}

	// Not fully approved yet
	c.JSON(http.StatusOK, gin.H{
		"message":            "Transfer approved! Waiting for more approvals.",
		"transferRequest":    transferRequest,
		"approvalCount":      transferRequest.ApprovalCount,
		"requiredApprovals":  transferRequest.RequiredApprovals,
		"remainingApprovals": transferRequest.RequiredApprovals - transferRequest.ApprovalCount,
		"transferCompleted":  false,
	})
}

// RejectTransfer rejects a transfer request
func RejectTransfer(c *gin.Context) {
	requestID := c.Param("id")
	id, err := uuid.Parse(requestID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request ID"})
		return
	}

	var input struct {
		Comment string `json:"comment" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj := user.(*models.User)

	// Only admins can reject
	if userObj.Role != "super_admin" && userObj.Role != "unit_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only admins can reject transfers"})
		return
	}

	var transferRequest models.TransferRequest
	if err := config.DB.First(&transferRequest, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Transfer request not found"})
		return
	}

	if transferRequest.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Transfer request is no longer pending"})
		return
	}

	// Check if user is from the same unit
	if userObj.UnitID == nil || *userObj.UnitID != transferRequest.FromUnitID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only reject transfers from your unit"})
		return
	}

	// Create rejection record
	approval := models.TransferApproval{
		TransferID:   transferRequest.ID,
		TransferType: transferRequest.TargetType,
		ApproverID:   userObj.ID,
		UnitID:       transferRequest.FromUnitID,
		Status:       "rejected",
		Comment:      input.Comment,
	}

	if err := config.DB.Create(&approval).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record rejection"})
		return
	}

	// Mark transfer as rejected
	transferRequest.Status = "rejected"
	config.DB.Save(&transferRequest)

	c.JSON(http.StatusOK, gin.H{
		"message":         "Transfer rejected",
		"transferRequest": transferRequest,
	})
}

// GetTransferRequests gets all transfer requests for a unit
func GetTransferRequests(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj := user.(*models.User)

	if userObj.Role != "unit_admin" && userObj.Role != "super_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only admins can view transfers"})
		return
	}
	if userObj.Role != "super_admin" && userObj.UnitID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "You are not assigned to a unit"})
		return
	}

	var requests []models.TransferRequest
	query := config.DB.Order("created_at desc").Limit(100)
	if userObj.Role != "super_admin" {
		query = query.Where("from_unit_id = ?", userObj.UnitID)
	}
	if err := query.Find(&requests).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch transfer requests"})
		return
	}

	rows := make([]gin.H, 0, len(requests))
	for _, request := range requests { rows = append(rows, transferSummary(request)) }
	c.JSON(http.StatusOK, gin.H{"transferRequests": rows})
}

// GetTransferApprovals gets all approvals for a transfer request
func GetTransferApprovals(c *gin.Context) {
	userValue, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj := userValue.(*models.User)
	if userObj.Role != "unit_admin" && userObj.Role != "super_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only admins can view transfer approvals"})
		return
	}
	requestID := c.Param("id")
	id, err := uuid.Parse(requestID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request ID"})
		return
	}

	var request models.TransferRequest
	if err := config.DB.First(&request, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Transfer request not found"})
		return
	}
	if userObj.Role != "super_admin" && (userObj.UnitID == nil || *userObj.UnitID != request.FromUnitID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only view approvals from your unit"})
		return
	}
	var approvals []models.TransferApproval
	if err := config.DB.Where("transfer_id = ?", id).Find(&approvals).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch approvals"})
		return
	}

	rows := make([]gin.H, 0, len(approvals))
	for _, approval := range approvals {
		rows = append(rows, gin.H{"id": approval.ID, "approverId": approval.ApproverID,
			"status": approval.Status, "comment": approval.Comment, "createdAt": approval.CreatedAt})
	}
	c.JSON(http.StatusOK, gin.H{"approvals": rows})
}
