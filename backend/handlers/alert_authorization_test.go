//go:build integration

package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"security-solution/internal/testutil"
	"security-solution/models"
)

func makeConfirmAlert(t *testing.T, unitID *uuid.UUID, createdBy uuid.UUID) *models.CommunityAlert {
	t.Helper()

	alert := &models.CommunityAlert{
		Title:     "Test alert",
		Content:   "Test alert content",
		Type:      "security",
		Severity:  "high",
		UnitID:    unitID,
		Location:  "Test location",
		Latitude:  6.5244,
		Longitude: 3.3792,
		Radius:    10,
		Status:    "active",
		CreatedBy: createdBy,
	}
	if err := testutil.GetDB().Create(alert).Error; err != nil {
		t.Fatalf("create alert: %v", err)
	}
	return alert
}

func postConfirmAlert(t *testing.T, server http.Handler, alertID uuid.UUID, token string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/alerts/"+alertID.String()+"/confirm", nil)
	if token != "" {
		req.Header.Set("Authorization", token)
	}
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)
	return w
}

func requireConfirmAlertStatus(t *testing.T, w *httptest.ResponseRecorder, want int) {
	t.Helper()

	if w.Code != want {
		t.Fatalf("status = %d, want %d. body=%s", w.Code, want, w.Body.String())
	}
}

func requireAlertUnconfirmed(t *testing.T, alertID uuid.UUID) {
	t.Helper()

	var alert models.CommunityAlert
	if err := testutil.GetDB().First(&alert, "id = ?", alertID).Error; err != nil {
		t.Fatalf("reload alert: %v", err)
	}
	if alert.ConfirmedBy != nil || alert.ConfirmedAt != nil {
		t.Fatalf("alert was mutated: confirmedBy=%v confirmedAt=%v", alert.ConfirmedBy, alert.ConfirmedAt)
	}
}

func requireAlertConfirmedBy(t *testing.T, alertID, userID uuid.UUID) {
	t.Helper()

	var alert models.CommunityAlert
	if err := testutil.GetDB().First(&alert, "id = ?", alertID).Error; err != nil {
		t.Fatalf("reload alert: %v", err)
	}
	if alert.ConfirmedBy == nil || *alert.ConfirmedBy != userID {
		t.Fatalf("confirmedBy = %v, want %s", alert.ConfirmedBy, userID)
	}
	if alert.ConfirmedAt == nil {
		t.Fatalf("confirmedAt is nil")
	}
}

