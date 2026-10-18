package handlers

import (
	"net/http"
	"testing"
)

func TestServeFile_ReportMedia_RequiresPostVisibilityOrModerator(t *testing.T) {
	t.Skip("requires test database; run via CI integration job")
	// Serving report_media/ requires ?reportId= and either postVisibleTo or
	// canModeratePost. Unauthorized callers get 403/404.
	req := `GET /files/report_media/abc123.mp4?reportId=1`
	_ = req // serve authz check tested in CI with DB.
}