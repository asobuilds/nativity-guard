package handlers

import (
	"errors"
	"math"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"security-solution/config"
	"security-solution/models"
)

const sosLocationMaxAge = 2 * time.Minute

// Deliberate response types keep audit actors, identities and user records out
// of the citizen response. Only the managing admin receives AssignedUserID.
type sosResponderView struct {
	ID         uuid.UUID `json:"id"`
	UnitID     uuid.UUID `json:"unitId"`
	Role       string    `json:"role"`
	AcceptedAt time.Time `json:"acceptedAt"`
	Unit       struct {
		ID   uuid.UUID `json:"id"`
		Name string    `json:"name"`
	} `json:"unit"`
	Assigned       bool       `json:"assigned"`
	AssignedUserID *uuid.UUID `json:"assignedUserId,omitempty"`
	IsMyAssignment bool       `json:"isMyAssignment"`
	CanManage      bool       `json:"canManage"`
}

type sosLocationView struct {
	ID         uuid.UUID `json:"id"`
	UnitID     uuid.UUID `json:"unitId"`
	Role       string    `json:"role"`
	Latitude   float64   `json:"latitude"`
	Longitude  float64   `json:"longitude"`
	Accuracy   float64   `json:"accuracy"`
	RecordedAt time.Time `json:"recordedAt"`
}

func sosActive(s models.SOSAlert) bool {
	return s.Status != "resolved" && s.Status != "cancelled" && s.DispatchState != "resolved"
}

func sosSuper(u *models.User) bool { return u.IsSuperAdmin || u.Role == "super_admin" }

func sosCaller(c *gin.Context) (*models.User, bool) {
	u, ok := callerUser(c)
	if !ok {
		return nil, false
	}
	if u.Status != "active" {
		c.JSON(http.StatusForbidden, gin.H{"error": "An active account is required"})
		return nil, false
	}
	c.Header("Cache-Control", "no-store")
	return u, true
}

func sosAdminUnits(db *gorm.DB, userID uuid.UUID) *gorm.DB {
	return db.Model(&models.UnitMembership{}).Select("unit_id").
		Where("user_id = ? AND status = ? AND (role = ? OR is_head_admin = ?)", userID, models.MembershipActive, models.UnitRoleAdmin, true)
}

// Unit membership alone never grants officer access to an SOS. Administrators
// may triage their addressed alerts, their unit's accepted alerts, or an open
// SOS for which the existing dispatcher explicitly notified them.
func sosReadScope(db *gorm.DB, u *models.User) *gorm.DB {
	q := db.Model(&models.SOSAlert{})
	if sosSuper(u) {
		return q
	}
	responding := db.Model(&models.SOSResponder{}).Select("sos_responders.sos_id").
		Joins("JOIN unit_memberships m ON m.unit_id = sos_responders.unit_id AND m.user_id = ?", u.ID).
		Where("m.deleted_at IS NULL AND m.status = ?", models.MembershipActive).
		Where("m.role = ? OR m.is_head_admin = ? OR (m.role = ? AND sos_responders.assigned_user_id = ?)", models.UnitRoleAdmin, true, models.UnitRoleOfficer, u.ID)
	notifications := db.Model(&models.Notification{}).Select("entity_id").
		Where("user_id = ? AND entity_type = ? AND type IN ?", u.ID, "sos", []string{"sos_alert", "dispatch"})
	return q.Where("sos_alerts.user_id = ? OR sos_alerts.id IN (?) OR sos_alerts.unit_id IN (?) OR (sos_alerts.dispatch_state = ? AND sos_alerts.status IN ? AND sos_alerts.id IN (?) AND EXISTS (?))",
		u.ID, responding, sosAdminUnits(db, u.ID), "open", []string{"pending", "escalated"}, notifications, sosAdminUnits(db, u.ID))
}

func sosCanManage(db *gorm.DB, u *models.User, unitID uuid.UUID) bool {
	if sosSuper(u) {
		return true
	}
	var n int64
	return sosAdminUnits(db, u.ID).Where("unit_id = ?", unitID).Count(&n).Error == nil && n > 0
}

