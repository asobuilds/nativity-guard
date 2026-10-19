package handlers

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"

	"security-solution/config"
	"security-solution/internal/testutil"
	"security-solution/models"
)

// TestServeFile_ReportMedia_RequiresPostVisibilityOrModerator pins the serve
// authorization for report attachments: the media is readable by anyone who
// can see the reported post, and by a moderator who can action the report.
func TestServeFile_ReportMedia_RequiresPostVisibilityOrModerator(t *testing.T) {
	skipWithoutDB(t)
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	store := newMediaStore(t)

	moderator := testutil.MakeUser(t, "unit_admin")
	unit := testutil.MakeUnit(t, moderator)
	if err := config.DB.Model(&models.SecurityUnit{}).Where("id = ?", unit.ID).
		Updates(map[string]any{"state": "FCT", "lga": "Garki", "ward": "Garki"}).Error; err != nil {
		t.Fatalf("place unit: %v", err)
	}
	mediaAssignUserToUnit(t, moderator, unit)

	author := testutil.MakeUser(t, "citizen")
	post := &models.ForumPost{
		UnitID:   &unit.ID,
		Title:    "Reported post",
		Content:  "content",
		AuthorID: author.ID,
		Status:   "published",
		GeoPlace: models.GeoPlace{Scope: ScopeLGA, State: "FCT", LGA: "Garki"},
	}
	if err := config.DB.Create(post).Error; err != nil {
		t.Fatalf("create post: %v", err)
	}

	// The reporter is in the post's unit, so the post is open to them.
	reporter := testutil.MakeUser(t, "citizen")
	testutil.MakeMembership(t, reporter, unit, models.UnitRoleOfficer, false)
	mediaAssignUserToUnit(t, reporter, unit)

	// A citizen with no unit and no place of their own.
	stranger := testutil.MakeUser(t, "citizen")

	_, hash := store.put("report_media", "clip.mp4", []byte("clip bytes"))

	report := models.Report{
		ReportedBy: reporter.ID,
		TargetType: reportTargetForumPost,
		TargetID:   post.ID.String(),
		Reason:     "spam",
		Status:     reportStatusPending,
	}
	if err := config.DB.Create(&report).Error; err != nil {
		t.Fatalf("create report: %v", err)
	}

	serve := func(t *testing.T, user *models.User, query string) (int, string) {
		t.Helper()
		ctx, recorder := mediaContext(t, "GET", "/files/report_media/"+hash+query,
			gin.Params{{Key: "category", Value: "report_media"}, {Key: "hash", Value: hash}}, user, nil)
		ServeFile(ctx)
		return recorder.Code, recorder.Body.String()
	}

	// The reporter can see the post, so the reporter can see the media.
	code, body := serve(t, reporter, "?reportId="+report.ID.String())
	if code != http.StatusOK {
		t.Fatalf("the reporter must be able to read the media, got %d %s", code, body)
	}
	if body != "clip bytes" {
		t.Errorf("want the stored bytes, got %q", body)
	}

	// A moderator of the post's unit can action the report, so the moderator
	// can see the media.
	code, body = serve(t, moderator, "?reportId="+report.ID.String())
	if code != http.StatusOK {
		t.Fatalf("a moderator must be able to read the media, got %d %s", code, body)
	}

	// A caller who can neither see the post nor moderate it cannot.
	code, body = serve(t, stranger, "?reportId="+report.ID.String())
	if code != http.StatusForbidden {
		t.Fatalf("want 403 for a caller who cannot see the post and cannot moderate it, got %d %s", code, body)
	}

	// The reportId is what carries the scope, so it is required.
	code, body = serve(t, reporter, "")
	if code != http.StatusBadRequest {
		t.Fatalf("want 400 without a reportId, got %d %s", code, body)
	}
}