func TestAlertConfirmationAuthorization(t *testing.T) {
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	server := FreshServer(t)

	t.Run("requires authentication", func(t *testing.T) {
		creator := testutil.MakeUser(t, "citizen")
		unit := testutil.MakeUnit(t, nil)
		alert := makeConfirmAlert(t, &unit.ID, creator.ID)

		w := postConfirmAlert(t, server, alert.ID, "")
		requireConfirmAlertStatus(t, w, http.StatusUnauthorized)
		requireAlertUnconfirmed(t, alert.ID)
	})

	t.Run("rejects admin from another unit", func(t *testing.T) {
		creator := testutil.MakeUser(t, "citizen")
		unitA := testutil.MakeUnit(t, nil)
		unitB := testutil.MakeUnit(t, nil)
		alert := makeConfirmAlert(t, &unitB.ID, creator.ID)
		caller := testutil.MakeUser(t, models.UnitRoleAdmin)
		testutil.MakeMembership(t, caller, unitA, models.UnitRoleAdmin, false)

		w := postConfirmAlert(t, server, alert.ID, testutil.AuthHeader(t, caller))
		requireConfirmAlertStatus(t, w, http.StatusForbidden)
		requireAlertUnconfirmed(t, alert.ID)
	})

	t.Run("rejects head admin from another unit", func(t *testing.T) {
		creator := testutil.MakeUser(t, "citizen")
		unitA := testutil.MakeUnit(t, nil)
		unitB := testutil.MakeUnit(t, nil)
		alert := makeConfirmAlert(t, &unitB.ID, creator.ID)
		caller := testutil.MakeUser(t, models.UnitRoleAdmin)
		testutil.MakeMembership(t, caller, unitA, models.UnitRoleAdmin, true)

		w := postConfirmAlert(t, server, alert.ID, testutil.AuthHeader(t, caller))
		requireConfirmAlertStatus(t, w, http.StatusForbidden)
		requireAlertUnconfirmed(t, alert.ID)
	})

	t.Run("allows head admin of the alert unit", func(t *testing.T) {
		unit := testutil.MakeUnit(t, nil)
		caller := testutil.MakeUser(t, models.UnitRoleAdmin)
		testutil.MakeMembership(t, caller, unit, models.UnitRoleAdmin, true)
		alert := makeConfirmAlert(t, &unit.ID, caller.ID)

		w := postConfirmAlert(t, server, alert.ID, testutil.AuthHeader(t, caller))
		requireConfirmAlertStatus(t, w, http.StatusOK)
		requireAlertConfirmedBy(t, alert.ID, caller.ID)
	})

	t.Run("rejects non-head admin in the alert unit", func(t *testing.T) {
		unit := testutil.MakeUnit(t, nil)
		caller := testutil.MakeUser(t, models.UnitRoleAdmin)
		testutil.MakeMembership(t, caller, unit, models.UnitRoleAdmin, false)
		alert := makeConfirmAlert(t, &unit.ID, caller.ID)

		w := postConfirmAlert(t, server, alert.ID, testutil.AuthHeader(t, caller))
		requireConfirmAlertStatus(t, w, http.StatusForbidden)
		requireAlertUnconfirmed(t, alert.ID)
	})

	t.Run("rejects unverified head admin", func(t *testing.T) {
		unit := testutil.MakeUnit(t, nil)
		caller := testutil.MakeUser(t, models.UnitRoleAdmin)
		membership := testutil.MakeMembership(t, caller, unit, models.UnitRoleAdmin, true)
		if err := testutil.GetDB().Model(membership).Update("verified_at", nil).Error; err != nil {
			t.Fatalf("clear verified_at: %v", err)
		}
		alert := makeConfirmAlert(t, &unit.ID, caller.ID)

		w := postConfirmAlert(t, server, alert.ID, testutil.AuthHeader(t, caller))
		requireConfirmAlertStatus(t, w, http.StatusForbidden)
		requireAlertUnconfirmed(t, alert.ID)
	})

	t.Run("rejects revoked head admin", func(t *testing.T) {
		unit := testutil.MakeUnit(t, nil)
		caller := testutil.MakeUser(t, models.UnitRoleAdmin)
		membership := testutil.MakeMembership(t, caller, unit, models.UnitRoleAdmin, true)
		if err := testutil.GetDB().Model(membership).Update("status", models.MembershipRevoked).Error; err != nil {
			t.Fatalf("revoke membership: %v", err)
		}
		alert := makeConfirmAlert(t, &unit.ID, caller.ID)

		w := postConfirmAlert(t, server, alert.ID, testutil.AuthHeader(t, caller))
		requireConfirmAlertStatus(t, w, http.StatusForbidden)
		requireAlertUnconfirmed(t, alert.ID)
	})

	t.Run("allows super admin without membership", func(t *testing.T) {
		creator := testutil.MakeUser(t, "citizen")
		unit := testutil.MakeUnit(t, nil)
		alert := makeConfirmAlert(t, &unit.ID, creator.ID)
		caller := testutil.MakeUser(t, "super_admin")

		w := postConfirmAlert(t, server, alert.ID, testutil.AuthHeader(t, caller))
		requireConfirmAlertStatus(t, w, http.StatusOK)
		requireAlertConfirmedBy(t, alert.ID, caller.ID)
	})

	t.Run("rejects non-superadmin for global alerts", func(t *testing.T) {
		creator := testutil.MakeUser(t, "citizen")
		unit := testutil.MakeUnit(t, nil)
		alert := makeConfirmAlert(t, nil, creator.ID)
		caller := testutil.MakeUser(t, models.UnitRoleAdmin)
		testutil.MakeMembership(t, caller, unit, models.UnitRoleAdmin, true)

		w := postConfirmAlert(t, server, alert.ID, testutil.AuthHeader(t, caller))
		requireConfirmAlertStatus(t, w, http.StatusForbidden)
		requireAlertUnconfirmed(t, alert.ID)
	})

	t.Run("allows super admin for global alerts", func(t *testing.T) {
		creator := testutil.MakeUser(t, "citizen")
		alert := makeConfirmAlert(t, nil, creator.ID)
		caller := testutil.MakeUser(t, "super_admin")

		w := postConfirmAlert(t, server, alert.ID, testutil.AuthHeader(t, caller))
		requireConfirmAlertStatus(t, w, http.StatusOK)
		requireAlertConfirmedBy(t, alert.ID, caller.ID)
	})

	t.Run("returns not found for unknown alerts", func(t *testing.T) {
		caller := testutil.MakeUser(t, "super_admin")

		w := postConfirmAlert(t, server, uuid.New(), testutil.AuthHeader(t, caller))
		requireConfirmAlertStatus(t, w, http.StatusNotFound)
	})

	t.Run("rejects invalid alert IDs", func(t *testing.T) {
		caller := testutil.MakeUser(t, "super_admin")

		req := httptest.NewRequest(http.MethodPost, "/api/v1/alerts/not-a-uuid/confirm", nil)
		req.Header.Set("Authorization", testutil.AuthHeader(t, caller))
		w := httptest.NewRecorder()
		server.ServeHTTP(w, req)

		requireConfirmAlertStatus(t, w, http.StatusBadRequest)
	})
}
