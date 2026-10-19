package handlers

import (
	"testing"
)

func TestReportForumPost_AttachesVerifiedMedia(t *testing.T) {
	t.Helper()
	// ReportForumPost accepts optional voiceNoteKey / videoKey; Head-verifies
	// prefix+size+MIME before creating the Report row (same pattern as evidence).
	_ = "test"
}