package services

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"security-solution/config"
	"security-solution/models"
)

type LedgerService struct{}

func NewLedgerService() *LedgerService {
	return &LedgerService{}
}

// LedgerEntryInput describes a new ledger entry to append.
type LedgerEntryInput struct {
	UnitID       uuid.UUID
	Direction    string  // "in" or "out"
	EntryType    string  // donation, expense, salary, refund, adjustment, ...
	Amount       float64 // positive magnitude; direction carries the sign
	Counterparty string
	Description  string
	SourceType   string // donation, transaction, manual
	SourceID     *uuid.UUID
	Metadata     string // optional JSON blob
	CreatedBy    uuid.UUID
}

// Append writes a new ledger entry. Append-only: never updates or deletes.
// Reference format: NG-YYYY-NNNNNNNN (year-scoped, monotonic).
func (s *LedgerService) Append(entry LedgerEntryInput) (*models.FinancialLedger, error) {
	if err := validateLedgerEntry(entry); err != nil {
		return nil, err
	}
	ledger, err := s.appendWithRetry(entry, 3)
	if err != nil {
		return nil, err
	}
	s.NotifyAdmins(ledger)
	return ledger, nil
}

func (s *LedgerService) AppendTx(tx *gorm.DB, entry LedgerEntryInput) (*models.FinancialLedger, error) {
	if tx == nil {
		return nil, errors.New("transaction is required")
	}
	if err := validateLedgerEntry(entry); err != nil {
		return nil, err
	}
	return s.appendToDB(tx, entry)
}

func (s *LedgerService) NotifyAdmins(ledger *models.FinancialLedger) {
	if ledger == nil {
		return
	}
	go func(uID uuid.UUID, eType, dir string, amt float64, curr string) {
		var admins []models.User
		config.DB.Where("unit_id = ? AND role IN (?)", uID, []string{"unit_admin", "super_admin"}).Find(&admins)
		for _, admin := range admins {
			notif := models.Notification{
				UserID:     admin.ID,
				Title:      "New Financial Ledger Entry",
				Message:    fmt.Sprintf("A new %s entry (%s) of %s %.2f was recorded.", eType, dir, curr, amt),
				Type:       "finance",
				Status:     "unread",
				EntityType: "unit",
				EntityID:   uID,
				LinkTo:     fmt.Sprintf("/units/%s/ledger", uID.String()),
			}
			config.DB.Create(&notif)
		}
	}(ledger.UnitID, ledger.EntryType, ledger.Direction, ledger.Amount, ledger.Currency)
}

func validateLedgerEntry(entry LedgerEntryInput) error {
	if entry.UnitID == uuid.Nil {
		return errors.New("unitId is required")
	}
	if entry.Amount <= 0 {
		return errors.New("amount must be positive")
	}
	if entry.Direction != "in" && entry.Direction != "out" {
		return errors.New("direction must be 'in' or 'out'")
	}
	if entry.EntryType == "" {
		return errors.New("entryType is required")
	}
	if entry.CreatedBy == uuid.Nil {
		return errors.New("createdBy is required")
	}
	return nil
}

func (s *LedgerService) appendWithRetry(entry LedgerEntryInput, remaining int) (*models.FinancialLedger, error) {
	var ledger *models.FinancialLedger
	err := config.DB.Transaction(func(tx *gorm.DB) error {
		var err error
		ledger, err = s.appendToDB(tx, entry)
		return err
	})
	if err != nil {
		if remaining <= 1 {
			return nil, err
		}
		return s.appendWithRetry(entry, remaining-1)
	}
	return ledger, nil
}

func (s *LedgerService) appendToDB(tx *gorm.DB, entry LedgerEntryInput) (*models.FinancialLedger, error) {
	now := time.Now().UTC()
	year := now.Year()

	// Primary correctness: atomic counter increment inside the same
	// transaction as the ledger insert. The INSERT ... ON CONFLICT
	// ensures the counter row exists, and UPDATE ... RETURNING
	// serializes concurrent appends for the same year.
	// The 3-retry loop in appendWithRetry is defense-in-depth only.
	var seq int64
	if err := tx.Exec(
		"INSERT INTO ledger_sequences(year, last_seq) VALUES (?, 0) ON CONFLICT (year) DO NOTHING",
		year,
	).Error; err != nil {
		return nil, err
	}
	if err := tx.Raw(
		"UPDATE ledger_sequences SET last_seq = last_seq + 1 WHERE year = ? RETURNING last_seq",
		year,
	).Scan(&seq).Error; err != nil {
		return nil, err
	}
	ledger := &models.FinancialLedger{
		Reference:      fmt.Sprintf("NG-%d-%08d", year, seq),
		UnitID:         entry.UnitID,
		Year:           year,
		SequenceNumber: seq,
		Direction:      entry.Direction,
		EntryType:      entry.EntryType,
		Amount:         entry.Amount,
		Currency:       "NGN",
		Counterparty:   entry.Counterparty,
		Description:    entry.Description,
		SourceType:     entry.SourceType,
		SourceID:       entry.SourceID,
		Status:         "posted",
		Metadata:       entry.Metadata,
		CreatedBy:      entry.CreatedBy,
		CreatedAt:      now,
	}
	if err := tx.Create(ledger).Error; err != nil {
		return nil, err
	}
	return ledger, nil
}

