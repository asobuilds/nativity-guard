package handlers

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"

	"security-solution/config"
	"security-solution/internal/testutil"
	"security-solution/models"
)

// TestServeFile_SOSMedia_RequiresSOSVisibility pins the serve authorization
// for SOS attachments: the object is only as readable as the alert it belongs
// to, and the check reuses the alert's own read scope.
func TestServeFile_SOSMedia_RequiresSOSVisibility(t *testing.T) {
	skipWithoutDB(t)
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	store := newMediaStore(t)

	owner := testutil.MakeUser(t, "citizen")
	stranger := testutil.MakeUser(t, "citizen")

	_, hash := store.put("sos_media", "clip.mp4", []byte("clip bytes"))

	alert := models.SOSAlert{
		UserID:        owner.ID,
		Status:        "pending",
		Priority:      "high",
		Severity:      "normal",
		DispatchState: "open",
		Latitude:      9.0765,
		Longitude:     7.3986,
		Description:   "help me",
	}
	if err := config.DB.Create(&alert).Error; err != nil {
		t.Fatalf("create alert: %v", err)
	}

	serve := func(t *testing.T, user *models.User, query string) (int, string) {
		t.Helper()
		ctx, recorder := mediaContext(t, "GET", "/files/sos_media/"+hash+query,
			gin.Params{{Key: "category", Value: "sos_media"}, {Key: "hash", Value: hash}}, user, nil)
		ServeFile(ctx)
		return recorder.Code, recorder.Body.String()
	}

	// The owner of the alert can read the attachment.
	code, body := serve(t, owner, "?sosId="+alert.ID.String())
	if code != http.StatusOK {
		t.Fatalf("the owner of the alert must be able to read the attachment, got %d %s", code, body)
	}
	if body != "clip bytes" {
		t.Errorf("want the stored bytes, got %q", body)
	}

	// A caller with no relation to the alert cannot.
	code, body = serve(t, stranger, "?sosId="+alert.ID.String())
	if code != http.StatusNotFound {
		t.Fatalf("want 404 for a caller outside the alert's scope, got %d %s", code, body)
	}

	// The sosId is what carries the scope, so it is required.
	code, body = serve(t, owner, "")
	if code != http.StatusBadRequest {
		t.Fatalf("want 400 without a sosId, got %d %s", code, body)
	}
}
