package handlers

import (
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"security-solution/config"
	"security-solution/models"
	"security-solution/services"
)

// callerIsUnitAdminOfUnit reports whether user is a super admin or an active
// unit_admin for the given unit.
func callerIsUnitAdminOfUnit(user *models.User, unitID uuid.UUID) bool {
	if user == nil {
		return false
	}
	if user.IsSuperAdmin || user.Role == "super_admin" {
		return true
	}
	if user.Role != "unit_admin" {
		return false
	}
	if user.UnitID != nil && *user.UnitID == unitID {
		return true
	}
	var membership models.UnitMembership
	return config.DB.
		Where("unit_id = ? AND user_id = ? AND status = ?", unitID, user.ID, models.MembershipActive).
		First(&membership).Error == nil
}

func maskAccountNumber(num string) string {
	trimmed := strings.TrimSpace(num)
	if len(trimmed) <= 4 {
		return "****"
	}
	return strings.Repeat("*", len(trimmed)-4) + trimmed[len(trimmed)-4:]
}

type bankAccountResponse struct {
	ID            uuid.UUID `json:"id"`
	UnitID        uuid.UUID `json:"unitId"`
	BankName      string    `json:"bankName"`
	AccountNumber string    `json:"accountNumber"`
	AccountName   string    `json:"accountName"`
	BankCode      string    `json:"bankCode,omitempty"`
	Branch        string    `json:"branch,omitempty"`
	SwiftCode     string    `json:"swiftCode,omitempty"`
	IsVerified    bool      `json:"isVerified"`
	IsDefault     bool      `json:"isDefault"`
	IsPublic      bool      `json:"isPublic"`
	DisplayOrder  int       `json:"displayOrder"`
	Status        string    `json:"status"`
	Notes         string    `json:"notes,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
}

func toBankAccountResponse(a models.BankAccount) bankAccountResponse {
	return bankAccountResponse{
		ID:            a.ID,
		UnitID:        a.UnitID,
		BankName:      a.BankName,
		AccountNumber: maskAccountNumber(a.AccountNumber),
		AccountName:   a.AccountName,
		BankCode:      a.BankCode,
		Branch:        a.Branch,
		SwiftCode:     a.SwiftCode,
		IsVerified:    a.IsVerified,
		IsDefault:     a.IsDefault,
		IsPublic:      a.IsPublic,
		DisplayOrder:  a.DisplayOrder,
		Status:        a.Status,
		Notes:         a.Notes,
		CreatedAt:     a.CreatedAt,
	}
}

// AddBankAccount adds a bank account for a unit
func AddBankAccount(c *gin.Context) {
	var input struct {
		UnitID        string `json:"unitId" binding:"required"`
		BankName      string `json:"bankName" binding:"required"`
		AccountNumber string `json:"accountNumber" binding:"required"`
		AccountName   string `json:"accountName" binding:"required"`
		BankCode      string `json:"bankCode"`
		Branch        string `json:"branch"`
		SwiftCode     string `json:"swiftCode"`
		IsDefault     bool   `json:"isDefault"`
		Notes         string `json:"notes"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj, ok := user.(*models.User)
	if !ok || userObj == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user"})
		return
	}

	unitID, err := uuid.Parse(input.UnitID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid unit ID"})
		return
	}

	if !callerIsUnitAdminOfUnit(userObj, unitID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only unit admins can add bank accounts for this unit"})
		return
	}

	accountNumber := strings.TrimSpace(input.AccountNumber)
	if accountNumber == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Account number is required"})
		return
	}

	// Check if account already exists by comparing decrypted account numbers
	// (AES-GCM uses a random nonce per encryption, so ciphertext equality does not match).
	var existingAccounts []models.BankAccount
	if err := config.DB.Where("unit_id = ? AND status = ?", unitID, "active").Find(&existingAccounts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify existing bank accounts"})
		return
	}
	for _, existing := range existingAccounts {
		if strings.TrimSpace(existing.AccountNumber) == accountNumber {
			c.JSON(http.StatusConflict, gin.H{"error": "This bank account is already registered"})
			return
		}
	}

	bankAccount := models.BankAccount{
		UnitID:        unitID,
		BankName:      input.BankName,
		AccountNumber: accountNumber,
		AccountName:   input.AccountName,
		BankCode:      input.BankCode,
		Branch:        input.Branch,
		SwiftCode:     input.SwiftCode,
		IsDefault:     input.IsDefault,
		Status:        "active",
		Notes:         input.Notes,
		CreatedBy:     userObj.ID,
	}

	if err := config.DB.Transaction(func(tx *gorm.DB) error {
		if input.IsDefault {
			if err := tx.Model(&models.BankAccount{}).Where("unit_id = ?", unitID).Update("is_default", false).Error; err != nil {
				return err
			}
		}
		return tx.Create(&bankAccount).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add bank account"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":     "Bank account added successfully",
		"bankAccount": toBankAccountResponse(bankAccount),
	})
}

