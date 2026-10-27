package handlers

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"security-solution/config"
	"security-solution/internal/testutil"
	"security-solution/models"
)

const donationWebhookTestSecret = "test-webhook-secret"

func donationWebhookPayload(unitID, eventID string) []byte {
	return []byte(fmt.Sprintf(
		`{"eventId":%q,"provider":"paystack","eventType":"charge.success","unitId":%q,"amount":5000,"donorName":"Test Donor","description":"Test donation"}`,
		eventID, unitID,
	))
}

func signDonationWebhookBody(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func newDonationWebhookContext(body []byte, signature string) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/donation", bytes.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	if signature != "" {
		ctx.Request.Header.Set("X-Webhook-Signature", signature)
	}
	return ctx, recorder
}

func TestDonationWebhook_RejectsMissingSignature(t *testing.T) {
	t.Setenv("WEBHOOK_SECRET", donationWebhookTestSecret)

	body := donationWebhookPayload("00000000-0000-0000-0000-000000000001", "evt-missing")
	ctx, recorder := newDonationWebhookContext(body, "")

	DonationWebhook(ctx)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("want 401 without a signature, got %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestDonationWebhook_RejectsInvalidSignature(t *testing.T) {
	t.Setenv("WEBHOOK_SECRET", donationWebhookTestSecret)

	body := donationWebhookPayload("00000000-0000-0000-0000-000000000001", "evt-invalid")
	signature := signDonationWebhookBody("wrong-secret", body)
	ctx, recorder := newDonationWebhookContext(body, signature)

	DonationWebhook(ctx)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("want 401 with an invalid signature, got %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestDonationWebhook_RejectsTamperedBody(t *testing.T) {
	t.Setenv("WEBHOOK_SECRET", donationWebhookTestSecret)

	body := donationWebhookPayload("00000000-0000-0000-0000-000000000001", "evt-tampered")
	signature := signDonationWebhookBody(donationWebhookTestSecret, body)

	tampered := bytes.Replace(body, []byte(`"amount":5000`), []byte(`"amount":999999`), 1)
	if bytes.Equal(tampered, body) {
		t.Fatal("test setup failed to tamper with the request body")
	}

	ctx, recorder := newDonationWebhookContext(tampered, signature)

	DonationWebhook(ctx)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("want 401 when the body does not match the signature, got %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestDonationWebhook_FailsClosedWithoutSecret(t *testing.T) {
	t.Setenv("WEBHOOK_SECRET", "")

	body := donationWebhookPayload("00000000-0000-0000-0000-000000000001", "evt-no-secret")
	signature := signDonationWebhookBody(donationWebhookTestSecret, body)
	ctx, recorder := newDonationWebhookContext(body, signature)

	DonationWebhook(ctx)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("want 401 when WEBHOOK_SECRET is unset, got %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestDonationWebhook_AcceptsValidSignature(t *testing.T) {
	skipWithoutDB(t)
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	t.Setenv("WEBHOOK_SECRET", donationWebhookTestSecret)

	head := testutil.MakeUser(t, "unit_admin")
	unit := testutil.MakeUnit(t, head)

	const eventID = "evt-valid-001"
	body := donationWebhookPayload(unit.ID.String(), eventID)
	signature := signDonationWebhookBody(donationWebhookTestSecret, body)
	ctx, recorder := newDonationWebhookContext(body, signature)

	DonationWebhook(ctx)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("want 201 with a valid signature, got %d %s", recorder.Code, recorder.Body.String())
	}

	var event models.WebhookEvent
	if err := config.DB.Where("idempotency_key = ?", eventID).First(&event).Error; err != nil {
		t.Fatalf("load webhook event: %v", err)
	}
	if event.Provider != "paystack" {
		t.Fatalf("want provider paystack, got %q", event.Provider)
	}

	var entry models.FinancialLedger
	if err := config.DB.Where("source_type = ? AND source_id = ?", "donation", event.ID).First(&entry).Error; err != nil {
		t.Fatalf("load ledger entry: %v", err)
	}
	if entry.UnitID != unit.ID {
		t.Fatalf("want ledger unit %s, got %s", unit.ID, entry.UnitID)
	}
	if entry.Direction != "in" {
		t.Fatalf("want direction in, got %q", entry.Direction)
	}
	if entry.EntryType != "donation" {
		t.Fatalf("want entry type donation, got %q", entry.EntryType)
	}
	if entry.Amount != 5000 {
		t.Fatalf("want amount 5000, got %v", entry.Amount)
	}
	if !strings.Contains(recorder.Body.String(), `"ledgerEntry"`) {
		t.Fatalf("want ledger entry in response, got %s", recorder.Body.String())
	}
}
