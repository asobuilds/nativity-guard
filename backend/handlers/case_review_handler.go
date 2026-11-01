package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"security-solution/config"
	"security-solution/models"
	"security-solution/services"
)

type CaseReviewRequest struct {
	Comment string `json:"comment" binding:"required"`
}

type WeeklyUpdateRequest struct {
	Summary            string `json:"summary" binding:"required"`
	Investigation      string `json:"investigation" binding:"required"`
	ActionsTaken       string `json:"actionsTaken"`
	Findings           string `json:"findings"`
	EvidenceSummary    string `json:"evidenceSummary"`
	OutstandingActions string `json:"outstandingActions"`
	NextSteps          string `json:"nextSteps"`
}

func getAuthenticatedUser(c *gin.Context) (*models.User, bool) {
	value, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return nil, false
	}

	user, ok := value.(*models.User)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authenticated user"})
		return nil, false
	}

	return user, true
}

func isGlobalAdmin(user *models.User) bool {
	if user == nil {
		return false
	}

	return user.IsSuperAdmin ||
		strings.EqualFold(user.Role, "admin") ||
		strings.EqualFold(user.Role, "super_admin")
}

func isCaseAdmin(user *models.User, caseRecord *models.Case) bool {
	if user == nil || caseRecord == nil {
		return false
	}

	if isGlobalAdmin(user) {
		return true
	}

	// Head Admin of the case's unit
	var membership models.UnitMembership
	if err := config.DB.
		Where("unit_id = ? AND user_id = ? AND status = ?", caseRecord.UnitID, user.ID, models.MembershipActive).
		First(&membership).Error; err == nil {
		if membership.IsHeadAdmin {
			return true
		}

		// Regular Admin: only if the case was submitted to them
		if strings.EqualFold(membership.Role, models.UnitRoleAdmin) {
			var assignment models.CaseAdminAssignment
			if err := config.DB.
				Where("case_id = ? AND admin_id = ? AND status IN ?", caseRecord.ID, user.ID, []string{"pending", "approved"}).
				First(&assignment).Error; err == nil {
				return true
			}
		}
	}

	return false
}

func isAssignedOfficer(user *models.User, caseRecord *models.Case) bool {
	if user == nil || caseRecord == nil {
		return false
	}

	// Primary via Case.AssignedTo
	if caseRecord.AssignedTo != nil && *caseRecord.AssignedTo == user.ID {
		return true
	}

	// Paired via CaseOfficer with role primary or paired
	var caseOfficer models.CaseOfficer
	if err := config.DB.
		Where("case_id = ? AND officer_id = ?", caseRecord.ID, user.ID).
		First(&caseOfficer).Error; err == nil {
		role := strings.ToLower(caseOfficer.Role)
		if role == "primary" || role == "paired" {
			return true
		}
	}

	return false
}

func SubmitCaseForReview(c *gin.Context) {
	caseID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid case ID"})
		return
	}

	user, ok := getAuthenticatedUser(c)
	if !ok {
		return
	}

	var caseRecord models.Case
	if err := config.DB.First(&caseRecord, "id = ?", caseID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Case not found"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load case"})
		return
	}

	if !isAssignedOfficer(user, &caseRecord) {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Only the assigned officer can submit this case for review",
		})
		return
	}

	if caseRecord.Status != models.CaseStatusInvestigating &&
		caseRecord.Status != models.CaseStatusAdminChangesRequested {
		c.JSON(http.StatusConflict, gin.H{
			"error":  "Case must be under investigation before it can be submitted for admin review",
			"status": caseRecord.Status,
		})
		return
	}

	if strings.TrimSpace(caseRecord.FinalReport) == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Final investigation report is required before submission",
		})
		return
	}

	now := time.Now().UTC()

	caseRecord.Status = models.CaseStatusPendingAdminReview

	if err := config.DB.Model(&caseRecord).Updates(map[string]interface{}{
		"status":     caseRecord.Status,
		"updated_at": now,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit case for review"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Case submitted for admin review",
		"case":    caseRecord,
	})
}

func GetCaseReview(c *gin.Context) {
	caseID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid case ID"})
		return
	}

	user, ok := getAuthenticatedUser(c)
	if !ok {
		return
	}

	var caseRecord models.Case
	if err := config.DB.First(&caseRecord, "id = ?", caseID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Case not found"})
		return
	}

	if !isCaseAdmin(user, &caseRecord) &&
		!isAssignedOfficer(user, &caseRecord) &&
		caseRecord.ReportedBy != user.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You do not have access to this case review"})
		return
	}

	var reviews []models.CaseReview

	if err := config.DB.
		Where("case_id = ?", caseID).
		Order("created_at ASC").
		Find(&reviews).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load case reviews"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"caseId":  caseID,
		"status":  caseRecord.Status,
		"reviews": reviews,
	})
}

