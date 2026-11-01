package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"security-solution/config"
	"security-solution/models"
	"security-solution/services"
	"security-solution/utils"
)

// maskIfExpunged blanks PII on a suspect when ExpungedAt is set, preserving
// the audit row while hiding name, id number, photo, phone, email, and
// address from officer search. Status is set to "expunged" consistently.
func maskIfExpunged(s *models.Suspect) {
	if s.ExpungedAt == nil {
		return
	}
	s.FirstName = ""
	s.LastName = ""
	s.Alias = ""
	s.IDNumber = ""
	s.PhotoURL = ""
	s.Phone = ""
	s.Email = ""
	s.Address = ""
	s.Status = "expunged"
}

// checkSuspectAccess checks if user has access to this suspect
func checkSuspectAccess(user *models.User, suspect *models.Suspect) bool {
	if user.Role == "super_admin" {
		return true
	}
	if user.Role == "unit_admin" && user.UnitID != nil {
		if suspect.UnitID != nil && *suspect.UnitID == *user.UnitID {
			return true
		}
		if suspect.TransferStatus == "pending" && suspect.TransferToUnit != nil && *suspect.TransferToUnit == *user.UnitID {
			return true
		}
		return false
	}
	if user.Role == "officer" && user.UnitID != nil {
		if suspect.UnitID == nil || *suspect.UnitID != *user.UnitID {
			return false
		}
		var count int64
		config.DB.Model(&models.SuspectCase{}).
			Joins("JOIN cases ON cases.id = suspect_cases.case_id").
			Where("suspect_cases.suspect_id = ? AND cases.assigned_to = ? AND cases.unit_id = ?", suspect.ID, user.ID, *user.UnitID).
			Count(&count)
		if count > 0 {
			return true
		}
		return false
	}
	return false
}

// canReadSuspect is the authoritative read check for any suspect-scoped
// endpoint. It narrows checkSuspectAccess to the four roles allowed to view
// suspect PII:
//
//   - super_admin: every suspect
//   - unit_admin:  only suspects in their own unit (a pending cross-unit
//     transfer is NOT a read grant — it stays a write-only concern)
//   - officer:     only suspects they are personally assigned to via a case
//     in their own unit
//   - the linked citizen (Suspect.UserID == caller): their own active record
//
// Write paths must not use this — they stay gated by checkAdminAccess.
func canReadSuspect(user *models.User, suspect *models.Suspect) bool {
	if suspect.UserID != nil && *suspect.UserID == user.ID {
		return true
	}
	if user.Role == "super_admin" {
		return true
	}
	if user.Role == "unit_admin" && user.UnitID != nil {
		return suspect.UnitID != nil && *suspect.UnitID == *user.UnitID
	}
	if user.Role == "officer" && user.UnitID != nil {
		if suspect.UnitID == nil || *suspect.UnitID != *user.UnitID {
			return false
		}
		var count int64
		config.DB.Model(&models.SuspectCase{}).
			Joins("JOIN cases ON cases.id = suspect_cases.case_id").
			Where("suspect_cases.suspect_id = ? AND cases.assigned_to = ? AND cases.unit_id = ?",
				suspect.ID, user.ID, *user.UnitID).
			Count(&count)
		return count > 0
	}
	return false
}

// checkAdminAccess checks if user is admin
func checkAdminAccess(user *models.User) bool {
	return user.Role == "super_admin" || user.Role == "unit_admin"
}

// requireMinorApproved allows an action unless the caller is a minor without a
// super-admin-granted minor exception. On denial it writes 403 and aborts the
// request. Must be called after AuthMiddleware has populated the "user".
func requireMinorApproved(c *gin.Context) bool {
	userValue, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusForbidden, gin.H{"error": "Minor access restriction applies"})
		c.Abort()
		return false
	}
	userObj, ok := userValue.(*models.User)
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "Minor access restriction applies"})
		c.Abort()
		return false
	}
	if userObj.MinorStatus == "minor" && !userObj.MinorExceptionGranted {
		c.JSON(http.StatusForbidden, gin.H{"error": "Minors need a verified guardian or a super-admin exception to perform this action"})
		c.Abort()
		return false
	}
	return true
}

