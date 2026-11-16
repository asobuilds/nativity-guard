package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"security-solution/config"
	"security-solution/models"
	"security-solution/services"
)

type assignCaseRequest struct {
	OfficerID uuid.UUID `json:"officerId" binding:"required"`
	Role      string    `json:"role"`
}

// AssignCase assigns a case to an officer and records the assignment.
func AssignCase(c *gin.Context) {
	caseID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid case ID",
		})
		return
	}

	accessLevel := c.GetString("case_access_level")
	if accessLevel != "super_admin" && accessLevel != "head_admin" && accessLevel != "assigned_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "only admins may assign officers to a case"})
		c.Abort()
		return
	}

	var req assignCaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "officerId is required",
		})
		return
	}

	role := req.Role
	if role == "" {
		role = "primary"
	}

	var caseRecord models.Case
	if err := config.DB.First(&caseRecord, "id = ?", caseID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "case not found",
		})
		return
	}

	var officer models.Officer
	if err := config.DB.First(&officer, "id = ?", req.OfficerID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "officer not found",
		})
		return
	}

	if officer.UnitID != caseRecord.UnitID {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "officer does not belong to the case unit",
		})
		return
	}

	var assignment models.CaseOfficer

	err = config.DB.Where(
		"case_id = ? AND officer_id = ?",
		caseID,
		req.OfficerID,
	).First(&assignment).Error

	if err == nil {
		assignment.Role = role

		if err := config.DB.Save(&assignment).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to update assignment",
			})
			return
		}
	} else {
		assignment = models.CaseOfficer{
			CaseID:    caseID,
			OfficerID: req.OfficerID,
			Role:      role,
		}

		if err := config.DB.Create(&assignment).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to create assignment",
			})
			return
		}
	}

	now := assignment.CreatedAt

	if caseRecord.AssignedTo == nil || role == "primary" {
		caseRecord.AssignedTo = &req.OfficerID
		caseRecord.AssignedAt = &now

		if caseRecord.Status == "pending" {
			caseRecord.Status = "assigned"
		}

if err := config.DB.Save(&caseRecord).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "assignment created but case update failed",
		})
		return
	}
	}

	if actorValue, exists := c.Get("user"); exists {
		if actor, ok := actorValue.(*models.User); ok && actor != nil {
			auditSvc := services.NewAuditService()
			_ = auditSvc.LogAction(
				actor.ID,
				"case.assign",
				"case",
				caseID.String(),
				map[string]interface{}{
					"officerId": caseRecord.AssignedTo,
					"role":      role,
				},
				map[string]interface{}{
					"officerId": &req.OfficerID,
					"role":      role,
				},
				c.ClientIP(),
				c.Request.UserAgent(),
			)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "case assigned successfully",
		"assignment": assignment,
		"case":       caseRecord,
	})
}

// GetCaseAssignments returns all officers assigned to a case.
func GetCaseAssignments(c *gin.Context) {
	caseID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid case ID",
		})
		return
	}

	var assignments []models.CaseOfficer

	if err := config.DB.
		Where("case_id = ?", caseID).
		Order("created_at ASC").
		Find(&assignments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to retrieve case assignments",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"assignments": assignments,
	})
}