func RequestCaseChanges(c *gin.Context) {
	caseID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid case ID"})
		return
	}

	user, ok := getAuthenticatedUser(c)
	if !ok {
		return
	}

	var req CaseReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "A review comment is required"})
		return
	}

	var caseRecord models.Case
	if err := config.DB.First(&caseRecord, "id = ?", caseID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Case not found"})
		return
	}

	if !isCaseAdmin(user, &caseRecord) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only an authorized admin can request changes"})
		return
	}

	if caseRecord.Status != models.CaseStatusPendingAdminReview {
		c.JSON(http.StatusConflict, gin.H{
			"error":  "Case is not currently awaiting admin review",
			"status": caseRecord.Status,
		})
		return
	}

	review := models.CaseReview{
		CaseID:   caseID,
		AdminID:  user.ID,
		Decision: models.CaseReviewDecisionRequestChanges,
		Comment:  strings.TrimSpace(req.Comment),
	}

	if err := config.DB.Create(&review).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record review decision"})
		return
	}

	now := time.Now().UTC()

	if err := config.DB.Model(&caseRecord).Updates(map[string]interface{}{
		"status":     models.CaseStatusAdminChangesRequested,
		"updated_at": now,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Review recorded but case status could not be updated"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Additional investigation or changes requested",
		"review":  review,
		"status":  caseRecord.Status,
	})
}

func ApproveCaseClosure(c *gin.Context) {
	caseID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid case ID"})
		return
	}

	user, ok := getAuthenticatedUser(c)
	if !ok {
		return
	}

	var req CaseReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Approval comment is required"})
		return
	}

	var caseRecord models.Case
	if err := config.DB.First(&caseRecord, "id = ?", caseID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Case not found"})
		return
	}

	if !isCaseAdmin(user, &caseRecord) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only an authorized admin can approve case closure"})
		return
	}

	if caseRecord.Status != models.CaseStatusPendingAdminReview {
		c.JSON(http.StatusConflict, gin.H{
			"error":  "Case must be awaiting admin review before closure",
			"status": caseRecord.Status,
		})
		return
	}

	if caseRecord.AssignedTo != nil && *caseRecord.AssignedTo == user.ID {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "An assigned officer cannot approve their own case closure",
		})
		return
	}

	review := models.CaseReview{
		CaseID:   caseID,
		AdminID:  user.ID,
		Decision: models.CaseReviewDecisionApprove,
		Comment:  strings.TrimSpace(req.Comment),
	}

	now := time.Now().UTC()

	tx := config.DB.Begin()

	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to begin case closure transaction"})
		return
	}

	if err := tx.Create(&review).Error; err != nil {
		tx.Rollback()

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record approval"})
		return
	}

	updates := map[string]interface{}{
		"status":      models.CaseStatusClosed,
		"closed_at":   now,
		"closed_by":   user.ID,
		"approved_by": user.ID,
		"updated_at":  now,
	}

	if err := tx.Model(&caseRecord).Updates(updates).Error; err != nil {
		tx.Rollback()

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to close case"})
		return
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to finalize case closure"})
		return
	}

	// Mark the case as closed via platform (counts toward unit ranking).
	config.DB.Model(&models.Case{}).
		Where("id = ?", caseRecord.ID).
		Update("closed_via_platform", true)

	// Archive linked suspects that no longer have an open case. Runs outside
	// the closure tx because it has its own transactional unit (suspect rows
	// + audit rows); failure here does not roll back the closure.
	if err := services.ArchiveSuspectsForCase(caseRecord.ID, user.ID); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"message":  "Case approved and closed",
			"review":   review,
			"status":   models.CaseStatusClosed,
			"closedAt": now,
			"warning":  "failed to archive linked suspects: " + err.Error(),
		})
		return
	}

	// Notify the reporter they can now rate the officer and unit.
	if caseRecord.ReportedBy != uuid.Nil {
		notification := models.Notification{
			UserID:  caseRecord.ReportedBy,
			Title:   "Case closed — rate your experience",
			Message: "Your case has been closed. You can now rate the officer and unit that handled it.",
			Type:    "case_closed_rate",
			Status:  "unread",
		}
		config.DB.Create(&notification)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Case approved and closed",
		"review":   review,
		"status":   models.CaseStatusClosed,
		"closedAt": now,
	})
}

