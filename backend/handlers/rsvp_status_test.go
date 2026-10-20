//go:build integration

package handlers

import (
	"testing"
	"time"

	"security-solution/config"
	"security-solution/internal/testutil"
	"security-solution/models"
)

func TestRSVPRejectsCompletedOrNonUpcomingEvent(t *testing.T) {
	if config.DB == nil {
		t.Skip("integration DB unavailable")
	}
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	author := testutil.MakeUser(t, "unit_admin")
	user := testutil.MakeUser(t, "citizen")

	// Create a completed event
	event := &models.CommunityEvent{
		Title:       "Past Meeting",
		Description: "Already finished",
		Location:    "HQ",
		EventDate:   time.Now().UTC().Add(-48 * time.Hour),
		Status:      "completed",
		AuthorID:    author.ID,
		GeoPlace:    models.GeoPlace{Scope: "world"},
	}
	if err := config.DB.Create(event).Error; err != nil {
		t.Fatalf("create event: %v", err)
	}

	ctx, rec := rsvpContext(event.ID, user)
	RSVPToEvent(ctx)

	if rec.Code != 409 {
		t.Fatalf("expected 409 Conflict for completed event RSVP, got %d %s", rec.Code, rec.Body.String())
	}
}

func TestRSVPRejectsOutOfPlaceEvent(t *testing.T) {
	if config.DB == nil {
		t.Skip("integration DB unavailable")
	}
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	author := testutil.MakeUser(t, "unit_admin")
	user := testutil.MakeUser(t, "citizen")

	// Create a verified unit in Lagos for user, so user's place is Lagos / Ikeja
	lagosUnit := &models.SecurityUnit{
		Name:       "Lagos Unit",
		Type:       "community",
		Status:     "active",
		IsVerified: true,
		State:      "Lagos",
		LGA:        "Ikeja",
	}
	if err := config.DB.Create(lagosUnit).Error; err != nil {
		t.Fatalf("create unit: %v", err)
	}
	testutil.MakeMembership(t, user, lagosUnit, "member", false)

	// Event scoped to Kano State (outside user's area)
	event := &models.CommunityEvent{
		Title:       "Kano Briefing",
		Description: "Northern security meeting",
		Location:    "Kano HQ",
		EventDate:   time.Now().UTC().Add(48 * time.Hour),
		Status:      "upcoming",
		GeoPlace:    models.GeoPlace{Scope: "state", State: "Kano"},
		AuthorID:    author.ID,
	}
	if err := config.DB.Create(event).Error; err != nil {
		t.Fatalf("create event: %v", err)
	}

	ctx, rec := rsvpContext(event.ID, user)
	RSVPToEvent(ctx)

	if rec.Code != 403 {
		t.Fatalf("expected 403 Forbidden for out-of-place event RSVP, got %d %s", rec.Code, rec.Body.String())
	}
}
