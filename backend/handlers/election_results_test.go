//go:build integration

package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"security-solution/internal/testutil"
	"security-solution/models"
)

type electionResultsResponse struct {
	Election models.UnitAdminElection `json:"election"`
	Seats    []models.UnitAdminSeat   `json:"seats"`
}

func makeResultsElection(t *testing.T, unit *models.SecurityUnit, status string, resultFinalizedAt *time.Time) *models.UnitAdminElection {
	t.Helper()

	now := time.Now().UTC()
	election := &models.UnitAdminElection{
		UnitID:                unit.ID,
		ElectionType:          "admin",
		RotationGroup:         "A",
		SeatCount:             1,
		MemberCountAtElection: 1,
		EligibleVoterCount:    1,
		QuorumCount:           1,
		TermStart:             now,
		TermEnd:               now.AddDate(1, 0, 0),
		ResultFinalizedAt:     resultFinalizedAt,
		Status:                status,
		CreatedBy:             uuid.New(),
	}
	if err := testutil.GetDB().Create(election).Error; err != nil {
		t.Fatalf("create election: %v", err)
	}

	seat := &models.UnitAdminSeat{
		UnitID:        unit.ID,
		ElectionID:    election.ID,
		SeatNumber:    1,
		RotationGroup: "A",
		TermStart:     now,
		TermEnd:       now.AddDate(1, 0, 0),
		Status:        "active",
		ElectedAt:     now,
	}
	if err := testutil.GetDB().Create(seat).Error; err != nil {
		t.Fatalf("create seat: %v", err)
	}

	return election
}

func getElectionResults(t *testing.T, server http.Handler, electionID uuid.UUID, token string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/elections/"+electionID.String()+"/results", nil)
	if token != "" {
		req.Header.Set("Authorization", token)
	}
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)
	return w
}

func requireElectionResultsStatus(t *testing.T, w *httptest.ResponseRecorder, want int) {
	t.Helper()

	if w.Code != want {
		t.Fatalf("status = %d, want %d. body=%s", w.Code, want, w.Body.String())
	}
}

func requireElectionResultsForbidden(t *testing.T, w *httptest.ResponseRecorder) {
	t.Helper()

	requireElectionResultsStatus(t, w, http.StatusForbidden)
	if bytes.Contains(w.Body.Bytes(), []byte(`"seats"`)) {
		t.Fatalf("forbidden response leaked seats: %s", w.Body.String())
	}
}

func requireElectionResultsAllowed(t *testing.T, w *httptest.ResponseRecorder, electionID uuid.UUID) {
	t.Helper()

	requireElectionResultsStatus(t, w, http.StatusOK)

	var response electionResultsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal results: %v", err)
	}
	if response.Election.ID != electionID {
		t.Fatalf("election ID = %s, want %s", response.Election.ID, electionID)
	}
	if len(response.Seats) != 1 {
		t.Fatalf("seat count = %d, want 1", len(response.Seats))
	}
}

