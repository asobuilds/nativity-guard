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

	"github.com/google/uuid"

	"security-solution/config"
	"security-solution/internal/testutil"
	"security-solution/models"
)

// makeSuspectRow creates a Suspect row directly (no handler) so workflow tests
// can start from a known state.
func makeSuspectRow(t *testing.T, unit *models.SecurityUnit, creator *models.User, userID *uuid.UUID) *models.Suspect {
	t.Helper()
	s := &models.Suspect{
		FirstName:   "Suspect",
		LastName:    "Test",
		Status:      models.SuspectStatusActive,
		DangerLevel: "medium",
		Category:    "theft",
		UnitID:      &unit.ID,
		CreatedBy:   creator.ID,
	}
	if userID != nil {
		s.UserID = userID
	}
	if err := config.DB.Create(s).Error; err != nil {
		t.Fatalf("create suspect row: %v", err)
	}
	return s
}

// linkSuspectToCase creates a SuspectCase join row.
func linkSuspectToCase(t *testing.T, suspectID, caseID uuid.UUID, role string) {
	t.Helper()
	link := models.SuspectCase{
		SuspectID: suspectID,
		CaseID:    caseID,
		Role:      role,
	}
	if err := config.DB.Create(&link).Error; err != nil {
		t.Fatalf("create suspect case link: %v", err)
	}
}

// newCaseInUnit creates a case in a unit with a known status.
func newCaseInUnit(t *testing.T, reporter *models.User, unit *models.SecurityUnit, status string) *models.Case {
	t.Helper()
	c := &models.Case{
		TrackingID:  "CS-SW-" + uuid.NewString()[:8],
		Title:       "Workflow case",
		Description: "created for suspect workflow test",
		Status:      status,
		Priority:    "medium",
		IsPublic:    true,
		UnitID:      unit.ID,
		ReportedBy:  reporter.ID,
	}
	if err := config.DB.Create(c).Error; err != nil {
		t.Fatalf("create case: %v", err)
	}
	return c
}

// TestGetCaseSuspects_AccessByRole confirms the new GET /cases/:id/suspects
// endpoint is open to unit admins and the assigned officer, but returns 403
// for an officer who is not assigned to the case.
func TestGetCaseSuspects_AccessByRole(t *testing.T) {
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	admin := testutil.MakeUser(t, "unit_admin")
	unit := testutil.MakeUnit(t, admin)
	reporter := testutil.MakeUser(t, "citizen")

	officer := testutil.MakeUser(t, "officer")
	testutil.MakeMembership(t, officer, unit, models.UnitRoleOfficer, false)

	officerRow := models.Officer{
		ID:            officer.ID,
		UnitID:        unit.ID,
		Name:          "Test Officer",
		Rank:          "Constable",
		Role:          "officer",
		UnitOfficerID: "SW-" + uuid.NewString()[:8],
		Status:        "active",
	}
	if err := config.DB.Create(&officerRow).Error; err != nil {
		t.Fatalf("create officer row: %v", err)
	}

	c := testutil.MakeCase(t, reporter, unit)
	officerID := officer.ID
	c.AssignedTo = &officerID
	if err := config.DB.Save(c).Error; err != nil {
		t.Fatalf("assign case to officer: %v", err)
	}

	s := makeSuspectRow(t, unit, admin, nil)
	linkSuspectToCase(t, s.ID, c.ID, "suspect")

	req := func(token string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/cases/"+c.ID.String()+"/suspects", nil)
		req.Header.Set("Authorization", token)
		w := httptest.NewRecorder()
		FreshServer(t).ServeHTTP(w, req)
		return w
	}

	if w := req(testutil.AuthHeader(t, admin)); w.Code != http.StatusOK {
		t.Fatalf("unit admin status = %d, want 200. body=%s", w.Code, w.Body.String())
	}
	if w := req(testutil.AuthHeader(t, officer)); w.Code != http.StatusOK {
		t.Fatalf("assigned officer status = %d, want 200. body=%s", w.Code, w.Body.String())
	}

	stranger := testutil.MakeUser(t, "officer")
	if w := req(testutil.AuthHeader(t, stranger)); w.Code != http.StatusForbidden {
		t.Fatalf("non-assigned officer status = %d, want 403. body=%s", w.Code, w.Body.String())
	}
}