func SubmitWeeklyCaseUpdate(c *gin.Context) {
	caseID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid case ID"})
		return
	}

	user, ok := getAuthenticatedUser(c)
	if !ok {
		return
	}

	var req WeeklyUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Summary and investigation update are required",
		})
		return
	}

	var caseRecord models.Case
	if err := config.DB.First(&caseRecord, "id = ?", caseID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Case not found"})
		return
	}

	if !isAssignedOfficer(user, &caseRecord) {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Only the assigned officer can submit the weekly case update",
		})
		return
	}

	if caseRecord.Status == models.CaseStatusClosed {
		c.JSON(http.StatusConflict, gin.H{"error": "Closed cases cannot receive weekly updates"})
		return
	}

	now := time.Now().UTC()

	// Monday 00:00 UTC is the start of the current reporting week.
	weekday := int(now.Weekday())
	daysSinceMonday := (weekday + 6) % 7

	weekStart := time.Date(
		now.Year(),
		now.Month(),
		now.Day()-daysSinceMonday,
		0, 0, 0, 0,
		time.UTC,
	)

	weekEnd := weekStart.AddDate(0, 0, 6)

	var existing models.CaseWeeklyUpdate

	err = config.DB.
		Where(
			"case_id = ? AND officer_id = ? AND week_start = ?",
			caseID,
			user.ID,
			weekStart,
		).
		First(&existing).Error

	if err == nil {
		c.JSON(http.StatusConflict, gin.H{
			"error":  "Weekly update already submitted for the current reporting week",
			"update": existing,
		})
		return
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check weekly update"})
		return
	}

	update := models.CaseWeeklyUpdate{
		CaseID:             caseID,
		OfficerID:          user.ID,
		WeekStart:          weekStart,
		WeekEnd:            weekEnd,
		Summary:            strings.TrimSpace(req.Summary),
		Investigation:      strings.TrimSpace(req.Investigation),
		ActionsTaken:       strings.TrimSpace(req.ActionsTaken),
		Findings:           strings.TrimSpace(req.Findings),
		EvidenceSummary:    strings.TrimSpace(req.EvidenceSummary),
		OutstandingActions: strings.TrimSpace(req.OutstandingActions),
		NextSteps:          strings.TrimSpace(req.NextSteps),
		CitizenVisible:     true,
		SubmittedAt:        &now,
	}

	if err := config.DB.Create(&update).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save weekly update"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Weekly case update submitted",
		"update":  update,
	})
}

func GetCaseWeeklyUpdates(c *gin.Context) {
	caseID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid case ID"})
		return
	}

	user, ok := getAuthenticatedUser(c)
	if !ok {
		return
	}

	var caseRecord models.Case
	if err := config.DB.First(&caseRecord, "id = ?", caseID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Case not found"})
		return
	}

	if !isCaseAdmin(user, &caseRecord) &&
		!isAssignedOfficer(user, &caseRecord) &&
		caseRecord.ReportedBy != user.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You do not have access to these updates"})
		return
	}

	query := config.DB.
		Where("case_id = ?", caseID).
		Order("week_start ASC")

	// Citizens only receive updates explicitly marked citizen-visible.
	if caseRecord.ReportedBy == user.ID &&
		!isCaseAdmin(user, &caseRecord) &&
		!isAssignedOfficer(user, &caseRecord) {
		query = query.Where("citizen_visible = ?", true)
	}

	var updates []models.CaseWeeklyUpdate

	if err := query.Find(&updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load weekly updates"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"caseId":  caseID,
		"updates": updates,
	})
}

func GetCaseAccountability(c *gin.Context) {
	caseID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid case ID",
		})
		return
	}

	user, ok := getAuthenticatedUser(c)
	if !ok {
		return
	}

	var caseRecord models.Case
	if err := config.DB.First(&caseRecord, "id = ?", caseID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Case not found",
		})
		return
	}

	if !isCaseAdmin(user, &caseRecord) &&
		!isAssignedOfficer(user, &caseRecord) &&
		caseRecord.ReportedBy != user.ID {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "You do not have access to this case",
		})
		return
	}

	accountability, err := services.GetCaseAccountability(caseID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to calculate case accountability",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"accountability": accountability,
	})
}
