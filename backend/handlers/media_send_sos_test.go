package handlers

import (
	"net/http"
	"testing"

	"security-solution/config"
	"security-solution/internal/testutil"
	"security-solution/models"
)

// TestSendSOSAlert_AttachesVerifiedMedia pins the SOS attach path: a private
// sos_media/ key is re-verified against the store before the alert is
// persisted, and the link row is written against the new alert.
func TestSendSOSAlert_AttachesVerifiedMedia(t *testing.T) {
	skipWithoutDB(t)
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	store := newMediaStore(t)

	caller := testutil.MakeUser(t, "citizen")
	key, _ := store.put("sos_media", "voice.mp3", []byte("voice bytes"))

	ctx, recorder := mediaContext(t, "POST", "/sos/send", nil, caller, map[string]any{
		"latitude":     9.0765,
		"longitude":    7.3986,
		"description":  "help me",
		"voiceNoteKey": key,
	})
	SendSOSAlert(ctx)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("want 201 sending the alert, got %d %s", recorder.Code, recorder.Body.String())
	}

	var alert models.SOSAlert
	if err := config.DB.Where("user_id = ?", caller.ID).Order("created_at desc").
		First(&alert).Error; err != nil {
		t.Fatalf("load alert: %v", err)
	}

	var link models.MediaLink
	if err := config.DB.Where("object_key = ? AND parent_type = ?", key, MediaParentSOS).
		First(&link).Error; err != nil {
		t.Fatalf("load media link: %v", err)
	}
	if link.ParentID != alert.ID {
		t.Errorf("the link must point at the new alert: got %s, want %s", link.ParentID, alert.ID)
	}
}

// TestSendSOSAlert_RejectsKeyOutsidePrefix pins the storage boundary on the
// attach path: an SOS alert may only carry sos_media/ keys, so an evidence
// object cannot be smuggled in as an attachment.
func TestSendSOSAlert_RejectsKeyOutsidePrefix(t *testing.T) {
	skipWithoutDB(t)
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	store := newMediaStore(t)

	caller := testutil.MakeUser(t, "citizen")
	// The bytes are real evidence, but evidence/ is not an SOS media prefix.
	key, _ := store.put("evidence", "clip.mp4", []byte("evidence bytes"))

	ctx, recorder := mediaContext(t, "POST", "/sos/send", nil, caller, map[string]any{
		"latitude":     9.0765,
		"longitude":    7.3986,
		"description":  "help me",
		"voiceNoteKey": key,
	})
	SendSOSAlert(ctx)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("want 400 for a key outside the sos_media prefix, got %d %s", recorder.Code, recorder.Body.String())
	}

	var alerts int64
	if err := config.DB.Model(&models.SOSAlert{}).Where("user_id = ?", caller.ID).Count(&alerts).Error; err != nil {
		t.Fatalf("count alerts: %v", err)
	}
	if alerts != 0 {
		t.Errorf("a refused attachment must not create an alert, got %d", alerts)
	}
	if got := mediaLinkCount(t, key, nil); got != 0 {
		t.Errorf("a refused key must not be linked, got %d link rows", got)
	}
}
