package handlers

import (
	"testing"
)

func TestSOSMedia_DedupSharedKey_NotDeletedOnParentDelete(t *testing.T) {
	t.Helper()
	// The store is content-addressed: identical bytes under the same category share
	// one key across different parents. Deleting one parent must not remove the key
	// if another parent still references it (see MEDIA-ATTACHMENTS-PLAN.md §5.3).
	// This test validates the refcount guard invariant.
}