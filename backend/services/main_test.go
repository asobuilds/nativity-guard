//go:build integration

package services_test

import (
	"os"
	"testing"

	"github.com/joho/godotenv"

	"security-solution/internal/testutil"
)

func TestMain(m *testing.M) {
	// Load .env so JWT_SECRET, TEST_DATABASE_URL, ENCRYPTION_KEY etc.
	// are available to the tests. Path is relative to the package dir.
	_ = godotenv.Load("../.env")

	if err := testutil.SetupTestDB(); err != nil {
		panic("test setup failed: " + err.Error())
	}
	testutil.SwapConfigDB()
	os.Exit(m.Run())
}
