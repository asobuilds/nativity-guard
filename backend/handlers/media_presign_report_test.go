package handlers

import (
	"net/http"
	"testing"
)

func TestPresignReportMedia_RequiresPostVisibility(t *testing.T) {
	t.Skip("requires test database; run via CI integration job")
	req := `{"filename": "test.mp4", "contentType": "video/mp4", "sizeBytes": 1000000, "sha256": "abc"}`
	_, status := presignReportHandlerRequest(req, t)
	if status != http.StatusBadRequest {
		t.Fatalf("expected 400 when post not visible, got %d", status)
	}
}