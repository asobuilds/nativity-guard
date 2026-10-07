//go:build integration

package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"security-solution/config"
	"security-solution/internal/testutil"
	"security-solution/models"
)

// Exercises the real router, JWT middleware, database queries and session
// lifecycle against TEST_DATABASE_URL. Uses test accounts only, never sends SOS.
func TestSOSTrackingLifecycle(t *testing.T) {
	testutil.TruncateAll(t)
	admin := testutil.MakeUser(t, "unit_admin")
	unit := testutil.MakeUnit(t, admin)
	admin.UnitID = &unit.ID
	if err := config.DB.Save(admin).Error; err != nil {
		t.Fatal(err)
	}
	reporter := testutil.MakeUser(t, "citizen")
	officer := testutil.MakeUser(t, "officer")
	peer := testutil.MakeUser(t, "officer")
	outsider := testutil.MakeUser(t, "citizen")
	otherAdmin := testutil.MakeUser(t, "unit_admin")
	otherUnit := testutil.MakeUnit(t, otherAdmin)
	otherAdmin.UnitID = &otherUnit.ID
	config.DB.Save(otherAdmin)
	testutil.MakeMembership(t, officer, unit, "officer", false)
	testutil.MakeMembership(t, peer, unit, "officer", false)
	officer.LocationSharingEnabled = true
	config.DB.Save(officer)
	sos := models.SOSAlert{UserID: reporter.ID, UnitID: &unit.ID, Latitude: 7.2, Longitude: 8.1, Status: "pending", Priority: "high", DispatchState: "open"}
	if err := config.DB.Create(&sos).Error; err != nil {
		t.Fatal(err)
	}
	router := FreshServer(t)
	base := "/api/v1/sos/" + sos.ID.String()
	responseBase := base + "/responders/" + unit.ID.String()
	request := func(user *models.User, method, path string, body any, want int) map[string]json.RawMessage {
		t.Helper()
		raw, _ := json.Marshal(body)
		req := httptest.NewRequest(method, path, bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		if user != nil {
			req.Header.Set("Authorization", testutil.AuthHeader(t, user))
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != want {
			t.Fatalf("%s %s: status %d want %d; %s", method, path, w.Code, want, w.Body.String())
		}
		var out map[string]json.RawMessage
		if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		return out
	}
	request(admin, "POST", base+"/accept", map[string]any{}, 200)
	t.Run("unrelated and unassigned users cannot read", func(t *testing.T) {
		request(nil, "GET", base+"/responders/locations", nil, http.StatusUnauthorized)
		request(outsider, "GET", base+"/responders/locations", nil, http.StatusNotFound)
		request(otherAdmin, "GET", base, nil, http.StatusNotFound)
		request(peer, "GET", base, nil, http.StatusNotFound)
	})
	t.Run("only unit administrators assign eligible officer accounts", func(t *testing.T) {
		request(otherAdmin, "PUT", responseBase+"/officer", map[string]any{"userId": officer.ID}, 403)
		request(peer, "PUT", responseBase+"/officer", map[string]any{"userId": officer.ID}, 403)
		request(admin, "PUT", responseBase+"/officer", map[string]any{"userId": outsider.ID}, 400)
		request(admin, "PUT", responseBase+"/officer", map[string]any{"userId": officer.ID}, 200)
	})
	t.Run("compatible detail envelope and citizen privacy", func(t *testing.T) {
		out := request(reporter, "GET", base, nil, 200)
		if out["sos"] == nil || out["alert"] == nil || out["responders"] == nil {
			t.Fatal("missing response contract")
		}
		data := string(out["responders"])
		for _, forbidden := range []string{officer.ID.String(), admin.ID.String(), "assignedUserId", "acceptedBy", "email", "phone"} {
			if strings.Contains(data, forbidden) {
				t.Fatalf("citizen identity leak: %s", data)
			}
		}
		request(officer, "GET", base, nil, 200)
		request(admin, "POST", responseBase+"/tracking", nil, 403)
	})
	var session string
	start := func() {
		out := request(officer, "POST", responseBase+"/tracking", nil, 200)
		if err := json.Unmarshal(out["sessionId"], &session); err != nil {
			t.Fatal(err)
		}
	}
	upload := func(want int) {
		request(officer, "PUT", responseBase+"/location", map[string]any{"sessionId": session, "latitude": 0, "longitude": 0, "accuracy": 12}, want)
	}
	locations := func(want int) {
		t.Helper()
		out := request(reporter, "GET", base+"/responders/locations", nil, 200)
		var rows []json.RawMessage
		if err := json.Unmarshal(out["responders"], &rows); err != nil {
			t.Fatal(err)
		}
		if len(rows) != want {
			t.Fatalf("locations = %d want %d; %s", len(rows), want, out["responders"])
		}
	}
	t.Run("only explicit session uploads are visible", func(t *testing.T) {
		// A general saved position never appears merely because an officer is assigned.
		config.DB.Create(&models.UserLocation{UserID: officer.ID, Latitude: 7.3, Longitude: 8.2, RecordedAt: time.Now()})
		locations(0)
		start()
		upload(200)
		locations(1)
		request(peer, "PUT", responseBase+"/location", map[string]any{"sessionId": session, "latitude": 7, "longitude": 8, "accuracy": 3}, 403)
		request(officer, "PUT", responseBase+"/location", map[string]any{"sessionId": session, "latitude": 91, "longitude": 8, "accuracy": 3}, 400)
	})
	t.Run("stale locations disappear", func(t *testing.T) {
		config.DB.Model(&models.UserLocation{}).Where("user_id = ?", officer.ID).Update("recorded_at", time.Now().Add(-3*time.Minute))
		locations(0)
		upload(200)
		locations(1)
	})
	t.Run("stopping invalidates delayed uploads", func(t *testing.T) {
		request(officer, "DELETE", responseBase+"/tracking?sessionId="+session, nil, 200)
		locations(0)
		upload(409)
	})
	t.Run("an old tab cannot stop a newer session", func(t *testing.T) {
		start()
		old := session
		start()
		upload(200)
		request(officer, "DELETE", responseBase+"/tracking?sessionId="+old, nil, 200)
		locations(1)
	})
	t.Run("disabling sharing revokes sessions", func(t *testing.T) {
		request(officer, "PUT", "/api/v1/location/sharing", map[string]any{"enabled": false}, 200)
		locations(0)
		upload(403)
		request(officer, "PUT", "/api/v1/location/sharing", map[string]any{"enabled": true}, 200)
		locations(0)
		upload(409)
		start()
		upload(200)
	})
	t.Run("reassignment hides old locations", func(t *testing.T) {
		request(admin, "PUT", responseBase+"/officer", map[string]any{"userId": peer.ID}, 200)
		locations(0)
		upload(403)
		request(admin, "PUT", responseBase+"/officer", map[string]any{"userId": officer.ID}, 200)
		start()
		upload(200)
	})
	t.Run("revoked membership cannot view or publish", func(t *testing.T) {
		config.DB.Model(&models.UnitMembership{}).Where("unit_id = ? AND user_id = ?", unit.ID, officer.ID).Update("status", models.MembershipRevoked)
		locations(0)
		upload(403)
		request(officer, "GET", base, nil, 404)
		config.DB.Model(&models.UnitMembership{}).Where("unit_id = ? AND user_id = ?", unit.ID, officer.ID).Update("status", models.MembershipActive)
	})
	t.Run("closing an SOS stops tracking", func(t *testing.T) {
		request(otherAdmin, "PUT", base+"/status", map[string]any{"status": "resolved"}, 404)
		request(admin, "PUT", base+"/status", map[string]any{"status": "anything"}, 400)
		request(admin, "PUT", base+"/status", map[string]any{"status": "resolved"}, 200)
		locations(0)
		upload(409)
		request(admin, "PUT", base+"/status", map[string]any{"status": "dispatched"}, 409)
		request(admin, "POST", base+"/release", map[string]any{}, 409)
	})
	t.Run("invalid ids are rejected", func(t *testing.T) { request(reporter, "GET", "/api/v1/sos/not-a-uuid/responders/locations", nil, 400) })
	t.Logf("Verified lifecycle for test SOS %s", sos.ID)
}
