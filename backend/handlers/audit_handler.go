package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"security-solution/config"
	"security-solution/models"
)

// GetActivityLogs gets user activity logs
func GetActivityLogs(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj := user.(*models.User)

	var activities []models.ActivityLog
	query := config.DB.Preload("User").Order("created_at desc")

	if userObj.Role == "citizen" {
		query = query.Where("user_id = ?", userObj.ID)
	} else if userObj.Role == "officer" && userObj.UnitID != nil {
		query = query.Joins("JOIN users ON users.id = activity_logs.user_id").
			Where("users.unit_id = ?", userObj.UnitID)
	} else if userObj.Role == "unit_admin" && userObj.UnitID != nil {
		query = query.Joins("JOIN users ON users.id = activity_logs.user_id").
			Where("users.unit_id = ?", userObj.UnitID)
	}

	if err := query.Limit(100).Find(&activities).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch activities"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"activities": activities,
	})
}

// GetAuditLogs gets audit logs (admin only)
func GetAuditLogs(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj := user.(*models.User)

	if userObj.Role != "super_admin" && userObj.Role != "unit_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only admins can view audit logs"})
		return
	}

	var logs []models.AuditLog
	query := config.DB.Preload("User").Order("created_at desc")

	if userObj.Role == "unit_admin" && userObj.UnitID != nil {
		query = query.Joins("JOIN users ON users.id = audit_logs.user_id").
			Where("users.unit_id = ?", userObj.UnitID)
	}

	if err := query.Limit(200).Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch audit logs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"auditLogs": logs,
	})
}

// GetSystemHealth gets system health metrics (admin only)
func GetSystemHealth(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj := user.(*models.User)

	if userObj.Role != "super_admin" && userObj.Role != "unit_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only admins can view system health"})
		return
	}

	var health models.SystemHealth
	if err := config.DB.Order("created_at desc").First(&health).Error; err != nil {
		health = models.SystemHealth{
			CPUUsage:       0,
			MemoryUsage:    0,
			DiskUsage:      0,
			ActiveUsers:    0,
			TotalRequests:  0,
			ResponseTime:   0,
			DatabaseStatus: "healthy",
			ServerStatus:   "healthy",
			Uptime:         0,
			LastCheck:      time.Now(),
		}
	}

	var activeCount int64
	config.DB.Model(&models.User{}).Where("status = ?", "active").Count(&activeCount)

	var totalRequests int64
	config.DB.Model(&models.AuditLog{}).Count(&totalRequests)

	health.ActiveUsers = int(activeCount)
	health.TotalRequests = int(totalRequests)
	health.LastCheck = time.Now()

	c.JSON(http.StatusOK, gin.H{
		"health": health,
	})
}

// GetNotificationLogs gets notification logs (admin only)
func GetNotificationLogs(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj := user.(*models.User)

	if userObj.Role != "super_admin" && userObj.Role != "unit_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only admins can view notification logs"})
		return
	}

	var logs []models.NotificationLog
	query := config.DB.Preload("User").Order("created_at desc")

	if userObj.Role == "unit_admin" && userObj.UnitID != nil {
		query = query.Joins("JOIN users ON users.id = notification_logs.user_id").
			Where("users.unit_id = ?", userObj.UnitID)
	}

	if err := query.Limit(100).Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch notification logs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"notificationLogs": logs,
	})
}
