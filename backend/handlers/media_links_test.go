package handlers

import (
	"testing"

	"github.com/google/uuid"

	"security-solution/config"
)

func TestMediaLink_AttachCreatesRow(t *testing.T) {
	if config.DB == nil {
		t.Skip("requires database configuration")
	}
	t.Helper()
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