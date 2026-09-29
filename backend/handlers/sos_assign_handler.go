package handlers

import (
    "fmt"
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"

    "security-solution/config"
    "security-solution/models"
)

// AssignSOS lets a super_admin force a specific unit to respond to an SOS.
// Distinct from AcceptSOS because:
//   - The caller is a super_admin (not necessarily a unit_admin)
//   - The unit is specified in the body, not derived from the caller
//   - This bypasses the severity lock — a super admin can stack responders
//     on a normal SOS if the situation demands it
//
// If the unit is already a responder, returns 409. If the SOS is closed,
// returns 409. Otherwise creates the responder row and updates dispatch state.
func AssignSOS(c *gin.Context) {
    idStr := c.Param("id")
    sosID, err := uuid.Parse(idStr)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid SOS ID"})
        return
    }

    userVal, exists := c.Get("user")
    if !exists {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
        return
    }
    userObj := userVal.(*models.User)
    if userObj.Role != "super_admin" {
        c.JSON(http.StatusForbidden, gin.H{"error": "Only super admins can force-assign"})
        return
    }

    var input struct {
        UnitID string `json:"unitId" binding:"required"`
        Role   string `json:"role"` // "primary" | "support", defaults to "support"
    }
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    unitID, err := uuid.Parse(input.UnitID)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid unit ID"})
        return
    }

    var sos models.SOSAlert
    if err := config.DB.First(&sos, "id = ?", sosID).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "SOS not found"})
        return
    }
    if sos.Status == "resolved" || sos.Status == "cancelled" {
        c.JSON(http.StatusConflict, gin.H{"error": "SOS is already closed"})
        return
    }

    var unit models.SecurityUnit
    if err := config.DB.First(&unit, "id = ?", unitID).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Unit not found"})
        return
    }

    var existing models.SOSResponder
    if err := config.DB.Where("sos_id = ? AND unit_id = ?", sosID, unitID).First(&existing).Error; err == nil {
        c.JSON(http.StatusConflict, gin.H{"error": "This unit is already responding"})
        return
    }

    var responderCount int64
    config.DB.Model(&models.SOSResponder{}).Where("sos_id = ?", sosID).Count(&responderCount)

    role := input.Role
    if role == "" {
        if responderCount == 0 {
            role = "primary"
        } else {
            role = "support"
        }
    }

    responder := models.SOSResponder{
        SOSID:      sosID,
        UnitID:     unitID,
        Role:       role,
        AcceptedBy: userObj.ID,
        AcceptedAt: time.Now().UTC(),
    }
    if err := config.DB.Create(&responder).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to assign"})
        return
    }

    updates := map[string]interface{}{
        "dispatch_state": "locked",
        "status":         "dispatched",
    }
    if responderCount == 0 {
        updates["accepted_by_unit_id"] = unitID
        updates["accepted_at"] = time.Now().UTC()
    }
    if responderCount+1 >= 2 {
        updates["dispatch_state"] = "multi_responder"
    }
    config.DB.Model(&sos).Updates(updates)

    // Notify the assigned unit's admins.
    var admins []models.User
    config.DB.Where("unit_id = ? AND role = ?", unitID, "unit_admin").Find(&admins)
    for _, admin := range admins {
        config.DB.Create(&models.Notification{
            UserID:     admin.ID,
            Title:      "SOS assigned by super admin",
            Message:    fmt.Sprintf("You have been assigned to SOS %s. Open to respond.", sosID.String()[:8]),
            Type:       "dispatch",
            Status:     "unread",
            EntityType: "sos",
            EntityID:   sosID,
            LinkTo:     "/sos/" + sosID.String(),
        })
    }

    // Notify the reporter.
    config.DB.Create(&models.Notification{
        UserID:     sos.UserID,
        Title:      "Responder assigned",
        Message:    fmt.Sprintf("%s is responding to your SOS.", unit.Name),
        Type:       "sos_update",
        Status:     "unread",
        EntityType: "sos",
        EntityID:   sosID,
        LinkTo:     "/sos/" + sosID.String(),
    })

    c.JSON(http.StatusOK, gin.H{
        "message":        "SOS assigned",
        "responder":      responder,
        "dispatchState":  updates["dispatch_state"],
        "responderCount": responderCount + 1,
    })
}