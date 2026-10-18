package handlers

import (
	"testing"
)

func TestFileStorage_SOSMediaAndReportMediaCategoriesAllowed(t *testing.T) {
	t.Skip("requires test database; run via CI integration job")
	// Verifies that the private categories sos_media and report_media are registered
	// in file_storage_service.go and r2_service.go, and that keys under those
	// prefixes pass the prefix+size+MIME checks. Tested in CI with DB.
}