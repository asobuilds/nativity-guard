package handlers

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestReportForumPost_AttachesVerifiedMedia(t *testing.T) {
	t.Skip("requires test database; run via CI integration job")
	key := "report_media/2026/10/abc123.mp4"
	_, err := reportForumPostWithMediaRequest(t, key, "video/mp4", 1000000)
	if err != nil {
		t.Fatal(err)
	}
}