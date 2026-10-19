package handlers

import (
	"testing"

	"github.com/google/uuid"

	"security-solution/internal/testutil"
)

func TestMediaLink_AttachCreatesRow(t *testing.T) {
	skipWithoutDB(t)
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	store := newMediaStore(t)
	key, _ := store.put("sos_media", "clip.mp4", []byte("shared bytes"))

	parentA := uuid.New()
	parentB := uuid.New()

	if err := attachMediaLink(key, MediaParentSOS, parentA); err != nil {
		t.Fatalf("attach media link: %v", err)
	}
	if got := mediaLinkCount(t, key, &parentA); got != 1 {
		t.Fatalf("want 1 link row for the first parent, got %d", got)
	}

	// Re-attaching the same key to the same parent is a no-op, not an error:
	// the unique index on (object_key, parent_type, parent_id) makes a retried
	// request safe.
	if err := attachMediaLink(key, MediaParentSOS, parentA); err != nil {
		t.Fatalf("re-attach media link: %v", err)
	}
	if got := mediaLinkCount(t, key, &parentA); got != 1 {
		t.Fatalf("a repeated attach must not create a second row, got %d", got)
	}

	// The store is content-addressed, so the same bytes under a different
	// parent are a second link to the same object.
	if err := attachMediaLink(key, MediaParentSOS, parentB); err != nil {
		t.Fatalf("attach media link to second parent: %v", err)
	}
	if got := mediaLinkCount(t, key, &parentB); got != 1 {
		t.Fatalf("want 1 link row for the second parent, got %d", got)
	}
	if got := mediaLinkCount(t, key, nil); got != 2 {
		t.Fatalf("want 2 link rows for the shared key in total, got %d", got)
	}
}