// GetBankAccounts gets all bank accounts for a unit
func GetBankAccounts(c *gin.Context) {
	unitID := c.Param("id")
	id, err := uuid.Parse(unitID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid unit ID"})
		return
	}

	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj, ok := user.(*models.User)
	if !ok || userObj == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user"})
		return
	}

	if !callerIsUnitAdminOfUnit(userObj, id) {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to view these accounts"})
		return
	}

	var bankAccounts []models.BankAccount
	if err := config.DB.Where("unit_id = ? AND status = ?", id, "active").Order("created_at desc").Limit(100).Find(&bankAccounts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch bank accounts"})
		return
	}

	out := make([]bankAccountResponse, 0, len(bankAccounts))
	for _, a := range bankAccounts {
		out = append(out, toBankAccountResponse(a))
	}

	c.JSON(http.StatusOK, gin.H{
		"accounts":     out,
		"bankAccounts": out,
	})
}

// UpdateBankAccount updates a bank account
func UpdateBankAccount(c *gin.Context) {
	id := c.Param("id")
	accountID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid account ID"})
		return
	}

	var input struct {
		BankName    string `json:"bankName"`
		AccountName string `json:"accountName"`
		BankCode    string `json:"bankCode"`
		Branch      string `json:"branch"`
		SwiftCode   string `json:"swiftCode"`
		IsDefault   *bool  `json:"isDefault"`
		Notes       string `json:"notes"`
		Status      string `json:"status"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj, ok := user.(*models.User)
	if !ok || userObj == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user"})
		return
	}

	var bankAccount models.BankAccount
	if err := config.DB.First(&bankAccount, "id = ?", accountID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bank account not found"})
		return
	}

	if !callerIsUnitAdminOfUnit(userObj, bankAccount.UnitID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to update this account"})
		return
	}

	if input.BankName != "" {
		bankAccount.BankName = input.BankName
	}
	if input.AccountName != "" {
		bankAccount.AccountName = input.AccountName
	}
	if input.BankCode != "" {
		bankAccount.BankCode = input.BankCode
	}
	if input.Branch != "" {
		bankAccount.Branch = input.Branch
	}
	if input.SwiftCode != "" {
		bankAccount.SwiftCode = input.SwiftCode
	}
	if input.Notes != "" {
		bankAccount.Notes = input.Notes
	}
	if input.Status != "" {
		bankAccount.Status = input.Status
	}
	if input.IsDefault != nil {
		bankAccount.IsDefault = *input.IsDefault
	}

	if err := config.DB.Transaction(func(tx *gorm.DB) error {
		if input.IsDefault != nil && *input.IsDefault {
			if err := tx.Model(&models.BankAccount{}).
				Where("unit_id = ? AND id <> ?", bankAccount.UnitID, bankAccount.ID).
				Update("is_default", false).Error; err != nil {
				return err
			}
		}
		return tx.Save(&bankAccount).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update bank account"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Bank account updated successfully",
		"bankAccount": toBankAccountResponse(bankAccount),
	})
}

// DeleteBankAccount deletes a bank account (soft delete)
func DeleteBankAccount(c *gin.Context) {
	id := c.Param("id")
	accountID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid account ID"})
		return
	}

	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj, ok := user.(*models.User)
	if !ok || userObj == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user"})
		return
	}

	var bankAccount models.BankAccount
	if err := config.DB.First(&bankAccount, "id = ?", accountID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bank account not found"})
		return
	}

	if !callerIsUnitAdminOfUnit(userObj, bankAccount.UnitID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to delete this account"})
		return
	}

	// Soft delete
	bankAccount.Status = "inactive"
	if err := config.DB.Save(&bankAccount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete bank account"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Bank account deleted successfully",
	})
}

// RecordDonation records a donation made to a unit
func RecordDonation(c *gin.Context) {
	var input struct {
		UnitID        string  `json:"unitId" binding:"required"`
		DonorName     string  `json:"donorName" binding:"required"`
		DonorEmail    string  `json:"donorEmail"`
		DonorPhone    string  `json:"donorPhone"`
		Amount        float64 `json:"amount"`
		PaymentMethod string  `json:"paymentMethod"`
		Reference     string  `json:"reference"`
		Notes         string  `json:"notes"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj := user.(*models.User)

	unitID, err := uuid.Parse(input.UnitID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid unit ID"})
		return
	}

	var unit models.SecurityUnit
	if err := config.DB.First(&unit, "id = ?", unitID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Unit not found"})
		return
	}

	if !callerIsHeadAdminOfUnit(userObj, unit.ID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only the unit's verified head admin can record donations"})
		return
	}

	if !validDonationAmount(input.Amount) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "amount must be a positive integer"})
		return
	}

	input.Reference = strings.TrimSpace(input.Reference)
	if input.Reference == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "reference is required"})
		return
	}

	input.PaymentMethod = strings.TrimSpace(input.PaymentMethod)
	if input.PaymentMethod == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "paymentMethod is required"})
		return
	}

	now := time.Now()
	donation := models.Donation{
		ID:            uuid.New(),
		UnitID:        unit.ID,
		DonorName:     input.DonorName,
		DonorEmail:    input.DonorEmail,
		DonorPhone:    input.DonorPhone,
		Amount:        input.Amount,
		PaymentMethod: input.PaymentMethod,
		Reference:     input.Reference,
		Status:        "confirmed",
		DonationDate:  now,
		Notes:         input.Notes,
		ConfirmedBy:   &userObj.ID,
		ConfirmedAt:   &now,
	}
	transaction := models.Transaction{
		UnitID:            unit.ID,
		Type:              "donation",
		Category:          "income",
		Amount:            input.Amount,
		Description:       "Donation from " + input.DonorName,
		InitiatedBy:       userObj.ID,
		ApprovedBy:        &userObj.ID,
		ApprovalCount:     1,
		RequiredApprovals: 1,
		Status:            "approved",
		TransactionDate:   now,
		PaymentMethod:     input.PaymentMethod,
		ReferenceID:       input.Reference,
	}

	ledgerSvc := services.NewLedgerService()
	var ledger *models.FinancialLedger
	if err := config.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&donation).Error; err != nil {
			return err
		}
		if err := tx.Create(&transaction).Error; err != nil {
			return err
		}
		var err error
		ledger, err = ledgerSvc.AppendTx(tx, services.LedgerEntryInput{
			UnitID:       unit.ID,
			Direction:    "in",
			EntryType:    "donation",
			Amount:       input.Amount,
			Counterparty: input.DonorName,
			Description:  "Donation from " + input.DonorName,
			SourceType:   "donation",
			SourceID:     &donation.ID,
			CreatedBy:    userObj.ID,
		})
		return err
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record donation"})
		return
	}
	ledgerSvc.NotifyAdmins(ledger)

	c.JSON(http.StatusCreated, gin.H{
		"message":  "Donation recorded and confirmed",
		"donation": donation,
	})
}

