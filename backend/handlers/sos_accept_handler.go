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

const maxCriticalResponders = 3

// AcceptSOS lets a unit_admin claim an SOS.
//
// Normal severity: first accept locks the alert; later accepts get 409.
// Critical severity: up to maxCriticalResponders accepts are allowed;
//   DispatchState flips to multi_responder after the second.
//
// The responder unit is derived from the caller's User.UnitID — never
// from the request body, so a user cannot claim on behalf of another unit.
func AcceptSOS(c *gin.Context) {
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

    if userObj.Role != "unit_admin" && userObj.Role != "super_admin" {
        c.JSON(http.StatusForbidden, gin.H{"error": "Only unit admins can accept an SOS"})
        return
    }
    if userObj.UnitID == nil {
        c.JSON(http.StatusForbidden, gin.H{"error": "Your account is not linked to a unit"})
        return
    }
    unitID := *userObj.UnitID

    var sos models.SOSAlert
    if err := config.DB.First(&sos, "id = ?", sosID).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "SOS not found"})
        return
    }
    if sos.Status == "resolved" || sos.Status == "cancelled" {
        c.JSON(http.StatusConflict, gin.H{"error": "SOS is already closed"})
        return
    }

    // Existing responder for this unit?
    var existing models.SOSResponder
    if err := config.DB.Where("sos_id = ? AND unit_id = ?", sosID, unitID).First(&existing).Error; err == nil {
        c.JSON(http.StatusConflict, gin.H{"error": "Your unit is already responding to this SOS"})
        return
    }

    // Enforce the dispatch-state lock.
    var responderCount int64
    config.DB.Model(&models.SOSResponder{}).Where("sos_id = ?", sosID).Count(&responderCount)

    if sos.Severity != "critical" && responderCount > 0 {
        c.JSON(http.StatusConflict, gin.H{"error": "Another unit is already responding"})
        return
    }
    if sos.Severity == "critical" && responderCount >= maxCriticalResponders {
        c.JSON(http.StatusConflict, gin.H{"error": "Maximum responders already accepted"})
        return
    }

    // Create the responder row.
    responder := models.SOSResponder{
        SOSID:      sosID,
        UnitID:     unitID,
        Role:       "primary",
        AcceptedBy: userObj.ID,
        AcceptedAt: time.Now().UTC(),
    }
    if responderCount > 0 {
        responder.Role = "support"
    }
    if err := config.DB.Create(&responder).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record acceptance"})
        return
    }

    // Update the SOS row.
    updates := map[string]interface{}{
        "dispatch_state": "locked",
        "status":         "dispatched",
    }
    if responderCount == 0 {
        updates["accepted_by_unit_id"] = unitID
        updates["accepted_at"] = time.Now().UTC()
    }
    if sos.Severity == "critical" && responderCount+1 >= 2 {
        updates["dispatch_state"] = "multi_responder"
    }
    if err := config.DB.Model(&sos).Updates(updates).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update SOS"})
        return
    }

    // Notify the reporter.
    config.DB.Create(&models.Notification{
        UserID:     sos.UserID,
        Title:      "Unit responding",
        Message:    fmt.Sprintf("A unit has accepted your SOS. Responders are on the way."),
        Type:       "sos_update",
        Status:     "unread",
        EntityType: "sos",
        EntityID:   sosID,
        LinkTo:     "/sos/" + sosID.String(),
    })

    c.JSON(http.StatusOK, gin.H{
        "message":        "SOS accepted",
        "responder":      responder,
        "dispatchState":  updates["dispatch_state"],
        "responderCount": responderCount + 1,
    })
}

// ReleaseSOS removes the caller's unit from the responders list.
// If the releasing unit was the primary (and there are no other
// responders), the SOS returns to the open pool.
func ReleaseSOS(c *gin.Context) {
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
    if userObj.UnitID == nil {
        c.JSON(http.StatusForbidden, gin.H{"error": "Your account is not linked to a unit"})
        return
    }
    unitID := *userObj.UnitID

    var responder models.SOSResponder
    if err := config.DB.Where("sos_id = ? AND unit_id = ?", sosID, unitID).First(&responder).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Your unit is not responding to this SOS"})
        return
    }
    if err := config.DB.Delete(&responder).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to release"})
        return
    }

    var remaining int64
    config.DB.Model(&models.SOSResponder{}).Where("sos_id = ?", sosID).Count(&remaining)

    updates := map[string]interface{}{
        "dispatch_state": "open",
    }
    if remaining == 0 {
        updates["status"] = "pending"
        updates["accepted_by_unit_id"] = nil
        updates["accepted_at"] = nil
    } else if remaining == 1 {
        updates["dispatch_state"] = "locked"
    }
    config.DB.Model(&models.SOSAlert{}).Where("id = ?", sosID).Updates(updates)

    c.JSON(http.StatusOK, gin.H{
        "message":        "Released",
        "responderCount": remaining,
        "dispatchState":  updates["dispatch_state"],
    })
}