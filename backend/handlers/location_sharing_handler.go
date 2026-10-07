package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"security-solution/config"
	"security-solution/models"
	"security-solution/services"
)

// GetLocationSharing returns the caller's location-sharing preference.
// GET /api/v1/location/sharing
func GetLocationSharing(c *gin.Context) {
	userValue, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	u, ok := userValue.(*models.User)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user identity"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"enabled": u.LocationSharingEnabled})
}

// UpdateLocationSharing toggles the caller's location-sharing preference.
// PUT /api/v1/location/sharing  body: {"enabled": bool}
func UpdateLocationSharing(c *gin.Context) {
	userValue, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	u, ok := userValue.(*models.User)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user identity"})
		return
	}

	var input struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: enabled (bool) is required"})
		return
	}

	oldValue := u.LocationSharingEnabled
	u.LocationSharingEnabled = input.Enabled

	if err := config.DB.Save(u).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update location sharing preference"})
		return
	}

    // Turning sharing off invalidates SOS sessions too. Turning it back on
    // must never revive an old session or publish a previous location.
    if !u.LocationSharingEnabled {
        if err := config.DB.Model(&models.SOSResponder{}).Where("assigned_user_id = ?", u.ID).
            Updates(map[string]interface{}{"tracking_session_id": nil, "location_id": nil}).Error; err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Sharing is disabled, but SOS session cleanup failed"})
            return
        }
    }

	// Audit: compliance trail for every toggle.
	auditSvc := services.NewAuditService()
	_ = auditSvc.LogAction(
		u.ID,
		"user.location_sharing_change",
		"user",
		u.ID.String(),
		map[string]interface{}{"enabled": oldValue},
		map[string]interface{}{"enabled": u.LocationSharingEnabled},
		c.ClientIP(),
		c.Request.UserAgent(),
	)

	c.JSON(http.StatusOK, gin.H{"enabled": u.LocationSharingEnabled})
}