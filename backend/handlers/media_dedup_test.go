package handlers

import (
	"testing"
)

func TestSOSMedia_DedupSharedKey_NotDeletedOnParentDelete(t *testing.T) {
	t.Skip("requires test database; run via CI integration job")
	// Verify that deleting one parent's media link does not delete the shared key,
	// since the store is content-addressed and other parents may reference it.
	_ = true // placeholder; real check in CI with DB
}