func sosOfficerActive(db *gorm.DB, userID, unitID uuid.UUID) bool {
	var n int64
	err := db.Model(&models.UnitMembership{}).
		Joins("JOIN users u ON u.id = unit_memberships.user_id").
		Where("unit_memberships.user_id = ? AND unit_memberships.unit_id = ? AND unit_memberships.role = ? AND unit_memberships.status = ?", userID, unitID, models.UnitRoleOfficer, models.MembershipActive).
		Where("u.deleted_at IS NULL AND u.status = ?", "active").Count(&n).Error
	return err == nil && n > 0
}

func sosView(s models.SOSAlert) gin.H {
	return gin.H{"id": s.ID, "userId": s.UserID, "unitId": s.UnitID,
		"latitude": s.Latitude, "longitude": s.Longitude, "description": s.Description,
		"status": s.Status, "priority": s.Priority, "severity": s.Severity,
		"dispatchState": s.DispatchState, "acceptedByUnitId": s.AcceptedByUnitID,
		"acceptedAt": s.AcceptedAt, "jurisdictionState": s.JurisdictionState,
		"jurisdictionLga": s.JurisdictionLGA, "createdAt": s.CreatedAt, "updatedAt": s.UpdatedAt}
}

func responderView(r models.SOSResponder, u *models.User, manage, mine bool) sosResponderView {
	v := sosResponderView{ID: r.ID, UnitID: r.UnitID, Role: r.Role, AcceptedAt: r.AcceptedAt,
		Assigned: r.AssignedUserID != nil, CanManage: manage, IsMyAssignment: mine}
	v.Unit.ID, v.Unit.Name = r.UnitID, r.Unit.Name
	if manage {
		v.AssignedUserID = r.AssignedUserID
	}
	return v
}

func sosRead(c *gin.Context, u *models.User) (models.SOSAlert, bool) {
	var s models.SOSAlert
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid SOS ID"})
		return s, false
	}
	err = sosReadScope(config.DB.WithContext(c.Request.Context()), u).First(&s, "sos_alerts.id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(404, gin.H{"error": "SOS not found or not accessible"})
		} else {
			c.JSON(500, gin.H{"error": "Could not load SOS"})
		}
		return s, false
	}
	return s, true
}

func GetVisibleSOSAlerts(c *gin.Context) {
	u, ok := sosCaller(c)
	if !ok {
		return
	}
	alerts := []models.SOSAlert{}
	if err := sosReadScope(config.DB, u).Order("created_at DESC").Find(&alerts).Error; err != nil {
		c.JSON(500, gin.H{"error": "Could not load SOS alerts"})
		return
	}
	out := make([]gin.H, 0, len(alerts))
	for _, s := range alerts {
		out = append(out, sosView(s))
	}
	c.JSON(200, gin.H{"alerts": out})
}

func GetSOSDetailWithResponders(c *gin.Context) {
	u, ok := sosCaller(c)
	if !ok {
		return
	}
	s, ok := sosRead(c, u)
	if !ok {
		return
	}
	var rows []models.SOSResponder
	if err := config.DB.Preload("Unit").Where("sos_id = ?", s.ID).Order("accepted_at ASC, id ASC").Find(&rows).Error; err != nil {
		c.JSON(500, gin.H{"error": "Could not load responders"})
		return
	}
	responders := make([]sosResponderView, 0, len(rows))
	for _, r := range rows {
		manage := sosCanManage(config.DB, u, r.UnitID)
		mine := r.AssignedUserID != nil && *r.AssignedUserID == u.ID && sosOfficerActive(config.DB, u.ID, r.UnitID)
		responders = append(responders, responderView(r, u, manage, mine))
	}
	// Keep alert for existing clients while fixing the sos envelope expected
	// by the current detail screen.
	view := sosView(s)
	c.JSON(200, gin.H{"sos": view, "alert": view, "responders": responders})
}

func freshSOSLocation(l models.UserLocation, now time.Time) bool {
	return !l.RecordedAt.IsZero() && now.Sub(l.RecordedAt) <= sosLocationMaxAge &&
		!l.RecordedAt.After(now.Add(10*time.Second)) &&
		!math.IsNaN(l.Latitude) && !math.IsNaN(l.Longitude) && !math.IsNaN(l.Accuracy) &&
		!math.IsInf(l.Latitude, 0) && !math.IsInf(l.Longitude, 0) && !math.IsInf(l.Accuracy, 0) &&
		l.Latitude >= -90 && l.Latitude <= 90 && l.Longitude >= -180 && l.Longitude <= 180 && l.Accuracy >= 0
}

