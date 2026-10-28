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
	"security-solution/services"
)

type ledgerBalanceResponse struct {
	UnitID   uuid.UUID `json:"unitId"`
	Balance  float64   `json:"balance"`
	TotalIn  float64   `json:"totalIn"`
	TotalOut float64   `json:"totalOut"`
	Currency string    `json:"currency"`
}

func appendLedgerEntry(t *testing.T, unit *models.SecurityUnit, createdBy uuid.UUID, direction string, amount float64) {
	t.Helper()

	svc := services.NewLedgerService()
	if _, err := svc.Append(services.LedgerEntryInput{
		UnitID:       unit.ID,
		Direction:    direction,
		EntryType:    "manual",
		Amount:       amount,
		Counterparty: "Test",
		Description:  "Authorization test entry",
		SourceType:   "manual",
		CreatedBy:    createdBy,
	}); err != nil {
		t.Fatalf("append ledger entry: %v", err)
	}
}

func getLedgerBalance(t *testing.T, server http.Handler, unitID uuid.UUID, token string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/finance/units/"+unitID.String()+"/ledger/balance", nil)
	if token != "" {
		req.Header.Set("Authorization", token)
	}
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)
	return w
}

func requireLedgerBalanceStatus(t *testing.T, w *httptest.ResponseRecorder, want int) {
	t.Helper()

	if w.Code != want {
		t.Fatalf("status = %d, want %d. body=%s", w.Code, want, w.Body.String())
	}
}

func requireLedgerBalanceForbidden(t *testing.T, w *httptest.ResponseRecorder) {
	t.Helper()

	requireLedgerBalanceStatus(t, w, http.StatusForbidden)
	if bytes.Contains(w.Body.Bytes(), []byte(`"balance"`)) ||
		bytes.Contains(w.Body.Bytes(), []byte(`"totalIn"`)) ||
		bytes.Contains(w.Body.Bytes(), []byte(`"totalOut"`)) {
		t.Fatalf("forbidden response leaked balance data: %s", w.Body.String())
	}
}

func decodeLedgerBalance(t *testing.T, w *httptest.ResponseRecorder) ledgerBalanceResponse {
	t.Helper()

	requireLedgerBalanceStatus(t, w, http.StatusOK)

	var response ledgerBalanceResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal ledger balance: %v", err)
	}
	return response
}

func TestLedgerBalanceAuthorization(t *testing.T) {
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	server := FreshServer(t)

	t.Run("requires authentication", func(t *testing.T) {
		unit := testutil.MakeUnit(t, nil)
		appendLedgerEntry(t, unit, uuid.New(), "in", 1000)

		w := getLedgerBalance(t, server, unit.ID, "")
		requireLedgerBalanceStatus(t, w, http.StatusUnauthorized)
	})

	t.Run("rejects authenticated non-members", func(t *testing.T) {
		unit := testutil.MakeUnit(t, nil)
		appendLedgerEntry(t, unit, uuid.New(), "in", 1000)
		caller := testutil.MakeUser(t, "citizen")

		w := getLedgerBalance(t, server, unit.ID, testutil.AuthHeader(t, caller))
		requireLedgerBalanceForbidden(t, w)
	})

	t.Run("allows active members and returns only that unit's balance", func(t *testing.T) {
		unitA := testutil.MakeUnit(t, nil)
		unitB := testutil.MakeUnit(t, nil)
		appendLedgerEntry(t, unitA, uuid.New(), "in", 10000)
		appendLedgerEntry(t, unitA, uuid.New(), "out", 2500)
		appendLedgerEntry(t, unitB, uuid.New(), "in", 999999)

		caller := testutil.MakeUser(t, "citizen")
		testutil.MakeMembership(t, caller, unitA, models.UnitRoleOfficer, false)

		w := getLedgerBalance(t, server, unitA.ID, testutil.AuthHeader(t, caller))
		response := decodeLedgerBalance(t, w)

		if response.UnitID != unitA.ID {
			t.Fatalf("unit ID = %s, want %s", response.UnitID, unitA.ID)
		}
		if response.TotalIn != 10000 || response.TotalOut != 2500 || response.Balance != 7500 {
			t.Fatalf("unexpected totals: %s", w.Body.String())
		}
		if response.Currency != "NGN" {
			t.Fatalf("currency = %q, want NGN", response.Currency)
		}
	})

	t.Run("rejects members of another unit", func(t *testing.T) {
		unitA := testutil.MakeUnit(t, nil)
		unitB := testutil.MakeUnit(t, nil)
		appendLedgerEntry(t, unitB, uuid.New(), "in", 1000)
		caller := testutil.MakeUser(t, "citizen")
		testutil.MakeMembership(t, caller, unitA, models.UnitRoleOfficer, false)

		w := getLedgerBalance(t, server, unitB.ID, testutil.AuthHeader(t, caller))
		requireLedgerBalanceForbidden(t, w)
	})

	t.Run("rejects pending and inactive memberships", func(t *testing.T) {
		for _, status := range []string{models.MembershipPending, models.MembershipRevoked} {
			status := status
			t.Run(status, func(t *testing.T) {
				unit := testutil.MakeUnit(t, nil)
				appendLedgerEntry(t, unit, uuid.New(), "in", 1000)
				caller := testutil.MakeUser(t, "citizen")
				membership := testutil.MakeMembership(t, caller, unit, models.UnitRoleOfficer, false)
				if err := testutil.GetDB().Model(membership).Update("status", status).Error; err != nil {
					t.Fatalf("update membership status: %v", err)
				}

				w := getLedgerBalance(t, server, unit.ID, testutil.AuthHeader(t, caller))
				requireLedgerBalanceForbidden(t, w)
			})
		}
	})

	t.Run("allows super admins without membership", func(t *testing.T) {
		unit := testutil.MakeUnit(t, nil)
		appendLedgerEntry(t, unit, uuid.New(), "in", 2000)
		appendLedgerEntry(t, unit, uuid.New(), "out", 500)
		caller := testutil.MakeUser(t, "super_admin")

		w := getLedgerBalance(t, server, unit.ID, testutil.AuthHeader(t, caller))
		response := decodeLedgerBalance(t, w)

		if response.Balance != 1500 {
			t.Fatalf("balance = %v, want 1500. body=%s", response.Balance, w.Body.String())
		}
	})

	t.Run("rejects invalid unit IDs", func(t *testing.T) {
		caller := testutil.MakeUser(t, "super_admin")

		req := httptest.NewRequest(http.MethodGet, "/api/v1/finance/units/not-a-uuid/ledger/balance", nil)
		req.Header.Set("Authorization", testutil.AuthHeader(t, caller))
		w := httptest.NewRecorder()
		server.ServeHTTP(w, req)

		requireLedgerBalanceStatus(t, w, http.StatusBadRequest)
	})
}
