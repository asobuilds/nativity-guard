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
)

// TriggerPanic is the officer distress endpoint. The authenticated officer
// reports their current position; a priority notification is written for the
// unit admin and the police observer, an SMS is dispatched, and an audit row
// is recorded in officer_panics.
func TriggerPanic(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj := user.(*models.User)

	var input struct {
		Latitude  float64  `json:"latitude" binding:"required"`
		Longitude float64  `json:"longitude" binding:"required"`
		CaseID    string   `json:"caseId"`
		Reason    string   `json:"reason"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var caseID *uuid.UUID
	if input.CaseID != "" {
		parsed, err := uuid.Parse(input.CaseID)
		if err == nil {
			caseID = &parsed
		}
	}

	// The officer's unit is authoritative for notification scoping.
	unitID := userObj.UnitID
	if unitID == nil {
		// Fall back to an officer profile row bound to the same unit via
		// a membership record, when the user row does not carry one.
		var membership models.UnitMembership
		if err := config.DB.Where("user_id = ?", userObj.ID).First(&membership).Error; err == nil {
			id := membership.UnitID
			unitID = &id
		}
	}

	audit := models.OfficerPanic{
		OfficerID: userObj.ID,
		CaseID:    caseID,
		Latitude:  input.Latitude,
		Longitude: input.Longitude,
		Status:    "active",
	}
	if err := config.DB.Create(&audit).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to record panic event"})
		return
	}

	title := "🚨 OFFICER PANIC"
	notificationType := "officer_panic"
	message := fmt.Sprintf("Officer %s %s triggered a panic at %.5f, %.5f.",
		userObj.FirstName, userObj.LastName, input.Latitude, input.Longitude)
	if input.Reason != "" {
		message += " Reason: " + input.Reason
	}
	if caseID != nil {
		message += " Case: " + caseID.String()
	}

	notifyPanicRecipients(c, userObj, unitID, audit.ID, title, message, notificationType)

	// SMS is best-effort: it logs and returns nil on this build; the
	// notification rows above are the authoritative record.
	_ = services.SendSMS(userObj.Phone, message)

	log.Printf("officer panic %s recorded for officer %s at %.5f, %.5f",
		audit.ID.String(), userObj.ID.String(), input.Latitude, input.Longitude)

	c.JSON(http.StatusCreated, gin.H{
		"message": "Officer panic recorded and dispatched",
		"panic":   audit,
		"time":    time.Now().UTC().Format(time.RFC3339),
	})
}

// notifyPanicRecipients writes the priority notification rows. Unit admins are
// found by their UnitID + role; police observers by the agency_observer role.
// Web push is fired for each recipient without blocking the response.
func notifyPanicRecipients(c *gin.Context, userObj *models.User, unitID *uuid.UUID, panicID uuid.UUID, title, message, notifType string) {
	var recipients []models.User
	q := config.DB.Model(&models.User{})
	if unitID != nil {
		q = q.Where("unit_id = ? AND role IN (?)", *unitID, []string{"unit_admin", "super_admin"})
	}
	q.Find(&recipients)

	seen := make(map[uuid.UUID]bool, len(recipients))
	for _, r := range recipients {
		seen[r.ID] = true
	}

	// Police / agency observers: role on the user row.
	var observers []models.User
	config.DB.Where("role = ?", models.RoleAgencyObserver).Find(&observers)
	for _, o := range observers {
		if !seen[o.ID] {
			seen[o.ID] = true
			recipients = append(recipients, o)
		}
	}

	for _, r := range recipients {
		notification := models.Notification{
			UserID:     r.ID,
			Title:      title,
			Message:    message,
			Type:       notifType,
			Status:     "unread",
			EntityType: "officer_panic",
			EntityID:   panicID,
			LinkTo:     "/officer-panic/" + panicID.String(),
		}
		if err := config.DB.Create(&notification).Error; err != nil {
			log.Printf("officer panic: failed to notify %s: %v", r.ID.String(), err)
			continue
		}
		go dispatchWebPush(notification.UserID, notification.Title, notification.Message)
	}
}