func GetSOSResponderLocations(c *gin.Context) {
	u, ok := sosCaller(c)
	if !ok {
		return
	}
	s, ok := sosRead(c, u)
	if !ok {
		return
	}
	out := make([]sosLocationView, 0)
	if !sosActive(s) {
		c.JSON(200, gin.H{"sosId": s.ID, "responders": out, "serverTime": time.Now().UTC()})
		return
	}
	var rows []models.SOSResponder
	if err := config.DB.Where("sos_id = ? AND assigned_user_id IS NOT NULL AND tracking_session_id IS NOT NULL AND location_id IS NOT NULL", s.ID).Find(&rows).Error; err != nil {
		c.JSON(500, gin.H{"error": "Could not load responder locations"})
		return
	}
	now := time.Now().UTC()
	for _, r := range rows {
		if !sosOfficerActive(config.DB, *r.AssignedUserID, r.UnitID) {
			continue
		}
		var officer models.User
		if err := config.DB.Select("id", "location_sharing_enabled").First(&officer, "id = ?", *r.AssignedUserID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			c.JSON(500, gin.H{"error": "Could not load sharing preferences"})
			return
		}
		if !officer.LocationSharingEnabled {
			continue
		}
		var l models.UserLocation
		err := config.DB.First(&l, "id = ? AND user_id = ?", *r.LocationID, *r.AssignedUserID).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			continue
		}
		if err != nil {
			c.JSON(500, gin.H{"error": "Could not load responder location"})
			return
		}
		if !freshSOSLocation(l, now) {
			continue
		}
		out = append(out, sosLocationView{r.ID, r.UnitID, r.Role, l.Latitude, l.Longitude, l.Accuracy, l.RecordedAt})
	}
	c.JSON(200, gin.H{"sosId": s.ID, "responders": out, "serverTime": time.Now().UTC()})
}

type sosHTTPError struct {
	code    int
	message string
}

func (e sosHTTPError) Error() string         { return e.message }
func sosFail(code int, message string) error { return sosHTTPError{code, message} }

// All writers lock the SOS before the responder. Session tokens make a delayed
// upload after stop/reassignment harmless rather than reviving tracking.
func mutateSOSResponder(c *gin.Context, action func(*gorm.DB, *models.User, *models.SOSAlert, *models.SOSResponder) (any, error)) {
	u, ok := sosCaller(c)
	if !ok {
		return
	}
	sosID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid SOS ID"})
		return
	}
	unitID, err := uuid.Parse(c.Param("unitId"))
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid unit ID"})
		return
	}
	var out any
	err = config.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		var s models.SOSAlert
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&s, "id = ?", sosID).Error; e != nil {
			return e
		}
		var r models.SOSResponder
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&r, "sos_id = ? AND unit_id = ?", sosID, unitID).Error; e != nil {
			return e
		}
		var e error
		out, e = action(tx, u, &s, &r)
		return e
	})
	if err != nil {
		var e sosHTTPError
		if errors.As(err, &e) {
			c.JSON(e.code, gin.H{"error": e.message})
		} else if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(404, gin.H{"error": "SOS responder not found"})
		} else {
			c.JSON(500, gin.H{"error": "Could not update SOS responder"})
		}
		return
	}
	c.JSON(200, out)
}

