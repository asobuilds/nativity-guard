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
    "security-solution/models"
)

func sosUploadRequest(t *testing.T, filename, contentType string, content []byte) (*gin.Context, *httptest.ResponseRecorder) {
    t.Helper()
    var body bytes.Buffer
    writer := multipart.NewWriter(&body)
    header := make(map[string][]string)
    header["Content-Disposition"] = []string{`form-data; name="file"; filename="` + filename + `"`}
    header["Content-Type"] = []string{contentType}
    part, err := writer.CreatePart(header)
    if err != nil { t.Fatal(err) }
    if _, err := part.Write(content); err != nil { t.Fatal(err) }
    if err := writer.Close(); err != nil { t.Fatal(err) }
    recorder := httptest.NewRecorder()
    ctx, _ := gin.CreateTestContext(recorder)
    ctx.Request = httptest.NewRequest("POST", "/api/v1/sos/media/upload", &body)
    ctx.Request.Header.Set("Content-Type", writer.FormDataContentType())
    ctx.Set("user", &models.User{MinorStatus: "adult"})
    return ctx, recorder
}

func TestUploadSOSMediaFileLocalFallback(t *testing.T) {
    store := newMediaStore(t)
    ctx, rec := sosUploadRequest(t, "note.m4a", "audio/mp4", []byte("short audio for test"))
    UploadSOSMediaFile(ctx)
    if rec.Code != http.StatusCreated { t.Fatalf("want 201, got %d %s", rec.Code, rec.Body.String()) }
    var response struct { Key string `json:"key"` }
    if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil { t.Fatal(err) }
    if !strings.HasPrefix(response.Key, "sos_media/") || !store.exists(response.Key) {
        t.Errorf("media should be stored privately, got %q", response.Key)
    }
    if err := verifyMediaObject(response.Key, sosMediaPrefix, voiceMediaMaxBytes); err != nil {
        t.Errorf("uploaded key must be verifiable for SOS attach: %v", err)
    }
}

func TestUploadSOSMediaFileRejectsInvalidFormats(t *testing.T) {
    newMediaStore(t)
    for _, tt := range []struct{ filename, mime string }{
        {"malware.html", "text/html"},
        {"mismatch.jpg", "audio/mp4"},
        {"clip.mp4", "application/octet-stream"},
    } {
        t.Run(tt.filename, func(t *testing.T) {
            ctx, rec := sosUploadRequest(t, tt.filename, tt.mime, []byte("test"))
            UploadSOSMediaFile(ctx)
            if rec.Code != http.StatusBadRequest { t.Fatalf("got %d %s", rec.Code, rec.Body.String()) }
        })
    }
}

func TestMediaKindAcceptsSafariVoiceType(t *testing.T) {
    if kind, ok := mediaKind("audio/mp4"); !ok || kind != "voice" {
        t.Fatalf("Safari audio/mp4 must be classified as voice: %q %v", kind, ok)
    }
}
