package handlers

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestSendSOSAlert_AttachesVerifiedMedia(t *testing.T) {
	t.Skip("requires test database; run via CI integration job")
	now := time.Now().UTC()
	key := "sos_media/2026/10/abc123.mp4"
	_, err := sendSOSAlertWithMediaRequest(t, key, "video/mp4", 1000000)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSendSOSAlert_RejectsKeyOutsidePrefix(t *testing.T) {
	t.Skip("requires test database; run via CI integration job")
	req := `{"latitude": 0, "longitude": 0, "description": "test", "unitId": "1", "emergencyContacts": [], "voiceNoteKey": "evidence/somekey.mp4", "videoKey": ""}`
	_, status := sendSOSAlertHandlerRequest(req, t)
	if status != http.StatusBadRequest {
		t.Fatalf("expected 400 for key outside prefix, got %d", status)
	}
}