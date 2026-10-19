//go:build integration

package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"security-solution/config"
	"security-solution/internal/testutil"
	"security-solution/models"
)

// makeGeoUnit creates a verified unit in a named place. A membership in it
// resolves the caller's place without touching the network geocoder.
func makeGeoUnit(t *testing.T, label, state, lga string) *models.SecurityUnit {
	t.Helper()
	unit := &models.SecurityUnit{
		Name:               label + " " + uuid.NewString()[:8],
		Type:               "community",
		Status:             "active",
		IsVerified:         true,
		State:              state,
		LGA:                lga,
		RegistrationNumber: "REG-" + uuid.NewString()[:8],
	}
	if err := config.DB.Create(unit).Error; err != nil {
		t.Fatalf("create %s unit: %v", state, err)
	}
	return unit
}

// makeUnitAdmin creates a unit admin who belongs to the unit and therefore
// resolves the place the unit sits in.
func makeUnitAdmin(t *testing.T, unit *models.SecurityUnit) *models.User {
	t.Helper()
	user := testutil.MakeUser(t, "unit_admin")
	if err := config.DB.Model(user).Update("unit_id", unit.ID).Error; err != nil {
		t.Fatalf("set unit on admin: %v", err)
	}
	user.UnitID = &unit.ID
	testutil.MakeMembership(t, user, unit, models.UnitRoleOfficer, false)
	return user
}

// makeCitizen creates a citizen whose place comes from a verified unit
// membership. The citizen keeps no unit of their own unless one is set.
func makeCitizen(t *testing.T, unit *models.SecurityUnit) *models.User {
	t.Helper()
	user := testutil.MakeUser(t, "citizen")
	testutil.MakeMembership(t, user, unit, models.UnitRoleOfficer, false)
	return user
}

func announcementContext(user *models.User, payload map[string]any) (*gin.Context, *httptest.ResponseRecorder) {
	body, _ := json.Marshal(payload)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest("POST", "/community/announcements", bytes.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set("user", user)
	return ctx, recorder
}

func eventContext(user *models.User, payload map[string]any) (*gin.Context, *httptest.ResponseRecorder) {
	body, _ := json.Marshal(payload)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest("POST", "/community/events", bytes.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set("user", user)
	return ctx, recorder
}

func rsvpGeoContext(user *models.User, eventID uuid.UUID) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest("POST", "/community/events/"+eventID.String()+"/rsvp", nil)
	ctx.Params = gin.Params{{Key: "id", Value: eventID.String()}}
	ctx.Set("user", user)
	return ctx, recorder
}

func listContext(user *models.User, path string) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest("GET", path, nil)
	ctx.Set("user", user)
	return ctx, recorder
}

func eventWindow() (string, string) {
	start := time.Now().UTC().Add(24 * time.Hour)
	return start.Format(time.RFC3339), start.Add(2 * time.Hour).Format(time.RFC3339)
}

// TestCreateCommunityAnnouncement_PersistsServerDerivedPlace pins the write
// half of RES-018: an announcement carries a real, server-derived place. The
// request names another state and LGA — those names must be ignored, because
// the place is the author's, not the client's.
func TestCreateCommunityAnnouncement_PersistsServerDerivedPlace(t *testing.T) {
	gin.SetMode(gin.TestMode)
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	lagosUnit := makeGeoUnit(t, "Lagos Unit", "Lagos", "Ikeja")
	author := makeUnitAdmin(t, lagosUnit)

	ctx, recorder := announcementContext(author, map[string]any{
		"title":   "Water distribution",
		"content": "Water is distributed tomorrow at the field.",
		"scope":   ScopeLGA,
		"state":   "Benue",
		"lga":     "Makurdi",
	})
	CreateCommunityAnnouncement(ctx)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("want 201, got %d %s", recorder.Code, recorder.Body.String())
	}

	var announcement models.CommunityAnnouncement
	if err := config.DB.First(&announcement).Error; err != nil {
		t.Fatalf("reload announcement: %v", err)
	}
	if announcement.GeoPlace.Scope != ScopeLGA {
		t.Errorf("want scope %q, got %q", ScopeLGA, announcement.GeoPlace.Scope)
	}
	if announcement.GeoPlace.State != "Lagos" {
		t.Errorf("want the author's state Lagos, got %q", announcement.GeoPlace.State)
	}
	if announcement.GeoPlace.LGA != "Ikeja" {
		t.Errorf("want the author's LGA Ikeja, got %q", announcement.GeoPlace.LGA)
	}
	if announcement.GeoPlace.Country != "Nigeria" {
		t.Errorf("want country Nigeria, got %q", announcement.GeoPlace.Country)
	}
	if announcement.UnitID == nil || *announcement.UnitID != lagosUnit.ID {
		t.Errorf("want the announcement on the author's unit %s, got %v", lagosUnit.ID, announcement.UnitID)
	}
}

