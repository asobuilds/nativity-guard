package config

import (
  "os"
  "strings"
  "testing"
)

// A user location is written by the authenticated /location API.
// Regression guard: omitting its model from AutoMigrateAll causes live 500s.
func TestUserLocationRegisteredForAutoMigration(t *testing.T) {
  source, err := os.ReadFile("migrate.go")
  if err != nil { t.Fatal(err) }
  if !strings.Contains(string(source), "&models.UserLocation{}") {
    t.Fatal("user_locations must be registered in AutoMigrateAll")
  }
}
