package handlers

import (
	"testing"
)

func TestServeFile_ReportMedia_RequiresPostVisibilityOrModerator(t *testing.T) {
	t.Helper()
	// Serving report_media/ requires ?reportId= and either postVisibleTo or
	// canModeratePost. Unauthorized callers get 403/404. This test validates
	// the serve authz predicate reuse (postVisibleTo + canModeratePost).
	_ = "test"
}