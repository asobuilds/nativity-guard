package testutil

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"security-solution/config"
	"security-solution/models"
)

var testDB *gorm.DB

// SetupTestDB connects to TEST_DATABASE_URL, validates it is different
// from DATABASE_URL (fatal safety check), and runs AutoMigrate once.
// Call from TestMain in each package that needs a DB.
//
// IMPORTANT: Do NOT call t.Parallel() in tests that use this package —
// config.DB is swapped at package level, so tests must run serially.
func SetupTestDB() error {
	testURL := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	prodURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))

	if testURL == "" {
		return errors.New("TEST_DATABASE_URL is not set (integration tests require it)")
	}
	if testURL == prodURL {
		return errors.New("TEST_DATABASE_URL must differ from DATABASE_URL — refusing to run")
	}

	db, err := gorm.Open(postgres.Open(testURL), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("connect test db: %w", err)
	}

	if err := db.AutoMigrate(
		&models.User{},
		&models.SecurityUnit{},
		&models.UnitMembership{},
		&models.Officer{},
		&models.Case{},
		&models.Evidence{},
		&models.CaseOfficer{},
		&models.CaseTimeline{},
		&models.Progress{},
		&models.CaseProgress{},
		&models.CaseFeedback{},
		&models.CaseReview{},
		&models.CaseWeeklyUpdate{},
		&models.CaseAdminAssignment{},
		&models.Notification{},
		&models.UnitAuth{},
		&models.UnitInvite{},
		&models.UnitAdminSeat{},
		&models.UnitAdminElection{},
		&models.AdminVote{},
		&models.HeadAdminVote{},
		&models.RevocationCycle{},
		&models.RevocationVote{},
		&models.Appeal{},
		&models.ExpungementRequest{},
		&models.CounterStatement{},
		&models.Rating{},
		&models.OfficerScore{},
		&models.UnitScore{},
		&models.FinancialLedger{},
		&models.LedgerSequence{},
		&models.UnitFinancialYear{},
		&models.PlatformDonation{},
		&models.BankAccount{},
		&models.Donation{},
		&models.Transaction{},
		&models.TransactionApproval{},
		&models.Budget{},
		&models.FinancialReport{},
		&models.Suspect{},
		&models.SuspectCase{},
		&models.SuspectSighting{},
		&models.SuspectAssociation{},
		&models.UserLocation{},
		&models.GeocodeCache{},
		&models.UserSession{},
		&models.RefreshToken{},
		&models.RevokedToken{},
		&models.IdempotencyRecord{},
		&models.AgeAudit{},
		&models.AuditLog{},
		&models.SOSAlert{},
		&models.SOSResponder{},
	); err != nil {
		return fmt.Errorf("automigrate test db: %w", err)
	}

	testDB = db
	return nil
}

// SwapConfigDB points config.DB at the test database. Call once in TestMain.
func SwapConfigDB() {
	config.DB = testDB
}

// TruncateAll wipes every table in the current schema. Uses Postgres's
// own catalog so the list never drifts when new models are added.
func TruncateAll(t *testing.T) {
	t.Helper()

	var tables []string
	if err := testDB.Raw(`
		SELECT tablename
		FROM pg_tables
		WHERE schemaname = 'public'
		  AND tablename NOT LIKE 'pg_%'
		  AND tablename != 'schema_migrations'
	`).Scan(&tables).Error; err != nil {
		t.Fatalf("list tables: %v", err)
	}

	if len(tables) == 0 {
		return
	}

	// Single TRUNCATE with CASCADE handles FKs in one round trip.
	stmt := "TRUNCATE TABLE " + strings.Join(tables, ", ") + " RESTART IDENTITY CASCADE"
	if err := testDB.Exec(stmt).Error; err != nil {
		t.Fatalf("truncate all: %v", err)
	}
}

// GetDB returns the test DB handle for direct use in tests.
func GetDB() *gorm.DB {
	return testDB
}
