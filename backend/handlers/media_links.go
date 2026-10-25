package handlers

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm/clause"

	"security-solution/config"
	"security-solution/models"
	"security-solution/services"
)

// Media parent types for media_links.parent_type. Constant so the writer and any
// future reader agree on one spelling.
const (
	MediaParentSOS    = "sos"
	MediaParentReport = "report"
)

// Media object prefixes. Both are private (never served as a public CDN URL):
// they are only reachable through ServeFile's per-category access check.
const (
	sosMediaPrefix    = "sos_media/"
	reportMediaPrefix = "report_media/"
)

// Media size caps. A voice note mirrors MaxImageSize (5 MB); video is a fifth of
// the evidence cap (20 MB), deliberately below evidence — this is a situational
// hint, not evidence of record (docs/community/MEDIA-ATTACHMENTS-PLAN.md §4.2).
//
// Duration is deliberately NOT enforced: there is no prober (ffprobe/ffmpeg) in
// the stack, so a server-side time limit would be a lie. The size cap is the
// honest proxy and the client owns the 60s/30s limits.
const (
	voiceMediaMaxBytes int64 = 5 << 20  // 5 MB
	videoMediaMaxBytes int64 = 20 << 20 // 20 MB
)

// mediaKind classifies a declared MIME type into the two attachment slots.
// ok=false means the type is neither voice nor video and must be rejected.
func mediaKind(contentType string) (kind string, ok bool) {
	switch strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0])) {
	case "audio/mpeg", "audio/mp4", "audio/wav", "audio/ogg", "audio/webm":
		return "voice", true
	case "video/mp4", "video/webm", "video/quicktime":
		return "video", true
	default:
		return "", false
	}
}

// mediaSizeCap returns the server-enforced byte cap for a media kind.
func mediaSizeCap(kind string) int64 {
	if kind == "voice" {
		return voiceMediaMaxBytes
	}
	return videoMediaMaxBytes
}

// presignMedia is the shared body of the SOS and report presign endpoints: it
// validates the declared type/size/hash, then issues a short-lived R2 PUT URL.
// The only difference between the two callers is the key prefix, which is the
// storage boundary that later authorizes the read.
func presignMedia(c *gin.Context, prefix string) {
	if !requireMinorApproved(c) {
		return
	}

	var input presignInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	filename := strings.TrimSpace(input.Filename)
	if filename == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "filename is required"})
		return
	}

	contentType := strings.TrimSpace(input.ContentType)
	kind, ok := mediaKind(contentType)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "content type must be an allowed audio or video type"})
		return
	}

	if input.SizeBytes <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sizeBytes must be greater than zero"})
		return
	}
	if input.SizeBytes > mediaSizeCap(kind) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file exceeds the maximum size for this media type"})
		return
	}

	sha := strings.ToLower(strings.TrimSpace(input.SHA256))
	if !hexSHA256.MatchString(sha) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sha256 must be a 64 character hex digest"})
		return
	}

	ext := strings.ToLower(path.Ext(filename))
	if ext == "" || ext == "." {
		ext = ".bin"
	}

	storageSvc := services.NewFileStorageService()
	r2 := storageSvc.R2()

	now := time.Now().UTC()
	dirName := strings.TrimSuffix(prefix, "/")
	key := filepath.ToSlash(filepath.Join(dirName, now.Format("2006"), now.Format("01"), sha+ext))
	expiresAt := time.Now().UTC().Add(presignTTL)

	if r2 == nil {
		uploadEndpoint := "/sos/media/upload"
		if strings.Contains(prefix, "report") {
			uploadEndpoint = "/community/posts/report/upload"
		} else if strings.Contains(prefix, "evidence") {
			uploadEndpoint = "/evidence/upload"
		}
		c.JSON(http.StatusOK, gin.H{
			"uploadUrl": uploadEndpoint,
			"key":       key,
			"expiresAt": expiresAt,
		})
		return
	}

	uploadURL, err := r2.PresignPut(c.Request.Context(), key, contentType, input.SizeBytes, map[string]string{
		"sha256":            sha,
		"original-filename": filename,
	}, presignTTL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to presign upload"})
		return
	}

	// Both prefixes are private: the key is returned, never a public URL.
	c.JSON(http.StatusOK, gin.H{
		"uploadUrl": uploadURL,
		"key":       key,
		"expiresAt": expiresAt,
	})
}

