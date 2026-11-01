//go:build integration

package handlers_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"security-solution/internal/testutil"
	"security-solution/models"
)

type agencyTenantResponse struct {
	Tenant models.AgencyTenant `json:"tenant"`
}

type agencyObserverResponse struct {
	User struct {
		ID     uuid.UUID `json:"id"`
		Role   string    `json:"role"`
		Status string    `json:"status"`
	} `json:"user"`
	Observer models.AgencyObserver `json:"observer"`
}

type agencyObserverListItem struct {
	ID       uuid.UUID `json:"id"`
	UserID   uuid.UUID `json:"userId"`
	TenantID uuid.UUID `json:"tenantId"`
	Rank     string    `json:"rank"`
}

type agencyObserverListResponse struct {
	Observers []agencyObserverListItem `json:"observers"`
}

func agencyJSONRequest(t *testing.T, server http.Handler, method, path, token string, payload any) *httptest.ResponseRecorder {
	t.Helper()

	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal request: %v", err)
		}
		body = bytes.NewReader(encoded)
	}

	req := httptest.NewRequest(method, path, body)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", token)
	}

	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)
	return w
}

func requireAgencyStatus(t *testing.T, w *httptest.ResponseRecorder, want int) {
	t.Helper()

	if w.Code != want {
		t.Fatalf("status = %d, want %d. body=%s", w.Code, want, w.Body.String())
	}
}

func createAgencyTenantForTest(t *testing.T, server http.Handler, token, agencyType, divisionName string) models.AgencyTenant {
	t.Helper()

	w := agencyJSONRequest(t, server, http.MethodPost, "/api/v1/super/agency-tenants", token, map[string]any{
		"agencyType":   agencyType,
		"state":        "Lagos",
		"lga":          "Ikeja",
		"divisionName": divisionName,
	})
	requireAgencyStatus(t, w, http.StatusCreated)

	var response agencyTenantResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode tenant response: %v", err)
	}
	if response.Tenant.ID == uuid.Nil {
		t.Fatalf("created tenant has no ID: %s", w.Body.String())
	}
	return response.Tenant
}

func assignAgencyDivisionAdminForTest(t *testing.T, server http.Handler, token string, tenantID, userID uuid.UUID, rank string) *httptest.ResponseRecorder {
	t.Helper()

	payload := map[string]any{"userId": userID.String()}
	if rank != "" {
		payload["rank"] = rank
	}
	return agencyJSONRequest(
		t,
		server,
		http.MethodPost,
		"/api/v1/super/agency-tenants/"+tenantID.String()+"/admin",
		token,
		payload,
	)
}

func observerPayload(email, phone string) map[string]any {
	return map[string]any{
		"email":     email,
		"phone":     phone,
		"firstName": "Agency",
		"lastName":  "Observer",
		"password":  "observer-password-123",
	}
}

func TestPublicRegistrationRejectsAgencyRoles(t *testing.T) {
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	t.Setenv("TERMS_CONSENT_ENFORCED", "false")
	server := FreshServer(t)

	for i, role := range []string{
		models.RoleAgencyObserver,
		models.RoleAgencyDivisionAdmin,
		models.RoleSuperAdmin,
	} {
		email := "blocked-role-" + role + "@example.test"
		w := agencyJSONRequest(t, server, http.MethodPost, "/api/v1/auth/register", "", map[string]any{
			"email":       email,
			"phone":       "0801000000" + string(rune('0'+i)),
			"firstName":   "Blocked",
			"lastName":    "Role",
			"password":    "hunter2hunter2",
			"role":        role,
			"dateOfBirth": "1990-01-01",
		})
		requireAgencyStatus(t, w, http.StatusBadRequest)
		if !strings.Contains(w.Body.String(), "citizen") {
			t.Fatalf("role rejection did not explain citizen-only registration: %s", w.Body.String())
		}

		var count int64
		if err := testutil.GetDB().Model(&models.User{}).
			Where("LOWER(email) = ?", strings.ToLower(email)).
			Count(&count).Error; err != nil {
			t.Fatalf("count rejected users: %v", err)
		}
		if count != 0 {
			t.Fatalf("rejected role %q still created %d user(s)", role, count)
		}
	}

	email := "citizen-role@example.test"
	w := agencyJSONRequest(t, server, http.MethodPost, "/api/v1/auth/register", "", map[string]any{
		"email":       email,
		"phone":       "08019999999",
		"firstName":   "Citizen",
		"lastName":    "Role",
		"password":    "hunter2hunter2",
		"role":        models.RoleCitizen,
		"dateOfBirth": "1990-01-01",
	})
	requireAgencyStatus(t, w, http.StatusCreated)

	var created models.User
	if err := testutil.GetDB().First(&created, "LOWER(email) = ?", strings.ToLower(email)).Error; err != nil {
		t.Fatalf("reload citizen: %v", err)
	}
	if created.Role != models.RoleCitizen {
		t.Fatalf("created role = %q, want %q", created.Role, models.RoleCitizen)
	}
}

