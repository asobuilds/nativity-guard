package handlers

import (
    "net/http"
    "path/filepath"
    "strings"

    "github.com/gin-gonic/gin"
    "security-solution/models"
    "security-solution/services"
)

// UploadSOSMediaFile is an authenticated, same-API-origin fallback when a
// browser cannot PUT to the storage provider's presigned URL (often CORS).
// It NEVER sends an SOS. Only SendSOSAlert can attach the returned private key.
func UploadSOSMediaFile(c *gin.Context) {
    if !requireMinorApproved(c) {
        return
    }
    if _, ok := c.Get("user"); !ok {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
        return
    }
    userValue, _ := c.Get("user")
    if _, ok := userValue.(*models.User); !ok {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
        return
    }

    // Bound multipart parsing. A short SOS clip may be up to 20 MiB.
    c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, videoMediaMaxBytes+(1<<20))
    file, err := c.FormFile("file")
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "a voice recording or video file is required"})
        return
    }
    contentType := strings.ToLower(strings.TrimSpace(strings.Split(file.Header.Get("Content-Type"), ";")[0]))
    kind, allowed := mediaKind(contentType)
    if !allowed {
        c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported audio or video format"})
        return
    }
    if file.Size <= 0 || file.Size > mediaSizeCap(kind) {
        c.JSON(http.StatusBadRequest, gin.H{"error": "recording exceeds the allowed size"})
        return
    }
    ext := strings.ToLower(filepath.Ext(file.Filename))
    if ext == "" || strings.ContainsAny(file.Filename, "/\\") {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid recording filename"})
        return
    }
    allowedExt := map[string][]string{
        "audio/mpeg": {".mp3"}, "audio/mp4": {".mp4", ".m4a"},
        "audio/wav": {".wav"}, "audio/ogg": {".ogg"},
        "audio/webm": {".webm"}, "video/mp4": {".mp4"},
        "video/webm": {".webm"}, "video/quicktime": {".mov"},
    }
    validExt := false
    for _, expected := range allowedExt[contentType] {
        if ext == expected { validExt = true; break }
    }
    if !validExt {
        c.JSON(http.StatusBadRequest, gin.H{"error": "filename does not match recording format"})
        return
    }

    stored, err := services.NewFileStorageService().Save(file, "sos_media")
    if err != nil {
        c.JSON(http.StatusServiceUnavailable, gin.H{"error": "recording storage is unavailable; try again or send SOS without attachment"})
        return
    }
    c.JSON(http.StatusCreated, gin.H{
        "key": stored.RelativePath,
        "sizeBytes": stored.Size,
        "contentType": contentType,
    })
}
