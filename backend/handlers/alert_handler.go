package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"security-solution/config"
	"security-solution/models"
)

// CreateCommunityAlert creates a new community alert
func CreateCommunityAlert(c *gin.Context) {
	var input struct {
		Title     string  `json:"title" binding:"required"`
		Content   string  `json:"content" binding:"required"`
		Type      string  `json:"type" binding:"required"`
		Severity  string  `json:"severity" binding:"required"`
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

	// Only admins and officers can create alerts
	if userObj.Role != "super_admin" && userObj.Role != "unit_admin" && userObj.Role != "officer" {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to create alerts"})
		return
	}

	if input.Severity == "" {
		input.Severity = "medium"
	}

	var expiresAt *time.Time
	if input.ExpiresAt != "" {
		parsed, err := time.Parse(time.RFC3339, input.ExpiresAt)
		if err == nil {
			expiresAt = &parsed
		}
	}

	alert := models.CommunityAlert{
		Title:     input.Title,
		Content:   input.Content,
		Type:      input.Type,
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

	// Notify subscribers (async)
	go notifyAlertSubscribers(alert)

	c.JSON(http.StatusCreated, gin.H{
		"message": "Alert created successfully",
		"alert":   alert,
	})
}

// GetCommunityAlerts gets all active alerts
func GetCommunityAlerts(c *gin.Context) {
	var alerts []models.CommunityAlert
	query := config.DB.Preload("Author").Where("status = ?", "active")

	// Role-based filtering
	user, exists := c.Get("user")
	if exists {
		userObj := user.(*models.User)
		if userObj.Role == "citizen" {
			// Citizens see only non-critical alerts or alerts in their area
			// Guard against citizens with no unit — UnitID is a *uuid.UUID and
// can be nil, which would panic on .String(). Fall back to a
// location match against the user's city or skip the filter.
                if userObj.UnitID != nil {
                        // Guard against citizens with no unit — UnitID is a *uuid.UUID and
// can be nil, which would panic on .String(). Fall back to a
// location match against the user's city or skip the filter.
                if userObj.UnitID != nil {
                        query = query.Where("severity != ? OR location ILIKE ?", "critical", "%"+userObj.UnitID.String()+"%")
                } else {
                        query = query.Where("severity != ?", "critical")
                }
                } else {
                        query = query.Where("severity != ?", "critical")
                }
		}
	}

	// If expires_at is set, only show non-expired
	query = query.Where("expires_at IS NULL OR expires_at > ?", time.Now())

	if err := query.Order("severity DESC, created_at DESC").Find(&alerts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch alerts"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"alerts": alerts,
	})
}

// GetAlertByID gets a specific alert
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

	c.JSON(http.StatusOK, gin.H{
		"alert": alert,
	})
}

// ConfirmAlert confirms an alert (admin only)
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

// SubscribeToAlerts subscribes a user to alerts
func SubscribeToAlerts(c *gin.Context) {
	var input struct {
		Type     string  `json:"type" binding:"required"`
		Channel  string  `json:"channel"`
		Location string  `json:"location"`
		Radius   float64 `json:"radius"`
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

	if input.Channel == "" {
		input.Channel = "in_app"
	}

	subscription := models.AlertSubscription{
		UserID:   userObj.ID,
		UnitID:   userObj.UnitID,
		Type:     input.Type,
		Channel:  input.Channel,
		Location: input.Location,
		Radius:   input.Radius,
		IsActive: true,
	}

	if err := config.DB.Create(&subscription).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to subscribe"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":      "Subscribed successfully",
		"subscription": subscription,
	})
}

// GetAlertSubscriptions gets user's subscriptions
func GetAlertSubscriptions(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj := user.(*models.User)

	var subscriptions []models.AlertSubscription
	if err := config.DB.Where("user_id = ?", userObj.ID).Find(&subscriptions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch subscriptions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"subscriptions": subscriptions,
	})
}

// notifyAlertSubscribers sends notifications to subscribers (async)
func notifyAlertSubscribers(alert models.CommunityAlert) {
	var subscriptions []models.AlertSubscription
	config.DB.Where("is_active = ? AND type IN (?)", true, []string{alert.Type, "all"}).Find(&subscriptions)

	for _, sub := range subscriptions {
		// Create in-app notification
		notification := models.Notification{
			UserID:  sub.UserID,
			Title:   "🚨 " + alert.Title,
			Message: alert.Content,
			Type:    "alert",
			Status:  "unread",
		}
		config.DB.Create(&notification)

		// In production, this would also send email/SMS based on sub.Channel
	}
}
