//go:build integration

package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"security-solution/config"
	"security-solution/internal/testutil"
	"security-solution/models"
)

// arriveAtCase posts the officer's GPS position to the case arrival endpoint.
func arriveAtCase(t *testing.T, token, caseID string, lat, lon float64) *httptest.ResponseRecorder {
	t.Helper()

	body, err := json.Marshal(map[string]any{
		"latitude":  lat,
		"longitude": lon,
	})
	if err != nil {
		t.Fatalf("marshal arrive payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/cases/"+caseID+"/arrive", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", token)
	w := httptest.NewRecorder()
	FreshServer(t).ServeHTTP(w, req)
	return w
}

// triggerPanic posts a distress event with the officer's current position.
func triggerPanic(t *testing.T, token string, lat, lon float64) *httptest.ResponseRecorder {
	t.Helper()

	body, err := json.Marshal(map[string]any{
		"latitude":  lat,
		"longitude": lon,
	})
	if err != nil {
		t.Fatalf("marshal panic payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/officer/panic", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", token)
	w := httptest.NewRecorder()
	FreshServer(t).ServeHTTP(w, req)
	return w
}

// setupArrivalCase builds a Lagos case in the "dispatched" state, assigned to an
// officer whose ID matches the caller's user ID so the arrival geofence check is
// the only thing left to exercise.
func setupArrivalCase(t *testing.T) (*models.User, *models.Case) {
	t.Helper()

	admin := testutil.MakeUser(t, "unit_admin")
	unit := testutil.MakeUnit(t, admin)

	officer := testutil.MakeUser(t, "officer")
	testutil.MakeMembership(t, officer, unit, models.UnitRoleOfficer, false)

	// The workflow authorizes the caller by their user ID, so the officer row
	// and the user row share the same primary key.
	officerRow := models.Officer{
		ID:            officer.ID,
		UnitID:        unit.ID,
		Name:          "Test Officer",
		Rank:          "Sergeant",
		Role:          "officer",
		UnitOfficerID: "WO-" + uuid.NewString()[:8],
		Status:        "active",
	}
	if err := config.DB.Create(&officerRow).Error; err != nil {
		t.Fatalf("create officer row: %v", err)
	}

	c := testutil.MakeCase(t, officer, unit)
	officerID := officer.ID
	c.AssignedTo = &officerID
	c.Status = "dispatched"
	// Lagos case coordinates.
	c.Latitude = 6.5244
	c.Longitude = 3.3792
	if err := config.DB.Save(c).Error; err != nil {
		t.Fatalf("save dispatched case: %v", err)
	}
	return officer, c
}

func TestArriveAtCase_RejectsWhenTooFar(t *testing.T) {
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	officer, c := setupArrivalCase(t)

	// ~10 km from the case location — well outside the 200 m geofence.
	w := arriveAtCase(t, testutil.AuthHeader(t, officer), c.ID.String(), 6.6, 3.5)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400. body=%s", w.Code, w.Body.String())
	}

	var reloaded models.Case
	if err := testutil.GetDB().First(&reloaded, "id = ?", c.ID).Error; err != nil {
		t.Fatalf("reload case: %v", err)
	}
	if reloaded.Status != "dispatched" {
		t.Fatalf("case status = %q, want dispatched (arrival must be rejected)", reloaded.Status)
	}
	if reloaded.ArrivedAt != nil {
		t.Fatalf("arrived_at = %v, want nil after a rejected arrival", reloaded.ArrivedAt)
	}
}

func TestArriveAtCase_AcceptsWhenClose(t *testing.T) {
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	officer, c := setupArrivalCase(t)

	// ~50 m from the case location — inside the 200 m geofence.
	w := arriveAtCase(t, testutil.AuthHeader(t, officer), c.ID.String(), 6.5248, 3.3795)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200. body=%s", w.Code, w.Body.String())
	}

	var reloaded models.Case
	if err := testutil.GetDB().First(&reloaded, "id = ?", c.ID).Error; err != nil {
		t.Fatalf("reload case: %v", err)
	}
	if reloaded.Status != "on_scene" {
		t.Fatalf("case status = %q, want on_scene", reloaded.Status)
	}
	if reloaded.ArrivedAt == nil {
		t.Fatalf("arrived_at is nil, want a timestamp recorded on arrival")
	}
}

func TestTriggerPanic_CreatesRecord(t *testing.T) {
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	officer, _ := setupArrivalCase(t)

	w := triggerPanic(t, testutil.AuthHeader(t, officer), 6.5244, 3.3792)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201. body=%s", w.Code, w.Body.String())
	}

	var panics []models.OfficerPanic
	if err := testutil.GetDB().
		Where("officer_id = ?", officer.ID).
		Find(&panics).Error; err != nil {
		t.Fatalf("query officer_panics: %v", err)
	}
	if len(panics) != 1 {
		t.Fatalf("officer_panics rows = %d, want 1", len(panics))
	}
	if panics[0].Latitude != 6.5244 || panics[0].Longitude != 3.3792 {
		t.Fatalf("panic location = (%v, %v), want (6.5244, 3.3792)",
			panics[0].Latitude, panics[0].Longitude)
	}
	if panics[0].Status != "active" {
		t.Fatalf("panic status = %q, want active", panics[0].Status)
	}
}
