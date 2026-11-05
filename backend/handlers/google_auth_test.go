package handlers

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"security-solution/config"
	"security-solution/internal/testutil"
	"security-solution/models"
	"security-solution/services"
)

func googleMockToken(t *testing.T, claims map[string]any) string {
	t.Helper()
	raw, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("marshal mock claims: %v", err)
	}
	return "mock:" + base64.RawURLEncoding.EncodeToString(raw)
}

func validGoogleMockToken(t *testing.T, subject, email string) string {
	t.Helper()
	return googleMockToken(t, map[string]any{
		"sub":            subject,
		"email":          email,
		"email_verified": true,
		"given_name":     "Ada",
		"family_name":    "Okafor",
	})
}

func TestGoogleAuthMockTokenRequiresExplicitDevelopmentFlag(t *testing.T) {
	t.Setenv("GOOGLE_AUTH_MOCK_ENABLED", "false")
	t.Setenv("APP_ENV", "development")

	_, err := services.NewAuthService().VerifyGoogleIDToken(
		context.Background(),
		validGoogleMockToken(t, "google-1", "ada@example.com"),
	)
	if err == nil || !strings.Contains(err.Error(), "invalid Google ID token") {
		t.Fatalf("expected disabled mock token rejection, got %v", err)
	}
}

func TestGoogleAuthMockTokenIsNeverAcceptedInProduction(t *testing.T) {
	t.Setenv("GOOGLE_AUTH_MOCK_ENABLED", "true")
	t.Setenv("APP_ENV", "production")

	_, err := services.NewAuthService().VerifyGoogleIDToken(
		context.Background(),
		validGoogleMockToken(t, "google-2", "ada@example.com"),
	)
	if err == nil {
		t.Fatal("expected production mock token rejection")
	}
}

func TestGoogleAuthMockTokenReturnsVerifiedIdentity(t *testing.T) {
	t.Setenv("GOOGLE_AUTH_MOCK_ENABLED", "true")
	t.Setenv("APP_ENV", "development")

	identity, err := services.NewAuthService().VerifyGoogleIDToken(
		context.Background(),
		validGoogleMockToken(t, "google-3", "ADA@Example.COM"),
	)
	if err != nil {
		t.Fatalf("verify mock token: %v", err)
	}
	if identity.Subject != "google-3" || identity.Email != "ada@example.com" {
		t.Fatalf("unexpected identity: %#v", identity)
	}
	if identity.FirstName != "Ada" || identity.LastName != "Okafor" {
		t.Fatalf("unexpected profile names: %#v", identity)
	}
}

func TestGoogleAuthRejectsUnverifiedGoogleEmail(t *testing.T) {
	t.Setenv("GOOGLE_AUTH_MOCK_ENABLED", "true")
	t.Setenv("APP_ENV", "test")
	token := googleMockToken(t, map[string]any{
		"sub":            "google-4",
		"email":          "ada@example.com",
		"email_verified": false,
	})

	_, err := services.NewAuthService().VerifyGoogleIDToken(context.Background(), token)
	if err == nil || !strings.Contains(err.Error(), "not verified") {
		t.Fatalf("expected unverified-email rejection, got %v", err)
	}
}

func TestGoogleAuthVerifiesLiveTokenClaims(t *testing.T) {
	var receivedToken string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedToken = r.URL.Query().Get("id_token")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"aud":            "nativity-web-client",
			"iss":            "https://accounts.google.com",
			"exp":            time.Now().Add(time.Hour).Unix(),
			"sub":            "google-live-1",
			"email":          "verified@example.com",
			"email_verified": "true",
			"name":           "Chidi Eze",
		})
	}))
	defer server.Close()

	t.Setenv("GOOGLE_CLIENT_ID", "nativity-web-client")
	t.Setenv("GOOGLE_TOKENINFO_URL", server.URL)
	t.Setenv("APP_ENV", "test")
	identity, err := services.NewAuthService().VerifyGoogleIDToken(context.Background(), "signed-google-token")
	if err != nil {
		t.Fatalf("verify live token claims: %v", err)
	}
	if receivedToken != "signed-google-token" {
		t.Fatalf("token-info request received %q", receivedToken)
	}
	if identity.Subject != "google-live-1" || identity.FirstName != "Chidi" || identity.LastName != "Eze" {
		t.Fatalf("unexpected identity: %#v", identity)
	}
}