func TestAgencyTenancyEndpoints(t *testing.T) {
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	server := FreshServer(t)
	superUser := testutil.MakeUser(t, models.RoleSuperAdmin)
	superToken := testutil.AuthHeader(t, superUser)
	citizen := testutil.MakeUser(t, models.RoleCitizen)
	citizenToken := testutil.AuthHeader(t, citizen)

	t.Run("requires authentication", func(t *testing.T) {
		w := agencyJSONRequest(t, server, http.MethodGet, "/api/v1/agency/observers", "", nil)
		requireAgencyStatus(t, w, http.StatusUnauthorized)

		w = agencyJSONRequest(t, server, http.MethodPost, "/api/v1/super/agency-tenants", "", map[string]any{
			"agencyType":   "police",
			"state":        "Lagos",
			"lga":          "Ikeja",
			"divisionName": "Ikeja Police Division",
		})
		requireAgencyStatus(t, w, http.StatusUnauthorized)
	})

	t.Run("rejects non-super-admin tenant creation", func(t *testing.T) {
		w := agencyJSONRequest(t, server, http.MethodPost, "/api/v1/super/agency-tenants", citizenToken, map[string]any{
			"agencyType":   "police",
			"state":        "Lagos",
			"lga":          "Ikeja",
			"divisionName": "Ikeja Police Division",
		})
		requireAgencyStatus(t, w, http.StatusForbidden)
	})

	t.Run("rejects invalid agency type", func(t *testing.T) {
		w := agencyJSONRequest(t, server, http.MethodPost, "/api/v1/super/agency-tenants", superToken, map[string]any{
			"agencyType":   "army",
			"state":        "Lagos",
			"lga":          "Ikeja",
			"divisionName": "Ikeja Army Division",
		})
		requireAgencyStatus(t, w, http.StatusBadRequest)
	})

	var tenantA models.AgencyTenant
	t.Run("creates agency tenant", func(t *testing.T) {
		tenantA = createAgencyTenantForTest(t, server, superToken, "police", "Ikeja Police Division")
		if tenantA.AgencyType != "police" || tenantA.State != "Lagos" || tenantA.LGA != "Ikeja" {
			t.Fatalf("unexpected tenant: %+v", tenantA)
		}
	})

	t.Run("assigns division admin and creates observer record", func(t *testing.T) {
		w := assignAgencyDivisionAdminForTest(t, server, superToken, tenantA.ID, citizen.ID, "")
		requireAgencyStatus(t, w, http.StatusOK)

		var admin models.User
		if err := testutil.GetDB().First(&admin, "id = ?", citizen.ID).Error; err != nil {
			t.Fatalf("reload assigned admin: %v", err)
		}
		if admin.Role != models.RoleAgencyDivisionAdmin {
			t.Fatalf("role = %q, want %q", admin.Role, models.RoleAgencyDivisionAdmin)
		}

		var observer models.AgencyObserver
		if err := testutil.GetDB().First(&observer, "user_id = ?", citizen.ID).Error; err != nil {
			t.Fatalf("load admin observer record: %v", err)
		}
		if observer.TenantID != tenantA.ID {
			t.Fatalf("observer tenant = %s, want %s", observer.TenantID, tenantA.ID)
		}
		if observer.Rank != "Division Admin" {
			t.Fatalf("observer rank = %q, want Division Admin", observer.Rank)
		}

		citizen.Role = admin.Role
		citizenToken = testutil.AuthHeader(t, citizen)
	})

	t.Run("rejects assigning a super admin", func(t *testing.T) {
		w := assignAgencyDivisionAdminForTest(t, server, superToken, tenantA.ID, superUser.ID, "")
		requireAgencyStatus(t, w, http.StatusBadRequest)
	})

	t.Run("rejects missing tenant and user", func(t *testing.T) {
		w := assignAgencyDivisionAdminForTest(t, server, superToken, uuid.New(), citizen.ID, "")
		requireAgencyStatus(t, w, http.StatusNotFound)

		w = assignAgencyDivisionAdminForTest(t, server, superToken, tenantA.ID, uuid.New(), "")
		requireAgencyStatus(t, w, http.StatusNotFound)
	})

	var tenantB models.AgencyTenant
	t.Run("creates second tenant", func(t *testing.T) {
		tenantB = createAgencyTenantForTest(t, server, superToken, "fire", "Ikeja Fire Division")
	})

	t.Run("rejects cross-tenant reassignment", func(t *testing.T) {
		w := assignAgencyDivisionAdminForTest(t, server, superToken, tenantB.ID, citizen.ID, "")
		requireAgencyStatus(t, w, http.StatusConflict)

		var observer models.AgencyObserver
		if err := testutil.GetDB().First(&observer, "user_id = ?", citizen.ID).Error; err != nil {
			t.Fatalf("reload observer after conflict: %v", err)
		}
		if observer.TenantID != tenantA.ID {
			t.Fatalf("observer tenant changed to %s, want %s", observer.TenantID, tenantA.ID)
		}
	})

	t.Run("rejects division admin without tenant record", func(t *testing.T) {
		unassignedAdmin := testutil.MakeUser(t, models.RoleAgencyDivisionAdmin)
		w := agencyJSONRequest(
			t,
			server,
			http.MethodGet,
			"/api/v1/agency/observers",
			testutil.AuthHeader(t, unassignedAdmin),
			nil,
		)
		requireAgencyStatus(t, w, http.StatusForbidden)
	})

	t.Run("rejects non-admin observer creation and listing", func(t *testing.T) {
		otherCitizen := testutil.MakeUser(t, models.RoleCitizen)
		otherToken := testutil.AuthHeader(t, otherCitizen)

		w := agencyJSONRequest(
			t,
			server,
			http.MethodPost,
			"/api/v1/agency/observers",
			otherToken,
			observerPayload("blocked-observer@example.test", "08020000001"),
		)
		requireAgencyStatus(t, w, http.StatusForbidden)

		w = agencyJSONRequest(t, server, http.MethodGet, "/api/v1/agency/observers", otherToken, nil)
		requireAgencyStatus(t, w, http.StatusForbidden)
	})

	t.Run("rejects observer creation for another tenant", func(t *testing.T) {
		payload := observerPayload("mismatch-observer@example.test", "08020000002")
		payload["tenantId"] = tenantB.ID.String()

		w := agencyJSONRequest(t, server, http.MethodPost, "/api/v1/agency/observers", citizenToken, payload)
		requireAgencyStatus(t, w, http.StatusForbidden)

		var count int64
		if err := testutil.GetDB().Model(&models.User{}).
			Where("email = ?", "mismatch-observer@example.test").
			Count(&count).Error; err != nil {
			t.Fatalf("count mismatched observer: %v", err)
		}
		if count != 0 {
			t.Fatalf("mismatched tenant created %d observer account(s)", count)
		}
	})

	var observerA models.AgencyObserver
	t.Run("creates observer in the admin tenant", func(t *testing.T) {
		w := agencyJSONRequest(
			t,
			server,
			http.MethodPost,
			"/api/v1/agency/observers",
			citizenToken,
			observerPayload("observer-a@example.test", "08020000003"),
		)
		requireAgencyStatus(t, w, http.StatusCreated)

		var response agencyObserverResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("decode observer response: %v", err)
		}
		if response.Observer.TenantID != tenantA.ID {
			t.Fatalf("observer tenant = %s, want %s", response.Observer.TenantID, tenantA.ID)
		}
		if response.Observer.Rank != "Observer" {
			t.Fatalf("observer rank = %q, want Observer", response.Observer.Rank)
		}
		if response.User.Role != models.RoleAgencyObserver {
			t.Fatalf("user role = %q, want %q", response.User.Role, models.RoleAgencyObserver)
		}
		if response.User.Status != "active" {
			t.Fatalf("user status = %q, want active", response.User.Status)
		}
		observerA = response.Observer

		var stored models.User
		if err := testutil.GetDB().First(&stored, "id = ?", response.User.ID).Error; err != nil {
			t.Fatalf("reload observer user: %v", err)
		}
		if stored.Password == "observer-password-123" {
			t.Fatal("observer password was stored in plaintext")
		}
	})

	t.Run("rejects duplicate observer email or phone", func(t *testing.T) {
		payload := observerPayload("observer-a@example.test", "08020000099")
		w := agencyJSONRequest(t, server, http.MethodPost, "/api/v1/agency/observers", citizenToken, payload)
		requireAgencyStatus(t, w, http.StatusConflict)
	})

	adminBUser := testutil.MakeUser(t, models.RoleCitizen)
	var adminBToken string
	t.Run("assigns second tenant admin", func(t *testing.T) {
		w := assignAgencyDivisionAdminForTest(t, server, superToken, tenantB.ID, adminBUser.ID, "")
		requireAgencyStatus(t, w, http.StatusOK)

		var adminB models.User
		if err := testutil.GetDB().First(&adminB, "id = ?", adminBUser.ID).Error; err != nil {
			t.Fatalf("reload tenant B admin: %v", err)
		}
		if adminB.Role != models.RoleAgencyDivisionAdmin {
			t.Fatalf("tenant B admin role = %q, want %q", adminB.Role, models.RoleAgencyDivisionAdmin)
		}
		adminBToken = testutil.AuthHeader(t, &adminB)
	})

	var observerB models.AgencyObserver
	t.Run("creates observer in the second tenant", func(t *testing.T) {
		w := agencyJSONRequest(
			t,
			server,
			http.MethodPost,
			"/api/v1/agency/observers",
			adminBToken,
			observerPayload("observer-b@example.test", "08020000004"),
		)
		requireAgencyStatus(t, w, http.StatusCreated)

		var response agencyObserverResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("decode observer B response: %v", err)
		}
		if response.Observer.TenantID != tenantB.ID {
			t.Fatalf("observer B tenant = %s, want %s", response.Observer.TenantID, tenantB.ID)
		}
		observerB = response.Observer
	})

	t.Run("lists only observers in the admin tenant", func(t *testing.T) {
		w := agencyJSONRequest(t, server, http.MethodGet, "/api/v1/agency/observers", citizenToken, nil)
		requireAgencyStatus(t, w, http.StatusOK)
		if bytes.Contains(w.Body.Bytes(), []byte(`"password"`)) {
			t.Fatalf("observer list leaked password field: %s", w.Body.String())
		}

		var response agencyObserverListResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("decode observer list: %v", err)
		}
		if len(response.Observers) != 2 {
			t.Fatalf("tenant A observer count = %d, want 2. body=%s", len(response.Observers), w.Body.String())
		}
		for _, observer := range response.Observers {
			if observer.TenantID != tenantA.ID {
				t.Fatalf("tenant A list leaked observer from tenant %s", observer.TenantID)
			}
			if observer.ID == observerB.ID {
				t.Fatal("tenant A list included tenant B observer")
			}
		}

		w = agencyJSONRequest(t, server, http.MethodGet, "/api/v1/agency/observers", adminBToken, nil)
		requireAgencyStatus(t, w, http.StatusOK)

		response = agencyObserverListResponse{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("decode tenant B observer list: %v", err)
		}
		if len(response.Observers) != 2 {
			t.Fatalf("tenant B observer count = %d, want 2. body=%s", len(response.Observers), w.Body.String())
		}
		for _, observer := range response.Observers {
			if observer.TenantID != tenantB.ID {
				t.Fatalf("tenant B list leaked observer from tenant %s", observer.TenantID)
			}
			if observer.ID == observerA.ID {
				t.Fatal("tenant B list included tenant A observer")
			}
		}
	})

	t.Run("lists agency tenants with assigned admin", func(t *testing.T) {
		w := agencyJSONRequest(t, server, http.MethodGet, "/api/v1/super/agency-tenants", citizenToken, nil)
		requireAgencyStatus(t, w, http.StatusForbidden)

		w = agencyJSONRequest(t, server, http.MethodGet, "/api/v1/super/agency-tenants", superToken, nil)
		requireAgencyStatus(t, w, http.StatusOK)

		var res struct {
			Tenants []struct {
				ID           uuid.UUID `json:"id"`
				AgencyType   string    `json:"agencyType"`
				DivisionName string    `json:"divisionName"`
				Admin        *struct {
					UserID uuid.UUID `json:"userId"`
					Email  string    `json:"email"`
					Rank   string    `json:"rank"`
				} `json:"admin"`
			} `json:"tenants"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
			t.Fatalf("decode tenants: %v", err)
		}
		if len(res.Tenants) < 2 {
			t.Fatalf("expected at least 2 tenants, got %d", len(res.Tenants))
		}
		var foundA bool
		for _, tenant := range res.Tenants {
			if tenant.ID == tenantA.ID {
				foundA = true
				if tenant.Admin == nil {
					t.Fatal("tenant A missing admin")
				}
				if tenant.Admin.UserID != citizen.ID {
					t.Fatalf("tenant A admin userId = %s, want %s", tenant.Admin.UserID, citizen.ID)
				}
			}
		}
		if !foundA {
			t.Fatal("tenant A not found in listing")
		}
	})

	t.Run("assigns division admin by email", func(t *testing.T) {
		newCitizen := testutil.MakeUser(t, models.RoleCitizen)
		tenantC := createAgencyTenantForTest(t, server, superToken, "nscdc", "Alausa NSCDC Division")

		w := agencyJSONRequest(
			t,
			server,
			http.MethodPost,
			"/api/v1/super/agency-tenants/"+tenantC.ID.String()+"/admin",
			superToken,
			map[string]any{
				"email": newCitizen.Email,
				"rank":  "Commander",
			},
		)
		requireAgencyStatus(t, w, http.StatusOK)

		var reloaded models.User
		if err := testutil.GetDB().First(&reloaded, "id = ?", newCitizen.ID).Error; err != nil {
			t.Fatalf("reload user: %v", err)
		}
		if reloaded.Role != models.RoleAgencyDivisionAdmin {
			t.Fatalf("reloaded role = %s, want %s", reloaded.Role, models.RoleAgencyDivisionAdmin)
		}
	})
}
