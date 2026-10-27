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

// TestPresignReportMedia_RequiresPostVisibility pins the D-014 boundary on
// the media endpoint: a caller can only presign media for a post they can
// already open, so the presign cannot be used to probe content outside the
// caller's area.
func TestPresignReportMedia_RequiresPostVisibility(t *testing.T) {
	skipWithoutDB(t)
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	newMediaStore(t)

	author := testutil.MakeUser(t, "citizen")
	hidden := &models.ForumPost{
		Title:    "Outside your area",
		Content:  "content",
		AuthorID: author.ID,
		Status:   "published",
		GeoPlace: models.GeoPlace{Scope: ScopeLGA, State: "FCT", LGA: "Kuje"},
	}
	if err := config.DB.Create(hidden).Error; err != nil {
		t.Fatalf("create post: %v", err)
	}

	open := &models.ForumPost{
		Title:    "Open to everyone",
		Content:  "content",
		AuthorID: author.ID,
		Status:   "published",
		GeoPlace: models.GeoPlace{Scope: ScopeWorld},
	}
	if err := config.DB.Create(open).Error; err != nil {
		t.Fatalf("create post: %v", err)
	}

	// The caller has no place of their own, so the LGA-scoped post is closed
	// to them.
	caller := testutil.MakeUser(t, "citizen")
	body := map[string]any{
		"filename":    "clip.mp4",
		"contentType": "video/mp4",
		"sizeBytes":   1024,
		"sha256":      strings.Repeat("e", 64),
	}

	ctx, recorder := mediaContext(t, "POST", "/community/posts/"+hidden.ID.String()+"/report/presign",
		gin.Params{{Key: "id", Value: hidden.ID.String()}}, caller, body)
	PresignReportMedia(ctx)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("want 403 presigning media for a post the caller cannot open, got %d %s",
			recorder.Code, recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), "key") {
		t.Errorf("a refused presign must not return a storage key, got %s", recorder.Body.String())
	}

	// A post the caller can open passes the gate and reaches the storage step.
	// With MEDIA_* unset, presignMedia returns the local upload fallback.
	ctx, recorder = mediaContext(t, "POST", "/community/posts/"+open.ID.String()+"/report/presign",
		gin.Params{{Key: "id", Value: open.ID.String()}}, caller, body)
	PresignReportMedia(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("a visible post must pass the visibility gate and reach the storage step, got %d %s",
			recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "/community/posts/report/upload") {
		t.Errorf("a visible post must return the report upload fallback, got %s", recorder.Body.String())
	}
}
