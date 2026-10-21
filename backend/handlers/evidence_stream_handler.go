package handlers

import (
    "io"
    "net/http"
    "strings"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"

    "security-solution/config"
    "security-solution/models"
    "security-solution/services"
)

// ServeCaseEvidence streams private case evidence only after a case-specific
// authentication and authorization check. This same-origin API path works
// when browser->object-storage GET is blocked by provider CORS.
func ServeCaseEvidence(c *gin.Context) {
    caseID, err := uuid.Parse(c.Param("caseId"))
    if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "invalid case ID"}); return }
    evidenceID, err := uuid.Parse(c.Param("evidenceId"))
    if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "invalid evidence ID"}); return }
    u, exists := c.Get("user")
    if !exists { c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"}); return }
    user, ok := u.(*models.User)
    if !ok { c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"}); return }

    var record models.Case
    if err := config.DB.First(&record, "id = ?", caseID).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "case not found"}); return
    }
    if !canAccessCase(user, &record) {
        c.JSON(http.StatusForbidden, gin.H{"error": "case evidence access denied"}); return
    }

    var evidence models.Evidence
    if err := config.DB.First(&evidence, "id = ? AND case_id = ?", evidenceID, caseID).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "evidence not found"}); return
    }
    key := evidence.FilePath
    if key == "" { key = evidence.FileURL }
    if !strings.HasPrefix(key, "evidence/") || !services.IsPrivateStorageKey(key) ||
       strings.Contains(key, "..") || strings.Contains(key, "//") {
        c.JSON(http.StatusBadRequest, gin.H{"error": "this evidence is a link, not a stored file"}); return
    }

    file, err := services.NewFileStorageService().Open(key)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "stored evidence is unavailable"}); return
    }
    defer file.Close()
    mime := strings.TrimSpace(evidence.MimeType)
    if mime == "" { mime = "application/octet-stream" }
    c.Header("Content-Type", mime)
    c.Header("Content-Disposition", "inline")
    c.Header("Cache-Control", "private, no-store")
    c.Header("X-Content-Type-Options", "nosniff")
    c.Status(http.StatusOK)
    _, _ = io.Copy(c.Writer, file)
}
