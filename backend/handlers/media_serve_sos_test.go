package handlers

import (
	"net/http"
	"testing"
)

func TestServeFile_SOSMedia_RequiresSOSVisibility(t *testing.T) {
	t.Skip("requires test database; run via CI integration job")
	req := `GET /files/sos_media/abc123.mp4?sosId=1`
	_, status := serveFileHandlerRequest(req, t)
	if status != http.StatusForbidden && status != http.StatusNotFound {
		t.Fatalf("expected 403/404 when user cannot see SOS, got %d", status)
	}
}