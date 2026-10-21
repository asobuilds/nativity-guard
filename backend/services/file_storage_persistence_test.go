package services

import (
	"bytes"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"
)

func preparedMediaHeader(t *testing.T) *multipart.FileHeader {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "test-voice.m4a")
	if err != nil { t.Fatal(err) }
	if _, err := part.Write([]byte("dummy voice recording for local storage test")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil { t.Fatal(err) }
	request := httptest.NewRequest("POST", "/", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	if err := request.ParseMultipartForm(1024); err != nil { t.Fatal(err) }
	return request.MultipartForm.File["file"][0]
}

func TestRemoteStorageRequiredPreventsEphemeralEvidence(t *testing.T) {
	t.Setenv("MEDIA_ENDPOINT", "")
	t.Setenv("MEDIA_ACCESS_KEY_ID", "")
	t.Setenv("MEDIA_SECRET_ACCESS_KEY", "")
	t.Setenv("MEDIA_BUCKET", "")
	t.Setenv("MEDIA_REQUIRE_REMOTE", "true")
	t.Setenv("STORAGE_ROOT", t.TempDir())

	stored, err := NewFileStorageService().Save(preparedMediaHeader(t), "sos_media")
	if stored != nil || err == nil || !strings.Contains(err.Error(), "remote media storage") {
		t.Fatalf("expected a clear nonpersistent-storage failure, got %+v, %v", stored, err)
	}
}

func TestLocalStorageRemainsAvailableForDevelopment(t *testing.T) {
	t.Setenv("MEDIA_ENDPOINT", "")
	t.Setenv("MEDIA_ACCESS_KEY_ID", "")
	t.Setenv("MEDIA_SECRET_ACCESS_KEY", "")
	t.Setenv("MEDIA_BUCKET", "")
	t.Setenv("MEDIA_REQUIRE_REMOTE", "false")
	t.Setenv("STORAGE_ROOT", t.TempDir())

	stored, err := NewFileStorageService().Save(preparedMediaHeader(t), "sos_media")
	if err != nil || stored == nil { t.Fatalf("development save failed: %v", err) }
	if !NewFileStorageService().Exists(stored.RelativePath) {
		t.Fatal("development file should exist on disk")
	}
}
