package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"security-solution/config"
	"security-solution/models"
)

// UpsertAlertSubscription creates or replaces the caller's alert subscription.
// A user has at most one active subscription at a time. Categories and
// channels arrive as JSON arrays and are stored comma-separated in the
// existing Type / Channel columns.
//
// Latitude/Longitude are where the subscriber wants to receive nearby
// alerts. Both zero means "no location set" — the subscriber then receives
// every alert in their chosen categories, nationwide.
func UpsertAlertSubscription(c *gin.Context) {
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
		Categories []string `json:"categories" binding:"required"`
		Channels   []string `json:"channels"`
		Location   string   `json:"location"`
		Latitude   float64  `json:"latitude"`
		Longitude  float64  `json:"longitude"`
		Radius     float64  `json:"radius"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "categories is required"})
		return
	}

	allowedCategories := map[string]bool{
		"security":  true,
		"community": true,
		"weather":   true,
		"general":   true,
	}
	cleanCategories := make([]string, 0, len(input.Categories))
	for _, cat := range input.Categories {
		cat = strings.TrimSpace(strings.ToLower(cat))
		if allowedCategories[cat] {
			cleanCategories = append(cleanCategories, cat)
		}
	}
	if len(cleanCategories) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "at least one valid category is required (security, community, weather, general)",
		})
		return
	}

	if len(input.Channels) == 0 {
		input.Channels = []string{"in_app"}
	}
	allowedChannels := map[string]bool{"in_app": true, "email": true, "sms": true}
	cleanChannels := make([]string, 0, len(input.Channels))
	for _, ch := range input.Channels {
		ch = strings.TrimSpace(strings.ToLower(ch))
		if allowedChannels[ch] {
			cleanChannels = append(cleanChannels, ch)
		}
	}
	if len(cleanChannels) == 0 {
		cleanChannels = []string{"in_app"}
	}

	if input.Radius < 1 {
		input.Radius = 10
	}
	if input.Radius > 50 {
		input.Radius = 50
	}

	// Validate coordinates. Outside these ranges means the client sent
	// garbage — treat as "no location" instead of rejecting, so the
	// subscription still saves (Option A fallback).
	validCoords := true
	if input.Latitude < -90 || input.Latitude > 90 {
		validCoords = false
	}
	if input.Longitude < -180 || input.Longitude > 180 {
		validCoords = false
	}
	// (0, 0) is the "no location set" sentinel.
	if input.Latitude == 0 && input.Longitude == 0 {
		validCoords = false
	}
	if !validCoords {
		input.Latitude = 0
		input.Longitude = 0
	}

	categoriesCSV := strings.Join(cleanCategories, ",")
	channelsCSV := strings.Join(cleanChannels, ",")

	var existing models.AlertSubscription
	err := config.DB.
		Where("user_id = ? AND is_active = ?", user.ID, true).
		Order("created_at DESC").
		First(&existing).Error

	if err == nil {
		existing.Type = categoriesCSV
		existing.Channel = channelsCSV
		existing.Location = strings.TrimSpace(input.Location)
		existing.Latitude = input.Latitude
		existing.Longitude = input.Longitude
		existing.Radius = input.Radius
		if err := config.DB.Save(&existing).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update subscription"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"subscription": toSubscriptionJSON(existing),
		})
		return
	}

	sub := models.AlertSubscription{
		UserID:    user.ID,
		UnitID:    user.UnitID,
		Type:      categoriesCSV,
		Channel:   channelsCSV,
		Location:  strings.TrimSpace(input.Location),
		Latitude:  input.Latitude,
		Longitude: input.Longitude,
		Radius:    input.Radius,
		IsActive:  true,
	}
	if err := config.DB.Create(&sub).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create subscription"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"subscription": toSubscriptionJSON(sub),
	})
}

// GetMyAlertSubscription returns the caller's active subscription, or
// `{subscription: null}` when none exists.
func GetMyAlertSubscription(c *gin.Context) {
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

	var sub models.AlertSubscription
	err := config.DB.
		Where("user_id = ? AND is_active = ?", user.ID, true).
		Order("created_at DESC").
		First(&sub).Error
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"subscription": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"subscription": toSubscriptionJSON(sub),
	})
}

// DeleteAlertSubscription deactivates the caller's subscription by id.
func DeleteAlertSubscription(c *gin.Context) {
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

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subscription id"})
		return
	}

	var sub models.AlertSubscription
	if err := config.DB.First(&sub, "id = ? AND user_id = ?", id, user.ID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
		return
	}

	if err := config.DB.Model(&sub).Update("is_active", false).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to remove subscription"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "subscription removed"})
}

// toSubscriptionJSON converts a stored row (CSV fields) into the shape the
// frontend expects (arrays).
func toSubscriptionJSON(s models.AlertSubscription) gin.H {
	cats := splitCSV(s.Type)
	chans := splitCSV(s.Channel)
	if len(chans) == 0 {
		chans = []string{"in_app"}
	}
	return gin.H{
		"id":         s.ID,
		"userId":     s.UserID,
		"categories": cats,
		"channels":   chans,
		"location":   s.Location,
		"latitude":   s.Latitude,
		"longitude":  s.Longitude,
		"radius":     s.Radius,
		"isActive":   s.IsActive,
		"createdAt":  s.CreatedAt,
		"updatedAt":  s.UpdatedAt,
	}
}

func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return []string{}
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}