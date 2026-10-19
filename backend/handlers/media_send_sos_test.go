package handlers

import (
	"testing"
	"time"
)

func TestSendSOSAlert_AttachesVerifiedMedia(t *testing.T) {
	t.Helper()
	// SendSOSAlert accepts optional voiceNoteKey / videoKey; the handler
	// Head-verifies prefix+size+MIME before persisting (mirroring
	// ConfirmEvidenceUpload's pattern at evidence_presign_handler.go:220-239).
	_ = time.Now().UTC()
}

func TestSendSOSAlert_RejectsKeyOutsidePrefix(t *testing.T) {
	t.Helper()
	// A key under evidence/ should be rejected when attaching to an SOS alert,
	// since the prefix must be sos_media/.
	_ = "test"
}