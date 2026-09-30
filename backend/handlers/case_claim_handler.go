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

// ClaimCase lets a unit_admin lock a case to their unit.
//
// Rule: a case can only be claimed once. Once claimed, later attempts
// get 409 and the case is invisible as available in the queue.
//
// This differs from the existing officer-assignment flow (AssignCase):
// assignment puts an officer on the case inside a unit; claiming locks
// the CASE to a UNIT so other units cannot take it.
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
        UnitID string `json:"unitId"` // super_admin may specify a unit; others use their own
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

    // Already claimed by another unit?
    if caseObj.UnitID != uuid.Nil && caseObj.UnitID != claimUnit {
        c.JSON(http.StatusConflict, gin.H{"error": "Another unit has already taken this case"})
        return
    }

    // Look up the unit for its name (used in the notification body).
    var unit models.SecurityUnit
    if err := config.DB.First(&unit, "id = ?", claimUnit).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Unit not found"})
        return
    }

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