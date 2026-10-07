package handlers

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"security-solution/models"
)

func TestSOSLocationFreshness(t *testing.T) {
	now := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)
	good := models.UserLocation{Latitude: 0, Longitude: 0, Accuracy: 10, RecordedAt: now}
	if !freshSOSLocation(good, now) {
		t.Fatal("valid equator/meridian coordinates must be supported")
	}
	cases := []struct {
		name   string
		change func(*models.UserLocation)
	}{
		{"stale", func(l *models.UserLocation) { l.RecordedAt = now.Add(-2*time.Minute - time.Second) }},
		{"future", func(l *models.UserLocation) { l.RecordedAt = now.Add(11 * time.Second) }},
		{"no timestamp", func(l *models.UserLocation) { l.RecordedAt = time.Time{} }},
		{"latitude", func(l *models.UserLocation) { l.Latitude = 91 }},
		{"longitude", func(l *models.UserLocation) { l.Longitude = -181 }},
		{"accuracy", func(l *models.UserLocation) { l.Accuracy = -1 }},
		{"nan", func(l *models.UserLocation) { l.Latitude = math.NaN() }},
		{"infinity", func(l *models.UserLocation) { l.Accuracy = math.Inf(1) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			l := good
			tc.change(&l)
			if freshSOSLocation(l, now) {
				t.Fatal("invalid location must be hidden")
			}
		})
	}
}

func TestSOSCitizenResponseHidesOfficerIdentity(t *testing.T) {
	officer, actor, session, loc := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	r := models.SOSResponder{ID: uuid.New(), UnitID: uuid.New(), AcceptedBy: actor, AssignedUserID: &officer, TrackingSessionID: &session, LocationID: &loc}
	viewer := &models.User{ID: uuid.New(), Role: "citizen"}
	b, err := json.Marshal(responderView(r, viewer, false, false))
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{officer.String(), actor.String(), session.String(), loc.String(), "acceptedBy", "assignedUserId"} {
		if strings.Contains(string(b), secret) {
			t.Fatalf("citizen response leaked %s: %s", secret, b)
		}
	}
	admin, _ := json.Marshal(responderView(r, viewer, true, false))
	if !strings.Contains(string(admin), officer.String()) {
		t.Fatal("managing admin needs assignment identity")
	}
}
