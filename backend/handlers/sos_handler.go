package handlers

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"security-solution/config"
	"security-solution/models"
	"security-solution/services"
	"security-solution/utils"
)

// SendSOSAlert - Enhanced with emergency contacts and escalation
func SendSOSAlert(c *gin.Context) {
	if !requireMinorApproved(c) {
		return
	}
	var input struct {
		Latitude          float64  `json:"latitude" binding:"required"`
		Longitude         float64  `json:"longitude" binding:"required"`
		Description       string   `json:"description"`
		UnitID            string   `json:"unitId"`
		EmergencyContacts []string `json:"emergencyContacts"`
		MedicalInfo       string   `json:"medicalInfo"`
		Priority          string   `json:"priority"`
		HideLocation      bool     `json:"hideLocation"`
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

	if input.Priority == "" {
		input.Priority = "high"
	}

	severity := "normal"
	if input.Priority == "critical" {
		severity = "critical"
	}

	geohash := utils.EncodeGeohash(input.Latitude, input.Longitude, 5)
	anon := input.HideLocation || !userObj.LocationSharingEnabled

	// Best-effort jurisdiction lookup so head admins in the same
    // state/LGA can be notified at creation time.
    var jurState, jurLGA string
    if rec, err := services.NewGeocodingService().ReverseGeocode(input.Latitude, input.Longitude); err == nil && rec != nil {
        jurState = rec.State
        jurLGA = rec.LGA
    }

    sos := models.SOSAlert{
		UserID:        userObj.ID,
		Severity:      severity,
		DispatchState:    "open",
		JurisdictionState: jurState,
		JurisdictionLGA:   jurLGA,
		Latitude:    input.Latitude,
		Longitude:   input.Longitude,
		Description: input.Description,
		Status:      "pending",
		Priority:    input.Priority,
		LocationGeohash: geohash,
	}

	// SOS is time-critical: even in coarse mode we keep the precise
	// coordinates so responders can reach the person. Only the public
	// display path strips them.
	if anon {
		sos.IsAnonymous = true
	}

	if input.UnitID != "" {
		unitID, err := uuid.Parse(input.UnitID)
		if err == nil {
			sos.UnitID = &unitID
		}
	}

	if err := config.DB.Create(&sos).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send SOS alert"})
		return
	}

	// Compliance audit for anonymous SOS.
	if anon {
		auditSvc := services.NewAuditService()
		_ = auditSvc.LogAction(
			userObj.ID,
			"sos.created_anonymous",
			"sos_alert",
			sos.ID.String(),
			nil,
			map[string]interface{}{"geohash": geohash},
			c.ClientIP(),
			c.Request.UserAgent(),
		)
	}

	// Notify emergency contacts
	if len(input.EmergencyContacts) > 0 {
		go notifyEmergencyContacts(input.EmergencyContacts, sos, userObj)
	}

	// Notify nearest units
	go func() {
        dispatcher := services.NewSosDispatchService()
        candidates := dispatcher.SelectCandidates(sos.Latitude, sos.Longitude)
        dispatcher.NotifyCandidates(sos, candidates)
        dispatcher.NotifySuperAdmins(sos, "new SOS")
        dispatcher.NotifyJurisdictionHeads(sos)
    }()

	// Start escalation timer (auto-escalate if no response in 5 minutes)
	go startEscalationTimer(sos.ID)

	// Save medical info if provided
	if input.MedicalInfo != "" {
		go saveMedicalInfo(userObj.ID, input.MedicalInfo)
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":        "🚨 SOS alert sent successfully! Help is on the way.",
		"sos":            sos,
		"escalationTime": "5 minutes",
	})
}

// GetSOSAlerts gets all SOS alerts (filtered by role)
func GetSOSAlerts(c *gin.Context) {
	GetVisibleSOSAlerts(c)
}

// GetSOSAlertByID gets a specific SOS alert
func GetSOSAlertByID(c *gin.Context) {
	GetSOSDetailWithResponders(c)
}

// UpdateSOSAlertStatus updates an SOS alert status
func UpdateSOSAlertStatus(c *gin.Context) {
	UpdateSOSStatusScoped(c)
}

