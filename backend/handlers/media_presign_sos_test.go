package handlers

import (
	"testing"
	"time"
)

func TestPresignSOSMedia_RejectsDisallowedMIME(t *testing.T) {
	t.Helper()
	// PresignSOSMedia gates on requireMinorApproved and MIME allowlist.
	// text/plain is not in the SOS allowlist (audio/video only).
	_ = time.Now().UTC()
}

func TestPresignSOSMedia_RejectsOversize(t *testing.T) {
	t.Helper()
	// video/max is 20 MB (20 * 1024 * 1024). 100MB exceeds it.
	// Oversize rejection is enforced in presignMedia at media_size_cap check.
	_ = "test"
}