// CreateSuspect - Enhanced with categories and risk score
func CreateSuspect(c *gin.Context) {
	if !requireMinorApproved(c) {
		return
	}
	var input struct {
		FirstName   string `json:"firstName" binding:"required"`
		LastName    string `json:"lastName" binding:"required"`
		Alias       string `json:"alias"`
		Gender      string `json:"gender"`
		DateOfBirth string `json:"dateOfBirth"`
		Nationality string `json:"nationality"`
		IDNumber    string `json:"idNumber"`
		Phone       string `json:"phone"`
		Email       string `json:"email"`
		Address     string `json:"address"`
		Description string `json:"description"`
		DangerLevel string `json:"dangerLevel"`
		Wanted      bool   `json:"wanted"`
		Category    string `json:"category"`
		SubCategory string `json:"subCategory"`
		UnitID      string `json:"unitId"`
		PhotoURL    string `json:"photoUrl"`
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

	if !checkAdminAccess(userObj) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only admins can create suspects"})
		return
	}

	var dob time.Time
	if input.DateOfBirth != "" {
		parsed, _ := time.Parse("2006-01-02", input.DateOfBirth)
		dob = parsed
	}

	var unitID *uuid.UUID
	if input.UnitID != "" {
		parsed, err := uuid.Parse(input.UnitID)
		if err == nil {
			unitID = &parsed
		}
	} else if userObj.UnitID != nil {
		unitID = userObj.UnitID
	}

	if input.Category == "" {
		input.Category = "general"
	}
	if input.DangerLevel == "" {
		input.DangerLevel = "medium"
	}

	// Calculate risk score based on danger level
	riskScore := 0.0
	switch input.DangerLevel {
	case "low":
		riskScore = 20
	case "medium":
		riskScore = 50
	case "high":
		riskScore = 75
	case "extreme":
		riskScore = 95
	}

	suspect := models.Suspect{
		FirstName:   input.FirstName,
		LastName:    input.LastName,
		Alias:       input.Alias,
		Gender:      input.Gender,
		DateOfBirth: dob,
		Nationality: input.Nationality,
		IDNumber:    input.IDNumber,
		Phone:       input.Phone,
		Email:       input.Email,
		Address:     input.Address,
		Description: input.Description,
		Status:      models.SuspectStatusActive,
		DangerLevel: input.DangerLevel,
		Wanted:      input.Wanted,
		Category:    input.Category,
		SubCategory: input.SubCategory,
		RiskScore:   riskScore,
		PhotoURL:    input.PhotoURL,
		CreatedBy:   userObj.ID,
		UnitID:      unitID,
	}

	if err := config.DB.Create(&suspect).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create suspect"})
		return
	}

	auditSvc := services.NewAuditService()
	_ = auditSvc.LogAction(
		userObj.ID,
		"suspect.create",
		"suspect",
		suspect.ID.String(),
		nil,
		suspect,
		c.ClientIP(),
		c.Request.UserAgent(),
	)

	c.JSON(http.StatusCreated, gin.H{
		"message":   "Suspect created successfully",
		"suspect":   suspect,
		"riskScore": riskScore,
	})
}

// GetAllSuspects gets all suspects (filtered by user access)
func GetAllSuspects(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj := user.(*models.User)

	var suspects []models.Suspect
	query := config.DB.Preload("CreatedByUser").Preload("Unit").Preload("TransferToUnitObj").Preload("Associations")

	if userObj.Role == "super_admin" {
		query = query.Order("created_at desc")
	} else if userObj.Role == "unit_admin" && userObj.UnitID != nil {
		query = query.Where("unit_id = ?", userObj.UnitID).Order("created_at desc")
	} else if userObj.Role == "officer" && userObj.UnitID != nil {
		// Officers see only suspects they are personally assigned to via a
		// case in their own unit — not every suspect in the unit.
		var caseSuspectIDs []string
		config.DB.Table("suspect_cases").
			Joins("JOIN cases ON cases.id = suspect_cases.case_id").
			Where("cases.assigned_to = ? AND cases.unit_id = ?", userObj.ID, userObj.UnitID).
			Pluck("suspect_cases.suspect_id", &caseSuspectIDs)
		if len(caseSuspectIDs) == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to view suspects"})
			return
		}
		query = query.Where("id IN (?) AND status IN (?, ?)",
			caseSuspectIDs, models.SuspectStatusActive, models.SuspectStatusExpunged).
			Order("created_at desc")
	} else {
		// Linked citizen: see their own active records only.
		var ownCount int64
		config.DB.Model(&models.Suspect{}).Where("user_id = ? AND status = ?", userObj.ID, models.SuspectStatusActive).Count(&ownCount)
		if ownCount == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to view suspects"})
			return
		}
		query = query.Where("user_id = ? AND status = ?", userObj.ID, models.SuspectStatusActive).
			Order("created_at desc")
	}

	if err := query.Find(&suspects).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch suspects"})
		return
	}

	for i := range suspects {
		maskIfExpunged(&suspects[i])
	}

	c.JSON(http.StatusOK, gin.H{
		"suspects": suspects,
	})
}

