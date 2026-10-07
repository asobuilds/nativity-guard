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
	"security-solution/models"
)

func communityWriteContext(body map[string]any, user *models.User) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	payload, _ := json.Marshal(body)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest("POST", "/", bytes.NewReader(payload))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set("user", user)
	return ctx, recorder
}

func TestCreateCommunityAnnouncementRejectsCitizen(t *testing.T) {
	user := &models.User{ID: uuid.New(), Role: "citizen"}
	ctx, recorder := communityWriteContext(map[string]any{"title": "Safety", "content": "Update"}, user)
	CreateCommunityAnnouncement(ctx)
	if recorder.Code != 403 {
		t.Fatalf("want 403 for citizen announcement, got %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestCreateCommunityEventRejectsCitizen(t *testing.T) {
	user := &models.User{ID: uuid.New(), Role: "citizen"}
	ctx, recorder := communityWriteContext(map[string]any{
		"title": "Meeting", "description": "Safety briefing", "location": "HQ",
		"eventDate": "2030-01-01T10:00:00Z", "endDate": "2030-01-01T11:00:00Z", "type": "meeting",
	}, user)
	CreateCommunityEvent(ctx)
	if recorder.Code != 403 {
		t.Fatalf("want 403 for citizen event creation, got %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestUnitAdminCannotPublishAnnouncementToAnotherUnit(t *testing.T) {
	own := uuid.New()
	other := uuid.New()
	user := &models.User{ID: uuid.New(), Role: "unit_admin", UnitID: &own}
	ctx, recorder := communityWriteContext(map[string]any{
		"unitId": other.String(), "title": "Safety", "content": "Update",
	}, user)
	CreateCommunityAnnouncement(ctx)
	if recorder.Code != 403 {
		t.Fatalf("want 403 for cross-unit announcement, got %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestUnitAdminCannotCreateEventForAnotherUnit(t *testing.T) {
	own := uuid.New()
	other := uuid.New()
	user := &models.User{ID: uuid.New(), Role: "unit_admin", UnitID: &own}
	ctx, recorder := communityWriteContext(map[string]any{
		"unitId": other.String(), "title": "Meeting", "description": "Safety briefing", "location": "HQ",
		"eventDate": "2030-01-01T10:00:00Z", "endDate": "2030-01-01T11:00:00Z", "type": "meeting",
	}, user)
	CreateCommunityEvent(ctx)
	if recorder.Code != 403 {
		t.Fatalf("want 403 for cross-unit event, got %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestCreateForumPostRejectsCrossUnitCitizen(t *testing.T) {
	if config.DB == nil {
		t.Skip("integration DB unavailable")
	}
	own := uuid.New()
	other := uuid.New()
	user := &models.User{ID: uuid.New(), Role: "citizen", UnitID: &own}
	ctx, recorder := communityWriteContext(map[string]any{
		"unitId": other.String(), "title": "Safety", "content": "Update",
	}, user)
	CreateForumPost(ctx)
	if recorder.Code != 403 {
		t.Fatalf("want 403 for cross-unit forum post, got %d %s", recorder.Code, recorder.Body.String())
	}
}
