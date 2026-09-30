package handlers

import (
    "fmt"
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"

    "security-solution/config"
    "security-solution/models"
    "security-solution/services"
)

// ClaimCase transfers a case to the caller's unit.
//
// Takeover rules:
//   - Same unit owns the case: idempotent success (no-op).
//   - Different unit owns it AND an officer is assigned: refuse.
//     Work in progress must not be stolen.
//   - Different unit owns it but no officer is assigned yet: allow
//     the takeover.
//   - A super_admin may also specify which unit should receive the case
//     via the request body's `unitId` field; other roles always use
//     their own unit.
//
// Every successful transfer (including the same-unit no-op) is written
// to the audit log as `case.transfer_claim`.
func ClaimCase(c *gin.Context) {
    idStr := c.Param("id")
    caseID, err := uuid.Parse(idStr)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid case ID"})
        return
    }

    userVal, exists := c.Get("user")
    if !exists {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
        return
    }
    userObj := userVal.(*models.User)
    if userObj.Role != "unit_admin" && userObj.Role != "super_admin" {
        c.JSON(http.StatusForbidden, gin.H{"error": "Only unit admins can claim a case"})
        return
    }
    if userObj.UnitID == nil && userObj.Role != "super_admin" {
        c.JSON(http.StatusForbidden, gin.H{"error": "Your account is not linked to a unit"})
        return
    }

    var input struct {
        UnitID string `json:"unitId"`
    }
    _ = c.ShouldBindJSON(&input)

    var claimUnit uuid.UUID
    if userObj.Role == "super_admin" && input.UnitID != "" {
        claimUnit, err = uuid.Parse(input.UnitID)
        if err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid unit ID"})
            return
        }
    } else {
        claimUnit = *userObj.UnitID
    }

    var caseObj models.Case
    if err := config.DB.First(&caseObj, "id = ?", caseID).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Case not found"})
        return
    }
    if caseObj.Status == "closed" {
        c.JSON(http.StatusConflict, gin.H{"error": "Case is already closed"})
        return
    }

    // Takeover guard.
    if caseObj.UnitID != uuid.Nil && caseObj.UnitID != claimUnit {
        if caseObj.AssignedTo != nil && userObj.Role != "super_admin" {
            c.JSON(http.StatusConflict, gin.H{
                "error": "Another unit is already working this case",
            })
            return
        }
    }

    var unit models.SecurityUnit
    if err := config.DB.First(&unit, "id = ?", claimUnit).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Unit not found"})
        return
    }

    previousOwner := caseObj.UnitID

    updates := map[string]interface{}{
        "unit_id": claimUnit,
    }
    if caseObj.Status == "pending" {
        updates["status"] = "assigned"
        updates["assigned_at"] = time.Now().UTC()
    }
    if err := config.DB.Model(&caseObj).Updates(updates).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to claim"})
        return
    }

    // Audit every transfer. Same-unit no-op still logs so the trail is complete.
    auditSvc := services.NewAuditService()
    _ = auditSvc.LogAction(
        userObj.ID,
        "case.transfer_claim",
        "case",
        caseID.String(),
        map[string]interface{}{"unitId": previousOwner},
        map[string]interface{}{"unitId": claimUnit, "claimedBy": userObj.ID},
        c.ClientIP(),
        c.Request.UserAgent(),
    )

    // Notify the reporter.
    config.DB.Create(&models.Notification{
        UserID:     caseObj.ReportedBy,
        Title:      "Case claimed",
        Message:    fmt.Sprintf("Case %s has been claimed by %s.", caseObj.TrackingID, unit.Name),
        Type:       "case_update",
        Status:     "unread",
        EntityType: "case",
        EntityID:   caseID,
        LinkTo:     "/cases/" + caseID.String(),
    })

    c.JSON(http.StatusOK, gin.H{
        "message": "Case claimed",
        "case":    caseObj,
    })
}