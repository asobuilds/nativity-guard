package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"security-solution/config"
	"security-solution/models"
)

// GetPublicCases - Public endpoint for landing page (no auth required)
func GetPublicCases(c *gin.Context) {
	var cases []models.Case
	if err := config.DB.Where("is_public = ? AND status != ?", true, "closed").Order("created_at desc").Limit(20).Find(&cases).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch cases"})
		return
	}

	// Privacy projection: anonymous reports (IsAnonymous or coarse-only
	// storage where Latitude == 0) must not leak precise coords to the
	// public. The frontend renders these via LocationGeohash instead.
	for i := range cases {
		if cases[i].IsAnonymous || cases[i].Latitude == 0 {
			cases[i].Latitude = 0
			cases[i].Longitude = 0
			cases[i].GISLatitude = 0
			cases[i].GISLongitude = 0
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"cases": publicCaseDTOs(cases),
	})
}

// GetPublicUnits - Public endpoint for landing page (no auth required)
func GetPublicUnits(c *gin.Context) {
	var units []models.SecurityUnit
	if err := config.DB.Where("status = ?", "active").Find(&units).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch units"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"units": publicUnitDTOs(units),
	})
}