func TestGoogleAuthRejectsWrongAudience(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"aud":            "other-client",
			"iss":            "accounts.google.com",
			"exp":            time.Now().Add(time.Hour).Unix(),
			"sub":            "google-live-2",
			"email":          "verified@example.com",
			"email_verified": true,
		})
	}))
	defer server.Close()

	t.Setenv("GOOGLE_CLIENT_ID", "nativity-web-client")
	t.Setenv("GOOGLE_TOKENINFO_URL", server.URL)
	t.Setenv("APP_ENV", "test")
	_, err := services.NewAuthService().VerifyGoogleIDToken(context.Background(), "signed-google-token")
	if err == nil || !strings.Contains(err.Error(), "audience") {
		t.Fatalf("expected audience rejection, got %v", err)
	}
}

func TestGoogleAuthEndpointRequiresIDToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/google", NewAuthHandler().GoogleAuth)

	req := httptest.NewRequest(http.MethodPost, "/google", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "idToken") {
		t.Fatalf("expected missing-token 400, got %d %s", response.Code, response.Body.String())
	}
}

func TestGoogleAuthAcceptsOnlyNigerianProfilePhones(t *testing.T) {
	valid := []string{"08031234567", "+2348031234567", "2348031234567"}
	for _, phone := range valid {
		if !nigerianPhonePattern.MatchString(phone) {
			t.Errorf("expected %q to be valid", phone)
		}
	}
	invalid := []string{"", "8031234567", "+2338031234567", "0803123456", "080312345678"}
	for _, phone := range invalid {
		if nigerianPhonePattern.MatchString(phone) {
			t.Errorf("expected %q to be invalid", phone)
		}
	}
}

func TestGoogleAuthProgressiveRegistrationAndLogin(t *testing.T) {
	if config.DB == nil {
		t.Skip("integration database is not configured")
	}
	testutil.TruncateAll(t)
	t.Setenv("GOOGLE_AUTH_MOCK_ENABLED", "true")
	t.Setenv("APP_ENV", "test")
	t.Setenv("TERMS_CONSENT_ENFORCED", "false")
	t.Setenv("JWT_SECRET", "google-auth-test-secret-that-is-long-enough")

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/google", NewAuthHandler().GoogleAuth)
	token := validGoogleMockToken(t, "google-progressive-1", "new-google@example.com")

	post := func(body map[string]any) *httptest.ResponseRecorder {
		t.Helper()
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request: %v", err)
		}
		req := httptest.NewRequest(http.MethodPost, "/google", bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		return response
	}

	profile := post(map[string]any{"idToken": token})
	if profile.Code != http.StatusOK || !strings.Contains(profile.Body.String(), `"requiresProfileCompletion":true`) {
		t.Fatalf("expected profile-completion response, got %d %s", profile.Code, profile.Body.String())
	}

	created := post(map[string]any{
		"idToken":     token,
		"phone":       "08031234567",
		"dateOfBirth": "1990-01-01",
	})
	if created.Code != http.StatusCreated || !strings.Contains(created.Body.String(), `"refreshToken"`) {
		t.Fatalf("expected account creation with tokens, got %d %s", created.Code, created.Body.String())
	}

	var user models.User
	if err := config.DB.Where("LOWER(email) = ?", "new-google@example.com").First(&user).Error; err != nil {
		t.Fatalf("load created Google user: %v", err)
	}
	if user.GoogleID == nil || *user.GoogleID != "google-progressive-1" || user.Status != "pending" {
		t.Fatalf("unexpected stored Google user: %#v", user)
	}

	loggedIn := post(map[string]any{"idToken": token})
	if loggedIn.Code != http.StatusOK || !strings.Contains(loggedIn.Body.String(), `"isNewUser":false`) {
		t.Fatalf("expected repeat Google login, got %d %s", loggedIn.Code, loggedIn.Body.String())
	}

	existing, err := services.NewAuthService().Register(&models.User{
		Email:     "existing@example.com",
		Phone:     "08031234568",
		FirstName: "Existing",
		LastName:  "Citizen",
		Password:  "existing-password",
		Role:      models.RoleCitizen,
		Status:    "active",
	})
	if err != nil {
		t.Fatalf("create existing local account: %v", err)
	}
	linkToken := validGoogleMockToken(t, "google-link-1", existing.Email)
	linked := post(map[string]any{"idToken": linkToken})
	if linked.Code != http.StatusOK || !strings.Contains(linked.Body.String(), `"linked":true`) {
		t.Fatalf("expected verified-email account link, got %d %s", linked.Code, linked.Body.String())
	}
	if err := config.DB.First(existing, "id = ?", existing.ID).Error; err != nil {
		t.Fatalf("reload linked local account: %v", err)
	}
	if existing.GoogleID == nil || *existing.GoogleID != "google-link-1" {
		t.Fatalf("Google subject was not linked: %#v", existing)
	}
}