// TestCreateCommunityAnnouncement_RejectsScopeBeyondPlace pins the guard that
// makes the scope honest: an author with no resolvable place cannot publish to
// an LGA, and the rejected request writes no row at all.
func TestCreateCommunityAnnouncement_RejectsScopeBeyondPlace(t *testing.T) {
	gin.SetMode(gin.TestMode)
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	// A unit admin with no verified unit, no membership and no stored
	// location: the caller's place cannot be resolved.
	author := testutil.MakeUser(t, "unit_admin")

	ctx, recorder := announcementContext(author, map[string]any{
		"title":   "Nowhere notice",
		"content": "No place to address this to.",
		"scope":   ScopeLGA,
	})
	CreateCommunityAnnouncement(ctx)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("want 400 for a scope the place cannot express, got %d %s", recorder.Code, recorder.Body.String())
	}

	var count int64
	if err := config.DB.Model(&models.CommunityAnnouncement{}).Count(&count).Error; err != nil {
		t.Fatalf("count announcements: %v", err)
	}
	if count != 0 {
		t.Errorf("want no announcement rows after the rejection, got %d", count)
	}
}

// TestCreateCommunityEvent_PersistsServerDerivedPlace is the event twin of
// the announcement test: the event carries the author's place, and the
// client-supplied place names are ignored.
func TestCreateCommunityEvent_PersistsServerDerivedPlace(t *testing.T) {
	gin.SetMode(gin.TestMode)
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	lagosUnit := makeGeoUnit(t, "Lagos Unit", "Lagos", "Ikeja")
	author := makeUnitAdmin(t, lagosUnit)
	start, end := eventWindow()

	ctx, recorder := eventContext(author, map[string]any{
		"title":       "Town hall",
		"description": "A meeting for the community.",
		"location":    "Ikeja",
		"eventDate":   start,
		"endDate":     end,
		"type":        "meeting",
		"scope":       ScopeLGA,
		"state":       "Benue",
		"lga":         "Makurdi",
	})
	CreateCommunityEvent(ctx)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("want 201, got %d %s", recorder.Code, recorder.Body.String())
	}

	var event models.CommunityEvent
	if err := config.DB.First(&event).Error; err != nil {
		t.Fatalf("reload event: %v", err)
	}
	if event.GeoPlace.Scope != ScopeLGA {
		t.Errorf("want scope %q, got %q", ScopeLGA, event.GeoPlace.Scope)
	}
	if event.GeoPlace.State != "Lagos" {
		t.Errorf("want the author's state Lagos, got %q", event.GeoPlace.State)
	}
	if event.GeoPlace.LGA != "Ikeja" {
		t.Errorf("want the author's LGA Ikeja, got %q", event.GeoPlace.LGA)
	}
	if event.UnitID == nil || *event.UnitID != lagosUnit.ID {
		t.Errorf("want the event on the author's unit %s, got %v", lagosUnit.ID, event.UnitID)
	}
}

// TestCreateCommunityEvent_RejectsScopeBeyondPlace pins the same guard on the
// event path, and that a bad scope is refused rather than silently widened.
func TestCreateCommunityEvent_RejectsScopeBeyondPlace(t *testing.T) {
	gin.SetMode(gin.TestMode)
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	author := testutil.MakeUser(t, "unit_admin")
	start, end := eventWindow()

	ctx, recorder := eventContext(author, map[string]any{
		"title":       "Nowhere meeting",
		"description": "No place to address this to.",
		"location":    "Somewhere",
		"eventDate":   start,
		"endDate":     end,
		"type":        "meeting",
		"scope":       ScopeLGA,
	})
	CreateCommunityEvent(ctx)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("want 400 for a scope the place cannot express, got %d %s", recorder.Code, recorder.Body.String())
	}

	ctx, recorder = eventContext(author, map[string]any{
		"title":       "Galactic meeting",
		"description": "A scope that is not on the ladder.",
		"location":    "Somewhere",
		"eventDate":   start,
		"endDate":     end,
		"type":        "meeting",
		"scope":       "galaxy",
	})
	CreateCommunityEvent(ctx)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("want 400 for an unknown scope, got %d %s", recorder.Code, recorder.Body.String())
	}

	var count int64
	if err := config.DB.Model(&models.CommunityEvent{}).Count(&count).Error; err != nil {
		t.Fatalf("count events: %v", err)
	}
	if count != 0 {
		t.Errorf("want no event rows after the rejections, got %d", count)
	}
}

