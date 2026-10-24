package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"security-solution/config"
	"security-solution/models"
	"security-solution/services"
)

// CreateTransaction creates a new transaction with multi-signature approval
func CreateTransaction(c *gin.Context) {
	var input struct {
		UnitID          string  `json:"unitId" binding:"required"`
		Type            string  `json:"type" binding:"required"`
		Category        string  `json:"category"`
		Amount          float64 `json:"amount" binding:"required"`
		Description     string  `json:"description" binding:"required"`
		PaymentMethod   string  `json:"paymentMethod"`
		ReferenceID     string  `json:"referenceId"`
		ReceiptURL      string  `json:"receiptUrl"`
		Notes           string  `json:"notes"`
		TransactionDate string  `json:"transactionDate"`
		DonorName       string  `json:"donorName"`
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
		c.JSON(http.StatusForbidden, gin.H{"error": "Only admins can create transactions"})
		return
	}

	unitID, err := uuid.Parse(input.UnitID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid unit ID"})
		return
	}

	if userObj.Role == "unit_admin" {
		if userObj.UnitID == nil || *userObj.UnitID != unitID {
			c.JSON(http.StatusForbidden, gin.H{"error": "You can only create transactions for your unit"})
			return
		}
	}

	requiredApprovals := 2
	if input.Amount > 1000000 {
		requiredApprovals = 3
	}
	if input.Amount > 5000000 {
		requiredApprovals = 4
	}
	if userObj.Role == "super_admin" {
		requiredApprovals = 1
	}

	transactionDate := time.Now()
	if input.TransactionDate != "" {
		parsed, err := time.Parse(time.RFC3339, input.TransactionDate)
		if err == nil {
			transactionDate = parsed
		}
	}

	if input.Category == "" {
		input.Category = "operational"
	}

	transaction := models.Transaction{
		UnitID:            unitID,
		Type:              input.Type,
		Category:          input.Category,
		Amount:            input.Amount,
		Description:       input.Description,
		InitiatedBy:       userObj.ID,
		RequiredApprovals: requiredApprovals,
		Status:            "pending",
		TransactionDate:   transactionDate,
		PaymentMethod:     input.PaymentMethod,
		ReferenceID:       input.ReferenceID,
		ReceiptURL:        input.ReceiptURL,
		Notes:             input.Notes,
	}

	if err := config.DB.Create(&transaction).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create transaction"})
		return
	}

	go checkBudgetAlert(unitID, input.Category, input.Amount)

	c.JSON(http.StatusCreated, gin.H{
		"message":           "Transaction created successfully",
		"transaction":       transaction,
		"requiredApprovals": requiredApprovals,
	})
}

// ApproveTransaction approves a transaction
func ApproveTransaction(c *gin.Context) {
	id := c.Param("id")
	transactionID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid transaction ID"})
		return
	}

	var input struct {
		Comment string `json:"comment"`
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
		c.JSON(http.StatusForbidden, gin.H{"error": "Only admins can approve transactions"})
		return
	}

	var transaction models.Transaction
	if err := config.DB.First(&transaction, "id = ?", transactionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Transaction not found"})
		return
	}

	if transaction.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Transaction is no longer pending"})
		return
	}

	if userObj.UnitID == nil || *userObj.UnitID != transaction.UnitID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only approve transactions from your unit"})
		return
	}

	var existingApproval models.TransactionApproval
	if err := config.DB.Where("transaction_id = ? AND approver_id = ?", transaction.ID, userObj.ID).First(&existingApproval).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "You have already approved this transaction"})
		return
	}

	approval := models.TransactionApproval{
		TransactionID: transaction.ID,
		ApproverID:    userObj.ID,
		UnitID:        transaction.UnitID,
		Status:        "approved",
		Comment:       input.Comment,
	}

	if err := config.DB.Create(&approval).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record approval"})
		return
	}

	transaction.ApprovalCount++
	config.DB.Save(&transaction)

	if transaction.ApprovalCount >= transaction.RequiredApprovals {
		transaction.Status = "approved"
		transaction.ApprovedBy = &userObj.ID
		config.DB.Save(&transaction)

		if transaction.Type == "expense" || transaction.Type == "salary" || transaction.Type == "maintenance" {
			var budget models.Budget
			if err := config.DB.Where("unit_id = ? AND category = ? AND status = ?",
				transaction.UnitID, transaction.Category, "active").First(&budget).Error; err == nil {
				budget.Spent += transaction.Amount
				budget.Remaining = budget.Amount - budget.Spent
				config.DB.Save(&budget)
			}
		}

		ledgerSvc := services.NewLedgerService()
		txID := transaction.ID
		_, _ = ledgerSvc.Append(services.LedgerEntryInput{
			UnitID:       transaction.UnitID,
			Direction:    "out",
			EntryType:    transaction.Type,
			Amount:       transaction.Amount,
			Counterparty: transaction.Description,
			Description:  transaction.Description,
			SourceType:   "transaction",
			SourceID:     &txID,
			CreatedBy:    userObj.ID,
		})

		c.JSON(http.StatusOK, gin.H{
			"message":           "Transaction fully approved!",
			"transaction":       transaction,
			"approvalCount":     transaction.ApprovalCount,
			"requiredApprovals": transaction.RequiredApprovals,
			"completed":         true,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":            "Transaction approved! Waiting for more approvals.",
		"transaction":        transaction,
		"approvalCount":      transaction.ApprovalCount,
		"requiredApprovals":  transaction.RequiredApprovals,
		"remainingApprovals": transaction.RequiredApprovals - transaction.ApprovalCount,
		"completed":          false,
	})
}

