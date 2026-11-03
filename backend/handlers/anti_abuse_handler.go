package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"security-solution/config"
	"security-solution/models"
	"security-solution/services"
)

const (
	falseAlertTrustPenalty = 20
	trustScoreSOSBanFloor  = 40
	sosBanDuration         = 24 * time.Hour
	defaultRestrictHours   = 24
	maxRestrictHours       = 168
)

// MarkFalseAlert lets an authorized responder mark an SOS as a false alarm.
// The reporter loses trust; crossing the threshold applies a temporary SOS ban.
// POST /sos/:id/false
func MarkFalseAlert(c *gin.Context) {
	u, ok := sosCaller(c)
	if !ok {
		return
	}
	s, ok := sosRead(c, u)
	if !ok {
		return
	}

	var reporter models.User
	var banned bool
	var previousStatus string
	err := config.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&s, "id = ?", s.ID).Error; err != nil {
			return err
		}

		var responders []models.SOSResponder
		if err := tx.Where("sos_id = ?", s.ID).Find(&responders).Error; err != nil {
			return err
		}
		allowed := sosSuper(u)
		if s.UnitID != nil && sosCanManage(tx, u, *s.UnitID) {
			allowed = true
		}
		for _, r := range responders {
			if sosCanManage(tx, u, r.UnitID) || (r.AssignedUserID != nil && *r.AssignedUserID == u.ID && sosOfficerActive(tx, u.ID, r.UnitID)) {
				allowed = true
			}
		}
		if !allowed {
			return sosFail(403, "Only authorized responders may mark this SOS as false")
		}
		if s.FalseAlert {
			return sosFail(409, "This SOS has already been marked as a false alarm")
		}
		previousStatus = s.Status

		now := time.Now().UTC()
		if err := tx.Model(&s).Updates(map[string]any{
			"false_alert":     true,
			"marked_false_at": now,
			"status":          "cancelled",
			"dispatch_state":  "resolved",
		}).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.SOSResponder{}).Where("sos_id = ?", s.ID).
			Updates(map[string]any{"tracking_session_id": nil, "location_id": nil}).Error; err != nil {
			return err
		}

		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&reporter, "id = ?", s.UserID).Error; err != nil {
			return err
		}
		reporter.TrustScore -= falseAlertTrustPenalty
		if reporter.TrustScore < 0 {
			reporter.TrustScore = 0
		}
		updates := map[string]any{"trust_score": reporter.TrustScore}
		if reporter.TrustScore < trustScoreSOSBanFloor {
			until := now.Add(sosBanDuration)
			updates["sos_banned_until"] = until
			reporter.SOSBannedUntil = &until
			banned = true
		}
		return tx.Model(&reporter).Updates(updates).Error
	})
	if err != nil {
		var e sosHTTPError
		if errors.As(err, &e) {
			c.JSON(e.code, gin.H{"error": e.message})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not mark SOS as false"})
		}
		return
	}

	_ = services.NewAuditService().LogAction(
		u.ID,
		"sos.false_alert",
		"sos",
		s.ID.String(),
		map[string]string{"falseAlert": "false", "status": previousStatus},
		map[string]string{"falseAlert": "true", "status": "cancelled"},
		c.ClientIP(),
		c.Request.UserAgent(),
	)

	c.JSON(http.StatusOK, gin.H{
		"message":        "SOS marked as a false alarm",
		"trustScore":     reporter.TrustScore,
		"sosBanned":      banned,
		"sosBannedUntil": reporter.SOSBannedUntil,
	})
}

