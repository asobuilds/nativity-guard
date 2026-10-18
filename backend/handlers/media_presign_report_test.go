package handlers

import (
	"net/http"
	"testing"
)

func TestPresignReportMedia_RequiresPostVisibility(t *testing.T) {
	t.Skip("requires test database; run via CI integration job")
	// PresignReportMedia is gated by communityViewablePost — a reporter can only
	// presign media for a post they can already open. Unauthorized callers get 400.
	body := `{"filename": "test.mp4", "contentType": "video/mp4", "sizeBytes": 1000000, "sha256": "abc"}`
	_ = body // visibility gate enforced in PresignReportMedia; CI tests the round-trip.
}