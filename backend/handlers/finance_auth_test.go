package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"security-solution/config"
	"security-solution/cryptoutil"
	"security-solution/internal/testutil"
	"security-solution/models"
)

func newFinanceTestContext(method, path string, body []byte, user *models.User, params ...gin.Param) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, path, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	c.Request = req
	if user != nil {
		c.Set("user", user)
	}
	if len(params) > 0 {
		c.Params = params
	}
	return c, w
}

func TestMaskAccountNumber(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"0123456789", "******6789"},
		{"12345", "*2345"},
		{"1234", "****"},
		{"12", "****"},
		{"", "****"},
		{"  9876543210  ", "******3210"},
	}
	for _, tc := range cases {
		if got := maskAccountNumber(tc.in); got != tc.want {
			t.Errorf("maskAccountNumber(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestCallerIsUnitAdminOfUnit_NonDBCases(t *testing.T) {
	unitA := uuid.New()
	unitB := uuid.New()

	if callerIsUnitAdminOfUnit(nil, unitA) {
		t.Fatal("nil user must not be unit admin")
	}

	super := &models.User{ID: uuid.New(), Role: "super_admin", IsSuperAdmin: true}
	if !callerIsUnitAdminOfUnit(super, unitA) {
		t.Fatal("super_admin must be allowed for any unit")
	}

	unitAdminA := &models.User{ID: uuid.New(), Role: "unit_admin", UnitID: &unitA}
	if !callerIsUnitAdminOfUnit(unitAdminA, unitA) {
		t.Fatal("unit_admin with matching UnitID must be allowed")
	}

	citizen := &models.User{ID: uuid.New(), Role: "citizen", UnitID: &unitA}
	if callerIsUnitAdminOfUnit(citizen, unitA) {
		t.Fatal("citizen must be denied even if UnitID matches")
	}

	officer := &models.User{ID: uuid.New(), Role: "officer", UnitID: &unitA}
	if callerIsUnitAdminOfUnit(officer, unitA) {
		t.Fatal("officer must be denied unit admin finance access")
	}

	if config.DB != nil {
		unitAdminB := &models.User{ID: uuid.New(), Role: "unit_admin", UnitID: &unitB}
		if callerIsUnitAdminOfUnit(unitAdminB, unitA) {
			t.Fatal("unit_admin of unit B must be denied access to unit A")
		}
	}
}

func TestFinanceHandlers_RejectUnauthenticatedAndUnauthorized(t *testing.T) {
	unitA := uuid.New()
	unitB := uuid.New()

	citizen := &models.User{ID: uuid.New(), Role: "citizen", UnitID: &unitA}
	officer := &models.User{ID: uuid.New(), Role: "officer", UnitID: &unitA}
	otherAdmin := &models.User{ID: uuid.New(), Role: "unit_admin", UnitID: &unitB}

	// Endpoints that take :id as unitID
	unitScopedEndpoints := []struct {
		name    string
		method  string
		handler gin.HandlerFunc
	}{
		{"GetBankAccounts", http.MethodGet, GetBankAccounts},
		{"GetDonations", http.MethodGet, GetDonations},
		{"GetTransactions", http.MethodGet, GetTransactions},
		{"GetTransactionSummary", http.MethodGet, GetTransactionSummary},
		{"GetBudgets", http.MethodGet, GetBudgets},
		{"GetFinancialReports", http.MethodGet, GetFinancialReports},
	}

	for _, ep := range unitScopedEndpoints {
		cUnauth, wUnauth := newFinanceTestContext(ep.method, "/", nil, nil, gin.Param{Key: "id", Value: unitA.String()})
		ep.handler(cUnauth)
		if wUnauth.Code != http.StatusUnauthorized {
			t.Errorf("%s unauthenticated: got %d, want 401", ep.name, wUnauth.Code)
		}

		for _, tc := range []struct {
			role string
			user *models.User
		}{
			{"citizen", citizen},
			{"officer", officer},
		} {
			c, w := newFinanceTestContext(ep.method, "/", nil, tc.user, gin.Param{Key: "id", Value: unitA.String()})
			ep.handler(c)
			if w.Code != http.StatusForbidden {
				t.Errorf("%s (%s): got %d, want 403. body=%s", ep.name, tc.role, w.Code, w.Body.String())
			}
		}

		if config.DB != nil {
			cOther, wOther := newFinanceTestContext(ep.method, "/", nil, otherAdmin, gin.Param{Key: "id", Value: unitA.String()})
			ep.handler(cOther)
			if wOther.Code != http.StatusForbidden {
				t.Errorf("%s (other unit_admin): got %d, want 403", ep.name, wOther.Code)
			}
		}
	}
}

func TestCreateTransaction_ValidationAndAuth(t *testing.T) {
	unitA := uuid.New()
	citizen := &models.User{ID: uuid.New(), Role: "citizen", UnitID: &unitA}
	admin := &models.User{ID: uuid.New(), Role: "unit_admin", UnitID: &unitA}

	validPayload := []byte(`{"unitId":"` + unitA.String() + `","type":"expense","amount":5000,"description":"Fuel"}`)

	// Unauthenticated
	c, w := newFinanceTestContext(http.MethodPost, "/api/v1/finance/transactions", validPayload, nil)
	CreateTransaction(c)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated: got %d, want 401", w.Code)
	}

	// Citizen forbidden
	c, w = newFinanceTestContext(http.MethodPost, "/api/v1/finance/transactions", validPayload, citizen)
	CreateTransaction(c)
	if w.Code != http.StatusForbidden {
		t.Fatalf("citizen: got %d, want 403", w.Code)
	}

	// Negative amount rejected
	negPayload := []byte(`{"unitId":"` + unitA.String() + `","type":"expense","amount":-100,"description":"Bad"}`)
	c, w = newFinanceTestContext(http.MethodPost, "/api/v1/finance/transactions", negPayload, admin)
	CreateTransaction(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("negative amount: got %d, want 400", w.Code)
	}

	// Invalid type rejected
	badTypePayload := []byte(`{"unitId":"` + unitA.String() + `","type":"hack","amount":100,"description":"Bad"}`)
	c, w = newFinanceTestContext(http.MethodPost, "/api/v1/finance/transactions", badTypePayload, admin)
	CreateTransaction(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid type: got %d, want 400", w.Code)
	}
}

func TestApproveAndRejectTransaction_RoleChecks(t *testing.T) {
	txID := uuid.New().String()
	citizen := &models.User{ID: uuid.New(), Role: "citizen"}
	officer := &models.User{ID: uuid.New(), Role: "officer"}

	for _, u := range []*models.User{citizen, officer} {
		c, w := newFinanceTestContext(http.MethodPost, "/", []byte(`{"comment":"ok"}`), u, gin.Param{Key: "id", Value: txID})
		ApproveTransaction(c)
		if w.Code != http.StatusForbidden {
			t.Errorf("ApproveTransaction (%s): got %d, want 403", u.Role, w.Code)
		}

		c2, w2 := newFinanceTestContext(http.MethodPost, "/", []byte(`{"comment":"no"}`), u, gin.Param{Key: "id", Value: txID})
		RejectTransaction(c2)
		if w2.Code != http.StatusForbidden {
			t.Errorf("RejectTransaction (%s): got %d, want 403", u.Role, w2.Code)
		}
	}
}

func TestPlatformDonations_SuperAdminDefenseInDepth(t *testing.T) {
	unitAdmin := &models.User{ID: uuid.New(), Role: "unit_admin"}
	citizen := &models.User{ID: uuid.New(), Role: "citizen"}
	donationID := uuid.New().String()

	for _, u := range []*models.User{nil, unitAdmin, citizen} {
		want := http.StatusForbidden
		if u == nil {
			want = http.StatusUnauthorized
		}

		c1, w1 := newFinanceTestContext(http.MethodGet, "/api/v1/super-admin/platform-donations", nil, u)
		ListPlatformDonations(c1)
		if w1.Code != want {
			t.Errorf("ListPlatformDonations (%v): got %d, want %d", u, w1.Code, want)
		}

		c2, w2 := newFinanceTestContext(http.MethodPost, "/api/v1/super-admin/platform-donations/"+donationID+"/confirm", nil, u, gin.Param{Key: "id", Value: donationID})
		ConfirmPlatformDonation(c2)
		if w2.Code != want {
			t.Errorf("ConfirmPlatformDonation (%v): got %d, want %d", u, w2.Code, want)
		}
	}
}

func TestDonationWebhook_RejectsNonPositiveAmount(t *testing.T) {
	t.Setenv("WEBHOOK_SECRET", donationWebhookTestSecret)

	body := []byte(`{"eventId":"evt-neg-1","provider":"paystack","unitId":"00000000-0000-0000-0000-000000000001","amount":-500}`)
	sig := signDonationWebhookBody(donationWebhookTestSecret, body)
	c, w := newDonationWebhookContext(body, sig)

	DonationWebhook(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("negative webhook amount: got %d, want 400. body=%s", w.Code, w.Body.String())
	}
}

func TestGetUnitLedger_BlocksCitizen(t *testing.T) {
	unitID := uuid.New().String()
	citizen := &models.User{ID: uuid.New(), Role: "citizen"}

	c, w := newFinanceTestContext(http.MethodGet, "/", nil, citizen, gin.Param{Key: "id", Value: unitID})
	GetUnitLedger(c)
	if w.Code != http.StatusForbidden {
		t.Fatalf("citizen GetUnitLedger: got %d, want 403", w.Code)
	}
}

func TestFinance_Integration_Hardening(t *testing.T) {
	skipWithoutDB(t)
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	t.Setenv("ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")

	super := testutil.MakeUser(t, "super_admin")
	headA := testutil.MakeUser(t, "unit_admin")
	unitA := testutil.MakeUnit(t, headA)
	headB := testutil.MakeUser(t, "unit_admin")
	unitB := testutil.MakeUnit(t, headB)

	// 1. GetPublicUnitLedger on empty unit must not panic
	cEmpty, wEmpty := newFinanceTestContext(http.MethodGet, "/", nil, nil, gin.Param{Key: "id", Value: unitA.ID.String()})
	GetPublicUnitLedger(cEmpty)
	if wEmpty.Code != http.StatusOK {
		t.Fatalf("empty GetPublicUnitLedger: got %d, want 200. body=%s", wEmpty.Code, wEmpty.Body.String())
	}

	// 2. AddBankAccount encrypts account_number in DB, rejects duplicates, masks in GetBankAccounts
	acctPayload := []byte(`{"unitId":"` + unitA.ID.String() + `","bankName":"First Bank","accountNumber":"0123456789","accountName":"Unit A Ops","isDefault":true}`)
	cAdd, wAdd := newFinanceTestContext(http.MethodPost, "/api/v1/bank/accounts", acctPayload, headA)
	AddBankAccount(cAdd)
	if wAdd.Code != http.StatusCreated {
		t.Fatalf("AddBankAccount: got %d, want 201. body=%s", wAdd.Code, wAdd.Body.String())
	}

	// Verify DB stores encrypted value, not plaintext
	var rawAcctNumber string
	if err := config.DB.Table("bank_accounts").Select("account_number").Where("unit_id = ?", unitA.ID).Scan(&rawAcctNumber).Error; err != nil {
		t.Fatalf("query raw account_number: %v", err)
	}
	if !strings.HasPrefix(rawAcctNumber, cryptoutil.EncryptedPrefix) || rawAcctNumber == "0123456789" {
		t.Fatalf("account_number was not encrypted in DB: %q", rawAcctNumber)
	}

	// Duplicate account number must return 409 Conflict
	cDup, wDup := newFinanceTestContext(http.MethodPost, "/api/v1/bank/accounts", acctPayload, headA)
	AddBankAccount(cDup)
	if wDup.Code != http.StatusConflict {
		t.Fatalf("duplicate AddBankAccount: got %d, want 409. body=%s", wDup.Code, wDup.Body.String())
	}

	// Cross-unit admin cannot view unitA bank accounts
	cCross, wCross := newFinanceTestContext(http.MethodGet, "/", nil, headB, gin.Param{Key: "id", Value: unitA.ID.String()})
	GetBankAccounts(cCross)
	if wCross.Code != http.StatusForbidden {
		t.Fatalf("cross-unit GetBankAccounts: got %d, want 403", wCross.Code)
	}

	// Owning admin gets masked accountNumber under both "accounts" and "bankAccounts"
	cList, wList := newFinanceTestContext(http.MethodGet, "/", nil, headA, gin.Param{Key: "id", Value: unitA.ID.String()})
	GetBankAccounts(cList)
	if wList.Code != http.StatusOK {
		t.Fatalf("GetBankAccounts: got %d, want 200", wList.Code)
	}
	var listResp struct {
		Accounts []struct {
			AccountNumber string `json:"accountNumber"`
		} `json:"accounts"`
		BankAccounts []struct {
			AccountNumber string `json:"accountNumber"`
		} `json:"bankAccounts"`
	}
	if err := json.Unmarshal(wList.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("unmarshal GetBankAccounts: %v", err)
	}
	if len(listResp.Accounts) != 1 || listResp.Accounts[0].AccountNumber != "******6789" {
		t.Fatalf("unexpected masked accounts: %+v", listResp.Accounts)
	}
	if len(listResp.BankAccounts) != 1 || listResp.BankAccounts[0].AccountNumber != "******6789" {
		t.Fatalf("unexpected masked bankAccounts: %+v", listResp.BankAccounts)
	}

	// 3. Super admin can approve a unit's pending transaction atomically
	tx := models.Transaction{
		UnitID:            unitA.ID,
		Type:              "expense",
		Category:          "operational",
		Amount:            10000,
		Description:       "Patrol fuel",
		InitiatedBy:       headA.ID,
		RequiredApprovals: 1,
		Status:            "pending",
	}
	if err := config.DB.Create(&tx).Error; err != nil {
		t.Fatalf("create transaction: %v", err)
	}

	cApp, wApp := newFinanceTestContext(http.MethodPost, "/", []byte(`{"comment":"approved"}`), super, gin.Param{Key: "id", Value: tx.ID.String()})
	ApproveTransaction(cApp)
	if wApp.Code != http.StatusOK {
		t.Fatalf("super_admin ApproveTransaction: got %d, want 200. body=%s", wApp.Code, wApp.Body.String())
	}

	var ledgerCount int64
	config.DB.Model(&models.FinancialLedger{}).Where("source_type = ? AND source_id = ?", "transaction", tx.ID).Count(&ledgerCount)
	if ledgerCount != 1 {
		t.Fatalf("expected 1 ledger entry for approved tx, got %d", ledgerCount)
	}

	// Re-approving must fail with 400
	cReApp, wReApp := newFinanceTestContext(http.MethodPost, "/", []byte(`{"comment":"again"}`), headA, gin.Param{Key: "id", Value: tx.ID.String()})
	ApproveTransaction(cReApp)
	if wReApp.Code != http.StatusBadRequest {
		t.Fatalf("re-approve completed tx: got %d, want 400", wReApp.Code)
	}
	_ = unitB
}
