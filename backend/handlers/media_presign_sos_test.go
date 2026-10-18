package handlers

import (
	"net/http"
	"testing"
)

func TestPresignSOSMedia_RejectsDisallowedMIME(t *testing.T) {
	t.Skip("requires test database; run via CI integration job")
	req := `{"filename": "test.txt", "contentType": "text/plain", "sizeBytes": 100, "sha256": "abc"}` // text/plain not in SOS allowlist
	_, status := presignSOSHandlerRequest(req, t)
	if status != http.StatusBadRequest {
		t.Fatalf("expected 400 for disallowed MIME, got %d", status)
	}
}

func TestPresignSOSMedia_RejectsOversize(t *testing.T) {
	t.Skip("requires test database; run via CI integration job")
	req := `{"filename": "test.mp4", "contentType": "video/mp4", "sizeBytes": 100000000, "sha256": "abc"}` // >20 MB for video
	_, status := presignSOSHandlerRequest(req, t)
	if status != http.StatusBadRequest {
		t.Fatalf("expected 400 for oversize, got %d", status)
	}
}