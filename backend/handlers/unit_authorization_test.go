//go:build integration

package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"security-solution/internal/testutil"
	"security-solution/models"
)

func createUnitRequest(t *testing.T, token, name string) *httptest.ResponseRecorder {
	t.Helper()

	body, err := json.Marshal(map[string]any{
		"name":      name,
		"type":      "community",
		"latitude":  7.19,
		"longitude": 8.14,
	})
	if err != nil {
		t.Fatalf("marshal create unit payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/units", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", token)
	w := httptest.NewRecorder()
	FreshServer(t).ServeHTTP(w, req)
	return w
}

func updateUnitRequest(t *testing.T, token, unitID string, payload map[string]any) *httptest.ResponseRecorder {
	t.Helper()

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal update unit payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPut, "/api/v1/units/"+unitID, strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", token)
	w := httptest.NewRecorder()
	FreshServer(t).ServeHTTP(w, req)
	return w
}

func TestCreateUnit_UnitAdminForbidden(t *testing.T) {
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	admin := testutil.MakeUser(t, "unit_admin")
	testutil.MakeUnit(t, admin)

	w := createUnitRequest(t, testutil.AuthHeader(t, admin), "Unauthorized New Unit")
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403. body=%s", w.Code, w.Body.String())
	}

	var count int64
	if err := testutil.GetDB().Model(&models.SecurityUnit{}).
		Where("name = ?", "Unauthorized New Unit").Count(&count).Error; err != nil {
		t.Fatalf("count created units: %v", err)
	}
	if count != 0 {
		t.Fatalf("unit_admin created %d unit(s), want 0", count)
	}
}

func TestUpdateUnit_UnitAdminCannotUpdateAnotherUnit(t *testing.T) {
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	adminA := testutil.MakeUser(t, "unit_admin")
	testutil.MakeUnit(t, adminA)

	adminB := testutil.MakeUser(t, "unit_admin")
	unitB := testutil.MakeUnit(t, adminB)

	w := updateUnitRequest(t, testutil.AuthHeader(t, adminA), unitB.ID.String(), map[string]any{
		"name": "Hijacked Unit",
	})
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403. body=%s", w.Code, w.Body.String())
	}

	var reloaded models.SecurityUnit
	if err := testutil.GetDB().First(&reloaded, "id = ?", unitB.ID).Error; err != nil {
		t.Fatalf("reload unit B: %v", err)
	}
	if reloaded.Name != unitB.Name {
		t.Fatalf("unit B name = %q, want unchanged %q", reloaded.Name, unitB.Name)
	}
}

func TestUpdateUnit_HeadAdminCanUpdateOwnUnit(t *testing.T) {
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	head := testutil.MakeUser(t, "unit_admin")
	unit := testutil.MakeUnit(t, head)

	w := updateUnitRequest(t, testutil.AuthHeader(t, head), unit.ID.String(), map[string]any{
		"contactPerson": "Updated Contact",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200. body=%s", w.Code, w.Body.String())
	}

	var reloaded models.SecurityUnit
	if err := testutil.GetDB().First(&reloaded, "id = ?", unit.ID).Error; err != nil {
		t.Fatalf("reload unit: %v", err)
	}
	if reloaded.ContactPerson != "Updated Contact" {
		t.Fatalf("contact person = %q, want %q", reloaded.ContactPerson, "Updated Contact")
	}
}

func TestUpdateUnit_RegularUnitAdminCannotUpdateOwnUnit(t *testing.T) {
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	admin := testutil.MakeUser(t, "unit_admin")
	unit := testutil.MakeUnit(t, nil)
	testutil.MakeMembership(t, admin, unit, models.UnitRoleAdmin, false)

	w := updateUnitRequest(t, testutil.AuthHeader(t, admin), unit.ID.String(), map[string]any{
		"name": "Unauthorized Rename",
	})
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403. body=%s", w.Code, w.Body.String())
	}

	var reloaded models.SecurityUnit
	if err := testutil.GetDB().First(&reloaded, "id = ?", unit.ID).Error; err != nil {
		t.Fatalf("reload unit: %v", err)
	}
	if reloaded.Name != unit.Name {
		t.Fatalf("unit name = %q, want unchanged %q", reloaded.Name, unit.Name)
	}
}

func TestUpdateUnit_SuperAdminCanUpdateAnyUnit(t *testing.T) {
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	head := testutil.MakeUser(t, "unit_admin")
	unit := testutil.MakeUnit(t, head)
	superAdmin := testutil.MakeUser(t, "super_admin")

	w := updateUnitRequest(t, testutil.AuthHeader(t, superAdmin), unit.ID.String(), map[string]any{
		"contactPerson": "Super Admin Contact",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200. body=%s", w.Code, w.Body.String())
	}

	var reloaded models.SecurityUnit
	if err := testutil.GetDB().First(&reloaded, "id = ?", unit.ID).Error; err != nil {
		t.Fatalf("reload unit: %v", err)
	}
	if reloaded.ContactPerson != "Super Admin Contact" {
		t.Fatalf("contact person = %q, want %q", reloaded.ContactPerson, "Super Admin Contact")
	}
}
