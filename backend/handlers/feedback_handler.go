package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"security-solution/config"
	"security-solution/models"
	"security-solution/services"
)

var validFeedbackCategories = map[string]bool{
	"bug":       true,
	"feature":   true,
	"complaint": true,
	"support":   true,
	"other":     true,
}

// CreateFeedback records a user's bug report, feature request, complaint,
// or support question. Authenticated users only — their account is the
// audit trail even if they ask to be contacted at a different address.
func CreateFeedback(c *gin.Context) {
	user, ok := userFromContext(c)
	if !ok {
		return
	}

	var input struct {
		Category     string `json:"category" binding:"required"`
		Subject      string `json:"subject" binding:"required"`
		Body         string `json:"body" binding:"required"`
		ContactEmail string `json:"contactEmail"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "category, subject and body are required"})
		return
	}

	input.Category = strings.ToLower(strings.TrimSpace(input.Category))
	input.Subject = strings.TrimSpace(input.Subject)
	input.Body = strings.TrimSpace(input.Body)
	input.ContactEmail = strings.TrimSpace(input.ContactEmail)

	if !validFeedbackCategories[input.Category] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "category must be one of: bug, feature, complaint, support, other",
		})
		return
	}
	if input.Subject == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "subject cannot be empty"})
		return
	}
	if len(input.Subject) > 200 {
		input.Subject = input.Subject[:200]
	}
	if input.Body == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "body cannot be empty"})
		return
	}
	if len(input.Body) > 5000 {
		input.Body = input.Body[:5000]
	}

	// Auto-derive priority: complaints and bugs are high, features are low,
	// everything else normal. Admins can see it and reorder their queue.
	priority := "normal"
	switch input.Category {
	case "bug", "complaint":
		priority = "high"
	case "feature":
		priority = "low"
	}

	fb := models.Feedback{
		UserID:       user.ID,
		Category:     input.Category,
		Priority:     priority,
		Status:       "open",
		Subject:      input.Subject,
		Body:         input.Body,
		ContactEmail: input.ContactEmail,
	}
	if err := config.DB.Create(&fb).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save feedback"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":  "Thank you — your feedback was received.",
		"feedback": fb,
	})
}

// ListMyFeedback returns the caller's own submissions, newest first.
func ListMyFeedback(c *gin.Context) {
	user, ok := userFromContext(c)
	if !ok {
		return
	}

	var items []models.Feedback
	if err := config.DB.
		Where("user_id = ?", user.ID).
		Order("created_at desc").
		Limit(100).
		Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load your feedback"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"feedback": items})
}

// ListAllFeedback — super-admin view. Filters: ?status=open|in_review|resolved|closed|all,
// ?category=bug|feature|complaint|support|other, ?priority=low|normal|high.
func ListAllFeedback(c *gin.Context) {
	if _, ok := callerIsSuperAdmin(c); !ok {
		return
	}

	q := config.DB.Model(&models.Feedback{}).Preload("User")

	if s := strings.TrimSpace(c.Query("status")); s != "" && s != "all" {
		q = q.Where("status = ?", s)
	}
	if cat := strings.TrimSpace(c.Query("category")); cat != "" && cat != "all" {
		q = q.Where("category = ?", cat)
	}
	if p := strings.TrimSpace(c.Query("priority")); p != "" && p != "all" {
		q = q.Where("priority = ?", p)
	}

	var items []models.Feedback
	if err := q.Order("priority desc, created_at desc").Limit(500).Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load feedback"})
		return
	}

	out := make([]gin.H, 0, len(items))
	for _, it := range items {
		out = append(out, gin.H{
			"id":            it.ID,
			"userId":        it.UserID,
			"userEmail":     it.User.Email,
			"userFirstName": it.User.FirstName,
			"userLastName":  it.User.LastName,
			"category":      it.Category,
			"priority":      it.Priority,
			"status":        it.Status,
			"subject":       it.Subject,
			"body":          it.Body,
			"contactEmail":  it.ContactEmail,
			"adminReply":    it.AdminReply,
			"repliedAt":     it.RepliedAt,
			"closedAt":      it.ClosedAt,
			"createdAt":     it.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"feedback": out})
}

// ReplyToFeedback — super admin writes a reply. Moves the ticket to
// in_review (if still open) so the queue reflects that it is being handled.
func ReplyToFeedback(c *gin.Context) {
	admin, ok := callerIsSuperAdmin(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid feedback id"})
		return
	}

	var input struct {
		Reply string `json:"reply" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "reply is required"})
		return
	}
	input.Reply = strings.TrimSpace(input.Reply)
	if input.Reply == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "reply cannot be empty"})
		return
	}
	if len(input.Reply) > 5000 {
		input.Reply = input.Reply[:5000]
	}

	var fb models.Feedback
	if err := config.DB.First(&fb, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "feedback not found"})
		return
	}
	if fb.Status == "closed" {
		c.JSON(http.StatusConflict, gin.H{"error": "feedback is closed"})
		return
	}

	now := time.Now().UTC()
	fb.AdminReply = input.Reply
	fb.RepliedBy = &admin.ID
	fb.RepliedAt = &now
	if fb.Status == "open" {
		fb.Status = "in_review"
	}
	if err := config.DB.Save(&fb).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save reply"})
		return
	}

	// Notify the submitter in-app.
	if fb.UserID != uuid.Nil {
		notification := models.Notification{
			UserID:     fb.UserID,
			Title:      "Reply to your feedback",
			Message:    "The team replied to: \"" + fb.Subject + "\"",
			Type:       "feedback",
			Status:     "unread",
			EntityType: "feedback",
			EntityID:   fb.ID,
			LinkTo:     "/feedback",
		}
		config.DB.Create(&notification)
	}

	_ = services.NewAuditService().LogAction(
		admin.ID,
		"feedback.reply",
		"feedback",
		fb.ID.String(),
		nil,
		map[string]string{"status": fb.Status, "userId": fb.UserID.String()},
		c.ClientIP(),
		c.Request.UserAgent(),
	)

	c.JSON(http.StatusOK, gin.H{
		"message":  "Reply sent",
		"feedback": fb,
	})
}

// CloseFeedback — super admin closes a ticket. Idempotent: closing twice
// does not error.
func CloseFeedback(c *gin.Context) {
	admin, ok := callerIsSuperAdmin(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid feedback id"})
		return
	}

	var fb models.Feedback
	if err := config.DB.First(&fb, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "feedback not found"})
		return
	}
	if fb.Status == "closed" {
		c.JSON(http.StatusOK, gin.H{"message": "Already closed", "feedback": fb})
		return
	}

	now := time.Now().UTC()
	fb.Status = "closed"
	fb.ClosedAt = &now
	if err := config.DB.Save(&fb).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to close"})
		return
	}

	_ = services.NewAuditService().LogAction(
		admin.ID,
		"feedback.close",
		"feedback",
		fb.ID.String(),
		nil,
		map[string]string{"status": "closed", "userId": fb.UserID.String()},
		c.ClientIP(),
		c.Request.UserAgent(),
	)

	c.JSON(http.StatusOK, gin.H{"message": "Feedback closed", "feedback": fb})
}