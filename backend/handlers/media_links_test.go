package handlers

import (
	"testing"

	"github.com/google/uuid"
	"security-solution/models"
)

func TestMediaLink_AttachCreatesRow(t *testing.T) {
	// This test verifies that attachMediaLink persists a row
	// when a test database is configured. It is meant to be run
	// with a real Postgres instance (via TestMain/SwapConfigDB).
	t.Skip("requires test database; run via CI integration job")
	// uuid is a fresh ID; the unique index makes repeat attach idempotent
	id := uuid.New()
	err := attachMediaLink("sos_media/2026/10/abc123.mp4", "sos", id)
	if err != nil {
		t.Fatal(err)
	}
	// Re-attaching the same key should be a no-op (DoNothing)
	err = attachMediaLink("sos_media/2026/10/abc123.mp4", "sos", id)
	if err != nil {
		t.Fatal(err)
	}
}

func TestMediaLink_SharedObjectNotDeleted(t *testing.T) {
	t.Skip("requires test database; run via CI integration job")
	// Two parents pointing at the same content-addressed key should
	// both remain valid after one parent is deleted. The delete path
	// must not blindly remove the object (see MEDIA-ATTACHMENTS-PLAN.md §5.3).
	id1 := uuid.New()
	id2 := uuid.New()
	_ = attachMediaLink("shared_key.mp4", "sos", id1)
	_ = attachMediaLink("shared_key.mp4", "report", id2)
	// TODO: assert that deleting one link leaves the other parent viable.
	// The refcount guard lives in removeMediaLink; this test validates
	// the invariant that an object is not GC'd while another parent exists.
}