func TestElectionResultsAuthorization(t *testing.T) {
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	server := FreshServer(t)

	t.Run("requires authentication", func(t *testing.T) {
		unit := testutil.MakeUnit(t, nil)
		election := makeResultsElection(t, unit, "open", nil)

		w := getElectionResults(t, server, election.ID, "")
		requireElectionResultsStatus(t, w, http.StatusUnauthorized)
	})

	t.Run("rejects authenticated non-members", func(t *testing.T) {
		unit := testutil.MakeUnit(t, nil)
		now := time.Now().UTC()
		election := makeResultsElection(t, unit, "finalized", &now)
		caller := testutil.MakeUser(t, "citizen")

		w := getElectionResults(t, server, election.ID, testutil.AuthHeader(t, caller))
		requireElectionResultsForbidden(t, w)
	})

	t.Run("rejects pending and inactive memberships", func(t *testing.T) {
		for _, status := range []string{models.MembershipPending, models.MembershipRevoked} {
			status := status
			t.Run(status, func(t *testing.T) {
				unit := testutil.MakeUnit(t, nil)
				now := time.Now().UTC()
				election := makeResultsElection(t, unit, "finalized", &now)
				caller := testutil.MakeUser(t, "citizen")
				membership := testutil.MakeMembership(t, caller, unit, models.UnitRoleOfficer, false)
				if err := testutil.GetDB().Model(membership).Update("status", status).Error; err != nil {
					t.Fatalf("update membership status: %v", err)
				}

				w := getElectionResults(t, server, election.ID, testutil.AuthHeader(t, caller))
				requireElectionResultsForbidden(t, w)
			})
		}
	})

	t.Run("rejects active non-admin before finalization", func(t *testing.T) {
		unit := testutil.MakeUnit(t, nil)
		election := makeResultsElection(t, unit, "open", nil)
		caller := testutil.MakeUser(t, "citizen")
		testutil.MakeMembership(t, caller, unit, models.UnitRoleOfficer, false)

		w := getElectionResults(t, server, election.ID, testutil.AuthHeader(t, caller))
		requireElectionResultsForbidden(t, w)
	})

	t.Run("allows active members after finalization", func(t *testing.T) {
		cases := []struct {
			name   string
			status string
			hasAt  bool
		}{
			{name: "finalized", status: "finalized"},
			{name: "finalized_low_turnout", status: "finalized_low_turnout"},
			{name: "result finalized timestamp", status: "open", hasAt: true},
		}

		for _, tc := range cases {
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				unit := testutil.MakeUnit(t, nil)
				var finalizedAt *time.Time
				if tc.hasAt {
					now := time.Now().UTC()
					finalizedAt = &now
				}
				election := makeResultsElection(t, unit, tc.status, finalizedAt)
				caller := testutil.MakeUser(t, "citizen")
				testutil.MakeMembership(t, caller, unit, models.UnitRoleOfficer, false)

				w := getElectionResults(t, server, election.ID, testutil.AuthHeader(t, caller))
				requireElectionResultsAllowed(t, w, election.ID)
			})
		}
	})

	t.Run("allows unit admins before finalization", func(t *testing.T) {
		cases := []struct {
			name        string
			role        string
			isHeadAdmin bool
		}{
			{name: "unit admin", role: models.UnitRoleAdmin},
			{name: "head admin", role: models.UnitRoleOfficer, isHeadAdmin: true},
		}

		for _, tc := range cases {
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				unit := testutil.MakeUnit(t, nil)
				election := makeResultsElection(t, unit, "open", nil)
				caller := testutil.MakeUser(t, tc.role)
				testutil.MakeMembership(t, caller, unit, tc.role, tc.isHeadAdmin)

				w := getElectionResults(t, server, election.ID, testutil.AuthHeader(t, caller))
				requireElectionResultsAllowed(t, w, election.ID)
			})
		}
	})

	t.Run("allows super admins without membership", func(t *testing.T) {
		unit := testutil.MakeUnit(t, nil)
		election := makeResultsElection(t, unit, "open", nil)
		caller := testutil.MakeUser(t, "super_admin")

		w := getElectionResults(t, server, election.ID, testutil.AuthHeader(t, caller))
		requireElectionResultsAllowed(t, w, election.ID)
	})

	t.Run("rejects members and admins of other units", func(t *testing.T) {
		unit := testutil.MakeUnit(t, nil)
		otherUnit := testutil.MakeUnit(t, nil)
		election := makeResultsElection(t, unit, "open", nil)

		member := testutil.MakeUser(t, "citizen")
		testutil.MakeMembership(t, member, otherUnit, models.UnitRoleOfficer, false)

		w := getElectionResults(t, server, election.ID, testutil.AuthHeader(t, member))
		requireElectionResultsForbidden(t, w)

		otherAdmin := testutil.MakeUser(t, "unit_admin")
		testutil.MakeMembership(t, otherAdmin, otherUnit, models.UnitRoleAdmin, true)

		w = getElectionResults(t, server, election.ID, testutil.AuthHeader(t, otherAdmin))
		requireElectionResultsForbidden(t, w)
	})

	t.Run("returns not found for unknown elections", func(t *testing.T) {
		caller := testutil.MakeUser(t, "super_admin")

		w := getElectionResults(t, server, uuid.New(), testutil.AuthHeader(t, caller))
		requireElectionResultsStatus(t, w, http.StatusNotFound)
	})
}
