package handlers

import (
	"testing"

	"github.com/google/uuid"

	"security-solution/internal/testutil"
)

// TestSOSMedia_DedupSharedKey_NotDeletedOnParentDelete guards the refcount
// invariant in media_links.go §5.3: an object shared across parents survives
// the deletion of its first link, and is removed only when the last one is.
func TestSOSMedia_DedupSharedKey_NotDeletedOnParentDelete(t *testing.T) {
	skipWithoutDB(t)
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	store := newMediaStore(t)
	key, _ := store.put("sos_media", "clip.mp4", []byte("identical bytes"))

	parentA := uuid.New()
	parentB := uuid.New()
	if err := attachMediaLink(key, MediaParentSOS, parentA); err != nil {
		t.Fatalf("attach to parent A: %v", err)
	}
	if err := attachMediaLink(key, MediaParentSOS, parentB); err != nil {
		t.Fatalf("attach to parent B: %v", err)
	}

	linkA := mediaLinkID(t, key, parentA)
	linkB := mediaLinkID(t, key, parentB)

	// Deleting one parent's link must not remove the shared object: another
	// parent still references it, so deleting would break that parent.
	if err := removeMediaLink(linkA); err != nil {
		t.Fatalf("removeMediaLink(linkA): %v", err)
	}
	if !store.exists(key) {
		t.Fatal("the shared object must survive while another parent still links to it")
	}
	if got := mediaLinkCount(t, key, nil); got != 1 {
		t.Fatalf("want 1 remaining link row for the shared key, got %d", got)
	}

	// The last link removes the object; a stale orphan cannot be cleaned up later.
	if err := removeMediaLink(linkB); err != nil {
		t.Fatalf("removeMediaLink(linkB): %v", err)
	}
	if store.exists(key) {
		t.Error("the object must be removed once the last link to it is gone")
	}
	if got := mediaLinkCount(t, key, nil); got != 0 {
		t.Fatalf("want 0 link rows after the last delete, got %d", got)
	}
}
