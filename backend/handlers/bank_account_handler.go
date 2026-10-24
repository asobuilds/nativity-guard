package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"security-solution/config"
	"security-solution/cryptoutil"
	"security-solution/models"
	"security-solution/services"
)

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
	userObj := user.(*models.User)

	// Only admins can add bank accounts
	if userObj.Role != "super_admin" && userObj.Role != "unit_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only admins can add bank accounts"})
		return
	}

	unitID, err := uuid.Parse(input.UnitID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid unit ID"})
		return
	}

	if userObj.Role == "unit_admin" {
		if userObj.UnitID == nil || *userObj.UnitID != unitID {
			c.JSON(http.StatusForbidden, gin.H{"error": "You can only add accounts for your unit"})
			return
		}
	}

	// Check if account already exists (compare against encrypted value)
	var existing models.BankAccount
	encAccountNum, _ := cryptoutil.Encrypt(input.AccountNumber)
	if err := config.DB.Where("unit_id = ? AND account_number = ?", unitID, encAccountNum).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "This bank account is already registered"})
		return
	}

	// If this is set as default, unset other defaults
	if input.IsDefault {
		config.DB.Model(&models.BankAccount{}).Where("unit_id = ?", unitID).Update("is_default", false)
	}

	bankAccount := models.BankAccount{
		UnitID:        unitID,
		BankName:      input.BankName,
		AccountNumber: input.AccountNumber,
		AccountName:   input.AccountName,
		BankCode:      input.BankCode,
		Branch:        input.Branch,
		SwiftCode:     input.SwiftCode,
		IsDefault:     input.IsDefault,
		Status:        "active",
		Notes:         input.Notes,
		CreatedBy:     userObj.ID,
	}

	if err := config.DB.Create(&bankAccount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add bank account"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":     "Bank account added successfully",
		"bankAccount": bankAccount,
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
	userObj := user.(*models.User)

	if userObj.Role != "super_admin" {
		if userObj.UnitID == nil || *userObj.UnitID != id {
			c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to view these accounts"})
			return
		}
	}

	var bankAccounts []models.BankAccount
	if err := config.DB.Where("unit_id = ? AND status = ?", id, "active").Order("created_at desc").Find(&bankAccounts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch bank accounts"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"bankAccounts": bankAccounts,
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
	userObj := user.(*models.User)

	if userObj.Role != "super_admin" && userObj.Role != "unit_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only admins can update bank accounts"})
		return
	}

	var bankAccount models.BankAccount
	if err := config.DB.First(&bankAccount, "id = ?", accountID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bank account not found"})
		return
	}

	if userObj.Role == "unit_admin" {
		if userObj.UnitID == nil || *userObj.UnitID != bankAccount.UnitID {
			c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to update this account"})
			return
		}
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
	if input.IsDefault != nil && *input.IsDefault {
		// Unset other defaults for this unit
		config.DB.Model(&models.BankAccount{}).Where("unit_id = ?", bankAccount.UnitID).Update("is_default", false)
		bankAccount.IsDefault = true
	} else if input.IsDefault != nil && !*input.IsDefault {
		bankAccount.IsDefault = false
	}

	if err := config.DB.Save(&bankAccount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update bank account"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Bank account updated successfully",
		"bankAccount": bankAccount,
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
	userObj := user.(*models.User)

	if userObj.Role != "super_admin" && userObj.Role != "unit_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only admins can delete bank accounts"})
		return
	}

	var bankAccount models.BankAccount
	if err := config.DB.First(&bankAccount, "id = ?", accountID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bank account not found"})
		return
	}

	if userObj.Role == "unit_admin" {
		if userObj.UnitID == nil || *userObj.UnitID != bankAccount.UnitID {
			c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to delete this account"})
			return
		}
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
		Amount        float64 `json:"amount" binding:"required"`
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

	// Anyone can record a donation (citizen or admin)
	// But if admin, they can confirm immediately

	donation := models.Donation{
		UnitID:        unitID,
		DonorName:     input.DonorName,
		DonorEmail:    input.DonorEmail,
		DonorPhone:    input.DonorPhone,
		Amount:        input.Amount,
		PaymentMethod: input.PaymentMethod,
		Reference:     input.Reference,
		Status:        "pending",
		DonationDate:  time.Now(),
		Notes:         input.Notes,
	}

	if donation.PaymentMethod == "" {
		donation.PaymentMethod = "bank_transfer"
	}

	if err := config.DB.Create(&donation).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record donation"})
		return
	}

	// If user is admin, auto-confirm
	if userObj.Role == "super_admin" || userObj.Role == "unit_admin" {
		now := time.Now()
		donation.Status = "confirmed"
		donation.ConfirmedBy = &userObj.ID
		donation.ConfirmedAt = &now
		config.DB.Save(&donation)

		// Create transaction record
		transaction := models.Transaction{
			UnitID:            unitID,
			Type:              "donation",
			Category:          "income",
			Amount:            input.Amount,
			Description:       "Donation from " + input.DonorName,
			InitiatedBy:       userObj.ID,
			ApprovedBy:        &userObj.ID,
			ApprovalCount:     1,
			RequiredApprovals: 1,
			Status:            "approved",
			TransactionDate:   time.Now(),
			PaymentMethod:     input.PaymentMethod,
			ReferenceID:       input.Reference,
		}
		config.DB.Create(&transaction)

		// Stamp the immutable ledger
		ledgerSvc := services.NewLedgerService()
		donationID := donation.ID
		_, _ = ledgerSvc.Append(services.LedgerEntryInput{
			UnitID:       unitID,
			Direction:    "in",
			EntryType:    "donation",
			Amount:       input.Amount,
			Counterparty: input.DonorName,
			Description:  "Donation from " + input.DonorName,
			SourceType:   "donation",
			SourceID:     &donationID,
			CreatedBy:    userObj.ID,
		})

		c.JSON(http.StatusCreated, gin.H{
			"message":  "Donation recorded and confirmed",
			"donation": donation,
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":  "Donation recorded successfully. Awaiting confirmation.",
		"donation": donation,
	})
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

	if userObj.Role != "super_admin" && userObj.Role != "unit_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only admins can confirm donations"})
		return
	}

	var donation models.Donation
	if err := config.DB.First(&donation, "id = ?", donationID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Donation not found"})
		return
	}

	if donation.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Donation is no longer pending"})
		return
	}

	if userObj.Role == "unit_admin" {
		if userObj.UnitID == nil || *userObj.UnitID != donation.UnitID {
			c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to confirm this donation"})
			return
		}
	}

	now := time.Now()
	donation.Status = "confirmed"
	donation.ConfirmedBy = &userObj.ID
	donation.ConfirmedAt = &now

	if err := config.DB.Save(&donation).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to confirm donation"})
		return
	}

	// Create transaction record
	transaction := models.Transaction{
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
		TransactionDate:   time.Now(),
		PaymentMethod:     donation.PaymentMethod,
		ReferenceID:       donation.Reference,
	}
	config.DB.Create(&transaction)

	// Stamp the immutable ledger
	ledgerSvc := services.NewLedgerService()
	donationID = donation.ID
	_, _ = ledgerSvc.Append(services.LedgerEntryInput{
		UnitID:       donation.UnitID,
		Direction:    "in",
		EntryType:    "donation",
		Amount:       donation.Amount,
		Counterparty: donation.DonorName,
		Description:  "Donation from " + donation.DonorName,
		SourceType:   "donation",
		SourceID:     &donationID,
		CreatedBy:    userObj.ID,
	})

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
	userObj := user.(*models.User)

	if userObj.Role != "super_admin" {
		if userObj.UnitID == nil || *userObj.UnitID != id {
			c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to view these donations"})
			return
		}
	}

	var donations []models.Donation
	if err := config.DB.Where("unit_id = ?", id).Order("created_at desc").Find(&donations).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch donations"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"donations": donations,
	})
}
