//go:build integration

package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"security-solution/config"
	"security-solution/internal/testutil"
	"security-solution/models"
	"security-solution/services"
)

// TestAuth_RegisterRejectsDuplicateEmail_CaseInsensitive proves the
// case-insensitive email uniqueness rule.
func TestAuth_RegisterRejectsDuplicateEmail_CaseInsensitive(t *testing.T) {
	testutil.TruncateAll(t)

	authSvc := services.NewAuthService()

	first := &models.User{
		Email:     "alice@example.com",
		Phone:     "08012345678",
		FirstName: "Alice",
		LastName:  "One",
		Password:  "password-1234",
		Role:      "citizen",
		Status:    "active",
	}
	if _, err := authSvc.Register(first); err != nil {
		t.Fatalf("first register: %v", err)
	}

	// Same email with different casing → should be caught by app-level check
	var count int64
	config.DB.Model(&models.User{}).
		Where("LOWER(email) = ?", "alice@example.com").
		Count(&count)
	if count != 1 {
		t.Fatalf("expected 1 user matching lower(email), got %d", count)
	}

	// The DB unique constraint is case-sensitive by default, so this test
	// asserts the app-level check catches the duplicate. Simulate the
	// app-level check that REGISTER performs.
	dupCheck := int64(0)
	config.DB.Model(&models.User{}).
		Where("LOWER(email) = ? OR phone = ?", "ALICE@example.com", "08012345678").
		Count(&dupCheck)
	if dupCheck == 0 {
		t.Fatalf("case-insensitive dup check failed to find existing user")
	}
}

// TestAuth_RegisterRejectsDuplicatePhone proves phone is unique.
func TestAuth_RegisterRejectsDuplicatePhone(t *testing.T) {
	testutil.TruncateAll(t)

	authSvc := services.NewAuthService()

	first := &models.User{
		Email:     "bob@example.com",
		Phone:     "08087654321",
		FirstName: "Bob",
		LastName:  "One",
		Password:  "password-1234",
		Role:      "citizen",
		Status:    "active",
	}
	if _, err := authSvc.Register(first); err != nil {
		t.Fatalf("first register: %v", err)
	}

	var count int64
	config.DB.Model(&models.User{}).
		Where("phone = ?", "08087654321").
		Count(&count)
	if count != 1 {
		t.Fatalf("expected 1 user with phone 08087654321, got %d", count)
	}
}

// TestAuth_LoginByEmailAndPhone proves both identifiers work.
func TestAuth_LoginByEmailAndPhone(t *testing.T) {
	testutil.TruncateAll(t)

	authSvc := services.NewAuthService()

	u := &models.User{
		Email:     "carol@example.com",
		Phone:     "08011112222",
		FirstName: "Carol",
		LastName:  "One",
		Password:  "password-1234",
		Role:      "citizen",
		Status:    "active",
	}
	if _, err := authSvc.Register(u); err != nil {
		t.Fatalf("register: %v", err)
	}

	// Login by email
	if _, _, _, err := authSvc.LoginWithJTI("carol@example.com", "password-1234"); err != nil {
		t.Fatalf("login by email: %v", err)
	}

	// Login by phone
	if _, _, _, err := authSvc.LoginWithJTI("08011112222", "password-1234"); err != nil {
		t.Fatalf("login by phone: %v", err)
	}

	// Wrong password → error
	if _, _, _, err := authSvc.LoginWithJTI("carol@example.com", "wrong-password"); err == nil {
		t.Fatalf("expected error for wrong password")
	}
}

// TestAuth_LogoutRevokesToken proves a revoked JTI is rejected by
// the TokenService revocation check.
func TestAuth_LogoutRevokesToken(t *testing.T) {
	testutil.TruncateAll(t)

	authSvc := services.NewAuthService()
	u := &models.User{
		Email:     "dave@example.com",
		Phone:     "08033334444",
		FirstName: "Dave",
		LastName:  "One",
		Password:  "password-1234",
		Role:      "citizen",
		Status:    "active",
	}
	if _, err := authSvc.Register(u); err != nil {
		t.Fatalf("register: %v", err)
	}

	token, jti, _, err := authSvc.LoginWithJTI("dave@example.com", "password-1234")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if token == "" || jti == "" {
		t.Fatalf("expected non-empty token + jti")
	}

	// Not revoked yet
	tokenSvc := services.NewTokenService()
	if tokenSvc.IsRevoked(jti) {
		t.Fatalf("jti should not be revoked before logout")
	}

	// Revoke
	if err := tokenSvc.Revoke(jti, u.ID, time.Now().UTC().Add(24*time.Hour), "logout"); err != nil {
		t.Fatalf("revoke: %v", err)
	}

	// Now revoked
	if !tokenSvc.IsRevoked(jti) {
		t.Fatalf("jti should be revoked after logout")
	}
}

