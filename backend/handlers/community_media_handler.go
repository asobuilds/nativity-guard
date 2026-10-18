package handlers

import "github.com/gin-gonic/gin"

// PresignReportMedia issues a short-lived PUT URL for a voice note or short
// video that will be attached to a community content report.
//
//	POST /community/posts/:id/report/presign
//	Body: { filename, contentType, sizeBytes, sha256 }
//	Resp: { uploadUrl, key, expiresAt }
//
// It reuses communityViewablePost, so a caller can only presign media for a post
// they can already open — the same D-014 boundary the report itself uses. The
// key it returns is only accepted by ReportForumPost when it is a private
// report_media/ key.
func PresignReportMedia(c *gin.Context) {
	if _, _, ok := communityViewablePost(c); !ok {
		return
	}
	presignMedia(c, reportMediaPrefix)
}
