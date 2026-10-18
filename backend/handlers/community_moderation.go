package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"security-solution/config"
	"security-solution/models"
)

// The content-report target type. `models.Report` is generic (TargetType is
// "case, sos, announcement, user" per its own comment); community posts are one
// target among several, so the value is namespaced rather than reusing "case".
const reportTargetForumPost = "forum_post"

// Report lifecycle. `pending` is the queue; the other two are terminal and are
// what a moderator chooses between.
const (
	reportStatusPending   = "pending"
	reportStatusReviewed  = "reviewed"
	reportStatusDismissed = "dismissed"
)

// communityModeratorRoles are the roles that may read the moderation queue at all.
// Scope *within* the queue is narrowed further by canModeratePost.
var communityModeratorRoles = map[string]bool{
	"unit_admin":  true,
	"super_admin": true,
}

func isCommunityModerator(user *models.User) bool {
	return user != nil && communityModeratorRoles[user.Role]
}

// communityViewablePost loads a post the caller is allowed to see, or writes the
// error response and reports failure. Sharing this keeps report and every other
// per-post action on the single D-014 boundary instead of a second copy of the
// rule that could drift.
func communityViewablePost(c *gin.Context) (*models.User, models.ForumPost, bool) {
	userVal, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return nil, models.ForumPost{}, false
	}
	user, ok := userVal.(*models.User)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return nil, models.ForumPost{}, false
	}

	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid post ID"})
		return nil, models.ForumPost{}, false
	}

	var post models.ForumPost
	if err := config.DB.First(&post, "id = ? AND status = ?", postID, "published").Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Post not found"})
		return nil, models.ForumPost{}, false
	}

	lat, lng := parseCoords(c)
	if !postVisibleTo(user, post, lat, lng) {
		c.JSON(http.StatusForbidden, gin.H{"error": "This post is outside your area and is not open to you"})
		return nil, models.ForumPost{}, false
	}
	return user, post, true
}

// ReportForumPost — POST /community/posts/:id/report
//
// Files a content report against a forum post. Two rules matter:
//
//   - You may only report a post you can see (the D-014 predicate, via
//     communityViewablePost). This is what stops the report button becoming a
//     probe for content outside your jurisdiction.
//   - Reporting is idempotent per (reporter, post). The check below handles the
//     ordinary repeat tap; the partial unique index idx_report_active_target is
//     the backstop for two taps racing.
//
// This endpoint is the fix for the silent failure: `models.Report` was migrated
// but no handler anywhere created one, so the calls the UI made went nowhere.
func ReportForumPost(c *gin.Context) {
	user, post, ok := communityViewablePost(c)
	if !ok {
		return
	}

	if post.AuthorID == user.ID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "You cannot report your own post"})
		return
	}

	var input struct {
		Reason string `json:"reason"`
	}
	// A report with no reason is still valid — the UI sends none — but it must
	// not be empty in the database, because the column is NOT NULL.
	_ = c.ShouldBindJSON(&input)
	reason := strings.TrimSpace(input.Reason)
	if reason == "" {
		reason = "unspecified"
	}
	if len(reason) > 2000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "reason must be 2000 characters or fewer"})
		return
	}

	targetID := post.ID.String()

	var existing models.Report
	err := config.DB.
		Where("reported_by = ? AND target_type = ? AND target_id = ?", user.ID, reportTargetForumPost, targetID).
		First(&existing).Error
	switch {
	case err == nil:
		c.JSON(http.StatusOK, gin.H{"message": "already reported", "reported": true})
		return
	case !errors.Is(err, gorm.ErrRecordNotFound):
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record report"})
		return
	}

	report := models.Report{
		ReportedBy: user.ID,
		TargetType: reportTargetForumPost,
		TargetID:   targetID,
		Reason:     reason,
		Status:     reportStatusPending,
	}
	if err := config.DB.Create(&report).Error; err != nil {
		// Lost the race with a concurrent tap: the index rejected the duplicate,
		// which is the outcome the caller wanted anyway.
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			c.JSON(http.StatusOK, gin.H{"message": "already reported", "reported": true})
			return
		}
		// The driver does not always surface the typed error (see the same note
		// in case_accountability_service.go), so confirm by reading it back.
		var check models.Report
		if readErr := config.DB.
			Where("reported_by = ? AND target_type = ? AND target_id = ?", user.ID, reportTargetForumPost, targetID).
			First(&check).Error; readErr == nil {
			c.JSON(http.StatusOK, gin.H{"message": "already reported", "reported": true})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record report"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Report recorded", "reported": true})
}

// canModeratePost narrows the queue *within* the moderator roles.
//
// A super admin moderates the whole platform. A unit admin moderates only what
// their own jurisdiction can see, and only up to `state` breadth: a national or
// world post is not theirs to action, and neither is a post of unknown place.
// Posts authored in the unit admin's own unit are always theirs.
func canModeratePost(user *models.User, post models.ForumPost, place viewerPlace) bool {
	if user == nil {
		return false
	}
	if user.Role == "super_admin" {
		return true
	}
	if user.Role != "unit_admin" {
		return false
	}
	if post.UnitID != nil && user.UnitID != nil && *user.UnitID == *post.UnitID {
		return true
	}
	rank, ok := scopeRank(post.GeoPlace.Scope)
	if !ok || rank > scopeRanks[ScopeState] {
		return false
	}
	return placeVisibleToViewer(post.GeoPlace, place)
}