// ListForUnit returns ledger entries for a unit, newest first.
// year=0 means "all years". limit is clamped to [1, 500], default 200.
func (s *LedgerService) ListForUnit(unitID uuid.UUID, year int, limit int) ([]models.FinancialLedger, error) {
	var entries []models.FinancialLedger
	q := config.DB.Where("unit_id = ?", unitID)
	if year > 0 {
		q = q.Where("year = ?", year)
	}
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	err := q.Order("sequence_number DESC").Limit(limit).Find(&entries).Error
	return entries, err
}

// LedgerSummary holds totals for a unit for one year.
type LedgerSummary struct {
	UnitID   uuid.UUID `json:"unitId"`
	Year     int       `json:"year"`
	TotalIn  float64   `json:"totalIn"`
	TotalOut float64   `json:"totalOut"`
	Net      float64   `json:"net"`
	InCount  int64     `json:"inCount"`
	OutCount int64     `json:"outCount"`
}

// SummaryForUnitYear aggregates totals for a unit in a given year.
func (s *LedgerService) SummaryForUnitYear(unitID uuid.UUID, year int) (*LedgerSummary, error) {
	type row struct {
		Direction string
		Total     float64
		Count     int64
	}
	var rows []row
	if err := config.DB.Model(&models.FinancialLedger{}).
		Select("direction, COALESCE(SUM(amount), 0) as total, COUNT(*) as count").
		Where("unit_id = ? AND year = ? AND status = ?", unitID, year, "posted").
		Group("direction").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	summary := &LedgerSummary{UnitID: unitID, Year: year}
	for _, r := range rows {
		switch r.Direction {
		case "in":
			summary.TotalIn = r.Total
			summary.InCount = r.Count
		case "out":
			summary.TotalOut = r.Total
			summary.OutCount = r.Count
		}
	}
	summary.Net = summary.TotalIn - summary.TotalOut
	return summary, nil
}

// CategoryTotal holds one category's total for a year.
type CategoryTotal struct {
	Category string  `json:"category"`
	Total    float64 `json:"total"`
	Count    int64   `json:"count"`
}

// TopCategoriesForYear returns the top expense categories for a unit in a year.
func (s *LedgerService) TopCategoriesForYear(unitID uuid.UUID, year int, limit int) []CategoryTotal {
	if limit <= 0 {
		limit = 5
	}
	var rows []CategoryTotal
	config.DB.Model(&models.FinancialLedger{}).
		Select("entry_type as category, COALESCE(SUM(amount), 0) as total, COUNT(*) as count").
		Where("unit_id = ? AND year = ? AND direction = ? AND status = ?",
			unitID, year, "out", "posted").
		Group("entry_type").
		Order("total DESC").
		Limit(limit).
		Scan(&rows)
	return rows
}

// CloseYear writes a frozen UnitFinancialYear snapshot for a unit.
// Idempotent — running twice for the same year is a no-op.
func (s *LedgerService) CloseYear(unitID uuid.UUID, year int, closedBy uuid.UUID) (*models.UnitFinancialYear, error) {
	var existing models.UnitFinancialYear
	if err := config.DB.Where("unit_id = ? AND year = ?", unitID, year).First(&existing).Error; err == nil {
		return &existing, nil
	}

	summary, err := s.SummaryForUnitYear(unitID, year)
	if err != nil {
		return nil, err
	}

	cats := s.TopCategoriesForYear(unitID, year, 5)
	catsJSON := ""
	if len(cats) > 0 {
		catsJSON = "["
		for i, c := range cats {
			if i > 0 {
				catsJSON += ","
			}
			catsJSON += `{"category":"` + c.Category + `","total":` +
				formatFloat(c.Total) + `,"count":` + formatInt(c.Count) + `}`
		}
		catsJSON += "]"
	}

	record := models.UnitFinancialYear{
		UnitID:        unitID,
		Year:          year,
		TotalIn:       summary.TotalIn,
		TotalOut:      summary.TotalOut,
		Net:           summary.Net,
		InCount:       summary.InCount,
		OutCount:      summary.OutCount,
		TopCategories: catsJSON,
		Status:        "closed",
		ClosedAt:      time.Now().UTC(),
		ClosedBy:      closedBy,
	}
	if err := config.DB.Create(&record).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

// ListYears returns all closed financial years for a unit, newest first.
func (s *LedgerService) ListYears(unitID uuid.UUID) ([]models.UnitFinancialYear, error) {
	var years []models.UnitFinancialYear
	err := config.DB.Where("unit_id = ?", unitID).
		Order("year DESC").
		Find(&years).Error
	return years, err
}

func formatFloat(f float64) string {
	return fmt.Sprintf("%.2f", f)
}

func formatInt(i int64) string {
	return fmt.Sprintf("%d", i)
}
