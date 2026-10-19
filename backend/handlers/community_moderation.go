package handlers

import (
	"errors"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"security-solution/config"
	"security-solution/models"
	"security-solution/services"
)

// The content-report target type. `models.Report` is generic (TargetType is
// "case, sos, announcement, user" per its own comment); community posts are one
// target among several, so the value is namespaced rather than reusing "case".
const reportTargetForumPost = "forum_post"

// Notification type for a newly filed content report. A named constant keeps the
// producer and any future consumer on one spelling, so future rows cannot drift
// into a second string that the client does not recognise.
const NotificationTypeReportCreated = "report_created"

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
		// Optional attachments, mirroring the SOS path: private report_media/
		// keys from POST /community/posts/:id/report/presign, re-verified
		// against the store before the report row is written.
		VoiceNoteKey string `json:"voiceNoteKey"`
		VideoKey     string `json:"videoKey"`
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

	// Re-verify declared attachments before recording anything, so a bad key does
	// not leave a report row pointing at an object that is not there.
	voiceNoteKey, ok := attachVerifiedMedia(c, input.VoiceNoteKey, reportMediaPrefix, voiceMediaMaxBytes)
	if !ok {
		return
	}
	videoKey, ok := attachVerifiedMedia(c, input.VideoKey, reportMediaPrefix, videoMediaMaxBytes)
	if !ok {
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

	// Record attachment links now the report has an id. Best-effort, on the same
	// reasoning as SOS: the report is already recorded.
	linkMedia(MediaParentReport, report.ID, voiceNoteKey, videoKey)

	// A report is only useful if a moderator learns about it. RES-014 made the
	// queue work but left it pull-only (audit finding 1, §4.4); this closes that
	// gap. The recipients are exactly the moderators who can action this report
	// in the queue, so nobody is pinged for work they cannot open.
	notifyReportModerators(post, user.ID)

	c.JSON(http.StatusCreated, gin.H{"message": "Report recorded", "reported": true})
}

// notifyReportModerators tells every moderator who can action this report that
// one has arrived. The recipient set mirrors GetCommunityReports' authorization
// exactly — super admins platform-wide, and unit admins only where
// canModeratePost holds — so a moderator is never notified about a report
// outside their jurisdiction and never silently left without one they can act
// on. The reporter is never notified, even if they are themselves a moderator.
//
// It reuses the existing per-user notification helper (notifyUser, sos_handler.go)
// rather than introducing a second notification path.
func notifyReportModerators(post models.ForumPost, reporterID uuid.UUID) {
	if config.DB == nil {
		return
	}

	var moderators []models.User
	if err := config.DB.
		Where("role IN ?", []string{"unit_admin", "super_admin"}).
		Find(&moderators).Error; err != nil {
		return
	}

	for i := range moderators {
		moderator := &moderators[i]
		if moderator.ID == reporterID {
			continue
		}
		if moderator.Role == "unit_admin" &&
			!canModeratePost(moderator, post, resolveViewerPlace(moderator, nil, nil)) {
			continue
		}
		notifyUser(moderator.ID, NotificationTypeReportCreated, "New content report",
			"A post has been reported and is waiting in the moderation queue.")
	}
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

	visible := make([]models.Report, 0, len(reports))
	visibleIDs := make([]uuid.UUID, 0, len(reports))
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
		visible = append(visible, report)
		visibleIDs = append(visibleIDs, report.ID)
	}

	// Attachments live in media_links keyed by the report; the Report row has no
	// media columns. Load them in one query for the page, then turn each key into
	// a short-lived read URL so a media element can play it without a bearer
	// header.
	linksByReport := make(map[uuid.UUID][]models.MediaLink, len(visible))
	if len(visibleIDs) > 0 {
		var links []models.MediaLink
		if err := config.DB.
			Where("parent_type = ? AND parent_id IN ?", MediaParentReport, visibleIDs).
			Find(&links).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load report attachments"})
			return
		}
		for _, link := range links {
			linksByReport[link.ParentID] = append(linksByReport[link.ParentID], link)
		}
	}

	storage := services.NewFileStorageService()
	results := make([]gin.H, 0, len(visible))
	for _, report := range visible {
		id, _ := uuid.Parse(report.TargetID)
		row := gin.H{
			"id":        report.ID,
			"reason":    report.Reason,
			"status":    report.Status,
			"createdAt": report.CreatedAt,
			"post":      forumPostResponse(posts[id], false),
		}
		for key, value := range reportMediaResponse(linksByReport[report.ID], storage) {
			row[key] = value
		}
		results = append(results, row)
	}

	c.JSON(http.StatusOK, gin.H{
		"reports": results,
		"count":   len(results),
		"status":  status,
		"limit":   limit,
		"offset":  offset,
	})
}

// reportMediaResponse classifies a report's attachments into the voice and video
// slots and mints a short-lived read URL for each.
//
// The object key alone cannot tell audio from video (.webm is used for both), so
// the stored Content-Type is consulted first. The extension is only a fallback
// for local-disk mode, which does not report a type.
func reportMediaResponse(links []models.MediaLink, storage *services.FileStorageService) gin.H {
	out := gin.H{}
	for _, link := range links {
		_, contentType, err := storage.Head(link.ObjectKey)
		if err != nil {
			continue
		}
		contentType = strings.ToLower(strings.TrimSpace(contentType))

		isVideo := strings.HasPrefix(contentType, "video/")
		if contentType == "" {
			isVideo = hasVideoExtension(link.ObjectKey)
		} else if !isVideo && !strings.HasPrefix(contentType, "audio/") {
			continue
		}

		url, err := storage.PresignGet(link.ObjectKey, privateReadTTL)
		if err != nil {
			continue
		}

		slot := "voiceNote"
		if isVideo {
			slot = "video"
		}
		if _, exists := out[slot+"Key"]; exists {
			continue
		}
		out[slot+"Key"] = link.ObjectKey
		out[slot+"Url"] = url
	}
	return out
}

// hasVideoExtension is the local-disk fallback classifier. A voice note recorded
// as .webm lands in the video slot here; it still plays, only the element differs.
func hasVideoExtension(key string) bool {
	switch strings.ToLower(path.Ext(key)) {
	case ".mp4", ".mov", ".webm", ".qt":
		return true
	default:
		return false
	}
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
