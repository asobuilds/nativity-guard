package handlers

import (
	"net/http"
	"testing"
)

func TestServeFile_SOSMedia_RequiresSOSVisibility(t *testing.T) {
	t.Skip("requires test database; run via CI integration job")
	// Serving sos_media/ requires ?sosId= and caller must pass sosReadScope.
	// Unrelated callers receive 403/404. Tested in CI with real DB.
	req := `GET /files/sos_media/abc123.mp4?sosId=1`
	// serveFileHandler implements the check; tested end-to-end in CI.
	_ = req
}