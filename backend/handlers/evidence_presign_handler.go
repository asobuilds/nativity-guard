package handlers

import (
	"fmt"
	"net/http"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"security-solution/config"
	"security-solution/models"
	"security-solution/services"
)

// evidenceMaxBytes mirrors the evidence cap enforced by
// middleware.UploadValidationMiddleware for the single-step upload path.
const evidenceMaxBytes int64 = 100 << 20 // 100 MB

// presignTTL is how long a presigned PUT stays valid.
const presignTTL = 15 * time.Minute

// hexSHA256 matches a lowercase or uppercase sha256 hex digest.
var hexSHA256 = regexp.MustCompile(`^[a-fA-F0-9]{64}$`)

// evidencePresignMIME is the allowlist for the presign path. The single-step
// endpoint accepts anything with a detectable type; the presign path is stricter
// because the client declares the type up front and we never see the bytes.
var evidencePresignMIME = map[string]bool{
	"image/jpeg":      true,
	"image/png":       true,
	"image/webp":      true,
	"image/gif":       true,
	"video/mp4":       true,
	"video/webm":      true,
	"video/quicktime": true,
	"audio/mpeg":      true,
	"audio/mp4":       true,
	"audio/wav":       true,
	"audio/ogg":       true,
	"audio/webm":      true,
	"application/pdf": true,
}

type presignInput struct {
	Filename    string `json:"filename"`
	ContentType string `json:"contentType"`
	SizeBytes   int64  `json:"sizeBytes"`
	SHA256      string `json:"sha256"`
}

type confirmInput struct {
	Key         string `json:"key"`
	Filename    string `json:"filename"`
	ContentType string `json:"contentType"`
	SizeBytes   int64  `json:"sizeBytes"`
	Description string `json:"description"`
	Type        string `json:"type"`
}

// PresignEvidenceUpload issues a short-lived R2 PUT URL so large evidence files
// travel from the client straight to the bucket.
//
//	POST /evidence/case/:caseId/presign
//	Body: { filename, contentType, sizeBytes, sha256 }
//	Resp: { uploadUrl, key, expiresAt }
func PresignEvidenceUpload(c *gin.Context) {
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
	if _, ok := userInterface.(*models.User); !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user"})
		return
	}

	var input presignInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	filename := strings.TrimSpace(input.Filename)
	if filename == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "filename is required"})
		return
	}

	contentType := strings.TrimSpace(input.ContentType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	if !evidencePresignMIME[contentType] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "content type not allowed for evidence"})
		return
	}

	if input.SizeBytes <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sizeBytes must be greater than zero"})
		return
	}
	if input.SizeBytes > evidenceMaxBytes {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file exceeds maximum allowed size"})
		return
	}

	sha := strings.ToLower(strings.TrimSpace(input.SHA256))
	if !hexSHA256.MatchString(sha) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sha256 must be a 64 character hex digest"})
		return
	}

	ext := strings.ToLower(path.Ext(filename))
	if ext == "" {
		ext = ".bin"
	}
	if ext == "." {
		ext = ".bin"
	}

	storageSvc := services.NewFileStorageService()
	r2 := storageSvc.R2()
	if r2 == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "direct upload is not configured"})
		return
	}

	now := time.Now().UTC()
	key := path.Join("evidence", now.Format("2006"), now.Format("01"), sha+ext)

	expiresAt := time.Now().UTC().Add(presignTTL)
	uploadURL, err := r2.PresignPut(c.Request.Context(), key, contentType, input.SizeBytes, map[string]string{
		"sha256":            sha,
		"original-filename": filename,
		"case-id":           caseID.String(),
	}, presignTTL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to presign upload"})
		return
	}

	// evidence/ is a private prefix: the key is returned, never a public URL.
	c.JSON(http.StatusOK, gin.H{
		"uploadUrl": uploadURL,
		"key":       key,
		"expiresAt": expiresAt,
	})
}

// ConfirmEvidenceUpload records the Evidence row after the client has PUT the
// bytes to R2. It verifies the object exists with the declared size and
// content type before writing the row.
//
//	POST /evidence/case/:caseId/confirm
//	Body: { key, filename, contentType, sizeBytes, description, type }
//	Resp: { evidence }
func ConfirmEvidenceUpload(c *gin.Context) {
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

	var input confirmInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	key := strings.TrimSpace(input.Key)
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "key is required"})
		return
	}
	// Only objects under the evidence prefix for this case may be confirmed.
	if !services.IsPrivateStorageKey(key) || !strings.HasPrefix(key, "evidence/") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "key must be an evidence object key"})
		return
	}
	if strings.Contains(key, "..") || strings.Contains(key, "//") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid key"})
		return
	}

	storageSvc := services.NewFileStorageService()
	r2 := storageSvc.R2()
	if r2 == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "direct upload is not configured"})
		return
	}

	actualSize, actualType, err := r2.Head(c.Request.Context(), key)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "uploaded object not found"})
		return
	}

	if input.SizeBytes > 0 && actualSize != input.SizeBytes {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("size mismatch: declared %d, stored %d", input.SizeBytes, actualSize),
		})
		return
	}

	declaredType := strings.TrimSpace(input.ContentType)
	if declaredType != "" && actualType != "" && !mimeMatches(declaredType, actualType) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("content type mismatch: declared %s, stored %s", declaredType, actualType),
		})
		return
	}

	evidenceType := strings.TrimSpace(input.Type)
	if evidenceType == "" {
		evidenceType = "other"
	}
	contentType := actualType
	if contentType == "" {
		contentType = declaredType
	}

	// The sha256 is the key stem for the presign flow.
	fileHash := keyHashFromKey(key)

	evidence := models.Evidence{
		CaseID:      caseID,
		UploadedBy:  userObj.ID,
		Type:        evidenceType,
		Description: input.Description,
		// evidence/ is private: FileURL holds the object key, not a public URL.
		FileURL:    key,
		FilePath:   key,
		MimeType:   contentType,
		SizeBytes:  actualSize,
		FileHash:   fileHash,
		UploadedAt: time.Now().UTC(),
	}

	if err := config.DB.Create(&evidence).Error; err != nil {
		_ = storageSvc.Delete(key)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save evidence"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"evidence": evidence})
}

// keyHashFromKey extracts the sha256 hex stem from evidence/<yyyy>/<mm>/<hash>.<ext>.
func keyHashFromKey(key string) string {
	base := path.Base(key)
	if i := strings.LastIndex(base, "."); i > 0 {
		base = base[:i]
	}
	if hexSHA256.MatchString(base) {
		return strings.ToLower(base)
	}
	return ""
}

// mimeMatches compares two media types ignoring parameters such as "; charset".
func mimeMatches(a string, b string) bool {
	na := strings.ToLower(strings.TrimSpace(strings.Split(a, ";")[0]))
	nb := strings.ToLower(strings.TrimSpace(strings.Split(b, ";")[0]))
	return na == nb
}