func GetSOSOfficerCandidates(c *gin.Context) {
	u, ok := sosCaller(c)
	if !ok {
		return
	}
	sosID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid SOS ID"})
		return
	}
	unitID, err := uuid.Parse(c.Param("unitId"))
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid unit ID"})
		return
	}
	if !sosCanManage(config.DB, u, unitID) {
		c.JSON(403, gin.H{"error": "Only this unit's administrators may assign officers"})
		return
	}
	var r models.SOSResponder
	if err := config.DB.First(&r, "sos_id = ? AND unit_id = ?", sosID, unitID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(404, gin.H{"error": "Unit is not responding to this SOS"})
		} else {
			c.JSON(500, gin.H{"error": "Could not load responder"})
		}
		return
	}
	type candidate struct {
		UserID    uuid.UUID `json:"userId"`
		FirstName string    `json:"firstName"`
		LastName  string    `json:"lastName"`
	}
	out := make([]candidate, 0)
	err = config.DB.Model(&models.UnitMembership{}).
		Select("unit_memberships.user_id, u.first_name, u.last_name").
		Joins("JOIN users u ON u.id = unit_memberships.user_id").
		Where("unit_memberships.unit_id = ? AND unit_memberships.role = ? AND unit_memberships.status = ? AND u.status = ? AND u.deleted_at IS NULL", unitID, models.UnitRoleOfficer, models.MembershipActive, "active").
		Order("u.first_name, u.last_name, u.id").Scan(&out).Error
	if err != nil {
		c.JSON(500, gin.H{"error": "Could not load eligible officers"})
		return
	}
	c.JSON(200, gin.H{"officers": out})
}

