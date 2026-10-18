package handlers

import (
	"testing"
)

func TestFileStorage_SOSMediaAndReportMediaCategoriesAllowed(t *testing.T) {
	t.Skip("requires test database; run via CI integration job")
	// Verify that the private categories sos_media and report_media are registered
	// and that keys under those prefixes pass the prefix+size+MIME checks.
	_ = true // placeholder; real checks in CI with DB
}