// RejectTransaction rejects a transaction
func RejectTransaction(c *gin.Context) {
	id := c.Param("id")
	transactionID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid transaction ID"})
		return
	}

	var input struct {
		Comment string `json:"comment" binding:"required"`
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
		c.JSON(http.StatusForbidden, gin.H{"error": "Only admins can reject transactions"})
		return
	}

	var transaction models.Transaction
	if err := config.DB.First(&transaction, "id = ?", transactionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Transaction not found"})
		return
	}

	if transaction.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Transaction is no longer pending"})
		return
	}

	if userObj.UnitID == nil || *userObj.UnitID != transaction.UnitID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only reject transactions from your unit"})
		return
	}

	transaction.Status = "rejected"
	config.DB.Save(&transaction)

	c.JSON(http.StatusOK, gin.H{
		"message":     "Transaction rejected",
		"transaction": transaction,
	})
}

// GetTransactions gets transactions for a unit (returns pending transactions by default)
func GetTransactions(c *gin.Context) {
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
			c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to view these transactions"})
			return
		}
	}

	statusFilter := c.Query("status")
	if statusFilter == "" {
		statusFilter = "pending"
	}

	var transactions []models.Transaction
	query := config.DB.Preload("Initiator").Preload("Approver").Where("unit_id = ?", id)
	if statusFilter != "all" {
		query = query.Where("status = ?", statusFilter)
	}
	if err := query.Order("created_at desc").Find(&transactions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch transactions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"transactions": transactions,
	})
}

// GetTransactionByID gets a specific transaction
func GetTransactionByID(c *gin.Context) {
	id := c.Param("id")
	transactionID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid transaction ID"})
		return
	}

	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj := user.(*models.User)

	var transaction models.Transaction
	if err := config.DB.Preload("Initiator").Preload("Approver").First(&transaction, "id = ?", transactionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Transaction not found"})
		return
	}

	if userObj.Role != "super_admin" {
		if userObj.UnitID == nil || *userObj.UnitID != transaction.UnitID {
			c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to view this transaction"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"transaction": transaction,
	})
}

// GetTransactionSummary gets transaction summary for a unit
func GetTransactionSummary(c *gin.Context) {
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
			c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to view this summary"})
			return
		}
	}

	var totalIncome float64
	var totalExpenses float64
	var pendingCount int64

	config.DB.Model(&models.Transaction{}).
		Where("unit_id = ? AND status = ? AND type IN (?)", id, "approved", []string{"donation", "gift", "tax"}).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&totalIncome)

	config.DB.Model(&models.Transaction{}).
		Where("unit_id = ? AND status = ? AND type IN (?)", id, "approved", []string{"expense", "salary", "maintenance"}).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&totalExpenses)

	config.DB.Model(&models.Transaction{}).
		Where("unit_id = ? AND status = ?", id, "pending").
		Count(&pendingCount)

	balance := totalIncome - totalExpenses

	var categoryBreakdown []struct {
		Category string
		Total    float64
	}
	config.DB.Model(&models.Transaction{}).
		Select("category, SUM(amount) as total").
		Where("unit_id = ? AND status = ?", id, "approved").
		Group("category").
		Scan(&categoryBreakdown)

	var monthlyTrends []struct {
		Month   string
		Income  float64
		Expense float64
	}
	config.DB.Model(&models.Transaction{}).
		Select("to_char(transaction_date, 'YYYY-MM') as month, "+
			"SUM(CASE WHEN type IN ('donation','gift','tax') THEN amount ELSE 0 END) as income, "+
			"SUM(CASE WHEN type IN ('expense','salary','maintenance') THEN amount ELSE 0 END) as expense").
		Where("unit_id = ? AND status = ?", id, "approved").
		Group("month").
		Order("month desc").
		Limit(6).
		Scan(&monthlyTrends)

	c.JSON(http.StatusOK, gin.H{
		"totalIncome":       totalIncome,
		"totalExpenses":     totalExpenses,
		"balance":           balance,
		"pendingCount":      pendingCount,
		"categoryBreakdown": categoryBreakdown,
		"monthlyTrends":     monthlyTrends,
	})
}

