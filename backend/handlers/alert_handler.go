package handlers

import (
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"security-solution/config"
	"security-solution/models"
)

// CreateCommunityAlert creates a new community alert.
func CreateCommunityAlert(c *gin.Context) {
	var input struct {
		Title     string  `json:"title" binding:"required"`
		Content   string  `json:"content" binding:"required"`
		Type      string  `json:"type" binding:"required"`
		Severity  string  `json:"severity"`
		Location  string  `json:"location" binding:"required"`
		Latitude  float64 `json:"latitude" binding:"required"`
		Longitude float64 `json:"longitude" binding:"required"`
		Radius    float64 `json:"radius" binding:"required"`
		ExpiresAt string  `json:"expiresAt"`
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

	if userObj.Role != "super_admin" && userObj.Role != "unit_admin" && userObj.Role != "officer" {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to create alerts"})
		return
	}

	if strings.TrimSpace(input.Severity) == "" {
		input.Severity = "medium"
	}
	if input.Radius < 1 {
		input.Radius = 10
	}

	var expiresAt *time.Time
	if input.ExpiresAt != "" {
		if parsed, err := time.Parse(time.RFC3339, input.ExpiresAt); err == nil {
			expiresAt = &parsed
		}
	}

	alert := models.CommunityAlert{
		Title:     input.Title,
		Content:   input.Content,
		Type:      strings.ToLower(strings.TrimSpace(input.Type)),
		Severity:  input.Severity,
		Location:  input.Location,
		Latitude:  input.Latitude,
		Longitude: input.Longitude,
		Radius:    input.Radius,
		Status:    "active",
		ExpiresAt: expiresAt,
		CreatedBy: userObj.ID,
	}

	if err := config.DB.Create(&alert).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create alert"})
		return
	}

	go notifyAlertSubscribers(alert)

	c.JSON(http.StatusCreated, gin.H{
		"message": "Alert created successfully",
		"alert":   alert,
	})
}

// GetCommunityAlerts gets all active alerts visible to the caller.
func GetCommunityAlerts(c *gin.Context) {
	var alerts []models.CommunityAlert
	query := config.DB.Preload("Author").Where("status = ?", "active")

	user, exists := c.Get("user")
	if exists {
		userObj := user.(*models.User)
		if userObj.Role == "citizen" {
			if userObj.UnitID != nil {
				query = query.Where(
					"severity != ? OR location ILIKE ?",
					"critical",
					"%"+userObj.UnitID.String()+"%",
				)
			} else {
				query = query.Where("severity != ?", "critical")
			}
		}
	}

	query = query.Where("expires_at IS NULL OR expires_at > ?", time.Now())

	if err := query.Order("severity DESC, created_at DESC").Find(&alerts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch alerts"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"alerts": alerts})
}

// GetAlertByID gets a specific alert.
func GetAlertByID(c *gin.Context) {
	id := c.Param("id")
	alertID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid alert ID"})
		return
	}

	var alert models.CommunityAlert
	if err := config.DB.Preload("Author").Preload("Confirmer").First(&alert, "id = ?", alertID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Alert not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"alert": alert})
}

// ConfirmAlert confirms an alert (admin only).
func ConfirmAlert(c *gin.Context) {
	id := c.Param("id")
	alertID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid alert ID"})
		return
	}

	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj := user.(*models.User)

	if userObj.Role != "super_admin" && userObj.Role != "unit_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only admins can confirm alerts"})
		return
	}

	var alert models.CommunityAlert
	if err := config.DB.First(&alert, "id = ?", alertID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Alert not found"})
		return
	}

	now := time.Now()
	alert.ConfirmedBy = &userObj.ID
	alert.ConfirmedAt = &now
	config.DB.Save(&alert)

	c.JSON(http.StatusOK, gin.H{
		"message": "Alert confirmed",
		"alert":   alert,
	})
}

// notifyAlertSubscribers runs after an alert is created. Rules:
//
//   - The subscriber's category list must include the alert's type
//     (or "all").
//   - If the subscriber has a location (lat/lng != 0), the alert must
//     be within the subscriber's radius.
//   - If the subscriber has turned alert notifications off in their
//     preferences, they are skipped.
func notifyAlertSubscribers(alert models.CommunityAlert) {
	var subs []models.AlertSubscription
	if err := config.DB.Where("is_active = ?", true).Find(&subs).Error; err != nil {
		return
	}

	alertType := strings.ToLower(strings.TrimSpace(alert.Type))

	for _, sub := range subs {
		if !subscriptionMatchesCategory(sub.Type, alertType) {
			continue
		}

		hasLocation := sub.Latitude != 0 || sub.Longitude != 0
		if hasLocation {
			radius := sub.Radius
			if radius <= 0 {
				radius = 10
			}
			dist := haversineKm(sub.Latitude, sub.Longitude, alert.Latitude, alert.Longitude)
			if dist > radius {
				continue
			}
		}

		// Check the subscriber's notification preferences. A missing
		// preferences row is treated as "notifications on".
		var prefs models.UserPreferences
		if err := config.DB.Where("user_id = ?", sub.UserID).First(&prefs).Error; err == nil {
			if !prefs.NotifyAlerts {
				continue
			}
		}

		notification := models.Notification{
			UserID:  sub.UserID,
			Title:   "🚨 " + alert.Title,
			Message: alert.Content,
			Type:    "alert",
			Status:  "unread",
		}
		config.DB.Create(&notification)
	}
}

func subscriptionMatchesCategory(subTypeCSV, alertType string) bool {
	if strings.TrimSpace(subTypeCSV) == "" {
		return false
	}
	for _, raw := range strings.Split(subTypeCSV, ",") {
		cat := strings.ToLower(strings.TrimSpace(raw))
		if cat == "all" || cat == alertType {
			return true
		}
	}
	return false
}

// haversineKm is the great-circle distance between two points in kilometres.
func haversineKm(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusKm = 6371.0

	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(dLon/2)*math.Sin(dLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusKm * c
}