// GetSuspectByID gets a specific suspect (with access check)
func GetSuspectByID(c *gin.Context) {
	id := c.Param("id")
	suspectID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid suspect ID"})
		return
	}

	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj := user.(*models.User)

	var suspect models.Suspect
	if err := config.DB.Preload("CreatedByUser").Preload("Unit").Preload("TransferToUnitObj").Preload("Associations").First(&suspect, "id = ?", suspectID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Suspect not found"})
		return
	}

	if !canReadSuspect(userObj, &suspect) {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to view this suspect"})
		return
	}

	maskIfExpunged(&suspect)

	c.JSON(http.StatusOK, gin.H{
		"suspect": suspect,
	})
}

// UpdateSuspect updates a suspect
func UpdateSuspect(c *gin.Context) {
	id := c.Param("id")
	suspectID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid suspect ID"})
		return
	}

	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj := user.(*models.User)

	if !checkAdminAccess(userObj) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only admins can update suspects"})
		return
	}

	var input struct {
		FirstName   string `json:"firstName"`
		LastName    string `json:"lastName"`
		Alias       string `json:"alias"`
		Gender      string `json:"gender"`
		Nationality string `json:"nationality"`
		IDNumber    string `json:"idNumber"`
		Phone       string `json:"phone"`
		Email       string `json:"email"`
		Address     string `json:"address"`
		Description string `json:"description"`
		Status      string `json:"status"`
		DangerLevel string `json:"dangerLevel"`
		Wanted      *bool  `json:"wanted"`
		Category    string `json:"category"`
		SubCategory string `json:"subCategory"`
		PhotoURL    string `json:"photoUrl"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var suspect models.Suspect
	if err := config.DB.First(&suspect, "id = ?", suspectID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Suspect not found"})
		return
	}

	// Snapshot pre-update values for the audit trail.
	oldSuspect := suspect

	if !checkSuspectAccess(userObj, &suspect) {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to update this suspect"})
		return
	}

	if input.FirstName != "" {
		suspect.FirstName = input.FirstName
	}
	if input.LastName != "" {
		suspect.LastName = input.LastName
	}
	if input.Alias != "" {
		suspect.Alias = input.Alias
	}
	if input.Gender != "" {
		suspect.Gender = input.Gender
	}
	if input.Nationality != "" {
		suspect.Nationality = input.Nationality
	}
	if input.IDNumber != "" {
		suspect.IDNumber = input.IDNumber
	}
	if input.Phone != "" {
		suspect.Phone = input.Phone
	}
	if input.Email != "" {
		suspect.Email = input.Email
	}
	if input.Address != "" {
		suspect.Address = input.Address
	}
	if input.Description != "" {
		suspect.Description = input.Description
	}
	if input.Status != "" {
		suspect.Status = input.Status
	}
	if input.DangerLevel != "" {
		suspect.DangerLevel = input.DangerLevel
		// Update risk score based on new danger level
		switch input.DangerLevel {
		case "low":
			suspect.RiskScore = 20
		case "medium":
			suspect.RiskScore = 50
		case "high":
			suspect.RiskScore = 75
		case "extreme":
			suspect.RiskScore = 95
		}
	}
	if input.Wanted != nil {
		suspect.Wanted = *input.Wanted
	}
	if input.Category != "" {
		suspect.Category = input.Category
	}
	if input.SubCategory != "" {
		suspect.SubCategory = input.SubCategory
	}
	if input.PhotoURL != "" {
		suspect.PhotoURL = input.PhotoURL
	}

	if err := config.DB.Save(&suspect).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update suspect"})
		return
	}

	auditSvc := services.NewAuditService()
	_ = auditSvc.LogAction(
		userObj.ID,
		"suspect.update",
		"suspect",
		suspectID.String(),
		oldSuspect,
		suspect,
		c.ClientIP(),
		c.Request.UserAgent(),
	)

	c.JSON(http.StatusOK, gin.H{
		"message": "Suspect updated successfully",
		"suspect": suspect,
	})
}

// DeleteSuspect deletes a suspect (admin only)
func DeleteSuspect(c *gin.Context) {
	id := c.Param("id")
	suspectID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid suspect ID"})
		return
	}

	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj := user.(*models.User)

	if userObj.Role != "super_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only super admin can delete suspects"})
		return
	}

	var suspect models.Suspect
	if err := config.DB.First(&suspect, "id = ?", suspectID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Suspect not found"})
		return
	}

	if err := config.DB.Delete(&models.Suspect{}, "id = ?", suspectID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete suspect"})
		return
	}

	auditSvc := services.NewAuditService()
	_ = auditSvc.LogAction(
		userObj.ID,
		"suspect.delete",
		"suspect",
		suspectID.String(),
		suspect,
		nil,
		c.ClientIP(),
		c.Request.UserAgent(),
	)

	c.JSON(http.StatusOK, gin.H{
		"message": "Suspect deleted successfully",
	})
}

// ReportSighting reports a suspect sighting
func ReportSighting(c *gin.Context) {
	var input struct {
		SuspectID    string  `json:"suspectId" binding:"required"`
		Latitude     float64 `json:"latitude" binding:"required"`
		Longitude    float64 `json:"longitude" binding:"required"`
		Location     string  `json:"location"`
		Description  string  `json:"description"`
		UnitID       string  `json:"unitId"`
		HideLocation bool    `json:"hideLocation"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	suspectID, err := uuid.Parse(input.SuspectID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid suspect ID"})
		return
	}

	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj := user.(*models.User)

	if userObj.Role == "citizen" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Citizens cannot report suspect sightings"})
		return
	}

	var suspect models.Suspect
	if err := config.DB.First(&suspect, "id = ?", suspectID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Suspect not found"})
		return
	}

	// Reporting a sighting reactivates the suspect, so only a caller with
	// read access to an active suspect may do it; a citizen's self-view is
	// not sufficient to change tracking state.
	if !checkSuspectAccess(userObj, &suspect) {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to report sightings for this suspect"})
		return
	}

	var unitID uuid.UUID
	if input.UnitID != "" {
		parsed, err := uuid.Parse(input.UnitID)
		if err == nil {
			unitID = parsed
		}
	} else if userObj.UnitID != nil {
		unitID = *userObj.UnitID
	}

	geohash := utils.EncodeGeohash(input.Latitude, input.Longitude, 5)
	anon := input.HideLocation || !userObj.LocationSharingEnabled

	sighting := models.SuspectSighting{
		SuspectID:       suspectID,
		ReportedBy:      userObj.ID,
		UnitID:          unitID,
		LocationGeohash: geohash,
		Description:     input.Description,
		Timestamp:       time.Now(),
	}

	if anon {
		// Anonymous sighting: zero precise coords and freeform location,
		// keep only the coarse geohash.
		sighting.Location = ""
		sighting.Latitude = 0
		sighting.Longitude = 0
		sighting.IsAnonymous = true
	} else {
		sighting.Latitude = input.Latitude
		sighting.Longitude = input.Longitude
		sighting.Location = input.Location
	}

	if err := config.DB.Create(&sighting).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to report sighting"})
		return
	}

	// Sighting audit. Anonymous sightings record only the coarse geohash
	// (no PII); precise sightings record the coordinates.
	auditSvc := services.NewAuditService()
	if anon {
		_ = auditSvc.LogAction(
			userObj.ID,
			"sighting.created_anonymous",
			"suspect_sighting",
			sighting.ID.String(),
			nil,
			map[string]interface{}{"geohash": geohash},
			c.ClientIP(),
			c.Request.UserAgent(),
		)
	} else {
		_ = auditSvc.LogAction(
			userObj.ID,
			"suspect.sighting",
			"suspect_sighting",
			sighting.ID.String(),
			nil,
			map[string]interface{}{
				"lat":     sighting.Latitude,
				"lon":     sighting.Longitude,
				"geohash": geohash,
			},
			c.ClientIP(),
			c.Request.UserAgent(),
		)
	}

	suspect.Status = models.SuspectStatusActive
	config.DB.Save(&suspect)

	c.JSON(http.StatusCreated, gin.H{
		"message":  "Sighting reported successfully",
		"sighting": sighting,
	})
}