// CreateBudget creates a budget for a unit
func CreateBudget(c *gin.Context) {
	var input struct {
		UnitID   string  `json:"unitId" binding:"required"`
		Category string  `json:"category" binding:"required"`
		Amount   float64 `json:"amount" binding:"required"`
		Period   string  `json:"period" binding:"required"`
		Year     int     `json:"year" binding:"required"`
		Month    int     `json:"month"`
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
		c.JSON(http.StatusForbidden, gin.H{"error": "Only admins can create budgets"})
		return
	}

	unitID, err := uuid.Parse(input.UnitID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid unit ID"})
		return
	}

	if userObj.Role == "unit_admin" {
		if userObj.UnitID == nil || *userObj.UnitID != unitID {
			c.JSON(http.StatusForbidden, gin.H{"error": "You can only create budgets for your unit"})
			return
		}
	}

	budget := models.Budget{
		UnitID:    unitID,
		Category:  input.Category,
		Amount:    input.Amount,
		Spent:     0,
		Remaining: input.Amount,
		Period:    input.Period,
		Year:      input.Year,
		Month:     input.Month,
		Status:    "active",
		CreatedBy: userObj.ID,
	}

	if err := config.DB.Create(&budget).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create budget"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Budget created successfully",
		"budget":  budget,
	})
}

// GetBudgets gets all budgets for a unit
func GetBudgets(c *gin.Context) {
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
			c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to view these budgets"})
			return
		}
	}

	var budgets []models.Budget
	if err := config.DB.Where("unit_id = ?", id).Order("created_at desc").Find(&budgets).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch budgets"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"budgets": budgets,
	})
}

// GenerateFinancialReport generates a financial report
func GenerateFinancialReport(c *gin.Context) {
	var input struct {
		UnitID      string `json:"unitId" binding:"required"`
		Title       string `json:"title" binding:"required"`
		Type        string `json:"type" binding:"required"`
		PeriodStart string `json:"periodStart" binding:"required"`
		PeriodEnd   string `json:"periodEnd" binding:"required"`
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
		c.JSON(http.StatusForbidden, gin.H{"error": "Only admins can generate reports"})
		return
	}

	unitID, err := uuid.Parse(input.UnitID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid unit ID"})
		return
	}

	if userObj.Role == "unit_admin" {
		if userObj.UnitID == nil || *userObj.UnitID != unitID {
			c.JSON(http.StatusForbidden, gin.H{"error": "You can only generate reports for your unit"})
			return
		}
	}

	periodStart, err := time.Parse(time.RFC3339, input.PeriodStart)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid period start"})
		return
	}

	periodEnd, err := time.Parse(time.RFC3339, input.PeriodEnd)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid period end"})
		return
	}

	var totalIncome float64
	var totalExpenses float64

	config.DB.Model(&models.Transaction{}).
		Where("unit_id = ? AND status = ? AND type IN (?) AND created_at BETWEEN ? AND ?",
			unitID, "approved", []string{"donation", "gift", "tax"}, periodStart, periodEnd).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&totalIncome)

	config.DB.Model(&models.Transaction{}).
		Where("unit_id = ? AND status = ? AND type IN (?) AND created_at BETWEEN ? AND ?",
			unitID, "approved", []string{"expense", "salary", "maintenance"}, periodStart, periodEnd).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&totalExpenses)

	balance := totalIncome - totalExpenses

	report := models.FinancialReport{
		UnitID:        unitID,
		Title:         input.Title,
		Type:          input.Type,
		PeriodStart:   periodStart,
		PeriodEnd:     periodEnd,
		TotalIncome:   totalIncome,
		TotalExpenses: totalExpenses,
		Balance:       balance,
		GeneratedBy:   userObj.ID,
		Status:        "generated",
	}

	if err := config.DB.Create(&report).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate report"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Report generated successfully",
		"report":  report,
	})
}

// GetFinancialReports gets all reports for a unit
func GetFinancialReports(c *gin.Context) {
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
			c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to view these reports"})
			return
		}
	}

	var reports []models.FinancialReport
	if err := config.DB.Preload("Generator").Where("unit_id = ?", id).Order("created_at desc").Find(&reports).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch reports"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"reports": reports,
	})
}

