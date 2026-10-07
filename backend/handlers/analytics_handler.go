package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"security-solution/config"
	"security-solution/models"
)

// GetUnitAnalytics returns analytics for a specific unit
func GetUnitAnalytics(c *gin.Context) {
	unitID := c.Param("id")

	userValue, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}
	user, ok := userValue.(*models.User)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	if user.Role != "super_admin" {
		if user.Role != "unit_admin" || user.UnitID == nil || user.UnitID.String() != unitID {
			c.JSON(http.StatusForbidden, gin.H{"error": "You do not have access to analytics for this unit"})
			return
		}
	}

	var totalCases int64
	var resolvedCases int64
	var pendingCases int64

	config.DB.Model(&models.Case{}).Where("unit_id = ?", unitID).Count(&totalCases)
	config.DB.Model(&models.Case{}).Where("unit_id = ? AND status = ?", unitID, "resolved").Count(&resolvedCases)
	config.DB.Model(&models.Case{}).Where("unit_id = ? AND status = ?", unitID, "pending").Count(&pendingCases)

	var monthlyStats []struct {
		Month string
		Count int64
	}
	config.DB.Model(&models.Case{}).
		Select("to_char(created_at, 'YYYY-MM') as month, count(*) as count").
		Where("unit_id = ?", unitID).
		Group("month").
		Order("month desc").
		Limit(6).
		Scan(&monthlyStats)

	c.JSON(http.StatusOK, gin.H{
		"totalCases":    totalCases,
		"resolvedCases": resolvedCases,
		"pendingCases":  pendingCases,
		"monthlyStats":  monthlyStats,
	})
}

// GetUserActivity returns user activity analytics
func GetUserActivity(c *gin.Context) {
	var userStats []struct {
		UserID    string
		Email     string
		CaseCount int64
	}
	config.DB.Table("users").
		Select("users.id as user_id, users.email, count(cases.id) as case_count").
		Joins("left join cases on cases.reported_by = users.id").
		Group("users.id, users.email").
		Order("case_count desc").
		Limit(10).
		Scan(&userStats)

	var dailyActivity []struct {
		Date  string
		Count int64
	}
	config.DB.Model(&models.Case{}).
		Select("to_char(created_at, 'YYYY-MM-DD') as date, count(*) as count").
		Group("date").
		Order("date desc").
		Limit(7).
		Scan(&dailyActivity)

	c.JSON(http.StatusOK, gin.H{
		"topUsers":      userStats,
		"dailyActivity": dailyActivity,
	})
}
