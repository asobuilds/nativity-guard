package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"security-solution/config"
	"security-solution/models"
)

var validThemes = map[string]bool{
	"forest":   true,
	"midnight": true,
	"daylight": true,
	"ocean":    true,
	"stone":    true,
}

var validTextSizes = map[string]bool{
	"standard": true,
	"large":    true,
	"xlarge":   true,
	"xxlarge":  true,
}

// GetMyPreferences returns the caller's preferences, creating defaults on
// first read.
func GetMyPreferences(c *gin.Context) {
	user, ok := userFromContext(c)
	if !ok {
		return
	}
	prefs := ensurePreferences(user.ID)
	c.JSON(http.StatusOK, gin.H{"preferences": prefs})
}

// UpdateMyPreferences replaces theme, text size, and notification flags.
// Every field is optional — send only what changed.
func UpdateMyPreferences(c *gin.Context) {
	user, ok := userFromContext(c)
	if !ok {
		return
	}

	var input struct {
		Theme                  string `json:"theme"`
		TextSize               string `json:"textSize"`
		NotifyAlerts           *bool  `json:"notifyAlerts"`
		NotifyCaseUpdates      *bool  `json:"notifyCaseUpdates"`
		NotifyCommunityReplies *bool  `json:"notifyCommunityReplies"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	prefs := ensurePreferences(user.ID)

	if input.Theme != "" {
		t := strings.ToLower(strings.TrimSpace(input.Theme))
		if !validThemes[t] {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "theme must be one of: forest, midnight, daylight, ocean, stone",
			})
			return
		}
		prefs.Theme = t
	}

	if input.TextSize != "" {
		s := strings.ToLower(strings.TrimSpace(input.TextSize))
		if !validTextSizes[s] {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "textSize must be one of: standard, large, xlarge, xxlarge",
			})
			return
		}
		prefs.TextSize = s
	}

	if input.NotifyAlerts != nil {
		prefs.NotifyAlerts = *input.NotifyAlerts
	}
	if input.NotifyCaseUpdates != nil {
		prefs.NotifyCaseUpdates = *input.NotifyCaseUpdates
	}
	if input.NotifyCommunityReplies != nil {
		prefs.NotifyCommunityReplies = *input.NotifyCommunityReplies
	}

	if err := config.DB.Save(&prefs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save preferences"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "preferences updated",
		"preferences": prefs,
	})
}

// ensurePreferences returns the caller's row, creating one on first use.
func ensurePreferences(userID uuid.UUID) models.UserPreferences {
	var prefs models.UserPreferences
	err := config.DB.Where("user_id = ?", userID).First(&prefs).Error
	if err == nil {
		return prefs
	}
	prefs = models.UserPreferences{
		UserID:                 userID,
		Theme:                  "forest",
		TextSize:               "standard",
		NotifyAlerts:           true,
		NotifyCaseUpdates:      true,
		NotifyCommunityReplies: true,
	}
	config.DB.Create(&prefs)
	return prefs
}

// userFromContext is a shared helper used by all preference handlers.
func userFromContext(c *gin.Context) (*models.User, bool) {
	value, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return nil, false
	}
	user, ok := value.(*models.User)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authenticated user"})
		return nil, false
	}
	return user, true
}