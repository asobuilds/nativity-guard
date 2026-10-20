//go:build integration

package handlers

import (
	"testing"

	"security-solution/config"
	"security-solution/internal/testutil"
	"security-solution/models"
)

func TestLocalIntelligenceContextInjection(t *testing.T) {
	if config.DB == nil {
		t.Skip("integration DB unavailable")
	}
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	// Create unit in Lagos / Ikeja
	unit := &models.SecurityUnit{
		Name:       "Ikeja Unit",
		Type:       "community",
		Status:     "active",
		IsVerified: true,
		State:      "Lagos",
		LGA:        "Ikeja",
	}
	if err := config.DB.Create(unit).Error; err != nil {
		t.Fatalf("create unit: %v", err)
	}

	user := testutil.MakeUser(t, "citizen")
	user.UnitID = &unit.ID
	config.DB.Save(user)

	// Create local intelligence record
	intel := models.LocalIntelligence{
		State:      "Lagos",
		LGA:        "Ikeja",
		Fact:       "Road blocked at Ikeja roundabout due to construction.",
		Category:   "road_block",
		Confidence: "high",
		SourceType: "report",
		SourceID:   "test-source-id",
	}
	if err := config.DB.Create(&intel).Error; err != nil {
		t.Fatalf("create intel: %v", err)
	}

	ctxStr := callerContext(user)
	if ctxStr == "" {
		t.Fatalf("expected non-empty caller context")
	}
	if !containsString(ctxStr, "Road blocked at Ikeja roundabout") {
		t.Fatalf("expected caller context to contain local intelligence fact, got:\n%s", ctxStr)
	}
}

func containsString(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || hasSubstring(s, sub))
}

func hasSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
