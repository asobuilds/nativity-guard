//go:build integration

package handlers

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"security-solution/config"
	"security-solution/internal/testutil"
	"security-solution/models"
)

// RES-019 proof test. Runs only with `-tags integration` and a TEST_DATABASE_URL;
// it exercises the real report -> notification path against Postgres. The claim
// being pinned is the audit's finding 1: correctly filing a report must actually
// tell a moderator, and must tell only the moderators who can action it.

// reportContext builds the gin.Context routes.go hands to ReportForumPost for
// POST /community/posts/:id/report (post id in Params, reporter in context).
func reportContext(postID uuid.UUID, user *models.User) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	payload, _ := json.Marshal(map[string]any{"reason": "spam"})
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest("POST", "/community/posts/"+postID.String()+"/report", bytes.NewReader(payload))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Params = gin.Params{{Key: "id", Value: postID.String()}}
	ctx.Set("user", user)
	return ctx, recorder
}

// assignUserToUnit persists user.unit_id. The notification recipient set is
// derived from rows re-read from the database, so the field must be stored, not
// only set on the in-memory struct.
func assignUserToUnit(t *testing.T, user *models.User, unit *models.SecurityUnit) {
	t.Helper()
	user.UnitID = &unit.ID
	if err := config.DB.Model(&models.User{}).Where("id = ?", user.ID).
		Update("unit_id", unit.ID).Error; err != nil {
		t.Fatalf("assign unit: %v", err)
	}
}

func notificationCount(t *testing.T, userID uuid.UUID) int64 {
	t.Helper()
	var n int64
	if err := config.DB.Model(&models.Notification{}).
		Where("user_id = ?", userID).Count(&n).Error; err != nil {
		t.Fatalf("count notifications for %s: %v", userID, err)
	}
	return n
}

func TestReportForumPost_NotifiesModerator(t *testing.T) {
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	// A unit admin in the reported post's own unit can moderate it
	// (canModeratePost's own-unit branch, reused unchanged) — so they are a
	// recipient. Give the unit a place too, so resolveViewerPlace does not have
	// to fall through to the geocoder.
	moderator := testutil.MakeUser(t, "unit_admin")
	unit := testutil.MakeUnit(t, moderator)
	if err := config.DB.Model(&models.SecurityUnit{}).Where("id = ?", unit.ID).
		Updates(map[string]any{"state": "FCT", "lga": "Abuja Municipal", "ward": "Garki"}).Error; err != nil {
		t.Fatalf("place unit: %v", err)
	}
	assignUserToUnit(t, moderator, unit)

	// A super admin moderates the whole platform — always a recipient.
	superAdmin := testutil.MakeUser(t, "super_admin")

	// The reporter can see the post (same unit) and files the report.
	reporter := testutil.MakeUser(t, "citizen")
	assignUserToUnit(t, reporter, unit)

	// A citizen who can neither moderate nor is the reporter — must stay silent.
	bystander := testutil.MakeUser(t, "citizen")

	// The post's author is someone other than the reporter.
	author := testutil.MakeUser(t, "citizen")
	post := &models.ForumPost{
		UnitID:   &unit.ID,
		Title:    "Reported post",
		Content:  "content",
		AuthorID: author.ID,
		Status:   "published",
		GeoPlace: models.GeoPlace{Scope: ScopeWorld},
	}
	if err := config.DB.Create(post).Error; err != nil {
		t.Fatalf("create post: %v", err)
	}

	ctx, recorder := reportContext(post.ID, reporter)
	ReportForumPost(ctx)

	if recorder.Code != 201 {
		t.Fatalf("want 201 creating report, got %d %s", recorder.Code, recorder.Body.String())
	}

	var reports int64
	if err := config.DB.Model(&models.Report{}).
		Where("reported_by = ? AND target_id = ?", reporter.ID, post.ID.String()).
		Count(&reports).Error; err != nil {
		t.Fatalf("count reports: %v", err)
	}
	if reports != 1 {
		t.Fatalf("want exactly 1 report row, got %d", reports)
	}

	if got := notificationCount(t, moderator.ID); got != 1 {
		t.Errorf("a unit_admin who can moderate must be notified exactly once, got %d", got)
	}
	// The type must be the report notification, not the helper's old hard-coded
	// "sos_update" — the client routes on it.
	var moderatorNotification models.Notification
	if err := config.DB.Where("user_id = ?", moderator.ID).First(&moderatorNotification).Error; err != nil {
		t.Fatalf("load moderator notification: %v", err)
	}
	if moderatorNotification.Type != NotificationTypeReportCreated {
		t.Errorf("notification type: got %q, want %q", moderatorNotification.Type, NotificationTypeReportCreated)
	}
	if got := notificationCount(t, superAdmin.ID); got != 1 {
		t.Errorf("a super_admin must be notified exactly once, got %d", got)
	}
	if got := notificationCount(t, bystander.ID); got != 0 {
		t.Errorf("a citizen who cannot moderate must not be notified, got %d", got)
	}
	if got := notificationCount(t, reporter.ID); got != 0 {
		t.Errorf("the reporter must never be notified, got %d", got)
	}
}
