package handlers

import "github.com/gin-gonic/gin"

// PresignSOSMedia issues a short-lived PUT URL for a voice note or short video
// that will be attached to an SOS alert.
//
//	POST /sos/media/presign
//	Body: { filename, contentType, sizeBytes, sha256 }
//	Resp: { uploadUrl, key, expiresAt }
//
// The alert does not exist yet — POST /sos/send is a single JSON call — so this
// endpoint is caller-scoped, exactly like PresignEvidenceUpload. The key it
// returns is only accepted by SendSOSAlert when it is a private sos_media/ key.
func PresignSOSMedia(c *gin.Context) {
	presignMedia(c, sosMediaPrefix)
}
