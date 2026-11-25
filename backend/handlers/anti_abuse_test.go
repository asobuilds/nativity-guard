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

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"security-solution/config"
	"security-solution/internal/testutil"
	"security-solution/middleware"
	"security-solution/models"
	"security-solution/services"
)

func antiAbuseRequest(t *testing.T, router *gin.Engine, user *models.User, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(method, path, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	if user != nil {
		req.Header.Set("Authorization", testutil.AuthHeader(t, user))
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestAntiAbuseRateLimits(t *testing.T) {
	testutil.TruncateAll(t)
	middleware.ResetRateLimitersForTests()
	router := FreshServer(t)
	user := testutil.MakeUser(t, "citizen")

	t.Run("sos limited to 3 per hour", func(t *testing.T) {
		middleware.ResetRateLimitersForTests()
		for i := 0; i < 3; i++ {
			w := antiAbuseRequest(t, router, user, "POST", "/api/v1/sos/send", map[string]any{})
			if w.Code == http.StatusTooManyRequests {
				t.Fatalf("request %d unexpectedly rate limited", i+1)
			}
		}
		w := antiAbuseRequest(t, router, user, "POST", "/api/v1/sos/send", map[string]any{})
		if w.Code != http.StatusTooManyRequests {
			t.Fatalf("4th SOS status %d want 429; %s", w.Code, w.Body.String())
		}
	})

	t.Run("cases limited to 5 per hour", func(t *testing.T) {
		middleware.ResetRateLimitersForTests()
		for i := 0; i < 5; i++ {
			w := antiAbuseRequest(t, router, user, "POST", "/api/v1/cases", map[string]any{})
			if w.Code == http.StatusTooManyRequests {
				t.Fatalf("request %d unexpectedly rate limited", i+1)
			}
		}
		w := antiAbuseRequest(t, router, user, "POST", "/api/v1/cases", map[string]any{})
		if w.Code != http.StatusTooManyRequests {
			t.Fatalf("6th case status %d want 429; %s", w.Code, w.Body.String())
		}
	})

	t.Run("community posts limited to 10 per hour", func(t *testing.T) {
		middleware.ResetRateLimitersForTests()
		for i := 0; i < 10; i++ {
			w := antiAbuseRequest(t, router, user, "POST", "/api/v1/community/posts", map[string]any{})
			if w.Code == http.StatusTooManyRequests {
				t.Fatalf("request %d unexpectedly rate limited", i+1)
			}
		}
		w := antiAbuseRequest(t, router, user, "POST", "/api/v1/community/posts", map[string]any{})
		if w.Code != http.StatusTooManyRequests {
			t.Fatalf("11th post status %d want 429; %s", w.Code, w.Body.String())
		}
	})
}

func TestFalseAlertTrustAndSOSBan(t *testing.T) {
	testutil.TruncateAll(t)
	middleware.ResetRateLimitersForTests()
	admin := testutil.MakeUser(t, "unit_admin")
	unit := testutil.MakeUnit(t, admin)
	admin.UnitID = &unit.ID
	if err := config.DB.Save(admin).Error; err != nil {
		t.Fatal(err)
	}
	reporter := testutil.MakeUser(t, "citizen")
	router := FreshServer(t)

	for i := 0; i < 4; i++ {
		sos := models.SOSAlert{
			UserID:        reporter.ID,
			UnitID:        &unit.ID,
			Latitude:      7.2,
			Longitude:     8.1,
			Status:        "pending",
			Priority:      "high",
			DispatchState: "open",
		}
		if err := config.DB.Create(&sos).Error; err != nil {
			t.Fatal(err)
		}
		w := antiAbuseRequest(t, router, admin, "POST", "/api/v1/sos/"+sos.ID.String()+"/false", map[string]any{})
		if w.Code != http.StatusOK {
			t.Fatalf("mark false %d: status %d; %s", i+1, w.Code, w.Body.String())
		}
		var out struct {
			TrustScore int  `json:"trustScore"`
			SOSBanned  bool `json:"sosBanned"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		wantTrust := 100 - 20*(i+1)
		if out.TrustScore != wantTrust {
			t.Fatalf("false alert %d trust %d want %d", i+1, out.TrustScore, wantTrust)
		}
		if i < 3 && out.SOSBanned {
			t.Fatalf("false alert %d banned too early (trust %d)", i+1, out.TrustScore)
		}
		if i == 3 && !out.SOSBanned {
			t.Fatalf("4th false alert did not trigger SOS ban (trust %d)", out.TrustScore)
		}
	}

	var fresh models.User
	if err := config.DB.First(&fresh, "id = ?", reporter.ID).Error; err != nil {
		t.Fatal(err)
	}
	if fresh.TrustScore != 20 {
		t.Fatalf("trust score %d want 20", fresh.TrustScore)
	}
	if fresh.SOSBannedUntil == nil || !fresh.SOSBannedUntil.After(time.Now()) {
		t.Fatalf("expected active SOS ban, got %v", fresh.SOSBannedUntil)
	}

	w := antiAbuseRequest(t, router, reporter, "POST", "/api/v1/sos/send", map[string]any{
		"latitude": 7.2, "longitude": 8.1, "description": "still need help",
	})
	if w.Code != http.StatusForbidden {
		t.Fatalf("banned SOS status %d want 403; %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "temporarily suspended") {
		t.Fatalf("unexpected ban response: %s", w.Body.String())
	}
}

func TestUnitAdminRestriction(t *testing.T) {
	testutil.TruncateAll(t)
	middleware.ResetRateLimitersForTests()
	admin := testutil.MakeUser(t, "unit_admin")
	unit := testutil.MakeUnit(t, admin)
	unit.State = "Lagos"
	unit.LGA = "Ikeja"
	if err := config.DB.Save(unit).Error; err != nil {
		t.Fatal(err)
	}
	admin.UnitID = &unit.ID
	if err := config.DB.Save(admin).Error; err != nil {
		t.Fatal(err)
	}

	otherAdmin := testutil.MakeUser(t, "unit_admin")
	otherUnit := testutil.MakeUnit(t, otherAdmin)
	otherUnit.State = "Oyo"
	otherUnit.LGA = "Ibadan"
	if err := config.DB.Save(otherUnit).Error; err != nil {
		t.Fatal(err)
	}
	otherAdmin.UnitID = &otherUnit.ID
	if err := config.DB.Save(otherAdmin).Error; err != nil {
		t.Fatal(err)
	}

	target := testutil.MakeUser(t, "citizen")
	target.UnitID = &unit.ID
	if err := config.DB.Save(target).Error; err != nil {
		t.Fatal(err)
	}
	testutil.MakeMembership(t, target, unit, "member", false)

	router := FreshServer(t)
	restrictPath := "/api/v1/admin/users/" + target.ID.String() + "/restrict"

	w := antiAbuseRequest(t, router, otherAdmin, "POST", restrictPath, map[string]any{"durationHours": 24})
	if w.Code != http.StatusForbidden {
		t.Fatalf("out-of-LGA restrict status %d want 403; %s", w.Code, w.Body.String())
	}

	w = antiAbuseRequest(t, router, admin, "POST", restrictPath, map[string]any{"durationHours": 24})
	if w.Code != http.StatusOK {
		t.Fatalf("unit admin restrict status %d want 200; %s", w.Code, w.Body.String())
	}

	var fresh models.User
	if err := config.DB.First(&fresh, "id = ?", target.ID).Error; err != nil {
		t.Fatal(err)
	}
	if fresh.RestrictedUntil == nil || !fresh.RestrictedUntil.After(time.Now()) {
		t.Fatalf("expected active restriction, got %v", fresh.RestrictedUntil)
	}
	if fresh.RestrictedState != "Lagos" || fresh.RestrictedLGA != "Ikeja" {
		t.Fatalf("restriction place = %q/%q want Lagos/Ikeja", fresh.RestrictedState, fresh.RestrictedLGA)
	}

	w = antiAbuseRequest(t, router, target, "POST", "/api/v1/community/posts", map[string]any{
		"title": "hello", "content": "world", "scope": "lga",
	})
	if w.Code != http.StatusForbidden {
		t.Fatalf("restricted post status %d want 403; %s", w.Code, w.Body.String())
	}

	past := time.Now().UTC().Add(-time.Hour)
	if err := config.DB.Model(&models.User{}).Where("id = ?", target.ID).Update("restricted_until", past).Error; err != nil {
		t.Fatal(err)
	}
	w = antiAbuseRequest(t, router, target, "POST", "/api/v1/community/posts", map[string]any{
		"title": "hello", "content": "world", "scope": "lga",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expired restriction post status %d want 201; %s", w.Code, w.Body.String())
	}
}

func TestSuperAdminBanAndTextValidation(t *testing.T) {
	testutil.TruncateAll(t)
	middleware.ResetRateLimitersForTests()
	super := testutil.MakeUser(t, "super_admin")
	super.IsSuperAdmin = true
	if err := config.DB.Save(super).Error; err != nil {
		t.Fatal(err)
	}
	unitAdmin := testutil.MakeUser(t, "unit_admin")
	target := testutil.MakeUser(t, "citizen")
	router := FreshServer(t)
	banPath := "/api/v1/admin/users/" + target.ID.String() + "/ban"

	w := antiAbuseRequest(t, router, unitAdmin, "POST", banPath, map[string]any{})
	if w.Code != http.StatusForbidden {
		t.Fatalf("non-super ban status %d want 403; %s", w.Code, w.Body.String())
	}

	w = antiAbuseRequest(t, router, super, "POST", banPath, map[string]any{})
	if w.Code != http.StatusOK {
		t.Fatalf("super ban status %d want 200; %s", w.Code, w.Body.String())
	}
	var banned models.User
	if err := config.DB.First(&banned, "id = ?", target.ID).Error; err != nil {
		t.Fatal(err)
	}
	if banned.Status != "banned" {
		t.Fatalf("status %q want banned", banned.Status)
	}

	if _, _, _, err := services.NewAuthService().LoginWithJTI(target.Email, "test-password-123"); err == nil {
		t.Fatal("expected banned user login to fail")
	}

	// A token minted after the ban must still be rejected by AuthMiddleware,
	// which reloads the user and checks status.
	w = antiAbuseRequest(t, router, target, "GET", "/api/v1/auth/profile", nil)
	if w.Code != http.StatusForbidden {
		t.Fatalf("banned token access status %d want 403; %s", w.Code, w.Body.String())
	}

	t.Run("case text validation", func(t *testing.T) {
		reporter := testutil.MakeUser(t, "citizen")
		body := map[string]any{
			"unitId": uuid.NewString(), "title": "x' OR 1=1 --", "description": "ok",
			"latitude": 7.2, "longitude": 8.1, "location": "Ikeja",
		}
		w := antiAbuseRequest(t, router, reporter, "POST", "/api/v1/cases", body)
		if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "disallowed") {
			t.Fatalf("sqli case status %d; %s", w.Code, w.Body.String())
		}

		body["title"] = strings.Repeat("a", 2001)
		w = antiAbuseRequest(t, router, reporter, "POST", "/api/v1/cases", body)
		if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "2000") {
			t.Fatalf("oversize case status %d; %s", w.Code, w.Body.String())
		}
	})

	t.Run("sos text validation", func(t *testing.T) {
		reporter := testutil.MakeUser(t, "citizen")
		w := antiAbuseRequest(t, router, reporter, "POST", "/api/v1/sos/send", map[string]any{
			"latitude": 7.2, "longitude": 8.1,
			"description": "x UNION SELECT password FROM users",
		})
		if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "disallowed") {
			t.Fatalf("sqli sos status %d; %s", w.Code, w.Body.String())
		}
	})
}