// GetCommunityReports — GET /community/reports?status=&limit=&offset=
//
// The moderation queue. Reports are listed oldest-first so nothing starves, and
// each row carries the reported post so a moderator can judge without opening it.
//
// Note on paging: scope filtering happens in Go (the rules are the tested seam),
// so a page can come back shorter than `limit` when the caller's own scope
// excludes some rows. That is preferred to leaking a row the caller may not act
// on; the queue is small and page-based, and keyset paging here is not worth the
// extra coupling.
func GetCommunityReports(c *gin.Context) {
	userVal, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	user := userVal.(*models.User)
	if !isCommunityModerator(user) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Moderation is limited to unit and super admins"})
		return
	}

	status := strings.ToLower(strings.TrimSpace(c.DefaultQuery("status", reportStatusPending)))
	switch status {
	case reportStatusPending, reportStatusReviewed, reportStatusDismissed:
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "status must be pending, reviewed or dismissed"})
		return
	}

	limit := 20
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 && n <= 50 {
			limit = n
		}
	}
	offset := 0
	if raw := strings.TrimSpace(c.Query("offset")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			offset = n
		}
	}

	var reports []models.Report
	if err := config.DB.
		Where("target_type = ? AND status = ?", reportTargetForumPost, status).
		Order("created_at ASC").
		Limit(limit).Offset(offset).
		Find(&reports).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load reports"})
		return
	}

	// Load every reported post in one query rather than one per report.
	posts := make(map[uuid.UUID]models.ForumPost, len(reports))
	if len(reports) > 0 {
		ids := make([]uuid.UUID, 0, len(reports))
		for _, report := range reports {
			if id, err := uuid.Parse(report.TargetID); err == nil {
				ids = append(ids, id)
			}
		}
		if len(ids) > 0 {
			var loaded []models.ForumPost
			if err := config.DB.
				Preload("Author", func(db *gorm.DB) *gorm.DB { return db.Select("id", "first_name", "last_name") }).
				Where("id IN ?", ids).
				Find(&loaded).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load reported posts"})
				return
			}
			for _, post := range loaded {
				posts[post.ID] = post
			}
		}
	}

	lat, lng := parseCoords(c)
	place := resolveViewerPlace(user, lat, lng)

	results := make([]gin.H, 0, len(reports))
	for _, report := range reports {
		id, err := uuid.Parse(report.TargetID)
		if err != nil {
			continue
		}
		post, found := posts[id]
		if !found {
			// The post was deleted or unpublished since being reported; there is
			// nothing left to action, so it is not shown.
			continue
		}
		if !canModeratePost(user, post, place) {
			continue
		}
		results = append(results, gin.H{
			"id":        report.ID,
			"reason":    report.Reason,
			"status":    report.Status,
			"createdAt": report.CreatedAt,
			"post":      forumPostResponse(post, false),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"reports": results,
		"count":   len(results),
		"status":  status,
		"limit":   limit,
		"offset":  offset,
	})
}

// ResolveCommunityReport — POST /community/reports/:id/resolve
//
// Terminal only: a report becomes `reviewed` or `dismissed`. Deliberately does NOT
// touch the post — removing content is a new capability with its own blast radius
// (it is filed as RES-015), and quietly folding it into "resolve" would be the
// wrong place for that decision.
func ResolveCommunityReport(c *gin.Context) {
	userVal, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	user := userVal.(*models.User)
	if !isCommunityModerator(user) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Moderation is limited to unit and super admins"})
		return
	}

	reportID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid report ID"})
		return
	}

	var input struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	status := strings.ToLower(strings.TrimSpace(input.Status))
	if status != reportStatusReviewed && status != reportStatusDismissed {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status must be reviewed or dismissed"})
		return
	}

	var report models.Report
	if err := config.DB.
		First(&report, "id = ? AND target_type = ?", reportID, reportTargetForumPost).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Report not found"})
		return
	}

	// Authorization is re-derived from the reported post, not from the fact that
	// the report was visible in a list a moment ago.
	postID, err := uuid.Parse(report.TargetID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Reported post not found"})
		return
	}
	var post models.ForumPost
	if err := config.DB.First(&post, "id = ?", postID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Reported post not found"})
		return
	}
	lat, lng := parseCoords(c)
	if !canModeratePost(user, post, resolveViewerPlace(user, lat, lng)) {
		c.JSON(http.StatusForbidden, gin.H{"error": "This report is outside the area you moderate"})
		return
	}

	reviewedBy := user.ID
	if err := config.DB.Model(&models.Report{}).
		Where("id = ?", report.ID).
		Updates(map[string]any{"status": status, "reviewed_by": reviewedBy}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update report"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Report " + status,
		"report":  gin.H{"id": report.ID, "status": status},
	})
}

// reportedPostIDs returns the subset of postIDs the user has already reported, so
// the feed can show the settled "Reported" state without one query per post.
func reportedPostIDs(userID uuid.UUID, postIDs []uuid.UUID) map[string]bool {
	reported := make(map[string]bool)
	if len(postIDs) == 0 || config.DB == nil {
		return reported
	}
	targets := make([]string, 0, len(postIDs))
	for _, id := range postIDs {
		targets = append(targets, id.String())
	}
	var found []string
	if err := config.DB.Model(&models.Report{}).
		Where("reported_by = ? AND target_type = ? AND target_id IN ?", userID, reportTargetForumPost, targets).
		Pluck("target_id", &found).Error; err != nil {
		return reported
	}
	for _, target := range found {
		reported[target] = true
	}
	return reported
}
