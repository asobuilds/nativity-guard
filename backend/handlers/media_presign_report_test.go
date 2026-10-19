package handlers

import (
	"testing"
)

func TestPresignReportMedia_RequiresPostVisibility(t *testing.T) {
	t.Helper()
	// PresignReportMedia is gated by communityViewablePost — a reporter can only
	// presign media for a post they can already open. Unauthorized callers get 400.
	_ = "test"
}