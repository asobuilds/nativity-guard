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

func officerAPIRequest(t *testing.T, server http.Handler, method, path, token string, payload any) *httptest.ResponseRecorder {
	t.Helper()
	var body *bytes.Reader
	if payload == nil {
		body = bytes.NewReader(nil)
	} else {
		encoded, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal officer request: %v", err)
		}
		body = bytes.NewReader(encoded)
	}
	req := httptest.NewRequest(method, path, body)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", token)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)
	return w
}

func TestCreateOfficer_BindsActiveUnitMemberToOperationalIdentity(t *testing.T) {
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	server := FreshServer(t)
	unitAdmin := testutil.MakeUser(t, "unit_admin")
	unit := testutil.MakeUnit(t, unitAdmin)
	officerUser := testutil.MakeUser(t, "officer")
	testutil.MakeMembership(t, officerUser, unit, models.UnitRoleOfficer, false)

	w := officerAPIRequest(t, server, http.MethodPost, "/api/v1/units/"+unit.ID.String()+"/officers",
		testutil.AuthHeader(t, unitAdmin), map[string]any{
			"userId": officerUser.ID.String(), "name": "Amina Bello", "rank": "Sergeant",
			"badgeNumber": "NG-TEST-"+uuid.NewString()[:8], "role": "field officer",
			"phone": officerUser.Phone, "email": officerUser.Email,
		})
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201. body=%s", w.Code, w.Body.String())
	}

	var response struct {
		Officer models.Officer `json:"officer"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode officer response: %v", err)
	}
	if response.Officer.ID != officerUser.ID {
		t.Fatalf("officer ID = %s, want authenticated user ID %s", response.Officer.ID, officerUser.ID)
	}
	if response.Officer.UnitID != unit.ID || response.Officer.UnitOfficerID == "" {
		t.Fatalf("officer roster identity incomplete: %+v", response.Officer)
	}

	reporter := testutil.MakeUser(t, "citizen")
	caseRecord := testutil.MakeCase(t, reporter, unit)
	w = officerAPIRequest(t, server, http.MethodPost, "/api/v1/cases/"+caseRecord.ID.String()+"/assign",
		testutil.AuthHeader(t, unitAdmin), map[string]any{
			"officerId": response.Officer.ID.String(), "role": "primary",
		})
	if w.Code != http.StatusOK {
		t.Fatalf("assignment status = %d, want 200. body=%s", w.Code, w.Body.String())
	}

	w = officerAPIRequest(t, server, http.MethodPost, "/api/v1/cases/"+caseRecord.ID.String()+"/dispatch",
		testutil.AuthHeader(t, officerUser), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("dispatch status = %d, want 200. body=%s", w.Code, w.Body.String())
	}
}

func TestCreateOfficer_RejectsUsersWithoutActiveOfficerMembership(t *testing.T) {
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	server := FreshServer(t)
	unitAdmin := testutil.MakeUser(t, "unit_admin")
	unit := testutil.MakeUnit(t, unitAdmin)
	unassignedOfficer := testutil.MakeUser(t, "officer")

	w := officerAPIRequest(t, server, http.MethodPost, "/api/v1/units/"+unit.ID.String()+"/officers",
		testutil.AuthHeader(t, unitAdmin), map[string]any{
			"userId": unassignedOfficer.ID.String(), "name": "Unassigned Officer",
			"rank": "Constable", "badgeNumber": "NG-UNASSIGNED-"+uuid.NewString()[:8],
			"role": "field officer",
		})
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403. body=%s", w.Code, w.Body.String())
	}

	var count int64
	if err := testutil.GetDB().Model(&models.Officer{}).Where("id = ?", unassignedOfficer.ID).Count(&count).Error; err != nil {
		t.Fatalf("count officer records: %v", err)
	}
	if count != 0 {
		t.Fatalf("created %d roster records for an unassigned user, want 0", count)
	}
}
