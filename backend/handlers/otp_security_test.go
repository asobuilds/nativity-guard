//go:build integration

package handlers_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"security-solution/internal/testutil"
	"security-solution/models"
)

var otpTestIPCounter int

func nextOTPTestIP() string {
	otpTestIPCounter++
	return fmt.Sprintf("198.51.100.%d:1234", (otpTestIPCounter%250)+1)
}

func postOTP(t *testing.T, server http.Handler, path, token string, payload any, remoteAddr string) *httptest.ResponseRecorder {
	t.Helper()

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal OTP payload: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", token)
	}
	req.RemoteAddr = remoteAddr
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)
	return w
}

func latestOTPForUser(t *testing.T, userID uuid.UUID) models.OTP {
	t.Helper()

	var otp models.OTP
	if err := testutil.GetDB().
		Where("user_id = ?", userID).
		Order("created_at DESC").
		First(&otp).Error; err != nil {
		t.Fatalf("load latest OTP: %v", err)
	}
	return otp
}

func otpCountForUser(t *testing.T, userID uuid.UUID) int64 {
	t.Helper()

	var count int64
	if err := testutil.GetDB().Model(&models.OTP{}).
		Where("user_id = ?", userID).
		Count(&count).Error; err != nil {
		t.Fatalf("count OTPs: %v", err)
	}
	return count
}