func AssignSOSOfficer(c *gin.Context) {
	var input struct {
		UserID *uuid.UUID `json:"userId"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"error": "Provide an officer userId, or null to unassign"})
		return
	}
	mutateSOSResponder(c, func(tx *gorm.DB, u *models.User, s *models.SOSAlert, r *models.SOSResponder) (any, error) {
		if !sosCanManage(tx, u, r.UnitID) {
			return nil, sosFail(403, "Only this unit's administrators may assign officers")
		}
		if !sosActive(*s) {
			return nil, sosFail(409, "This SOS is closed")
		}
		if input.UserID != nil && !sosOfficerActive(tx, *input.UserID, r.UnitID) {
			return nil, sosFail(400, "Choose an active officer account in this unit")
		}
		if err := tx.Model(r).Updates(map[string]any{"assigned_user_id": input.UserID, "tracking_session_id": nil, "location_id": nil}).Error; err != nil {
			return nil, err
		}
		if input.UserID != nil {
			n := models.Notification{UserID: *input.UserID, Title: "SOS response assignment", Message: "You have been assigned to an SOS. Open it to review the alert and choose whether to share your location.", Type: "dispatch", Status: "unread", EntityType: "sos", EntityID: s.ID, LinkTo: "/sos/" + s.ID.String()}
			if err := tx.Create(&n).Error; err != nil {
				return nil, err
			}
		}
		return gin.H{"message": "Officer assignment updated"}, nil
	})
}

func sosRequireAssigned(tx *gorm.DB, u *models.User, r *models.SOSResponder) error {
	if r.AssignedUserID == nil || *r.AssignedUserID != u.ID || !sosOfficerActive(tx, u.ID, r.UnitID) {
		return sosFail(403, "Only the assigned active officer can share a location")
	}
	return nil
}

func StartSOSTracking(c *gin.Context) {
	mutateSOSResponder(c, func(tx *gorm.DB, u *models.User, s *models.SOSAlert, r *models.SOSResponder) (any, error) {
		if err := sosRequireAssigned(tx, u, r); err != nil {
			return nil, err
		}
		if !sosActive(*s) {
			return nil, sosFail(409, "This SOS is closed")
		}
		if !u.LocationSharingEnabled {
			return nil, sosFail(403, "Enable location sharing in Settings first")
		}
		session := uuid.New()
		if err := tx.Model(r).Updates(map[string]any{"tracking_session_id": session, "location_id": nil}).Error; err != nil {
			return nil, err
		}
		return gin.H{"sessionId": session}, nil
	})
}

func StopSOSTracking(c *gin.Context) {
	session, err := uuid.Parse(c.Query("sessionId"))
	if err != nil {
		c.JSON(400, gin.H{"error": "A valid sessionId is required"})
		return
	}
	mutateSOSResponder(c, func(tx *gorm.DB, u *models.User, _ *models.SOSAlert, r *models.SOSResponder) (any, error) {
		if r.AssignedUserID == nil || *r.AssignedUserID != u.ID {
			return nil, sosFail(403, "Only the assigned officer can stop this session")
		}
		// A late stop from an old tab must not stop a newer tab's session.
		if err := tx.Model(r).Where("tracking_session_id = ?", session).Updates(map[string]any{"tracking_session_id": nil, "location_id": nil}).Error; err != nil {
			return nil, err
		}
		return gin.H{"message": "Location sharing stopped"}, nil
	})
}

func UpdateSOSResponderLocation(c *gin.Context) {
	var input struct {
		SessionID uuid.UUID `json:"sessionId"`
		Latitude  *float64  `json:"latitude"`
		Longitude *float64  `json:"longitude"`
		Accuracy  *float64  `json:"accuracy"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || input.SessionID == uuid.Nil || input.Latitude == nil || input.Longitude == nil || input.Accuracy == nil {
		c.JSON(400, gin.H{"error": "sessionId, latitude, longitude and accuracy are required"})
		return
	}
	location := models.UserLocation{Latitude: *input.Latitude, Longitude: *input.Longitude, Accuracy: *input.Accuracy, RecordedAt: time.Now().UTC()}
	if !freshSOSLocation(location, location.RecordedAt) {
		c.JSON(400, gin.H{"error": "Invalid coordinates or accuracy"})
		return
	}
	mutateSOSResponder(c, func(tx *gorm.DB, u *models.User, s *models.SOSAlert, r *models.SOSResponder) (any, error) {
		if err := sosRequireAssigned(tx, u, r); err != nil {
			return nil, err
		}
		if !sosActive(*s) {
			return nil, sosFail(409, "This SOS is closed")
		}
		if !u.LocationSharingEnabled {
			return nil, sosFail(403, "Location sharing is disabled")
		}
		if r.TrackingSessionID == nil || *r.TrackingSessionID != input.SessionID {
			return nil, sosFail(409, "Sharing session expired; start sharing again")
		}
		location.UserID = u.ID
		if err := tx.Create(&location).Error; err != nil {
			return nil, err
		}
		if err := tx.Model(r).Update("location_id", location.ID).Error; err != nil {
			return nil, err
		}
		return gin.H{"recordedAt": location.RecordedAt}, nil
	})
}

// UpdateSOSStatusScoped enforces assignment scope and prevents closed alerts
// from being reopened by a delayed status write. Closure clears every session.
func UpdateSOSStatusScoped(c *gin.Context) {
	u, ok := sosCaller(c)
	if !ok {
		return
	}
	s, ok := sosRead(c, u)
	if !ok {
		return
	}
	var input struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || (input.Status != "dispatched" && input.Status != "resolved" && input.Status != "cancelled") {
		c.JSON(400, gin.H{"error": "Status must be dispatched, resolved or cancelled"})
		return
	}
	err := config.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&s, "id = ?", s.ID).Error; err != nil {
			return err
		}
		var responders []models.SOSResponder
		if err := tx.Where("sos_id = ?", s.ID).Find(&responders).Error; err != nil {
			return err
		}
		allowed := sosSuper(u)
		for _, r := range responders {
			if sosCanManage(tx, u, r.UnitID) || (r.AssignedUserID != nil && *r.AssignedUserID == u.ID && sosOfficerActive(tx, u.ID, r.UnitID)) {
				allowed = true
			}
		}
		if !allowed {
			return sosFail(403, "Only authorized responders may update this SOS")
		}
		if !sosActive(s) && s.Status != input.Status {
			return sosFail(409, "This SOS is closed")
		}
		changes := map[string]any{"status": input.Status}
		if input.Status == "resolved" || input.Status == "cancelled" {
			changes["dispatch_state"] = "resolved"
			if err := tx.Model(&models.SOSResponder{}).Where("sos_id = ?", s.ID).Updates(map[string]any{"tracking_session_id": nil, "location_id": nil}).Error; err != nil {
				return err
			}
		}
		return tx.Model(&s).Updates(changes).Error
	})
	if err != nil {
		var e sosHTTPError
		if errors.As(err, &e) {
			c.JSON(e.code, gin.H{"error": e.message})
		} else {
			c.JSON(500, gin.H{"error": "Could not update SOS status"})
		}
		return
	}
	c.JSON(200, gin.H{"message": "SOS status updated", "alert": sosView(s)})
}
