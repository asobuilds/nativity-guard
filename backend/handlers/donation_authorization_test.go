//go:build integration

package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"security-solution/internal/testutil"
	"security-solution/models"
)

func donationPayload(unitID string, amount float64, reference, paymentMethod string) map[string]any {
	return map[string]any{
		"unitId":        unitID,
		"donorName":     "Test Donor",
		"amount":        amount,
		"reference":     reference,
		"paymentMethod": paymentMethod,
	}
}

func postDonation(t *testing.T, token string, payload map[string]any) *httptest.ResponseRecorder {
	t.Helper()

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal donation payload: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/bank/donations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", token)
	w := httptest.NewRecorder()
	FreshServer(t).ServeHTTP(w, req)
	return w
}

func postConfirmDonation(t *testing.T, token, donationID string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/bank/donations/"+donationID+"/confirm", nil)
	req.Header.Set("Authorization", token)
	w := httptest.NewRecorder()
	FreshServer(t).ServeHTTP(w, req)
	return w
}

func donationRecordCounts(t *testing.T) (int64, int64, int64) {
	t.Helper()

	db := testutil.GetDB()
	var donations int64
	var transactions int64
	var ledgers int64
	if err := db.Model(&models.Donation{}).Count(&donations).Error; err != nil {
		t.Fatalf("count donations: %v", err)
	}
	if err := db.Model(&models.Transaction{}).Count(&transactions).Error; err != nil {
		t.Fatalf("count transactions: %v", err)
	}
	if err := db.Model(&models.FinancialLedger{}).Count(&ledgers).Error; err != nil {
		t.Fatalf("count ledger entries: %v", err)
	}
	return donations, transactions, ledgers
}

func TestRecordDonation_RejectsUnauthorizedCallers(t *testing.T) {
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	owner := testutil.MakeUser(t, "unit_admin")
	unit := testutil.MakeUnit(t, owner)

	regularAdmin := testutil.MakeUser(t, "unit_admin")
	testutil.MakeMembership(t, regularAdmin, unit, models.UnitRoleAdmin, false)

	otherHead := testutil.MakeUser(t, "unit_admin")
	testutil.MakeUnit(t, otherHead)

	unverifiedHead := testutil.MakeUser(t, "unit_admin")
	testutil.MakeMembership(t, unverifiedHead, unit, models.UnitRoleAdmin, true)
	if err := testutil.GetDB().Model(&models.UnitMembership{}).
		Where("unit_id = ? AND user_id = ?", unit.ID, unverifiedHead.ID).
		Update("verified_at", nil).Error; err != nil {
		t.Fatalf("clear verified_at: %v", err)
	}

	citizen := testutil.MakeUser(t, "citizen")

	cases := []struct {
		name string
		user *models.User
	}{
		{name: "citizen", user: citizen},
		{name: "regular admin", user: regularAdmin},
		{name: "head admin of another unit", user: otherHead},
		{name: "unverified head admin", user: unverifiedHead},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := postDonation(t, testutil.AuthHeader(t, tc.user),
				donationPayload(unit.ID.String(), 5000, "REF-"+tc.name, "bank_transfer"))
			if w.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want 403. body=%s", w.Code, w.Body.String())
			}
		})
	}

	donations, transactions, ledgers := donationRecordCounts(t)
	if donations != 0 || transactions != 0 || ledgers != 0 {
		t.Fatalf("unauthorized requests created records: donations=%d transactions=%d ledgers=%d", donations, transactions, ledgers)
	}
}

