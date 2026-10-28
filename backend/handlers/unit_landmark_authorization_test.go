//go:build integration

package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"security-solution/internal/testutil"
	"security-solution/models"
)

type unitLandmarksResponse struct {
	Landmarks []models.UnitLandmark `json:"landmarks"`
}

func makeUnitLandmark(t *testing.T, unit *models.SecurityUnit, name string) *models.UnitLandmark {
	t.Helper()

	landmark := &models.UnitLandmark{
		UnitID:    unit.ID,
		Name:      name,
		Type:      "Safe House",
		Latitude:  6.5244,
		Longitude: 3.3792,
	}
	if err := testutil.GetDB().Create(landmark).Error; err != nil {
		t.Fatalf("create landmark: %v", err)
	}
	return landmark
}

func getUnitLandmarks(t *testing.T, server http.Handler, unitID uuid.UUID, token string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/units/"+unitID.String()+"/landmarks", nil)
	if token != "" {
		req.Header.Set("Authorization", token)
	}
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)
	return w
}

func requireUnitLandmarksStatus(t *testing.T, w *httptest.ResponseRecorder, want int) {
	t.Helper()

	if w.Code != want {
		t.Fatalf("status = %d, want %d. body=%s", w.Code, want, w.Body.String())
	}
}

func requireUnitLandmarksForbidden(t *testing.T, w *httptest.ResponseRecorder) {
	t.Helper()

	requireUnitLandmarksStatus(t, w, http.StatusForbidden)
	if bytes.Contains(w.Body.Bytes(), []byte(`"landmarks"`)) {
		t.Fatalf("forbidden response leaked landmarks: %s", w.Body.String())
	}
}

func decodeUnitLandmarks(t *testing.T, w *httptest.ResponseRecorder) []models.UnitLandmark {
	t.Helper()

	requireUnitLandmarksStatus(t, w, http.StatusOK)

	var response unitLandmarksResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal landmarks: %v", err)
	}
	return response.Landmarks
}

func TestUnitLandmarkAuthorization(t *testing.T) {
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	server := FreshServer(t)

	t.Run("requires authentication", func(t *testing.T) {
		unit := testutil.MakeUnit(t, nil)
		makeUnitLandmark(t, unit, "Safe House A")

		w := getUnitLandmarks(t, server, unit.ID, "")
		requireUnitLandmarksStatus(t, w, http.StatusUnauthorized)
	})

	t.Run("rejects authenticated non-members", func(t *testing.T) {
		unit := testutil.MakeUnit(t, nil)
		makeUnitLandmark(t, unit, "Safe House A")
		caller := testutil.MakeUser(t, "citizen")

		w := getUnitLandmarks(t, server, unit.ID, testutil.AuthHeader(t, caller))
		requireUnitLandmarksForbidden(t, w)
	})

	t.Run("allows active members and returns only that unit's landmarks", func(t *testing.T) {
		unitA := testutil.MakeUnit(t, nil)
		unitB := testutil.MakeUnit(t, nil)
		landmarkA1 := makeUnitLandmark(t, unitA, "Safe House A1")
		landmarkA2 := makeUnitLandmark(t, unitA, "Checkpoint A2")
		makeUnitLandmark(t, unitB, "Safe House B1")

		caller := testutil.MakeUser(t, "citizen")
		testutil.MakeMembership(t, caller, unitA, models.UnitRoleOfficer, false)

		w := getUnitLandmarks(t, server, unitA.ID, testutil.AuthHeader(t, caller))
		landmarks := decodeUnitLandmarks(t, w)

		if len(landmarks) != 2 {
			t.Fatalf("landmark count = %d, want 2. body=%s", len(landmarks), w.Body.String())
		}
		seen := map[uuid.UUID]bool{}
		for _, landmark := range landmarks {
			if landmark.UnitID != unitA.ID {
				t.Fatalf("landmark %s belongs to unit %s, want %s", landmark.ID, landmark.UnitID, unitA.ID)
			}
			seen[landmark.ID] = true
		}
		if !seen[landmarkA1.ID] || !seen[landmarkA2.ID] {
			t.Fatalf("expected both Unit A landmarks, got %s", w.Body.String())
		}
	})

	t.Run("rejects members of another unit", func(t *testing.T) {
		unitA := testutil.MakeUnit(t, nil)
		unitB := testutil.MakeUnit(t, nil)
		makeUnitLandmark(t, unitB, "Safe House B")
		caller := testutil.MakeUser(t, "citizen")
		testutil.MakeMembership(t, caller, unitA, models.UnitRoleOfficer, false)

		w := getUnitLandmarks(t, server, unitB.ID, testutil.AuthHeader(t, caller))
		requireUnitLandmarksForbidden(t, w)
	})

	t.Run("rejects pending and inactive memberships", func(t *testing.T) {
		for _, status := range []string{models.MembershipPending, models.MembershipRevoked} {
			status := status
			t.Run(status, func(t *testing.T) {
				unit := testutil.MakeUnit(t, nil)
				makeUnitLandmark(t, unit, "Safe House A")
				caller := testutil.MakeUser(t, "citizen")
				membership := testutil.MakeMembership(t, caller, unit, models.UnitRoleOfficer, false)
				if err := testutil.GetDB().Model(membership).Update("status", status).Error; err != nil {
					t.Fatalf("update membership status: %v", err)
				}

				w := getUnitLandmarks(t, server, unit.ID, testutil.AuthHeader(t, caller))
				requireUnitLandmarksForbidden(t, w)
			})
		}
	})

	t.Run("allows super admins without membership", func(t *testing.T) {
		unit := testutil.MakeUnit(t, nil)
		landmark := makeUnitLandmark(t, unit, "Safe House A")
		caller := testutil.MakeUser(t, "super_admin")

		w := getUnitLandmarks(t, server, unit.ID, testutil.AuthHeader(t, caller))
		landmarks := decodeUnitLandmarks(t, w)

		if len(landmarks) != 1 || landmarks[0].ID != landmark.ID {
			t.Fatalf("unexpected landmarks: %s", w.Body.String())
		}
	})

	t.Run("rejects invalid unit IDs", func(t *testing.T) {
		caller := testutil.MakeUser(t, "super_admin")

		req := httptest.NewRequest(http.MethodGet, "/api/v1/units/not-a-uuid/landmarks", nil)
		req.Header.Set("Authorization", testutil.AuthHeader(t, caller))
		w := httptest.NewRecorder()
		server.ServeHTTP(w, req)

		requireUnitLandmarksStatus(t, w, http.StatusBadRequest)
	})
}