func TestOTPSecurity(t *testing.T) {
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	server := FreshServer(t)

	t.Run("requires authentication", func(t *testing.T) {
		cases := []struct {
			path    string
			payload map[string]any
		}{
			{path: "/api/v1/otp/send", payload: map[string]any{"type": "phone"}},
			{path: "/api/v1/otp/verify", payload: map[string]any{"code": "123456"}},
			{path: "/api/v1/otp/resend", payload: map[string]any{"type": "phone"}},
		}
		for _, tc := range cases {
			w := postOTP(t, server, tc.path, "", tc.payload, nextOTPTestIP())
			if w.Code != http.StatusUnauthorized {
				t.Fatalf("POST %s status = %d, want 401. body=%s", tc.path, w.Code, w.Body.String())
			}
		}
	})

	t.Run("derives user from session and ignores caller user ID", func(t *testing.T) {
		userA := testutil.MakeUser(t, "citizen")
		userB := testutil.MakeUser(t, "citizen")

		w := postOTP(t, server, "/api/v1/otp/send", testutil.AuthHeader(t, userA), map[string]any{
			"userId": userB.ID.String(),
			"type":   "phone",
		}, nextOTPTestIP())
		if w.Code != http.StatusOK {
			t.Fatalf("send status = %d, want 200. body=%s", w.Code, w.Body.String())
		}

		otp := latestOTPForUser(t, userA.ID)
		if otp.UserID != userA.ID {
			t.Fatalf("OTP user = %s, want %s", otp.UserID, userA.ID)
		}
		if otpCountForUser(t, userB.ID) != 0 {
			t.Fatalf("caller-supplied user ID created an OTP for user B")
		}
		if len(otp.Code) != 6 {
			t.Fatalf("OTP length = %d, want 6", len(otp.Code))
		}

		w = postOTP(t, server, "/api/v1/otp/verify", testutil.AuthHeader(t, userB), map[string]any{
			"userId": userA.ID.String(),
			"code":   otp.Code,
		}, nextOTPTestIP())
		if w.Code != http.StatusBadRequest {
			t.Fatalf("user B verify status = %d, want 400. body=%s", w.Code, w.Body.String())
		}

		w = postOTP(t, server, "/api/v1/otp/verify", testutil.AuthHeader(t, userA), map[string]any{
			"userId": userB.ID.String(),
			"code":   otp.Code,
		}, nextOTPTestIP())
		if w.Code != http.StatusOK {
			t.Fatalf("user A verify status = %d, want 200. body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("send and resend return the same generic response", func(t *testing.T) {
		user := testutil.MakeUser(t, "citizen")
		token := testutil.AuthHeader(t, user)

		send := postOTP(t, server, "/api/v1/otp/send", token, map[string]any{"type": "phone"}, nextOTPTestIP())
		if send.Code != http.StatusOK {
			t.Fatalf("send status = %d, want 200. body=%s", send.Code, send.Body.String())
		}
		resend := postOTP(t, server, "/api/v1/otp/resend", token, map[string]any{"type": "phone"}, nextOTPTestIP())
		if resend.Code != http.StatusOK {
			t.Fatalf("resend status = %d, want 200. body=%s", resend.Code, resend.Body.String())
		}
		if !bytes.Equal(send.Body.Bytes(), resend.Body.Bytes()) {
			t.Fatalf("send and resend bodies differ: send=%s resend=%s", send.Body.String(), resend.Body.String())
		}
		if bytes.Contains(send.Body.Bytes(), []byte(user.ID.String())) {
			t.Fatalf("response leaked the user ID: %s", send.Body.String())
		}

		var response map[string]any
		if err := json.Unmarshal(send.Body.Bytes(), &response); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}
		if _, ok := response["otpId"]; ok {
			t.Fatalf("response must not include otpId: %s", send.Body.String())
		}
		if _, ok := response["delivered"]; ok {
			t.Fatalf("response must not include delivery state: %s", send.Body.String())
		}
		if message, ok := response["message"].(string); !ok || message == "" {
			t.Fatalf("response missing generic message: %s", send.Body.String())
		}
	})

	t.Run("enforces the five attempt limit", func(t *testing.T) {
		user := testutil.MakeUser(t, "citizen")
		token := testutil.AuthHeader(t, user)

		w := postOTP(t, server, "/api/v1/otp/send", token, map[string]any{"type": "phone"}, nextOTPTestIP())
		if w.Code != http.StatusOK {
			t.Fatalf("send status = %d, want 200. body=%s", w.Code, w.Body.String())
		}
		otp := latestOTPForUser(t, user.ID)

		wrongCode := "000000"
		if otp.Code == wrongCode {
			wrongCode = "000001"
		}
		for i := 0; i < maxTestOTPAttempts; i++ {
			w = postOTP(t, server, "/api/v1/otp/verify", token, map[string]any{"code": wrongCode}, nextOTPTestIP())
			if w.Code != http.StatusBadRequest {
				t.Fatalf("wrong attempt %d status = %d, want 400. body=%s", i+1, w.Code, w.Body.String())
			}
		}

		updated := latestOTPForUser(t, user.ID)
		if updated.Attempts != maxTestOTPAttempts {
			t.Fatalf("attempts = %d, want %d", updated.Attempts, maxTestOTPAttempts)
		}

		w = postOTP(t, server, "/api/v1/otp/verify", token, map[string]any{"code": otp.Code}, nextOTPTestIP())
		if w.Code != http.StatusBadRequest {
			t.Fatalf("verify after max attempts status = %d, want 400. body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("rejects expired OTPs", func(t *testing.T) {
		user := testutil.MakeUser(t, "citizen")
		token := testutil.AuthHeader(t, user)

		w := postOTP(t, server, "/api/v1/otp/send", token, map[string]any{"type": "phone"}, nextOTPTestIP())
		if w.Code != http.StatusOK {
			t.Fatalf("send status = %d, want 200. body=%s", w.Code, w.Body.String())
		}
		otp := latestOTPForUser(t, user.ID)
		if err := testutil.GetDB().Model(&models.OTP{}).
			Where("id = ?", otp.ID).
			Update("expires_at", time.Now().UTC().Add(-time.Minute)).Error; err != nil {
			t.Fatalf("expire OTP: %v", err)
		}

		w = postOTP(t, server, "/api/v1/otp/verify", token, map[string]any{"code": otp.Code}, nextOTPTestIP())
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expired OTP status = %d, want 400. body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("consumes OTP without changing user status", func(t *testing.T) {
		user := testutil.MakeUser(t, "citizen")
		if err := testutil.GetDB().Model(&models.User{}).
			Where("id = ?", user.ID).
			Update("status", "suspended").Error; err != nil {
			t.Fatalf("suspend user: %v", err)
		}
		token := testutil.AuthHeader(t, user)

		w := postOTP(t, server, "/api/v1/otp/send", token, map[string]any{"type": "phone"}, nextOTPTestIP())
		if w.Code != http.StatusOK {
			t.Fatalf("send status = %d, want 200. body=%s", w.Code, w.Body.String())
		}
		otp := latestOTPForUser(t, user.ID)

		w = postOTP(t, server, "/api/v1/otp/verify", token, map[string]any{"code": otp.Code}, nextOTPTestIP())
		if w.Code != http.StatusOK {
			t.Fatalf("verify status = %d, want 200. body=%s", w.Code, w.Body.String())
		}

		verified := latestOTPForUser(t, user.ID)
		if !verified.Verified {
			t.Fatalf("OTP was not marked verified")
		}

		var reloaded models.User
		if err := testutil.GetDB().First(&reloaded, "id = ?", user.ID).Error; err != nil {
			t.Fatalf("reload user: %v", err)
		}
		if reloaded.Status != "suspended" {
			t.Fatalf("user status = %q, want suspended", reloaded.Status)
		}

		w = postOTP(t, server, "/api/v1/otp/verify", token, map[string]any{"code": otp.Code}, nextOTPTestIP())
		if w.Code != http.StatusBadRequest {
			t.Fatalf("reused OTP status = %d, want 400. body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("rate limits OTP requests per IP", func(t *testing.T) {
		user := testutil.MakeUser(t, "citizen")
		token := testutil.AuthHeader(t, user)
		remoteAddr := "203.0.113.77:1234"

		for i := 0; i < maxTestOTPRateLimit; i++ {
			w := postOTP(t, server, "/api/v1/otp/send", token, map[string]any{"type": "phone"}, remoteAddr)
			if w.Code != http.StatusOK {
				t.Fatalf("request %d status = %d, want 200. body=%s", i+1, w.Code, w.Body.String())
			}
		}

		w := postOTP(t, server, "/api/v1/otp/send", token, map[string]any{"type": "phone"}, remoteAddr)
		if w.Code != http.StatusTooManyRequests {
			t.Fatalf("rate-limited request status = %d, want 429. body=%s", w.Code, w.Body.String())
		}
	})
}

const (
	maxTestOTPAttempts  = 5
	maxTestOTPRateLimit = 5
)
