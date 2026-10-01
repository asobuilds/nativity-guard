package handlers

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"security-solution/config"
	"security-solution/content"
	"security-solution/models"
)

func setupConsentRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/register", NewAuthHandler().Register)
	return r
}

func postRegister(r *gin.Engine, body map[string]any) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/register", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "consent-test/1.0")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func validBody(email string) map[string]any {
	short := strings.SplitN(email, "@", 2)[0]
	if len(short) > 4 {
		short = short[:4]
	}
	return map[string]any{
		"email":                 email,
		"phone":                 "+2348000000" + short,
		"firstName":             "Test",
		"lastName":              "User",
		"password":              "hunter2hunter2",
		"dateOfBirth":           "1990-01-01",
		"termsVersion":          content.TermsVersion,
		"consentTerms":          true,
		"consentDataProcessing": true,
	}
}

func TestRegister_RejectsMissingTermsConsent(t *testing.T) {
	r := setupConsentRouter()
	b := validBody("m1@x.com")
	delete(b, "consentTerms")
	w := postRegister(r, b)
	if w.Code != 400 || !strings.Contains(w.Body.String(), "terms_consent_required") {
		t.Fatalf("want 400 terms_consent_required, got %d %s", w.Code, w.Body.String())
	}
}

func TestRegister_RejectsFalseTermsConsent(t *testing.T) {
	r := setupConsentRouter()
	b := validBody("m2@x.com")
	b["consentTerms"] = false
	w := postRegister(r, b)
	if w.Code != 400 || !strings.Contains(w.Body.String(), "terms_consent_required") {
		t.Fatalf("got %d %s", w.Code, w.Body.String())
	}
}

func TestRegister_RejectsMissingDataProcessingConsent(t *testing.T) {
	r := setupConsentRouter()
	b := validBody("m3@x.com")
	delete(b, "consentDataProcessing")
	w := postRegister(r, b)
	if w.Code != 400 || !strings.Contains(w.Body.String(), "data_processing_consent_required") {
		t.Fatalf("got %d %s", w.Code, w.Body.String())
	}
}

func TestRegister_RejectsFalseDataProcessingConsent(t *testing.T) {
	r := setupConsentRouter()
	b := validBody("m4@x.com")
	b["consentDataProcessing"] = false
	w := postRegister(r, b)
	if w.Code != 400 || !strings.Contains(w.Body.String(), "data_processing_consent_required") {
		t.Fatalf("got %d %s", w.Code, w.Body.String())
	}
}

func TestRegister_RejectsMissingVersion(t *testing.T) {
	r := setupConsentRouter()
	b := validBody("m5@x.com")
	b["termsVersion"] = ""
	w := postRegister(r, b)
	if w.Code != 400 || !strings.Contains(w.Body.String(), "terms_version_required") {
		t.Fatalf("got %d %s", w.Code, w.Body.String())
	}
}

func TestRegister_RejectsStaleVersion(t *testing.T) {
	r := setupConsentRouter()
	b := validBody("m6@x.com")
	b["termsVersion"] = "1999-01-01-v0"
	w := postRegister(r, b)
	if w.Code != 409 || !strings.Contains(w.Body.String(), "terms_version_stale") {
		t.Fatalf("got %d %s", w.Code, w.Body.String())
	}
}

func TestRegister_LegacyServicePath_NoAcceptanceRows(t *testing.T) {
	if config.DB == nil {
		t.Skip("no DB configured in this test environment")
	}
	u := &models.User{
		Email:     "legacy@x.com",
		Phone:     "+2349000000001",
		FirstName: "L",
		LastName:  "U",
		Password:  "hunter2hunter2",
	}
	created, err := NewAuthHandler().authService.Register(u)
	if err != nil {
		t.Fatalf("legacy register failed: %v", err)
	}
	var n int64
	config.DB.Model(&models.TermsAcceptance{}).Where("user_id = ?", created.ID).Count(&n)
	if n != 0 {
		t.Fatalf("legacy path should create 0 acceptances, got %d", n)
	}
	config.DB.Unscoped().Delete(&created)
}

func TestClientIPForConsent_UsesFirstXFFHop(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/", nil)
	c.Request.Header.Set("X-Forwarded-For", "203.0.113.7, 10.0.0.1")
	if got := clientIPForConsent(c); got != "203.0.113.7" {
		t.Fatalf("want 203.0.113.7, got %q", got)
	}
}