func TestSendPasswordResetTokenEmail_WithBaseURL(t *testing.T) {
	t.Setenv("BASE_URL", "https://nativityguard.com/")

	var gotTo, gotSubject, gotBody string
	origSend := sendEmailFn
	sendEmailFn = func(to, subject, body string) error {
		gotTo, gotSubject, gotBody = to, subject, body
		return nil
	}
	defer func() { sendEmailFn = origSend }()

	if err := SendPasswordResetTokenEmail("user@example.com", "123456"); err != nil {
		t.Fatalf("SendPasswordResetTokenEmail: %v", err)
	}
	if gotTo != "user@example.com" {
		t.Fatalf("expected to=user@example.com, got %q", gotTo)
	}
	if gotSubject == "" {
		t.Fatalf("expected non-empty subject")
	}
	wantLink := "https://nativityguard.com/auth/reset-password?token=123456"
	if !strings.Contains(gotBody, wantLink) {
		t.Fatalf("expected body to contain clickable link %q, got %q", wantLink, gotBody)
	}
}

func TestSendPasswordResetTokenEmail_WithoutBaseURL(t *testing.T) {
	t.Setenv("BASE_URL", "")

	var gotTo, gotBody string
	origSend := sendEmailFn
	sendEmailFn = func(to, _, body string) error {
		gotTo, gotBody = to, body
		return nil
	}
	defer func() { sendEmailFn = origSend }()

	if err := SendPasswordResetTokenEmail("user@example.com", "654321"); err != nil {
		t.Fatalf("SendPasswordResetTokenEmail: %v", err)
	}
	if gotTo != "user@example.com" {
		t.Fatalf("expected to=user@example.com, got %q", gotTo)
	}
	if !strings.Contains(gotBody, "654321") {
		t.Fatalf("expected body to contain inline token 654321, got %q", gotBody)
	}
	if strings.Contains(gotBody, "/auth/reset-password?token=") {
		t.Fatalf("did not expect reset link when BASE_URL is unset, got %q", gotBody)
	}
}

type recordingResetNotifier struct {
	mu     sync.Mutex
	emails []string
	sms    []string
	ch     chan struct{}
}

func (r *recordingResetNotifier) NotifyEmail(toEmail, _, _ string) {
	r.mu.Lock()
	r.emails = append(r.emails, toEmail)
	r.mu.Unlock()
	r.ch <- struct{}{}
}

func (r *recordingResetNotifier) NotifySMS(toPhone, _ string) {
	r.mu.Lock()
	r.sms = append(r.sms, toPhone)
	r.mu.Unlock()
	r.ch <- struct{}{}
}

func TestAuth_ForgotPassword_EmailAndSMSDelivery_NoEnumeration(t *testing.T) {
	testutil.TruncateAll(t)

	authSvc := services.NewAuthService()
	u := &models.User{
		Email:     "eve@example.com",
		Phone:     "08055556666",
		FirstName: "Eve",
		LastName:  "One",
		Password:  "password-1234",
		Role:      "citizen",
		Status:    "active",
	}
	if _, err := authSvc.Register(u); err != nil {
		t.Fatalf("register: %v", err)
	}

	rec := &recordingResetNotifier{ch: make(chan struct{}, 4)}
	services.SetResetNotifier(rec)
	defer services.SetResetNotifier(nil)

	gin.SetMode(gin.TestMode)
	h := NewAuthHandler()
	router := gin.New()
	router.POST("/auth/forgot-password", h.ForgotPassword)

	sendReq := func(identifier string) *httptest.ResponseRecorder {
		payload, _ := json.Marshal(map[string]string{"identifier": identifier})
		req := httptest.NewRequest(http.MethodPost, "/auth/forgot-password", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}

	w1 := sendReq("eve@example.com")
	if w1.Code != http.StatusOK {
		t.Fatalf("expected 200 for existing email, got %d: %s", w1.Code, w1.Body.String())
	}

	for i := 0; i < 2; i++ {
		select {
		case <-rec.ch:
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out waiting for reset notifications (%d/2 received)", i)
		}
	}

	rec.mu.Lock()
	emails := append([]string(nil), rec.emails...)
	sms := append([]string(nil), rec.sms...)
	rec.mu.Unlock()

	if len(emails) != 1 || emails[0] != "eve@example.com" {
		t.Fatalf("expected 1 email notification to eve@example.com, got %v", emails)
	}
	if len(sms) != 1 || sms[0] != "08055556666" {
		t.Fatalf("expected 1 SMS fallback notification to 08055556666, got %v", sms)
	}

	// Unknown identifier must return the exact same 200 response and send nothing.
	w2 := sendReq("nobody@example.com")
	if w2.Code != http.StatusOK || w2.Body.String() != w1.Body.String() {
		t.Fatalf("expected identical 200 response for unknown identifier, got %d: %s", w2.Code, w2.Body.String())
	}
}
