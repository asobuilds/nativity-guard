package handlers

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"security-solution/config"
	"security-solution/internal/testutil"
	"security-solution/models"
)

func TestUploadReportMediaFileLocalFallback(t *testing.T) {
	skipWithoutDB(t)
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	store := newMediaStore(t)

	author := testutil.MakeUser(t, "citizen")
	post := &models.ForumPost{
		Title:    "Test Post",
		Content:  "content",
		AuthorID: author.ID,
		Status:   "published",
		GeoPlace: models.GeoPlace{Scope: ScopeWorld},
	}
	if err := config.DB.Create(post).Error; err != nil {
		t.Fatalf("create post: %v", err)
	}

	caller := testutil.MakeUser(t, "citizen")

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	header := make(map[string][]string)
	header["Content-Disposition"] = []string{`form-data; name="file"; filename="note.m4a"`}
	header["Content-Type"] = []string{"audio/mp4"}
	part, err := writer.CreatePart(header)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte("short audio for test")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest("POST", "/api/v1/community/posts/"+post.ID.String()+"/report/upload", &body)
	ctx.Request.Header.Set("Content-Type", writer.FormDataContentType())
	ctx.Params = gin.Params{{Key: "id", Value: post.ID.String()}}
	ctx.Set("user", caller)

	UploadReportMediaFile(ctx)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("want 201, got %d %s", recorder.Code, recorder.Body.String())
	}

	var response struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(response.Key, "report_media/") || !store.exists(response.Key) {
		t.Errorf("media should be stored privately under report_media/, got %q", response.Key)
	}
	if err := verifyMediaObject(response.Key, reportMediaPrefix, videoMediaMaxBytes); err != nil {
		t.Errorf("uploaded key must be verifiable for report attach: %v", err)
	}
}