// RestrictUser restricts a user's community posting within the acting unit
// admin's state/LGA. Super admins may call it too.
// POST /admin/users/:id/restrict
func RestrictUser(c *gin.Context) {
	actor, ok := callerUser(c)
	if !ok {
		return
	}
	targetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}
	if actor.ID == targetID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You cannot restrict your own account"})
		return
	}

	var target models.User
	if err := config.DB.First(&target, "id = ?", targetID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	var input struct {
		DurationHours int `json:"durationHours"`
	}
	_ = c.ShouldBindJSON(&input)
	hours := input.DurationHours
	if hours <= 0 {
		hours = defaultRestrictHours
	}
	if hours > maxRestrictHours {
		hours = maxRestrictHours
	}

	targetState, targetLGA, placeOK := resolvedUserPlace(target)
	if !sosSuper(actor) {
		if target.IsSuperAdmin || target.Role == "super_admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Only a super admin may restrict another super admin"})
			return
		}
		actorState, actorLGA, ok := unitAdminPlace(actor.ID)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "Only a unit admin may restrict users"})
			return
		}
		if !placeOK || !samePlace(actorState, targetState, "") || !samePlace(actorLGA, targetLGA, "") {
			c.JSON(http.StatusForbidden, gin.H{"error": "Target is outside your LGA"})
			return
		}
		targetState, targetLGA = actorState, actorLGA
	}

	until := time.Now().UTC().Add(time.Duration(hours) * time.Hour)
	if err := config.DB.Model(&target).Updates(map[string]any{
		"restricted_until": until,
		"restricted_state": targetState,
		"restricted_lga":   targetLGA,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not restrict user"})
		return
	}

	_ = services.NewAuditService().LogAction(
		actor.ID,
		"user.restrict",
		"user",
		targetID.String(),
		map[string]string{"restrictedUntil": ""},
		map[string]string{"restrictedUntil": until.Format(time.RFC3339), "state": targetState, "lga": targetLGA},
		c.ClientIP(),
		c.Request.UserAgent(),
	)

	c.JSON(http.StatusOK, gin.H{
		"message":         "User restricted",
		"restrictedUntil": until,
		"restrictedState": targetState,
		"restrictedLga":   targetLGA,
	})
}

// BanUser permanently bans a user and revokes their existing credentials.
// POST /admin/users/:id/ban
func BanUser(c *gin.Context) {
	actor, ok := callerUser(c)
	if !ok {
		return
	}
	targetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}
	if actor.ID == targetID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You cannot ban your own account"})
		return
	}

	var target models.User
	if err := config.DB.First(&target, "id = ?", targetID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	previous := target.Status
	target.Status = "banned"
	if err := config.DB.Save(&target).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not ban user"})
		return
	}

	_ = services.NewTokenService().RevokeAllForUser(targetID, "account_banned")
	_ = services.NewRefreshTokenService().RevokeAllForUser(targetID)
	_ = services.NewSessionService().RevokeAll(targetID, "account_banned")

	_ = services.NewAuditService().LogAction(
		actor.ID,
		"user.ban",
		"user",
		targetID.String(),
		map[string]string{"status": previous},
		map[string]string{"status": "banned"},
		c.ClientIP(),
		c.Request.UserAgent(),
	)

	c.JSON(http.StatusOK, gin.H{
		"message": "User banned",
		"user":    target,
	})
}

// isPostingRestricted reports whether u is currently barred from posting in the
// given place. An empty restricted jurisdiction means a global restriction.
func isPostingRestricted(u *models.User, place viewerPlace) bool {
	if u == nil || u.RestrictedUntil == nil {
		return false
	}
	if !time.Now().UTC().Before(*u.RestrictedUntil) {
		return false
	}
	if strings.TrimSpace(u.RestrictedState) == "" && strings.TrimSpace(u.RestrictedLGA) == "" {
		return true
	}
	return samePlace(u.RestrictedState, place.State, "") && samePlace(u.RestrictedLGA, place.LGA, "")
}

// unitAdminPlace returns the state/LGA of the first unit the user administers.
func unitAdminPlace(userID uuid.UUID) (state, lga string, ok bool) {
	var unit models.SecurityUnit
	err := config.DB.
		Where("id IN (?)", sosAdminUnits(config.DB, userID)).
		Where("state <> '' AND lga <> ''").
		Order("is_verified DESC, created_at ASC").
		First(&unit).Error
	if err != nil {
		return "", "", false
	}
	return strings.TrimSpace(unit.State), strings.TrimSpace(unit.LGA), true
}

// resolvedUserPlace resolves a user's jurisdiction from their verified unit or
// their attached unit, falling back to the stored membership/coordinates.
func resolvedUserPlace(u models.User) (state, lga string, ok bool) {
	place := resolveViewerPlace(&u, nil, nil)
	if place.State != "" && place.LGA != "" {
		return place.State, place.LGA, true
	}
	if u.UnitID != nil {
		var unit models.SecurityUnit
		if err := config.DB.First(&unit, "id = ?", *u.UnitID).Error; err == nil {
			s, l := strings.TrimSpace(unit.State), strings.TrimSpace(unit.LGA)
			if s != "" && l != "" {
				return s, l, true
			}
		}
	}
	return "", "", false
}