// GetSuspectSightings gets all sightings for a suspect
func GetSuspectSightings(c *gin.Context) {
	suspectID := c.Param("id")
	id, err := uuid.Parse(suspectID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid suspect ID"})
		return
	}

	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj := user.(*models.User)

	var suspect models.Suspect
	if err := config.DB.First(&suspect, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Suspect not found"})
		return
	}

	if !canReadSuspect(userObj, &suspect) {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to view sightings for this suspect"})
		return
	}

	var sightings []models.SuspectSighting
	if err := config.DB.Preload("Reporter").Preload("Unit").Where("suspect_id = ?", id).Order("timestamp desc").Find(&sightings).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch sightings"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"sightings": sightings,
	})
}

// GetSuspectCases gets all cases for a suspect
func GetSuspectCases(c *gin.Context) {
	suspectID := c.Param("id")
	id, err := uuid.Parse(suspectID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid suspect ID"})
		return
	}

	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj := user.(*models.User)

	var suspect models.Suspect
	if err := config.DB.First(&suspect, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Suspect not found"})
		return
	}

	if !canReadSuspect(userObj, &suspect) {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to view cases for this suspect"})
		return
	}

	var suspectCases []models.SuspectCase
	if err := config.DB.Preload("Case").Where("suspect_id = ?", id).Find(&suspectCases).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch suspect cases"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"suspectCases": suspectCases,
	})
}