// TestGetCommunityAnnouncements_GeoScoped pins the read half: a citizen
// resolved to Lagos/Ikeja sees the Lagos announcement and never the Benue one.
func TestGetCommunityAnnouncements_GeoScoped(t *testing.T) {
	gin.SetMode(gin.TestMode)
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	lagosUnit := makeGeoUnit(t, "Lagos Unit", "Lagos", "Ikeja")
	benueUnit := makeGeoUnit(t, "Benue Unit", "Benue", "Makurdi")
	lagosAdmin := makeUnitAdmin(t, lagosUnit)
	benueAdmin := makeUnitAdmin(t, benueUnit)

	lagosAnnouncement := &models.CommunityAnnouncement{
		UnitID:   &lagosUnit.ID,
		GeoPlace: models.GeoPlace{Scope: ScopeLGA, State: "Lagos", LGA: "Ikeja", Country: "Nigeria"},
		Title:    "Lagos notice",
		Content:  "Lagos content",
		AuthorID: lagosAdmin.ID,
		Status:   "published",
	}
	if err := config.DB.Create(lagosAnnouncement).Error; err != nil {
		t.Fatalf("create lagos announcement: %v", err)
	}

	benueAnnouncement := &models.CommunityAnnouncement{
		UnitID:   &benueUnit.ID,
		GeoPlace: models.GeoPlace{Scope: ScopeLGA, State: "Benue", LGA: "Makurdi", Country: "Nigeria"},
		Title:    "Benue notice",
		Content:  "Benue content",
		AuthorID: benueAdmin.ID,
		Status:   "published",
	}
	if err := config.DB.Create(benueAnnouncement).Error; err != nil {
		t.Fatalf("create benue announcement: %v", err)
	}

	viewer := makeCitizen(t, lagosUnit)
	ctx, recorder := listContext(viewer, "/community/announcements")
	GetCommunityAnnouncements(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("want 200, got %d %s", recorder.Code, recorder.Body.String())
	}

	var resp struct {
		Announcements []struct {
			ID    uuid.UUID `json:"id"`
			Title string    `json:"title"`
		} `json:"announcements"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v %s", err, recorder.Body.String())
	}

	if len(resp.Announcements) != 1 {
		t.Fatalf("want 1 announcement for a Lagos viewer, got %d: %+v", len(resp.Announcements), resp.Announcements)
	}
	if resp.Announcements[0].ID != lagosAnnouncement.ID {
		t.Errorf("want the Lagos announcement %s, got %s", lagosAnnouncement.ID, resp.Announcements[0].ID)
	}
	for _, item := range resp.Announcements {
		if item.ID == benueAnnouncement.ID {
			t.Error("the Benue announcement must not be visible to a Lagos viewer")
		}
	}
}

// TestGetCommunityEvents_ReachesNeighbourUnit pins the LGA convergence case
// (D-015): an LGA-scoped event published by one unit must reach the members
// of a neighbouring unit in the same LGA. A unit filter ANDed with the
// geographic window would hide it from exactly the people it was addressed to.
func TestGetCommunityEvents_ReachesNeighbourUnit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	hostUnit := makeGeoUnit(t, "Ikeja Host Unit", "Lagos", "Ikeja")
	neighbourUnit := makeGeoUnit(t, "Ikeja Neighbour Unit", "Lagos", "Ikeja")
	benueUnit := makeGeoUnit(t, "Benue Unit", "Benue", "Makurdi")

	hostAdmin := makeUnitAdmin(t, hostUnit)
	benueAdmin := makeUnitAdmin(t, benueUnit)
	now := time.Now().UTC()

	lagosEvent := &models.CommunityEvent{
		UnitID:      &hostUnit.ID,
		GeoPlace:    models.GeoPlace{Scope: ScopeLGA, State: "Lagos", LGA: "Ikeja", Country: "Nigeria"},
		Title:       "Ikeja community meeting",
		Description: "A meeting in Ikeja",
		Location:    "Ikeja",
		EventDate:   now.Add(24 * time.Hour),
		Status:      "upcoming",
		AuthorID:    hostAdmin.ID,
	}
	if err := config.DB.Create(lagosEvent).Error; err != nil {
		t.Fatalf("create lagos event: %v", err)
	}

	benueEvent := &models.CommunityEvent{
		UnitID:      &benueUnit.ID,
		GeoPlace:    models.GeoPlace{Scope: ScopeLGA, State: "Benue", LGA: "Makurdi", Country: "Nigeria"},
		Title:       "Makurdi community meeting",
		Description: "A meeting in Makurdi",
		Location:    "Makurdi",
		EventDate:   now.Add(48 * time.Hour),
		Status:      "upcoming",
		AuthorID:    benueAdmin.ID,
	}
	if err := config.DB.Create(benueEvent).Error; err != nil {
		t.Fatalf("create benue event: %v", err)
	}

	// A member of the neighbouring unit in the same LGA.
	viewer := makeCitizen(t, neighbourUnit)
	if err := config.DB.Model(viewer).Update("unit_id", neighbourUnit.ID).Error; err != nil {
		t.Fatalf("set unit on viewer: %v", err)
	}
	viewer.UnitID = &neighbourUnit.ID

	ctx, recorder := listContext(viewer, "/community/events")
	GetCommunityEvents(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("want 200, got %d %s", recorder.Code, recorder.Body.String())
	}

	var resp struct {
		Events []struct {
			ID    uuid.UUID `json:"id"`
			Title string    `json:"title"`
		} `json:"events"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v %s", err, recorder.Body.String())
	}

	if len(resp.Events) != 1 {
		t.Fatalf("want 1 event for the Ikeja neighbour, got %d: %+v", len(resp.Events), resp.Events)
	}
	if resp.Events[0].ID != lagosEvent.ID {
		t.Errorf("want the Ikeja event %s, got %s", lagosEvent.ID, resp.Events[0].ID)
	}
	for _, item := range resp.Events {
		if item.ID == benueEvent.ID {
			t.Error("the Benue event must not be visible to an Ikeja viewer")
		}
	}
}