// Helper: Check budget alert
func checkBudgetAlert(unitID uuid.UUID, category string, amount float64) {
	var budget models.Budget
	if err := config.DB.Where("unit_id = ? AND category = ? AND status = ?", unitID, category, "active").First(&budget).Error; err != nil {
		return
	}

	percentageUsed := (budget.Spent + amount) / budget.Amount * 100
	if percentageUsed >= 80 {
		var admins []models.User
		config.DB.Where("unit_id = ? AND role IN (?)", unitID, []string{"unit_admin", "super_admin"}).Find(&admins)

		for _, admin := range admins {
			notification := models.Notification{
				UserID: admin.ID,
				Title:  "⚠️ Budget Alert",
				Message: fmt.Sprintf("Budget for %s is at %.0f%% used. Current: %.2f, Budget: %.2f",
					category, percentageUsed, budget.Spent+amount, budget.Amount),
				Type:   "budget_alert",
				Status: "unread",
			}
			config.DB.Create(&notification)
		}
	}
}

// GetLedgerBalance computes the net balance from FinancialLedger for a unit
func GetLedgerBalance(c *gin.Context) {
	unitIDParam := c.Param("id")
	unitID, err := uuid.Parse(unitIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid unit ID"})
		return
	}

	var totalIn, totalOut float64
	config.DB.Model(&models.FinancialLedger{}).
		Select("COALESCE(SUM(CASE WHEN direction = 'in' THEN amount ELSE 0 END), 0)").
		Where("unit_id = ? AND status = ?", unitID, "posted").
		Scan(&totalIn)

	config.DB.Model(&models.FinancialLedger{}).
		Select("COALESCE(SUM(CASE WHEN direction = 'out' THEN amount ELSE 0 END), 0)").
		Where("unit_id = ? AND status = ?", unitID, "posted").
		Scan(&totalOut)

	balance := totalIn - totalOut

	c.JSON(http.StatusOK, gin.H{
		"unitId":   unitID,
		"balance":  balance,
		"totalIn":  totalIn,
		"totalOut": totalOut,
		"currency": "NGN",
	})
}

// DonationWebhook handles incoming donation webhooks idempotently
func DonationWebhook(c *gin.Context) {
	var input struct {
		EventID     string  `json:"eventId" binding:"required"`
		Provider    string  `json:"provider" binding:"required"`
		EventType   string  `json:"eventType"`
		UnitID      string  `json:"unitId" binding:"required"`
		Amount      float64 `json:"amount" binding:"required"`
		DonorName   string  `json:"donorName"`
		DonorEmail  string  `json:"donorEmail"`
		Reference   string  `json:"reference"`
		Description string  `json:"description"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	unitID, err := uuid.Parse(input.UnitID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid unit ID"})
		return
	}

	// Check idempotency in WebhookEvent table
	var existing models.WebhookEvent
	if err := config.DB.Where("idempotency_key = ?", input.EventID).First(&existing).Error; err == nil {
		c.JSON(http.StatusOK, gin.H{
			"message":        "Webhook already processed (idempotent)",
			"idempotencyKey": input.EventID,
			"status":         "success",
		})
		return
	}

	now := time.Now().UTC()
	webhookEvent := models.WebhookEvent{
		IdempotencyKey: input.EventID,
		Provider:       input.Provider,
		EventType:      input.EventType,
		Payload:        "{}",
		Status:         "processed",
		ProcessedAt:    &now,
	}

	if err := config.DB.Create(&webhookEvent).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "Webhook already processed"})
		return
	}

	var adminUser models.User
	if err := config.DB.Where("unit_id = ? AND role IN (?)", unitID, []string{"unit_admin", "super_admin"}).First(&adminUser).Error; err != nil {
		if err := config.DB.Where("role = ?", "super_admin").First(&adminUser).Error; err != nil {
			var anyUser models.User
			if err := config.DB.First(&anyUser).Error; err == nil {
				adminUser = anyUser
			}
		}
	}
	createdBy := adminUser.ID
	if createdBy == uuid.Nil {
		createdBy = uuid.New()
	}

	donorName := input.DonorName
	if donorName == "" {
		donorName = "Anonymous Donor"
	}
	description := input.Description
	if description == "" {
		description = "Donation via " + input.Provider
	}

	ledgerSvc := services.NewLedgerService()
	entryInput := services.LedgerEntryInput{
		UnitID:       unitID,
		Direction:    "in",
		EntryType:    "donation",
		Amount:       input.Amount,
		Counterparty: donorName,
		Description:  description,
		SourceType:   "donation",
		SourceID:     &webhookEvent.ID,
		CreatedBy:    createdBy,
	}

	ledgerEntry, err := ledgerSvc.Append(entryInput)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to append to ledger: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":      "Donation webhook processed successfully",
		"ledgerEntry":  ledgerEntry,
		"webhookEvent": webhookEvent,
	})
}
