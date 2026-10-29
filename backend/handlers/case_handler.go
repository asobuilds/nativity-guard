package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"security-solution/config"
	"security-solution/models"
	"security-solution/services"
	"security-solution/utils"
)

// Generate tracking ID
func generateTrackingID() string {
	return fmt.Sprintf("CS-%s-%d", time.Now().Format("20060102"), time.Now().UnixNano()%10000)
}

// CreateCase - Enhanced with automation
func CreateCase(c *gin.Context) {
	if !requireMinorApproved(c) {
		return
	}
	var input struct {
		UnitID       string  `json:"unitId" binding:"required"`
		Title        string  `json:"title" binding:"required"`
		Description  string  `json:"description" binding:"required"`
		Category     string  `json:"category"`
		Latitude     float64 `json:"latitude" binding:"required"`
		Longitude    float64 `json:"longitude" binding:"required"`
		Location     string  `json:"location" binding:"required"`
		Priority     string  `json:"priority"`
		IsSOS        bool    `json:"isSOS"`
		HideLocation bool    `json:"hideLocation"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	category, ok := normalizeIncidentCategory(input.Category)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "category must be one of infrastructure, vandalism, disaster, fire, road_accident, or traffic"})
		return
	}

	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj := user.(*models.User)

	// Generate tracking ID
	trackingID := generateTrackingID()

	// Determine priority level
	priorityLevel := "P3" // Default
	if input.IsSOS {
		priorityLevel = "P1" // Critical
	} else if input.Priority == "high" {
		priorityLevel = "P2" // High
	}

	// Create case
	geohash := utils.EncodeGeohash(input.Latitude, input.Longitude, 5)
	anon := input.HideLocation || !userObj.LocationSharingEnabled

	caseObj := models.Case{
		TrackingID:      trackingID,
		Title:           input.Title,
		Description:     input.Description,
		Category:        category,
		LocationGeohash: geohash,
		Status:          "pending",
		Priority:        input.Priority,
		PriorityLevel:   priorityLevel,
		IsPublic:        true,
		ReportedBy:      userObj.ID,
	}

	if anon {
		// Anonymous path: zero the precise coords and freeform location,
		// keep only the coarse geohash for clustering/heatmap.
		caseObj.Location = ""
		caseObj.Latitude = 0
		caseObj.Longitude = 0
		caseObj.GISLatitude = 0
		caseObj.GISLongitude = 0
		caseObj.IsAnonymous = true
	} else {
		caseObj.Latitude = input.Latitude
		caseObj.Longitude = input.Longitude
		caseObj.GISLatitude = input.Latitude
		caseObj.GISLongitude = input.Longitude
		caseObj.Location = input.Location
	}

	if input.UnitID != "" {
		unitID, err := uuid.Parse(input.UnitID)
		if err == nil {
			caseObj.UnitID = unitID
		}
	}

	if err := config.DB.Create(&caseObj).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create case"})
		return
	}

	// Log audit trail
	auditService := services.NewAuditService()
	go auditService.LogAction(
		userObj.ID,
		"CREATE",
		"CASE",
		caseObj.ID.String(),
		nil,
		caseObj,
		c.ClientIP(),
		c.GetHeader("User-Agent"),
	)

	// Compliance audit for anonymous reports: we need to know anonymized
	// reports happened without storing who filed them precisely.
	if anon {
		go auditService.LogAction(
			userObj.ID,
			"case.created_anonymous",
			"case",
			caseObj.ID.String(),
			nil,
			map[string]interface{}{"geohash": geohash},
			c.ClientIP(),
			c.GetHeader("User-Agent"),
		)
	}

	// If P1 (SOS), trigger immediate dispatch
	if priorityLevel == "P1" {
		go triggerImmediateDispatch(caseObj)
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":       "Case reported successfully",
		"case":          caseObj,
		"trackingId":    trackingID,
		"priorityLevel": priorityLevel,
	})
}

// Helper: Trigger immediate dispatch for P1 cases
func triggerImmediateDispatch(caseObj models.Case) {
	// Get all active units
	var units []models.SecurityUnit
	config.DB.Where("status = ?", "active").Find(&units)

	// Find nearest unit
	var nearestUnit models.SecurityUnit
	var minDistance float64 = -1

	for _, unit := range units {
		if unit.Latitude != 0 && unit.Longitude != 0 {
			distance := haversine(caseObj.Latitude, caseObj.Longitude, unit.Latitude, unit.Longitude)
			if minDistance == -1 || distance < minDistance {
				minDistance = distance
				nearestUnit = unit
			}
		}
	}

	if nearestUnit.ID != uuid.Nil {
		// Assign to nearest unit
		caseObj.UnitID = nearestUnit.ID
		now := time.Now()
		caseObj.AssignedAt = &now
		caseObj.Status = "assigned"
		config.DB.Save(&caseObj)

		// Notify unit officers
		notifyUnitOfficers(nearestUnit.ID, caseObj)

		log.Printf("🚨 P1 Case %s dispatched to unit %s (distance: %.2f km)", caseObj.TrackingID, nearestUnit.Name, minDistance)
	}
}

// Notify unit officers about dispatch
func notifyUnitOfficers(unitID uuid.UUID, caseObj models.Case) {
	var officers []models.Officer
	config.DB.Where("unit_id = ?", unitID).Find(&officers)

	for _, officer := range officers {
		notification := models.Notification{
			UserID:  officer.ID,
			Title:   "🚨 P1 Emergency Case Assigned",
			Message: fmt.Sprintf("Case %s: %s assigned to your unit. Immediate response required!", caseObj.TrackingID, caseObj.Title),
			Type:    "dispatch",
			Status:  "unread",
		}
		config.DB.Create(&notification)
	}

	// Also notify unit admins
	var admins []models.User
	config.DB.Where("unit_id = ? AND role IN (?)", unitID, []string{"unit_admin", "super_admin"}).Find(&admins)

	for _, admin := range admins {
		notification := models.Notification{
			UserID:  admin.ID,
			Title:   "🚨 P1 Emergency Case Assigned",
			Message: fmt.Sprintf("Case %s: %s assigned to your unit. Immediate response required!", caseObj.TrackingID, caseObj.Title),
			Type:    "dispatch",
			Status:  "unread",
		}
		config.DB.Create(&notification)
	}
}

// GetAllCases gets all cases
func GetAllCases(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj := user.(*models.User)

	limit := 50
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
			if limit > 100 {
				limit = 100
			}
		}
	}
	page := 1
	if p := c.Query("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if page > 10000 {
		page = 10000
	}

	var cases []models.Case
	query := config.DB.Preload("Evidence").Preload("Progress").Order("created_at desc").Limit(limit).Offset((page - 1) * limit)

	if userObj.Role == "citizen" {
		query = query.Where("reported_by = ?", userObj.ID)
	} else if userObj.Role == "officer" && userObj.UnitID != nil {
		query = query.Where("unit_id = ?", userObj.UnitID)
	} else if userObj.Role == "unit_admin" && userObj.UnitID != nil {
		query = query.Where("unit_id = ?", userObj.UnitID)
	}

	if err := query.Find(&cases).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch cases"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"cases": cases,
	})
}

// GetCaseByID gets a specific case
func GetCaseByID(c *gin.Context) {
	id := c.Param("id")
	caseID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid case ID"})
		return
	}

	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj := user.(*models.User)

	accessLevel := c.GetString("case_access_level")

	var caseObj models.Case
	if err := config.DB.Preload("Evidence").Preload("Progress").First(&caseObj, "id = ?", caseID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Case not found"})
		return
	}

	// Reporter / citizen: public-safe view only
	if accessLevel == "reporter" || (userObj.Role == "citizen" && caseObj.ReportedBy == userObj.ID) {
		if caseObj.ReportedBy != userObj.ID {
			c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to view this case"})
			return
		}

		var timeline []models.CaseTimeline
		config.DB.Where("case_id = ?", caseID).Order("created_at asc").Find(&timeline)

		var feedback []models.CaseFeedback
		config.DB.Where("case_id = ?", caseID).Find(&feedback)

		publicCase := gin.H{
			"id":            caseObj.ID,
			"trackingId":    caseObj.TrackingID,
			"title":         caseObj.Title,
			"description":   caseObj.Description,
			"status":        caseObj.Status,
			"priority":      caseObj.Priority,
			"priorityLevel": caseObj.PriorityLevel,
			"location":      caseObj.Location,
			"latitude":      caseObj.Latitude,
			"longitude":     caseObj.Longitude,
			"isPublic":      caseObj.IsPublic,
			"finalReport":   caseObj.FinalReport,
			"closedAt":      caseObj.ClosedAt,
			"createdAt":     caseObj.CreatedAt,
			"updatedAt":     caseObj.UpdatedAt,
		}

		c.JSON(http.StatusOK, gin.H{
			"case":     publicCase,
			"timeline": timeline,
			"feedback": feedback,
		})
		return
	}

	// Defense-in-depth: block non-owner citizens even if middleware is bypassed
	if userObj.Role == "citizen" && caseObj.ReportedBy != userObj.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to view this case"})
		return
	}

	// Authorized staff: full case view
	var timeline []models.CaseTimeline
	config.DB.Where("case_id = ?", caseID).Order("created_at asc").Find(&timeline)

	var feedback []models.CaseFeedback
	config.DB.Where("case_id = ?", caseID).Find(&feedback)

	c.JSON(http.StatusOK, gin.H{
		"case":     caseObj,
		"timeline": timeline,
		"feedback": feedback,
	})
}

// UpdateCaseStatus updates case status
func UpdateCaseStatus(c *gin.Context) {
	id := c.Param("id")
	caseID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid case ID"})
		return
	}

	var input struct {
		Status      string `json:"status" binding:"required"`
		Description string `json:"description"`
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

	if userObj.Role != "super_admin" && userObj.Role != "unit_admin" && userObj.Role != "officer" {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to update case status"})
		return
	}

	var caseObj models.Case
	if err := config.DB.First(&caseObj, "id = ?", caseID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Case not found"})
		return
	}

	oldStatus := caseObj.Status

	caseObj.Status = input.Status
	if err := config.DB.Save(&caseObj).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update case"})
		return
	}

	timeline := models.CaseTimeline{
		CaseID:      caseID,
		UserID:      userObj.ID,
		Action:      "status_update",
		Description: input.Description,
		Status:      input.Status,
	}
	config.DB.Create(&timeline)

	auditSvc := services.NewAuditService()
	_ = auditSvc.LogAction(
		userObj.ID,
		"case.status_change",
		"case",
		caseID.String(),
		map[string]string{"status": oldStatus},
		map[string]string{"status": input.Status},
		c.ClientIP(),
		c.Request.UserAgent(),
	)

	c.JSON(http.StatusOK, gin.H{
		"message": "Case status updated successfully",
		"case":    caseObj,
	})
}

// GetCaseAuditTrail - Get full audit trail for a case
func GetCaseAuditTrail(c *gin.Context) {
	caseID := c.Param("id")
	id, err := uuid.Parse(caseID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid case ID"})
		return
	}

	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj := user.(*models.User)

	if userObj.Role != "super_admin" && userObj.Role != "unit_admin" {
		var caseObj models.Case
		if err := config.DB.First(&caseObj, "id = ?", id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Case not found"})
			return
		}
		if caseObj.ReportedBy != userObj.ID && (userObj.UnitID == nil || caseObj.UnitID != *userObj.UnitID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}
	}

	auditService := services.NewAuditService()
	logs, err := auditService.GetAuditTrail("CASE", id.String())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get audit trail"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"auditTrail": logs,
	})
}

// AddCaseTimeline - Add manual timeline entry
func AddCaseTimeline(c *gin.Context) {
	id := c.Param("id")
	caseID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid case ID"})
		return
	}

	var input struct {
		Action      string `json:"action" binding:"required"`
		Description string `json:"description" binding:"required"`
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

	if userObj.Role != "super_admin" && userObj.Role != "unit_admin" && userObj.Role != "officer" {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to add timeline entries"})
		return
	}

	var caseObj models.Case
	if err := config.DB.First(&caseObj, "id = ?", caseID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Case not found"})
		return
	}

	timeline := models.CaseTimeline{
		CaseID:      caseID,
		UserID:      userObj.ID,
		Action:      input.Action,
		Description: input.Description,
		Status:      caseObj.Status,
	}
	config.DB.Create(&timeline)

	c.JSON(http.StatusCreated, gin.H{
		"message":  "Timeline entry added",
		"timeline": timeline,
	})
}

// GetCaseTimeline - Get case timeline
func GetCaseTimeline(c *gin.Context) {
	id := c.Param("id")
	caseID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid case ID"})
		return
	}

	var timeline []models.CaseTimeline
	if err := config.DB.Preload("User").Where("case_id = ?", caseID).Order("created_at asc").Find(&timeline).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch timeline"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"timeline": timeline,
	})
}

// SubmitCaseFeedback - Citizen feedback on case resolution
func SubmitCaseFeedback(c *gin.Context) {
	id := c.Param("id")
	caseID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid case ID"})
		return
	}

	var input struct {
		Rating     int    `json:"rating" binding:"required,min=1,max=5"`
		Comment    string `json:"comment"`
		Categories string `json:"categories"`
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

	var caseObj models.Case
	if err := config.DB.First(&caseObj, "id = ?", caseID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Case not found"})
		return
	}

	if caseObj.ReportedBy != userObj.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not authorized to provide feedback for this case"})
		return
	}

	var existing models.CaseFeedback
	if err := config.DB.Where("case_id = ? AND user_id = ?", caseID, userObj.ID).First(&existing).Error; err == nil {
		existing.Rating = input.Rating
		existing.Comment = input.Comment
		existing.Categories = input.Categories
		config.DB.Save(&existing)
		c.JSON(http.StatusOK, gin.H{
			"message":  "Feedback updated",
			"feedback": existing,
		})
		return
	}

	feedback := models.CaseFeedback{
		CaseID:     caseID,
		UserID:     userObj.ID,
		Rating:     input.Rating,
		Comment:    input.Comment,
		Categories: input.Categories,
		IsPublic:   true,
	}
	config.DB.Create(&feedback)

	c.JSON(http.StatusCreated, gin.H{
		"message":  "Feedback submitted",
		"feedback": feedback,
	})
}

// GetCaseAnalytics - Get case performance metrics
func GetCaseAnalytics(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj := user.(*models.User)

	if userObj.Role != "super_admin" && userObj.Role != "unit_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to view analytics"})
		return
	}

	query := config.DB.Model(&models.Case{})
	if userObj.Role == "unit_admin" && userObj.UnitID != nil {
		query = query.Where("unit_id = ?", userObj.UnitID)
	}

	var totalCases int64
	var resolvedCases int64
	var pendingCases int64

	query.Count(&totalCases)
	query.Where("status = ?", "resolved").Count(&resolvedCases)
	query.Where("status = ?", "pending").Count(&pendingCases)

	c.JSON(http.StatusOK, gin.H{
		"totalCases":     totalCases,
		"resolvedCases":  resolvedCases,
		"pendingCases":   pendingCases,
		"resolutionRate": float64(resolvedCases) / float64(totalCases) * 100,
	})
}
