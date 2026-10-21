package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"security-solution/config"
	"security-solution/models"
	"security-solution/services"
)

// categoryFromMime returns the storage category for a given MIME type.
func categoryFromMime(mime string) string {
	switch {
	case mime == "image/jpeg" || mime == "image/png" || mime == "image/webp" || mime == "image/gif":
		return "image"
	case mime == "video/mp4" || mime == "video/webm" || mime == "video/quicktime":
		return "video"
	case mime == "audio/mpeg" || mime == "audio/mp4" || mime == "audio/wav" || mime == "audio/ogg" || mime == "audio/webm":
		return "audio"
	default:
		return "document"
	}
}

// UploadEvidenceFile accepts a multipart file and attaches it as evidence.
// Requires :caseId param. Access is gated by CanAccessCase middleware.
func UploadEvidenceFile(c *gin.Context) {
	if !requireMinorApproved(c) {
		return
	}
	caseID, err := uuid.Parse(c.Param("caseId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid case ID"})
		return
	}

	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj, ok := userInterface.(*models.User)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user"})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}

	evidenceType := c.PostForm("type")
	if evidenceType == "" {
		evidenceType = categoryFromMime(file.Header.Get("Content-Type"))
	}

	description := c.PostForm("description")

	mime := file.Header.Get("Content-Type")
	if mime == "" {
		mime = "application/octet-stream"
	}
	category := categoryFromMime(mime)

	storageSvc := services.NewFileStorageService()
	stored, err := storageSvc.Save(file, "evidence")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	evidence := models.Evidence{
		CaseID:      caseID,
		UploadedBy:  userObj.ID,
		Type:        evidenceType,
		Description: description,
		FileURL:     stored.RelativePath,
		FilePath:    stored.RelativePath,
		MimeType:    mime,
		SizeBytes:   stored.Size,
		FileHash:    stored.Hash,
	}

	if err := config.DB.Create(&evidence).Error; err != nil {
		_ = storageSvc.Delete(stored.RelativePath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save evidence"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":  "Evidence uploaded",
		"evidence": evidence,
		"category": category,
	})
}
