package services

import (
	"fmt"
	"time"

	"gorm.io/gorm"

	"security-solution/models"
)

// RiskPredictionService handles AI predictions.
type RiskPredictionService struct{}

// GenerateLocationRisk produces a short risk assessment for a location.
//
// The prior version of this function slept for one second and returned a
// hardcoded string — it was a stub, not an analysis. This version counts
// real incidents in the window, feeds them to the LLM with the coordinates
// and place name, and returns whatever the model writes. If the LLM call
// fails, the numeric count is still returned so the caller has something
// truthful to show.
func GenerateLocationRisk(db *gorm.DB, lat, lng float64, location string) string {
	var count int64
	cutoff := time.Now().AddDate(0, 0, -30)

	if err := db.Model(&models.Case{}).
		Where("created_at > ?", cutoff).
		Where("(latitude BETWEEN ? AND ?) AND (longitude BETWEEN ? AND ?)",
			lat-0.1, lat+0.1, lng-0.1, lng+0.1).
		Count(&count).Error; err != nil {
		return fmt.Sprintf("Could not read recent incidents near %s.", location)
	}

	var recent []models.Case
	_ = db.Model(&models.Case{}).
		Where("created_at > ?", cutoff).
		Where("(latitude BETWEEN ? AND ?) AND (longitude BETWEEN ? AND ?)",
			lat-0.1, lat+0.1, lng-0.1, lng+0.1).
		Order("created_at desc").
		Limit(10).
		Find(&recent).Error

	incidentsText := ""
	for _, c := range recent {
		incidentsText += fmt.Sprintf("- %s (%s)\n", c.Title, c.Status)
	}

	ai := NewAIService()
	analysis, err := ai.AnalyzeLocationRisk(lat, lng, location, incidentsText)
	if err != nil {
		// Fall back to the count alone — do not fabricate a risk level.
		return fmt.Sprintf("%d recent incidents near %s in the last 30 days.", count, location)
	}
	return analysis
}