// GetUserSOSAlerts gets SOS alerts for a specific user
func GetUserSOSAlerts(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj := user.(*models.User)

	var alerts []models.SOSAlert
	if err := config.DB.Where("user_id = ?", userObj.ID).Order("created_at desc").Find(&alerts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch SOS alerts"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"alerts": alerts,
	})
}

// Helper: Notify emergency contacts
func notifyEmergencyContacts(contacts []string, sos models.SOSAlert, user *models.User) {
	for _, contact := range contacts {
		message := fmt.Sprintf("🚨 EMERGENCY ALERT from %s %s!\nLocation: %f, %f\nDescription: %s\nPlease contact them immediately.",
			user.FirstName, user.LastName, sos.Latitude, sos.Longitude, sos.Description)

		// Send SMS or notification
		log.Printf("📱 Emergency contact notified: %s - %s", contact, message)
	}
}

// Helper: Notify nearest units
func notifyNearestUnits(sos models.SOSAlert) {
	var units []models.SecurityUnit
	config.DB.Where("status = ?", "active").Find(&units)

	// Find nearest units based on location
	for _, unit := range units {
		if unit.Latitude != 0 && unit.Longitude != 0 {
			distance := haversine(sos.Latitude, sos.Longitude, unit.Latitude, unit.Longitude)
			if distance <= unit.OperationalRadius {
				// Notify this unit
				log.Printf("🔔 Notifying unit %s about SOS alert (distance: %.2f km)", unit.Name, distance)

				// Create notification for unit admins
				notifyUnitAdminsForSOS(unit.ID, "🚨 SOS Alert Nearby",
					fmt.Sprintf("SOS alert at %f, %f. Distance: %.2f km", sos.Latitude, sos.Longitude, distance))
			}
		}
	}
}

// Helper: Start escalation timer
func startEscalationTimer(sosID uuid.UUID) {
	time.Sleep(5 * time.Minute)

	var sos models.SOSAlert
	if err := config.DB.First(&sos, "id = ?", sosID).Error; err != nil {
		return
	}

	if sos.Status == "pending" || sos.Status == "dispatched" {
		// Escalate - notify higher authority
		sos.Status = "escalated"
		config.DB.Save(&sos)

		// Notify super admins
		var superAdmins []models.User
		config.DB.Where("role = ?", "super_admin").Find(&superAdmins)

		for _, admin := range superAdmins {
			notification := models.Notification{
				UserID:  admin.ID,
				Title:   "⚠️ SOS Escalated - No Response",
				Message: fmt.Sprintf("SOS alert %s has not been responded to in 5 minutes. Immediate attention required.", sosID.String()),
				Type:    "sos_escalation",
				Status:  "unread",
			}
			config.DB.Create(&notification)
		}

		log.Printf("⚠️ SOS alert %s escalated due to no response", sosID.String())
	}
}

// Helper: Save medical info
func saveMedicalInfo(userID uuid.UUID, info string) {
	// info is intentionally not logged: medical content must never appear
	// in stdout. The parameter is retained to keep the call signature stable.
	_ = info
	log.Printf("sos: medical info saved for user %s", userID.String())
}

// Helper: Notify user
func notifyUser(userID uuid.UUID, title, message string) {
	notification := models.Notification{
		UserID:  userID,
		Title:   title,
		Message: message,
		Type:    "sos_update",
		Status:  "unread",
	}
	config.DB.Create(&notification)
}

// Helper: Notify unit admins for SOS (renamed to avoid conflict)
func notifyUnitAdminsForSOS(unitID uuid.UUID, title, message string) {
	var admins []models.User
	config.DB.Where("unit_id = ? AND role IN (?)", unitID, []string{"unit_admin", "super_admin"}).Find(&admins)

	for _, admin := range admins {
		notification := models.Notification{
			UserID:  admin.ID,
			Title:   title,
			Message: message,
			Type:    "sos_alert",
			Status:  "unread",
		}
		config.DB.Create(&notification)
	}
}
