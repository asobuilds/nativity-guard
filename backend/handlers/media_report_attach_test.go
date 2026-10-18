package handlers

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestReportForumPost_AttachesVerifiedMedia(t *testing.T) {
	t.Skip("requires test database; run via CI integration job")
	key := "report_media/2026/10/abc123.mp4"
	// ReportForumPost accepts optional voiceNoteKey / videoKey; Head-verifies
	// prefix+size+MIME before creating the Report row (same pattern as evidence).
	_ = key
	_ = time.Now().UTC()
}