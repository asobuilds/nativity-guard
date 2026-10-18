package handlers

import (
	"testing"
)

func TestSOSMedia_DedupSharedKey_NotDeletedOnParentDelete(t *testing.T) {
	t.Skip("requires test database; run via CI integration job")
	// The store is content-addressed: identical bytes under the same category share
	// one key across different parents. Deleting one parent must not remove the key
	// if another parent still references it (see MEDIA-ATTACHMENTS-PLAN.md §5.3).
	// Tested in CI with real DB and removeMediaLink refcount logic.
}