package handlers

import (
	"strings"
	"testing"

	"security-solution/services"
)

func TestFileStorage_SOSMediaAndReportMediaCategoriesAllowed(t *testing.T) {
	store := newMediaStore(t)
	svc := services.NewFileStorageService()

	// Both media categories must be registered in the storage service, or the
	// upload the client makes against the presigned URL lands nowhere.
	for _, category := range []string{"sos_media", "report_media"} {
		t.Run(category, func(t *testing.T) {
			header := mediaMultipartFile(t, "clip.mp4", "video/mp4", []byte("media bytes"))

			stored, err := svc.Save(header, category)
			if err != nil {
				t.Fatalf("Save(%s): %v", category, err)
			}
			if !strings.HasPrefix(stored.RelativePath, category+"/") {
				t.Errorf("stored key %q must live under %s/", stored.RelativePath, category)
			}
			if !store.exists(stored.RelativePath) {
				t.Errorf("stored key %q is not on disk", stored.RelativePath)
			}
		})
	}

	// The two categories are private: a key under them must never be handed
	// out as a public CDN URL (services/r2_service.go).
	for _, category := range []string{"sos_media", "report_media"} {
		key := category + "/2026/10/" + strings.Repeat("a", 64) + ".mp4"
		if !services.IsPrivateStorageKey(key) {
			t.Errorf("%s must be a private storage key", key)
		}
		if services.IsPublicStorageKey(key) {
			t.Errorf("%s must never be a public storage key", key)
		}
	}

	// The category list is an allowlist, not a catch-all: an unregistered
	// category is still refused.
	header := mediaMultipartFile(t, "clip.mp4", "video/mp4", []byte("media bytes"))
	if _, err := svc.Save(header, "not_a_media_category"); err == nil {
		t.Error("an unregistered storage category must be refused")
	}
}