// CreateSuspectAssociation links a suspect to a case or another suspect
func CreateSuspectAssociation(c *gin.Context) {
	suspectID := c.Param("id")
	id, err := uuid.Parse(suspectID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid suspect ID"})
		return
	}

	var input struct {
		TargetID        string `json:"targetId" binding:"required"`
		TargetType      string `json:"targetType" binding:"required"` // case, suspect
		AssociationType string `json:"associationType"`
		Notes           string `json:"notes"`
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

	if !checkAdminAccess(userObj) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only admins can create suspect associations"})
		return
	}

	var suspect models.Suspect
	if err := config.DB.First(&suspect, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Suspect not found"})
		return
	}

	if !checkSuspectAccess(userObj, &suspect) {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to associate this suspect"})
		return
	}

	targetID, err := uuid.Parse(input.TargetID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid target ID"})
		return
	}

	if input.AssociationType == "" {
		input.AssociationType = "known"
	}

	association := models.SuspectAssociation{
		SuspectID:       id,
		TargetID:        targetID,
		TargetType:      input.TargetType,
		AssociationType: input.AssociationType,
		Notes:           input.Notes,
		CreatedBy:       userObj.ID,
	}

	if err := config.DB.Create(&association).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create association"})
		return
	}

	auditSvc := services.NewAuditService()
	_ = auditSvc.LogAction(
		userObj.ID,
		"suspect.association.created",
		"suspect_association",
		association.ID.String(),
		nil,
		map[string]interface{}{
			"suspectId":       id,
			"targetId":        targetID,
			"targetType":      input.TargetType,
			"associationType": input.AssociationType,
		},
		c.ClientIP(),
		c.Request.UserAgent(),
	)

	c.JSON(http.StatusCreated, gin.H{
		"message":     "Association created successfully",
		"association": association,
	})
}
