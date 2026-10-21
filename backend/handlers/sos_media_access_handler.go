package handlers

import (
    "io"
    "net/http"
    "path/filepath"
    "strings"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"

    "security-solution/config"
    "security-solution/models"
    "security-solution/services"
)

// ServeSOSAttachment permits a reporter or a properly scoped responder to read
// ONLY an attachment that is linked to the SOS they are allowed to view.
func ServeSOSAttachment(c *gin.Context) {
    u, ok := sosCaller(c)
    if !ok { return }
    sos, ok := sosRead(c, u)
    if !ok { return }
    linkID, err := uuid.Parse(c.Param("mediaId"))
    if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "invalid media ID"}); return }

    var link models.MediaLink
    if err := config.DB.Where("id = ? AND parent_type = ? AND parent_id = ?",
        linkID, MediaParentSOS, sos.ID).First(&link).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "SOS attachment not found"}); return
    }
    key := link.ObjectKey
    if !strings.HasPrefix(key, sosMediaPrefix) || !services.IsPrivateStorageKey(key) ||
       strings.Contains(key, "..") || strings.Contains(key, "//") {
        c.JSON(http.StatusNotFound, gin.H{"error": "SOS attachment not found"}); return
    }
    storage := services.NewFileStorageService()
    reader, err := storage.Open(key)
    if err != nil { c.JSON(http.StatusNotFound, gin.H{"error": "SOS attachment unavailable"}); return }
    defer reader.Close()

    _, mime, _ := storage.Head(key)
    if mime == "" {
        // Local test/development storage does not record MIME metadata.
        switch strings.ToLower(filepath.Ext(key)) {
        case ".mp3": mime = "audio/mpeg"
        case ".m4a": mime = "audio/mp4"
        case ".wav": mime = "audio/wav"
        case ".ogg": mime = "audio/ogg"
        case ".mp4": mime = "video/mp4"
        case ".mov": mime = "video/quicktime"
        case ".webm": mime = "video/webm"
        default: mime = "application/octet-stream"
        }
    }
    c.Header("Content-Type", mime)
    c.Header("Content-Disposition", "inline")
    c.Header("Cache-Control", "private, no-store")
    c.Header("X-Content-Type-Options", "nosniff")
    c.Status(http.StatusOK)
    _, _ = io.Copy(c.Writer, reader)
}
