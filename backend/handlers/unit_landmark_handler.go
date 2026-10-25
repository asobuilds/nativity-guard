package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"security-solution/config"
	"security-solution/models"
)

// GetUnitLandmarks returns all custom landmarks for a given unit
func GetUnitLandmarks(c *gin.Context) {
	unitIDParam := c.Param("id")
	unitID, err := uuid.Parse(unitIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid unit ID"})
		return
	}

	var landmarks []models.UnitLandmark
	if err := config.DB.Where("unit_id = ?", unitID).Order("created_at desc").Find(&landmarks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch unit landmarks"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"landmarks": landmarks,
	})
}

// CreateUnitLandmark creates a custom landmark for a unit (Unit Admin / Super Admin)
func CreateUnitLandmark(c *gin.Context) {
	unitIDParam := c.Param("id")
	unitID, err := uuid.Parse(unitIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid unit ID"})
		return
	}

	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj := user.(*models.User)

	if !userObj.IsSuperAdmin && userObj.Role != "super_admin" {
		if userObj.Role != "unit_admin" || userObj.UnitID == nil || *userObj.UnitID != unitID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Only admins of this unit may add landmarks"})
			return
		}
	}

	var input struct {
		Name      string  `json:"name" binding:"required"`
		Type      string  `json:"type" binding:"required"` // "Safe House", "Known Hotspot", "Checkpoint", "Post"
		Latitude  float64 `json:"latitude" binding:"required"`
		Longitude float64 `json:"longitude" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	landmark := models.UnitLandmark{
		UnitID:    unitID,
		Name:      input.Name,
		Type:      input.Type,
		Latitude:  input.Latitude,
		Longitude: input.Longitude,
	}

	if err := config.DB.Create(&landmark).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create landmark"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":  "Landmark created successfully",
		"landmark": landmark,
	})
}

// DeleteUnitLandmark deletes a custom landmark for a unit (Unit Admin / Super Admin)
func DeleteUnitLandmark(c *gin.Context) {
	unitIDParam := c.Param("id")
	unitID, err := uuid.Parse(unitIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid unit ID"})
		return
	}

	landmarkIDParam := c.Param("landmarkId")
	landmarkID, err := uuid.Parse(landmarkIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid landmark ID"})
		return
	}

	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj := user.(*models.User)

	if !userObj.IsSuperAdmin && userObj.Role != "super_admin" {
		if userObj.Role != "unit_admin" || userObj.UnitID == nil || *userObj.UnitID != unitID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Only admins of this unit may delete landmarks"})
			return
		}
	}

	var landmark models.UnitLandmark
	if err := config.DB.First(&landmark, "id = ? AND unit_id = ?", landmarkID, unitID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Landmark not found"})
		return
	}

	if err := config.DB.Delete(&landmark).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete landmark"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Landmark deleted successfully",
	})
}
