package handlers

import (
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"security-solution/config"
	"security-solution/models"
	"security-solution/services"
)

// privateReadTTL is how long a presigned GET for a private object stays valid.
const privateReadTTL = 10 * time.Minute

// ServeFile resolves a stored file by category and bare hash and hands the
// caller off to the right place.
//
// Stored keys use <category>/<yyyy>/<mm>/<hash>.<ext>. The route only carries
// :category and :hash, so resolution walks the yyyy/mm levels: ListObjectsV2 on
// R2, filepath.Glob on local disk.
//
// Delivery depends on the prefix:
//   - public  (avatars, covers, units, news, community) -> 302 to R2_PUBLIC_BASE
//   - private (evidence, gov_ids, voice_notes, sos_media, report_media) -> 302 to
//     a short-lived presigned GET URL. A public URL is never returned for these.
//
// Access rules:
//   - avatars/*, covers/*, units/*, news/*, community/*  : any authenticated user
//   - evidence/*, voice_notes/*  : requires case access via caseId query param
//   - sos_media/*                : requires the sosId to pass sosReadScope
//   - report_media/*             : requires reportId whose post is visible, or
//     the caller canModeratePost
//   - gov_ids/*                  : admin only
func ServeFile(c *gin.Context) {
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}
	userObj, ok := userInterface.(*models.User)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user"})
		return
	}

	category := c.Param("category")
	hash := c.Param("hash")

	if category == "" || hash == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "category and hash are required"})
		return
	}
	if strings.Contains(hash, "..") || strings.Contains(category, "..") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path"})
		return
	}

	if !authorizeFileCategory(c, category, userObj) {
		return
	}

	storageSvc := services.NewFileStorageService()
	key, err := storageSvc.FindByHash(category, strings.ToLower(hash))
	if err != nil || key == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	// Private keys must belong to the exact parent being accessed, not merely
	// share a category with some other readable parent. Otherwise a known media
	// hash from an unrelated case/SOS could be read via an accessible parent ID.
	switch category {
	case "sos_media", "report_media":
		parentType, parentParam := MediaParentSOS, "sosId"
		if category == "report_media" {
			parentType, parentParam = MediaParentReport, "reportId"
		}
		parentID, parseErr := uuid.Parse(c.Query(parentParam))
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid media parent"})
			return
		}
		var count int64
		err := config.DB.Model(&models.MediaLink{}).
			Where("object_key = ? AND parent_type = ? AND parent_id = ?", key, parentType, parentID).
			Count(&count).Error
		if err != nil || count == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "media is not attached to this record"})
			return
		}
	case "evidence":
		caseID, parseErr := uuid.Parse(c.Query("caseId"))
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid case ID"})
			return
		}
		var count int64
		err := config.DB.Model(&models.Evidence{}).
			Where("case_id = ? AND (file_path = ? OR file_url = ?)", caseID, key, key).
			Count(&count).Error
		if err != nil || count == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "evidence is not attached to this case"})
			return
		}
	}

	r2 := storageSvc.R2()
	if r2 == nil {
		// Local disk mode: stream straight from disk.
		serveStoredFile(c, storageSvc, key)
		return
	}

	if services.IsPublicStorageKey(key) {
		target := r2.PublicURL(key)
		if target == "" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "R2_PUBLIC_BASE is not configured"})
			return
		}
		c.Redirect(http.StatusFound, target)
		return
	}

	// Private prefix: short-lived presigned GET instead of a public URL.
	target, err := r2.PresignGet(c.Request.Context(), key, privateReadTTL)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}
	c.Redirect(http.StatusFound, target)
}

// serveStoredFile streams a key out of the local disk backend.
func serveStoredFile(c *gin.Context, storageSvc *services.FileStorageService, key string) {
	reader, err := storageSvc.Open(key)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}
	defer reader.Close()

	c.Header("Content-Type", contentTypeFromExt(key))
	c.Header("Cache-Control", "private, max-age=3600")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Status(http.StatusOK)
	_, _ = io.Copy(c.Writer, reader)
}

// authorizeFileCategory applies the per-category access rules. It writes the
// error response itself and returns false when access is denied.
func authorizeFileCategory(c *gin.Context, category string, userObj *models.User) bool {
	switch category {
	case "avatars", "covers", "units", "news", "community":
		// public-safe: any authed user may read
		return true
	case "evidence", "voice_notes":
		caseIDStr := c.Query("caseId")
		if caseIDStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "caseId query param required for evidence"})
			return false
		}
		caseID, err := uuid.Parse(caseIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid caseId"})
			return false
		}

		accessLevel := c.GetString("case_access_level")
		if accessLevel != "" {
			return true
		}

		// fall back to an explicit check
		var membership models.UnitMembership
		var caseObj models.Case
		if err := config.DB.First(&caseObj, "id = ?", caseID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Case not found"})
			return false
		}
		isReporter := caseObj.ReportedBy == userObj.ID
		hasMembership := config.DB.
			Where("unit_id = ? AND user_id = ? AND status = ?", caseObj.UnitID, userObj.ID, models.MembershipActive).
			First(&membership).Error == nil
		if !isReporter && !hasMembership && !userObj.IsSuperAdmin && userObj.Role != "super_admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return false
		}
		return true
	case "gov_ids":
		if !userObj.IsSuperAdmin && userObj.Role != "super_admin" && userObj.Role != "unit_admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			return false
		}
		return true
	case "sos_media":
		// The attachment is only as readable as its parent alert. sosReadScope is
		// the exact predicate the alert detail uses, so this cannot drift from it.
		sosIDStr := strings.TrimSpace(c.Query("sosId"))
		if sosIDStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "sosId query param required for SOS media"})
			return false
		}
		sosID, err := uuid.Parse(sosIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid sosId"})
			return false
		}
		var alert models.SOSAlert
		if err := sosReadScope(config.DB, userObj).
			First(&alert, "sos_alerts.id = ?", sosID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "SOS not found or not accessible"})
			return false
		}
		return true
	case "report_media":
		// Readable by anyone who can see the reported post, and by a moderator who
		// can action it — the same pair the report itself is gated by.
		reportIDStr := strings.TrimSpace(c.Query("reportId"))
		if reportIDStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "reportId query param required for report media"})
			return false
		}
		reportID, err := uuid.Parse(reportIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid reportId"})
			return false
		}
		var report models.Report
		if err := config.DB.First(&report, "id = ?", reportID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Report not found"})
			return false
		}
		postID, err := uuid.Parse(report.TargetID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Reported post not found"})
			return false
		}
		var post models.ForumPost
		if err := config.DB.First(&post, "id = ?", postID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Reported post not found"})
			return false
		}
		lat, lng := parseCoords(c)
		if !postVisibleTo(userObj, post, lat, lng) &&
			!canModeratePost(userObj, post, resolveViewerPlace(userObj, lat, lng)) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return false
		}
		return true
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown category"})
		return false
	}
}

func contentTypeFromExt(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	case ".pdf":
		return "application/pdf"
	case ".mp4":
		return "video/mp4"
	case ".webm":
		return "video/webm"
	case ".mov":
		return "video/quicktime"
	case ".mp3":
		return "audio/mpeg"
	case ".wav":
		return "audio/wav"
	case ".ogg":
		return "audio/ogg"
	case ".m4a":
		return "audio/mp4"
	}
	return "application/octet-stream"
}
