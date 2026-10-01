package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"security-solution/config"
	"security-solution/models"
)

// UpdateCoverPosition sets the vertical focus anchor for the caller's cover
// photo. Allowed values are "top", "center", "bottom". Anything else is
// rejected with 400 before touching the database.
//
// This is deliberately a small, single-purpose endpoint rather than a field
// on the general user update — the position changes often (each time a user
// experiments with cropping), and mixing it into PUT /users/me would mean
// resubmitting name and phone on every crop adjustment.
func UpdateCoverPosition(c *gin.Context) {
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
		Position string `json:"position" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "position is required"})
		return
	}

	switch input.Position {
	case "top", "center", "bottom":
		// accepted
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "position must be one of: top, center, bottom",
		})
		return
	}

	if err := config.DB.
		Model(&models.User{}).
		Where("id = ?", userObj.ID).
		Update("cover_position", input.Position).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save cover position"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Cover position updated",
		"coverPosition": input.Position,
	})
}