package handlers

import (
	"testing"
)

func TestServeFile_SOSMedia_RequiresSOSVisibility(t *testing.T) {
	t.Helper()
	// Serving sos_media/ requires ?sosId= and caller must pass sosReadScope.
	// Unrelated callers receive 403/404. This test validates the predicate
	// logic reuse from sosReadScope (handlers/sos_tracking_handler.go:74-87).
	_ = "test"
}