func TestRecordDonation_HeadAdminCreatesAtomicRecords(t *testing.T) {
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	owner := testutil.MakeUser(t, "unit_admin")
	unit := testutil.MakeUnit(t, owner)

	w := postDonation(t, testutil.AuthHeader(t, owner),
		donationPayload(unit.ID.String(), 5000, "REF-OK", "bank_transfer"))
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201. body=%s", w.Code, w.Body.String())
	}

	var response struct {
		Donation models.Donation `json:"donation"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal donation: %v", err)
	}
	if response.Donation.Status != "confirmed" {
		t.Errorf("donation status = %q, want confirmed", response.Donation.Status)
	}
	if response.Donation.UnitID != unit.ID {
		t.Errorf("donation unit = %s, want %s", response.Donation.UnitID, unit.ID)
	}
	if response.Donation.ConfirmedBy == nil || *response.Donation.ConfirmedBy != owner.ID {
		t.Errorf("confirmedBy = %v, want %s", response.Donation.ConfirmedBy, owner.ID)
	}

	var transaction models.Transaction
	if err := testutil.GetDB().Where("unit_id = ? AND type = ?", unit.ID, "donation").
		First(&transaction).Error; err != nil {
		t.Fatalf("load transaction: %v", err)
	}
	if transaction.Amount != 5000 {
		t.Errorf("transaction amount = %v, want 5000", transaction.Amount)
	}

	var ledger models.FinancialLedger
	if err := testutil.GetDB().
		Where("unit_id = ? AND source_type = ? AND source_id = ?", unit.ID, "donation", response.Donation.ID).
		First(&ledger).Error; err != nil {
		t.Fatalf("load ledger: %v", err)
	}
	if ledger.UnitID != unit.ID {
		t.Errorf("ledger unit = %s, want %s", ledger.UnitID, unit.ID)
	}
	if ledger.Amount != 5000 {
		t.Errorf("ledger amount = %v, want 5000", ledger.Amount)
	}

	donations, transactions, ledgers := donationRecordCounts(t)
	if donations != 1 || transactions != 1 || ledgers != 1 {
		t.Fatalf("unexpected record counts: donations=%d transactions=%d ledgers=%d", donations, transactions, ledgers)
	}
}

func TestRecordDonation_SuperAdminCreatesForAnyUnit(t *testing.T) {
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	superAdmin := testutil.MakeUser(t, "super_admin")
	owner := testutil.MakeUser(t, "unit_admin")
	unit := testutil.MakeUnit(t, owner)

	w := postDonation(t, testutil.AuthHeader(t, superAdmin),
		donationPayload(unit.ID.String(), 7500, "REF-SUPER", "cash"))
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201. body=%s", w.Code, w.Body.String())
	}

	var response struct {
		Donation models.Donation `json:"donation"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal donation: %v", err)
	}
	if response.Donation.UnitID != unit.ID {
		t.Errorf("donation unit = %s, want %s", response.Donation.UnitID, unit.ID)
	}

	var ledger models.FinancialLedger
	if err := testutil.GetDB().
		Where("unit_id = ? AND source_type = ? AND source_id = ?", unit.ID, "donation", response.Donation.ID).
		First(&ledger).Error; err != nil {
		t.Fatalf("load ledger: %v", err)
	}
	if ledger.UnitID != unit.ID {
		t.Errorf("ledger unit = %s, want %s", ledger.UnitID, unit.ID)
	}
}

func TestRecordDonation_ValidatesInput(t *testing.T) {
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	owner := testutil.MakeUser(t, "unit_admin")
	unit := testutil.MakeUnit(t, owner)

	cases := []struct {
		name   string
		mutate func(map[string]any)
	}{
		{name: "negative amount", mutate: func(p map[string]any) { p["amount"] = -1 }},
		{name: "zero amount", mutate: func(p map[string]any) { p["amount"] = 0 }},
		{name: "fractional amount", mutate: func(p map[string]any) { p["amount"] = 100.5 }},
		{name: "missing amount", mutate: func(p map[string]any) { delete(p, "amount") }},
		{name: "missing reference", mutate: func(p map[string]any) { p["reference"] = "" }},
		{name: "blank reference", mutate: func(p map[string]any) { p["reference"] = "   " }},
		{name: "missing payment method", mutate: func(p map[string]any) { p["paymentMethod"] = "" }},
		{name: "blank payment method", mutate: func(p map[string]any) { p["paymentMethod"] = "   " }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			payload := donationPayload(unit.ID.String(), 5000, "REF-VALID", "bank_transfer")
			tc.mutate(payload)
			w := postDonation(t, testutil.AuthHeader(t, owner), payload)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400. body=%s", w.Code, w.Body.String())
			}
		})
	}

	donations, transactions, ledgers := donationRecordCounts(t)
	if donations != 0 || transactions != 0 || ledgers != 0 {
		t.Fatalf("invalid requests created records: donations=%d transactions=%d ledgers=%d", donations, transactions, ledgers)
	}
}