func validDonationAmount(amount float64) bool {
	return !math.IsNaN(amount) && !math.IsInf(amount, 0) && amount > 0 && math.Trunc(amount) == amount
}

// ConfirmDonation confirms a donation (admin only)
func ConfirmDonation(c *gin.Context) {
	id := c.Param("id")
	donationID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid donation ID"})
		return
	}

	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj := user.(*models.User)

	var donation models.Donation
	if err := config.DB.First(&donation, "id = ?", donationID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Donation not found"})
		return
	}

	if !callerIsHeadAdminOfUnit(userObj, donation.UnitID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only the unit's verified head admin can confirm donations"})
		return
	}

	if donation.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Donation is no longer pending"})
		return
	}

	if !validDonationAmount(donation.Amount) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Donation amount is invalid"})
		return
	}

	ledgerSvc := services.NewLedgerService()
	var transaction models.Transaction
	var ledger *models.FinancialLedger
	alreadyConfirmed := false
	if err := config.DB.Transaction(func(tx *gorm.DB) error {
		var locked models.Donation
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&locked, "id = ?", donationID).Error; err != nil {
			return err
		}
		if locked.Status != "pending" {
			alreadyConfirmed = true
			return nil
		}

		now := time.Now()
		locked.Status = "confirmed"
		locked.ConfirmedBy = &userObj.ID
		locked.ConfirmedAt = &now
		if err := tx.Save(&locked).Error; err != nil {
			return err
		}
		donation = locked

		transaction = models.Transaction{
			UnitID:            donation.UnitID,
			Type:              "donation",
			Category:          "income",
			Amount:            donation.Amount,
			Description:       "Donation from " + donation.DonorName,
			InitiatedBy:       userObj.ID,
			ApprovedBy:        &userObj.ID,
			ApprovalCount:     1,
			RequiredApprovals: 1,
			Status:            "approved",
			TransactionDate:   now,
			PaymentMethod:     donation.PaymentMethod,
			ReferenceID:       donation.Reference,
		}
		if err := tx.Create(&transaction).Error; err != nil {
			return err
		}

		var err error
		ledger, err = ledgerSvc.AppendTx(tx, services.LedgerEntryInput{
			UnitID:       donation.UnitID,
			Direction:    "in",
			EntryType:    "donation",
			Amount:       donation.Amount,
			Counterparty: donation.DonorName,
			Description:  "Donation from " + donation.DonorName,
			SourceType:   "donation",
			SourceID:     &donation.ID,
			CreatedBy:    userObj.ID,
		})
		return err
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to confirm donation"})
		return
	}
	if alreadyConfirmed {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Donation is no longer pending"})
		return
	}
	ledgerSvc.NotifyAdmins(ledger)

	c.JSON(http.StatusOK, gin.H{
		"message":     "Donation confirmed successfully",
		"donation":    donation,
		"transaction": transaction,
	})
}

// GetDonations gets all donations for a unit
func GetDonations(c *gin.Context) {
	unitID := c.Param("id")
	id, err := uuid.Parse(unitID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid unit ID"})
		return
	}

	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj, ok := user.(*models.User)
	if !ok || userObj == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user"})
		return
	}

	if !callerIsUnitAdminOfUnit(userObj, id) {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to view these donations"})
		return
	}

	var donations []models.Donation
	if err := config.DB.Where("unit_id = ?", id).Order("created_at desc").Limit(200).Find(&donations).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch donations"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"donations": donations,
	})
}
