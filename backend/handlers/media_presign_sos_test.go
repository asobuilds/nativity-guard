package handlers

import (
	"net/http"
	"strings"
	"testing"

	"security-solution/models"
)

func TestPresignSOSMedia_RejectsDisallowedMIME(t *testing.T) {
	// No database needed: the presign gate reads the caller out of the context.
	body := map[string]any{
		"filename":    "note.txt",
		"contentType": "text/plain",
		"sizeBytes":   1024,
		"sha256":      strings.Repeat("a", 64),
	}

	// A minor without a guardian exception is refused before anything else.
	minor := &models.User{MinorStatus: "minor"}
	ctx, recorder := mediaContext(t, "POST", "/sos/media/presign", nil, minor, body)
	PresignSOSMedia(ctx)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("a minor without an exception must be refused, got %d %s", recorder.Code, recorder.Body.String())
	}

	// text/plain is not audio or video: the allowlist rejects it.
	adult := &models.User{MinorStatus: "adult"}
	ctx, recorder = mediaContext(t, "POST", "/sos/media/presign", nil, adult, body)
	PresignSOSMedia(ctx)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("want 400 for a disallowed content type, got %d %s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "content type") {
		t.Errorf("the error must name the content type, got %s", recorder.Body.String())
	}
}

func TestPresignSOSMedia_RejectsOversize(t *testing.T) {
	// Force the local-disk backend so the storage step is reachable but
	// unconfigured: the presign then fails with 503 instead of reaching the
	// network, which keeps the size check the only variable in this test.
	newMediaStore(t)

	adult := &models.User{MinorStatus: "adult"}

	// video is capped at 20 MB; 100 MB is over it.
	video := map[string]any{
		"filename":    "clip.mp4",
		"contentType": "video/mp4",
		"sizeBytes":   100 * 1024 * 1024,
		"sha256":      strings.Repeat("b", 64),
	}
	ctx, recorder := mediaContext(t, "POST", "/sos/media/presign", nil, adult, video)
	PresignSOSMedia(ctx)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("want 400 for an oversize video, got %d %s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "maximum size") {
		t.Errorf("the error must name the size cap, got %s", recorder.Body.String())
	}

	// A voice note is capped at 5 MB; 6 MB is over it.
	voice := map[string]any{
		"filename":    "note.mp3",
		"contentType": "audio/mpeg",
		"sizeBytes":   6 * 1024 * 1024,
		"sha256":      strings.Repeat("c", 64),
	}
	ctx, recorder = mediaContext(t, "POST", "/sos/media/presign", nil, adult, voice)
	PresignSOSMedia(ctx)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("want 400 for an oversize voice note, got %d %s", recorder.Code, recorder.Body.String())
	}

	// A video inside the cap passes validation and returns local fallback upload endpoint.
	within := map[string]any{
		"filename":    "clip.mp4",
		"contentType": "video/mp4",
		"sizeBytes":   1024 * 1024,
		"sha256":      strings.Repeat("d", 64),
	}
	ctx, recorder = mediaContext(t, "POST", "/sos/media/presign", nil, adult, within)
	PresignSOSMedia(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("a video within the cap must pass validation and return local fallback, got %d %s",
			recorder.Code, recorder.Body.String())
	}
}