func TestRecordDonation_RejectsUnknownUnit(t *testing.T) {
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	superAdmin := testutil.MakeUser(t, "super_admin")

	w := postDonation(t, testutil.AuthHeader(t, superAdmin),
		donationPayload(uuid.NewString(), 5000, "REF-UNKNOWN", "bank_transfer"))
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404. body=%s", w.Code, w.Body.String())
	}

	donations, transactions, ledgers := donationRecordCounts(t)
	if donations != 0 || transactions != 0 || ledgers != 0 {
		t.Fatalf("unknown unit created records: donations=%d transactions=%d ledgers=%d", donations, transactions, ledgers)
	}
}

func TestRecordDonation_RollsBackWhenLedgerFails(t *testing.T) {
	testutil.TruncateAll(t)
	t.Cleanup(func() { testutil.TruncateAll(t) })

	owner := testutil.MakeUser(t, "unit_admin")
	unit := testutil.MakeUnit(t, owner)

	if err := testutil.GetDB().Exec(
		"ALTER TABLE financial_ledgers RENAME TO financial_ledgers_atomicity_tmp",
	).Error; err != nil {
		t.Fatalf("rename ledger table: %v", err)
	}
	t.Cleanup(func() {
		if err := testutil.GetDB().Exec(
			"ALTER TABLE financial_ledgers_atomicity_tmp RENAME TO financial_ledgers",
		).Error; err != nil {
			t.Errorf("restore ledger table: %v", err)
		}
	})

	w := postDonation(t, testutil.AuthHeader(t, owner),
		donationPayload(unit.ID.String(), 5000, "REF-ROLLBACK", "bank_transfer"))
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500. body=%s", w.Code, w.Body.String())
	}

	var donations int64
	var transactions int64
	if err := testutil.GetDB().Model(&models.Donation{}).Count(&donations).Error; err != nil {
		t.Fatalf("count donations: %v", err)
	}
	if err := testutil.GetDB().Model(&models.Transaction{}).Count(&transactions).Error; err != nil {
		t.Fatalf("count transactions: %v", err)
	}
	if donations != 0 || transactions != 0 {
		t.Fatalf("ledger failure did not roll back: donations=%d transactions=%d", donations, transactions)
	}
}