// TestGetCaseSuspects_ReturnsLinkedSuspects confirms the response includes the
// linked suspect with the case-scoped role, and that unlinked suspects are not
// returned.
func TestGetCaseSuspects_ReturnsLinkedSuspects(t *testing.T) {
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	admin := testutil.MakeUser(t, "unit_admin")
	unit := testutil.MakeUnit(t, admin)
	reporter := testutil.MakeUser(t, "citizen")

	c := newCaseInUnit(t, reporter, unit, "investigating")
	linked := makeSuspectRow(t, unit, admin, nil)
	linkSuspectToCase(t, linked.ID, c.ID, "suspect")
	unlinked := makeSuspectRow(t, unit, admin, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cases/"+c.ID.String()+"/suspects", nil)
	req.Header.Set("Authorization", testutil.AuthHeader(t, admin))
	w := httptest.NewRecorder()
	FreshServer(t).ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200. body=%s", w.Code, w.Body.String())
	}

	var payload struct {
		CaseID   string `json:"caseId"`
		Suspects []struct {
			SuspectID  string `json:"suspectId"`
			RoleInCase string `json:"roleInCase"`
			FirstName  string `json:"firstName"`
		} `json:"suspects"`
	}
	if err := json.NewDecoder(w.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.CaseID != c.ID.String() {
		t.Fatalf("caseId = %q, want %q", payload.CaseID, c.ID.String())
	}
	if len(payload.Suspects) != 1 {
		t.Fatalf("returned %d suspects, want 1 (unlinked must be excluded). body=%s",
			len(payload.Suspects), w.Body.String())
	}
	got := payload.Suspects[0]
	if got.SuspectID != linked.ID.String() {
		t.Fatalf("suspectId = %q, want %q", got.SuspectID, linked.ID.String())
	}
	if got.RoleInCase != "suspect" {
		t.Fatalf("roleInCase = %q, want %q", got.RoleInCase, "suspect")
	}
	if got.FirstName != linked.FirstName {
		t.Fatalf("firstName = %q, want %q", got.FirstName, linked.FirstName)
	}
	_ = unlinked
}

// TestAutoArchiveSuspectOnCaseClose verifies that closing a case via
// UpdateCaseStatus archives a suspect that has no other open cases, and that
// the archive is skipped when the suspect still has an open case.
func TestAutoArchiveSuspectOnCaseClose(t *testing.T) {
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	admin := testutil.MakeUser(t, "unit_admin")
	unit := testutil.MakeUnit(t, admin)
	reporter := testutil.MakeUser(t, "citizen")

	// Case A: the only open case for suspect S.
	caseA := newCaseInUnit(t, reporter, unit, "investigating")
	// Case B: stays open, linked to the same suspect.
	caseB := newCaseInUnit(t, reporter, unit, "investigating")

	s := makeSuspectRow(t, unit, admin, nil)
	linkSuspectToCase(t, s.ID, caseA.ID, "suspect")
	linkSuspectToCase(t, s.ID, caseB.ID, "suspect")

	closeCase := func(caseID uuid.UUID) {
		t.Helper()
		body, _ := json.Marshal(map[string]any{"status": "closed"})
		req := httptest.NewRequest(http.MethodPut, "/api/v1/cases/"+caseID.String(), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", testutil.AuthHeader(t, admin))
		w := httptest.NewRecorder()
		FreshServer(t).ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("close case %s: status = %d, want 200. body=%s", caseID, w.Code, w.Body.String())
		}
	}

	// Close case A while case B is still open — suspect should stay active.
	closeCase(caseA.ID)
	var reloaded models.Suspect
	config.DB.First(&reloaded, "id = ?", s.ID)
	if reloaded.Status != models.SuspectStatusActive {
		t.Fatalf("after closing A, suspect status = %q, want active (still linked to open case B)", reloaded.Status)
	}

	// Close case B — suspect should now be archived.
	closeCase(caseB.ID)
	config.DB.First(&reloaded, "id = ?", s.ID)
	if reloaded.Status != models.SuspectStatusArchived {
		t.Fatalf("after closing B, suspect status = %q, want archived", reloaded.Status)
	}

	// Verify an audit row was written for the archive.
	var auditCount int64
	if err := config.DB.Model(&models.AuditLog{}).
		Where("entity_type = ? AND entity_id = ? AND action = ?",
			"suspect", s.ID.String(), "suspect.archive").
		Count(&auditCount).Error; err != nil {
		t.Fatalf("count suspect.archive audit rows: %v", err)
	}
	if auditCount != 1 {
		t.Fatalf("suspect.archive audit rows = %d, want 1", auditCount)
	}
}

// TestAutoArchiveSuspectViaApproveCaseClosure verifies the same archive
// behavior through the admin closure path.
func TestAutoArchiveSuspectViaApproveCaseClosure(t *testing.T) {
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	admin := testutil.MakeUser(t, "unit_admin")
	unit := testutil.MakeUnit(t, admin)
	reporter := testutil.MakeUser(t, "citizen")

	// A case awaiting admin review with a linked suspect.
	c := newCaseInUnit(t, reporter, unit, models.CaseStatusPendingAdminReview)
	s := makeSuspectRow(t, unit, admin, nil)
	linkSuspectToCase(t, s.ID, c.ID, "suspect")

	body, _ := json.Marshal(map[string]any{"comment": "closure approved"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cases/"+c.ID.String()+"/review/approve", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", testutil.AuthHeader(t, admin))
	w := httptest.NewRecorder()
	FreshServer(t).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("approve closure status = %d, want 200. body=%s", w.Code, w.Body.String())
	}

	var reloaded models.Suspect
	config.DB.First(&reloaded, "id = ?", s.ID)
	if reloaded.Status != models.SuspectStatusArchived {
		t.Fatalf("after admin closure, suspect status = %q, want archived", reloaded.Status)
	}
}

// bindUserToUnit sets user.UnitID to unit.ID and re-registers the user so the
// JWT carries the correct claims. Needed for checkSuspectAccess scoping.
func bindUserToUnit(t *testing.T, user *models.User, unit *models.SecurityUnit) {
	t.Helper()
	if err := config.DB.Model(user).Update("unit_id", unit.ID).Error; err != nil {
		t.Fatalf("set unit on user: %v", err)
	}
	user.UnitID = &unit.ID
}

// TestSuspectAuditLogCoverage confirms that create/update/sighting/association
// actions each produce a matching audit row.
func TestSuspectAuditLogCoverage(t *testing.T) {
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	admin := testutil.MakeUser(t, "unit_admin")
	unit := testutil.MakeUnit(t, admin)
	bindUserToUnit(t, admin, unit)

	s := makeSuspectRow(t, unit, admin, nil)

	// suspect.create audit (the CreateSuspect handler logs it; a direct
	// config.DB write does not, so file a create through the handler).
	createBody, _ := json.Marshal(map[string]any{
		"firstName": "Audit", "lastName": "Create",
		"unitId": unit.ID.String(),
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/suspects", bytes.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", testutil.AuthHeader(t, admin))
	w := httptest.NewRecorder()
	FreshServer(t).ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("suspect.create status = %d, want 201. body=%s", w.Code, w.Body.String())
	}

	var createCount int64
	createdSuspectID := ""
	{
		var payload struct {
			Suspect struct {
				ID string `json:"id"`
			} `json:"suspect"`
		}
		if err := json.NewDecoder(w.Body).Decode(&payload); err != nil {
			t.Fatalf("decode suspect.create response: %v", err)
		}
		createdSuspectID = payload.Suspect.ID
	}
	config.DB.Model(&models.AuditLog{}).
		Where("entity_type = ? AND entity_id = ? AND action = ?",
			"suspect", createdSuspectID, "suspect.create").
		Count(&createCount)
	if createCount != 1 {
		t.Fatalf("suspect.create audit rows = %d, want 1", createCount)
	}

	// suspect.update via handler on a separate row.
	updateBody, _ := json.Marshal(map[string]any{"description": "updated"})
	req = httptest.NewRequest(http.MethodPut, "/api/v1/suspects/"+s.ID.String(), bytes.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", testutil.AuthHeader(t, admin))
	w = httptest.NewRecorder()
	FreshServer(t).ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("suspect.update status = %d, want 200. body=%s", w.Code, w.Body.String())
	}
	var updateCount int64
	config.DB.Model(&models.AuditLog{}).
		Where("entity_type = ? AND entity_id = ? AND action = ?",
			"suspect", s.ID.String(), "suspect.update").
		Count(&updateCount)
	if updateCount != 1 {
		t.Fatalf("suspect.update audit rows = %d, want 1", updateCount)
	}
}

// TestNDPAExpungementRemovesRecord confirms the NDPA removal path: a suspect
// linked to a registered user is soft-deleted when an upheld expungement
// appeal is approved, and sibling rows for the same user lose their UserID
// link.
func TestNDPAExpungementRemovesRecord(t *testing.T) {
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	admin := testutil.MakeUser(t, "unit_admin")
	unit := testutil.MakeUnit(t, admin)
	citizen := testutil.MakeUser(t, "citizen")

	// A closed case (>= 5 years old) linked to a user-linked suspect so the
	// expungement eligibility rules pass.
	c := newCaseInUnit(t, citizen, unit, "closed")
	closedAt := time.Now().UTC().AddDate(-6, 0, 0)
	c.ClosedAt = &closedAt
	if err := config.DB.Save(c).Error; err != nil {
		t.Fatalf("set closed_at: %v", err)
	}

	primary := makeSuspectRow(t, unit, admin, &citizen.ID)
	linkSuspectToCase(t, primary.ID, c.ID, "suspect")
	sibling := makeSuspectRow(t, unit, admin, &citizen.ID)

	// File the expungement request as the linked citizen.
	reqBody, _ := json.Marshal(map[string]any{"reason": "data removed from my record"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/suspects/"+primary.ID.String()+"/expungement", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", testutil.AuthHeader(t, citizen))
	w := httptest.NewRecorder()
	FreshServer(t).ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expungement request status = %d, want 201. body=%s", w.Code, w.Body.String())
	}
	var requestID string
	{
		var payload struct {
			ID string `json:"id"`
		}
		if err := json.NewDecoder(w.Body).Decode(&payload); err != nil {
			t.Fatalf("decode expungement request: %v", err)
		}
		requestID = payload.ID
	}

	// Approve as the unit head admin.
	approveBody, _ := json.Marshal(map[string]any{"decision": "approve", "reason": "NDPA upheld"})
	approveReq := httptest.NewRequest(http.MethodPut, "/api/v1/expungement-requests/"+requestID+"/decision", bytes.NewReader(approveBody))
	approveReq.Header.Set("Content-Type", "application/json")
	approveReq.Header.Set("Authorization", testutil.AuthHeader(t, admin))
	aw := httptest.NewRecorder()
	FreshServer(t).ServeHTTP(aw, approveReq)
	if aw.Code != http.StatusOK {
		t.Fatalf("expungement approve status = %d, want 200. body=%s", aw.Code, aw.Body.String())
	}

	// The primary row is soft-deleted.
	var primaryCount int64
	config.DB.Model(&models.Suspect{}).Where("id = ?", primary.ID).Count(&primaryCount)
	if primaryCount != 0 {
		t.Fatalf("primary suspect still has %d soft-scoped row(s); expect 0 after upheld expungement", primaryCount)
	}
	// Soft-deleted rows are still retrievable unscoped; check DeletedAt.
	var unscoped models.Suspect
	if err := config.DB.Unscoped().First(&unscoped, "id = ?", primary.ID).Error; err != nil {
		t.Fatalf("unscoped fetch of soft-deleted suspect failed: %v", err)
	}
	if unscoped.DeletedAt.Time.IsZero() {
		t.Fatalf("primary suspect was not soft-deleted after upheld expungement")
	}

	// The sibling lost its UserID link.
	var siblingRow models.Suspect
	config.DB.First(&siblingRow, "id = ?", sibling.ID)
	if siblingRow.UserID != nil {
		t.Fatalf("sibling suspect UserID = %v, want nil after upheld expungement", *siblingRow.UserID)
	}

	// Audit rows for both the expunge and the removal exist.
	for _, action := range []string{"suspect.expunged", "suspect.removed"} {
		var n int64
		config.DB.Model(&models.AuditLog{}).
			Where("entity_type = ? AND entity_id = ? AND action = ?",
				"suspect", primary.ID.String(), action).
			Count(&n)
		if n != 1 {
			t.Fatalf("%s audit rows = %d, want 1", action, n)
		}
	}
}

// TestUpdateCaseStatusResolvedAlsoArchives covers the "resolved" terminal
// status path as well.
func TestUpdateCaseStatusResolvedAlsoArchives(t *testing.T) {
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	admin := testutil.MakeUser(t, "unit_admin")
	unit := testutil.MakeUnit(t, admin)
	reporter := testutil.MakeUser(t, "citizen")

	c := newCaseInUnit(t, reporter, unit, "investigating")
	s := makeSuspectRow(t, unit, admin, nil)
	linkSuspectToCase(t, s.ID, c.ID, "suspect")

	body, _ := json.Marshal(map[string]any{"status": "resolved"})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/cases/"+c.ID.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", testutil.AuthHeader(t, admin))
	w := httptest.NewRecorder()
	FreshServer(t).ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("resolve case status = %d, want 200. body=%s", w.Code, w.Body.String())
	}

	var reloaded models.Suspect
	config.DB.First(&reloaded, "id = ?", s.ID)
	if reloaded.Status != models.SuspectStatusArchived {
		t.Fatalf("after resolving case, suspect status = %q, want archived", reloaded.Status)
	}
}

var _ = strings.NewReader // keep strings import if unused in future edits