// verifyMediaObject re-checks an attached key against the store before the link
// is written: the object must exist, live under the expected private prefix, and
// be within the slot's size cap. This is the attach-time half of the plan's
// "enforce size at presign AND by Head on attach" rule — a client can PUT bytes
// that differ from what it declared, so the declared size is not trusted.
//
// It mirrors ConfirmEvidenceUpload's check (handlers/evidence_presign_handler.go)
// through the backend-agnostic FileStorageService.Head, so it works against R2
// and the local-disk fallback.
func verifyMediaObject(key, expectedPrefix string, cap int64) error {
	clean := strings.TrimSpace(key)
	if clean == "" {
		return errors.New("media key is empty")
	}
	if !services.IsPrivateStorageKey(clean) || !strings.HasPrefix(clean, expectedPrefix) {
		return fmt.Errorf("key must be a %s object key", strings.TrimSuffix(expectedPrefix, "/"))
	}
	if strings.Contains(clean, "..") || strings.Contains(clean, "//") {
		return errors.New("invalid key")
	}

	size, _, err := services.NewFileStorageService().Head(clean)
	if err != nil {
		return errors.New("uploaded object not found")
	}
	if size > cap {
		return fmt.Errorf("stored object exceeds the %d byte cap for this attachment", cap)
	}
	return nil
}

// attachMediaLink records one attachment. The unique index on
// (object_key, parent_type, parent_id) makes a repeated attach a no-op rather
// than an error, so a retried request cannot fail on a link that already exists.
func attachMediaLink(objectKey, parentType string, parentID uuid.UUID) error {
	if config.DB == nil {
		return errors.New("database not configured")
	}
	link := models.MediaLink{ObjectKey: objectKey, ParentType: parentType, ParentID: parentID}
	return config.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&link).Error
}

// removeMediaLink deletes one attachment link and, only when it was the last
// link to that object, the object itself. Identical bytes are shared across
// parents by the content-addressed store, so deleting the object while another
// parent still points at it would break that parent (plan §5.3).
//
// No route calls this yet: RES-020 ships the link rows and the attach path; the
// delete path is a later row. The refcount rule lives here now so that future
// delete cannot get it wrong by copying the avatar/cover shortcut.
func removeMediaLink(linkID uuid.UUID) error {
	if config.DB == nil {
		return errors.New("database not configured")
	}

	var link models.MediaLink
	if err := config.DB.First(&link, "id = ?", linkID).Error; err != nil {
		return err
	}
	if err := config.DB.Delete(&models.MediaLink{}, "id = ?", link.ID).Error; err != nil {
		return err
	}

	var remaining int64
	if err := config.DB.Model(&models.MediaLink{}).
		Where("object_key = ?", link.ObjectKey).
		Count(&remaining).Error; err != nil {
		return err
	}
	if remaining > 0 {
		return nil
	}
	return services.NewFileStorageService().Delete(link.ObjectKey)
}

// attachVerifiedMedia runs verify-then-link for one optional key. On a
// verification failure it writes the 400 and reports false, so the caller aborts
// before anything is persisted.
func attachVerifiedMedia(c *gin.Context, mediaKey, expectedPrefix string, cap int64) (string, bool) {
	mediaKey = strings.TrimSpace(mediaKey)
	if mediaKey == "" {
		return "", true
	}
	if err := verifyMediaObject(mediaKey, expectedPrefix, cap); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return "", false
	}
	return mediaKey, true
}

// linkMedia best-effort records the link rows for the verified keys of a parent
// that was just created. A link failure must not fail the parent write (an SOS
// is time-critical, and the report already exists), so it is logged and carried
// on; the object simply stays unattached until the retry or the orphan sweep.
func linkMedia(parentType string, parentID uuid.UUID, mediaKeys ...string) (attached, failed int) {
	for _, key := range mediaKeys {
		if strings.TrimSpace(key) == "" {
			continue
		}
		if err := attachMediaLink(key, parentType, parentID); err != nil {
			failed++
			log.Printf("media link attach failed for %s %s key %s: %v", parentType, parentID, key, err)
		} else {
			attached++
		}
	}
	return attached, failed
}