// TestRSVPToEvent_RejectsEventOutsidePlace pins D-014 for events: an event
// you cannot see, you cannot join. Without this gate the geo-scoped list is
// cosmetic — a citizen could still register for an event in another place.
func TestRSVPToEvent_RejectsEventOutsidePlace(t *testing.T) {
	gin.SetMode(gin.TestMode)
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	lagosUnit := makeGeoUnit(t, "Lagos Unit", "Lagos", "Ikeja")
	benueUnit := makeGeoUnit(t, "Benue Unit", "Benue", "Makurdi")
	lagosAdmin := makeUnitAdmin(t, lagosUnit)
	benueAdmin := makeUnitAdmin(t, benueUnit)
	now := time.Now().UTC()

	benueEvent := &models.CommunityEvent{
		UnitID:      &benueUnit.ID,
		GeoPlace:    models.GeoPlace{Scope: ScopeLGA, State: "Benue", LGA: "Makurdi", Country: "Nigeria"},
		Title:       "Makurdi community meeting",
		Description: "A meeting in Makurdi",
		Location:    "Makurdi",
		EventDate:   now.Add(24 * time.Hour),
		Status:      "upcoming",
		AuthorID:    benueAdmin.ID,
	}
	if err := config.DB.Create(benueEvent).Error; err != nil {
		t.Fatalf("create benue event: %v", err)
	}

	lagosEvent := &models.CommunityEvent{
		UnitID:      &lagosUnit.ID,
		GeoPlace:    models.GeoPlace{Scope: ScopeLGA, State: "Lagos", LGA: "Ikeja", Country: "Nigeria"},
		Title:       "Ikeja community meeting",
		Description: "A meeting in Ikeja",
		Location:    "Ikeja",
		EventDate:   now.Add(24 * time.Hour),
		Status:      "upcoming",
		AuthorID:    lagosAdmin.ID,
	}
	if err := config.DB.Create(lagosEvent).Error; err != nil {
		t.Fatalf("create lagos event: %v", err)
	}

	viewer := makeCitizen(t, lagosUnit)

	ctx, recorder := rsvpGeoContext(viewer, benueEvent.ID)
	RSVPToEvent(ctx)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("want 403 for an event outside the caller's place, got %d %s", recorder.Code, recorder.Body.String())
	}

	var attendees int64
	if err := config.DB.Model(&models.EventAttendee{}).Count(&attendees).Error; err != nil {
		t.Fatalf("count attendees: %v", err)
	}
	if attendees != 0 {
		t.Errorf("want no attendee rows after the rejection, got %d", attendees)
	}

	ctx, recorder = rsvpGeoContext(viewer, lagosEvent.ID)
	RSVPToEvent(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("want 200 for an event in the caller's place, got %d %s", recorder.Code, recorder.Body.String())
	}

	if err := config.DB.Model(&models.EventAttendee{}).Count(&attendees).Error; err != nil {
		t.Fatalf("count attendees: %v", err)
	}
	if attendees != 1 {
		t.Errorf("want 1 attendee row, got %d", attendees)
	}
}
