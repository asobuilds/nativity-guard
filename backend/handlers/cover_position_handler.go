package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"security-solution/config"
	"security-solution/models"
)

// UpdateCoverAdjustment saves the caller's cover-photo focus:
//   - x, y: object-position percentage, 0–100
//   - zoom: 100–300, where 100 = 1.0× and 300 = 3.0×
//
// Out-of-range values are clamped rather than rejected — a client that
// sends 105 while the user drags should not fail. Values are stored on
// the user record so the adjustment follows the account across devices.
func UpdateCoverAdjustment(c *gin.Context) {
	value, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj, ok := value.(*models.User)
	if !ok || userObj == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user"})
		return
	}

	var input struct {
		X    int `json:"x"`
		Y    int `json:"y"`
		Zoom int `json:"zoom"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "x, y and zoom are required"})
		return
	}

	x := clampInt(input.X, 0, 100)
	y := clampInt(input.Y, 0, 100)
	zoom := clampInt(input.Zoom, 100, 300)

	if err := config.DB.
		Model(&models.User{}).
		Where("id = ?", userObj.ID).
		Updates(map[string]interface{}{
			"cover_position_x": x,
			"cover_position_y": y,
			"cover_zoom":       zoom,
		}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save cover adjustment"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        "Cover adjustment updated",
		"coverPositionX": x,
		"coverPositionY": y,
		"coverZoom":      zoom,
	})
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}