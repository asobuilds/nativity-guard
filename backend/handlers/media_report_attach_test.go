package handlers

import (
	"net/http"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"security-solution/config"
	"security-solution/internal/testutil"
	"security-solution/models"
)

// TestReportForumPost_AttachesVerifiedMedia pins the report attach path: a
// private report_media/ key is re-verified against the store before the report
// is written, and the link row is written against the new report.
func TestReportForumPost_AttachesVerifiedMedia(t *testing.T) {
	skipWithoutDB(t)
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	store := newMediaStore(t)

	author := testutil.MakeUser(t, "citizen")
	post := &models.ForumPost{
		Title:    "Reported post",
		Content:  "content",
		AuthorID: author.ID,
		Status:   "published",
		GeoPlace: models.GeoPlace{Scope: ScopeWorld},
	}
	if err := config.DB.Create(post).Error; err != nil {
		t.Fatalf("create post: %v", err)
	}

	// fileReport files a report from a fresh reporter and returns the status
	// code and the response body.
	fileReport := func(t *testing.T, body map[string]any) (int, string, *models.User) {
		t.Helper()
		reporter := testutil.MakeUser(t, "citizen")
		ctx, recorder := mediaContext(t, "POST", "/community/posts/"+post.ID.String()+"/report",
			gin.Params{{Key: "id", Value: post.ID.String()}}, reporter, body)
		ReportForumPost(ctx)
		return recorder.Code, recorder.Body.String(), reporter
	}

	t.Run("verified media is attached", func(t *testing.T) {
		key, _ := store.put("report_media", "voice.mp3", []byte("voice bytes"))

		code, body, reporter := fileReport(t, map[string]any{"reason": "spam", "voiceNoteKey": key})
		if code != http.StatusCreated {
			t.Fatalf("want 201 creating the report, got %d %s", code, body)
		}

		var report models.Report
		if err := config.DB.Where("reported_by = ? AND target_id = ?", reporter.ID, post.ID.String()).
			First(&report).Error; err != nil {
			t.Fatalf("load report: %v", err)
		}

		var link models.MediaLink
		if err := config.DB.Where("object_key = ? AND parent_type = ?", key, MediaParentReport).
			First(&link).Error; err != nil {
			t.Fatalf("load media link: %v", err)
		}
		if link.ParentID != report.ID {
			t.Errorf("the link must point at the new report: got %s, want %s", link.ParentID, report.ID)
		}
	})

	t.Run("a key outside the prefix is refused", func(t *testing.T) {
		// The bytes are real, but they live under the SOS prefix: an SOS
		// attachment must not be readable as a report attachment.
		key, _ := store.put("sos_media", "voice.mp3", []byte("other bytes"))

		code, body, _ := fileReport(t, map[string]any{"reason": "spam", "voiceNoteKey": key})
		if code != http.StatusBadRequest {
			t.Fatalf("want 400 for a key outside the report_media prefix, got %d %s", code, body)
		}
		if got := mediaLinkCount(t, key, nil); got != 0 {
			t.Errorf("a refused key must not be linked, got %d link rows", got)
		}
	})

	t.Run("a key that is not in the store is refused", func(t *testing.T) {
		key := "report_media/2026/10/" + strings.Repeat("f", 64) + ".mp3"

		code, body, _ := fileReport(t, map[string]any{"reason": "spam", "voiceNoteKey": key})
		if code != http.StatusBadRequest {
			t.Fatalf("want 400 for a key that is not in the store, got %d %s", code, body)
		}
		if got := mediaLinkCount(t, key, nil); got != 0 {
			t.Errorf("a refused key must not be linked, got %d link rows", got)
		}
	})
}