func TestConfirmDonation_RejectsUnauthorizedCallers(t *testing.T) {
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	owner := testutil.MakeUser(t, "unit_admin")
	unit := testutil.MakeUnit(t, owner)

	regularAdmin := testutil.MakeUser(t, "unit_admin")
	testutil.MakeMembership(t, regularAdmin, unit, models.UnitRoleAdmin, false)

	otherHead := testutil.MakeUser(t, "unit_admin")
	testutil.MakeUnit(t, otherHead)

	unverifiedHead := testutil.MakeUser(t, "unit_admin")
	testutil.MakeMembership(t, unverifiedHead, unit, models.UnitRoleAdmin, true)
	if err := testutil.GetDB().Model(&models.UnitMembership{}).
		Where("unit_id = ? AND user_id = ?", unit.ID, unverifiedHead.ID).
		Update("verified_at", nil).Error; err != nil {
		t.Fatalf("clear verified_at: %v", err)
	}

	citizen := testutil.MakeUser(t, "citizen")

	donation := models.Donation{
		ID:            uuid.New(),
		UnitID:        unit.ID,
		DonorName:     "Pending Donor",
		Amount:        2500,
		PaymentMethod: "bank_transfer",
		Reference:     "REF-PENDING",
		Status:        "pending",
		DonationDate:  time.Now(),
	}
	if err := testutil.GetDB().Create(&donation).Error; err != nil {
		t.Fatalf("create pending donation: %v", err)
	}

	cases := []struct {
		name string
		user *models.User
	}{
		{name: "citizen", user: citizen},
		{name: "regular admin", user: regularAdmin},
		{name: "head admin of another unit", user: otherHead},
		{name: "unverified head admin", user: unverifiedHead},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := postConfirmDonation(t, testutil.AuthHeader(t, tc.user), donation.ID.String())
			if w.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want 403. body=%s", w.Code, w.Body.String())
			}
		})
	}

	var reloaded models.Donation
	if err := testutil.GetDB().First(&reloaded, "id = ?", donation.ID).Error; err != nil {
		t.Fatalf("reload donation: %v", err)
	}
	if reloaded.Status != "pending" {
		t.Errorf("donation status = %q, want pending", reloaded.Status)
	}

	donations, transactions, ledgers := donationRecordCounts(t)
	if donations != 1 || transactions != 0 || ledgers != 0 {
		t.Fatalf("unexpected record counts: donations=%d transactions=%d ledgers=%d", donations, transactions, ledgers)
	}
}

func TestConfirmDonation_HeadAdminConfirmsAtomically(t *testing.T) {
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	owner := testutil.MakeUser(t, "unit_admin")
	unit := testutil.MakeUnit(t, owner)

	donation := models.Donation{
		ID:            uuid.New(),
		UnitID:        unit.ID,
		DonorName:     "Pending Donor",
		Amount:        2500,
		PaymentMethod: "bank_transfer",
		Reference:     "REF-CONFIRM",
		Status:        "pending",
		DonationDate:  time.Now(),
	}
	if err := testutil.GetDB().Create(&donation).Error; err != nil {
		t.Fatalf("create pending donation: %v", err)
	}

	w := postConfirmDonation(t, testutil.AuthHeader(t, owner), donation.ID.String())
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200. body=%s", w.Code, w.Body.String())
	}

	var response struct {
		Donation    models.Donation    `json:"donation"`
		Transaction models.Transaction `json:"transaction"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal confirmation: %v", err)
	}
	if response.Donation.Status != "confirmed" {
		t.Errorf("donation status = %q, want confirmed", response.Donation.Status)
	}
	if response.Donation.ConfirmedBy == nil || *response.Donation.ConfirmedBy != owner.ID {
		t.Errorf("confirmedBy = %v, want %s", response.Donation.ConfirmedBy, owner.ID)
	}
	if response.Transaction.UnitID != unit.ID {
		t.Errorf("transaction unit = %s, want %s", response.Transaction.UnitID, unit.ID)
	}

	var ledger models.FinancialLedger
	if err := testutil.GetDB().
		Where("unit_id = ? AND source_type = ? AND source_id = ?", unit.ID, "donation", donation.ID).
		First(&ledger).Error; err != nil {
		t.Fatalf("load ledger: %v", err)
	}
	if ledger.UnitID != unit.ID {
		t.Errorf("ledger unit = %s, want %s", ledger.UnitID, unit.ID)
	}

	donations, transactions, ledgers := donationRecordCounts(t)
	if donations != 1 || transactions != 1 || ledgers != 1 {
		t.Fatalf("unexpected record counts: donations=%d transactions=%d ledgers=%d", donations, transactions, ledgers)
	}

	w = postConfirmDonation(t, testutil.AuthHeader(t, owner), donation.ID.String())
	if w.Code != http.StatusBadRequest {
		t.Fatalf("second confirm status = %d, want 400. body=%s", w.Code, w.Body.String())
	}
	donations, transactions, ledgers = donationRecordCounts(t)
	if donations != 1 || transactions != 1 || ledgers != 1 {
		t.Fatalf("second confirm changed record counts: donations=%d transactions=%d ledgers=%d", donations, transactions, ledgers)
	}
}
