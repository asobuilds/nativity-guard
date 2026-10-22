//go:build integration

package handlers

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

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

// makeLocalIntelUser creates a citizen attached to a Lagos/Ikeja unit so
// area-scoped fact queries resolve.
func makeLocalIntelUser(t *testing.T) *models.User {
	t.Helper()
	head := testutil.MakeUser(t, "unit_admin")
	unit := testutil.MakeUnit(t, head)
	unit.State = "Lagos"
	unit.LGA = "Ikeja"
	if err := config.DB.Save(unit).Error; err != nil {
		t.Fatalf("update unit area: %v", err)
	}
	user := testutil.MakeUser(t, "citizen")
	if err := config.DB.Model(&models.User{}).
		Where("id = ?", user.ID).
		Update("unit_id", unit.ID).Error; err != nil {
		t.Fatalf("attach user to unit: %v", err)
	}
	user.UnitID = &unit.ID
	return user
}

func makeLocalIntel(t *testing.T, fact string, expiresAt *time.Time) *models.LocalIntelligence {
	t.Helper()
	intel := &models.LocalIntelligence{
		State:      "Lagos",
		LGA:        "Ikeja",
		Fact:       fact,
		Category:   "general",
		Confidence: "high",
		SourceType: "report",
		SourceID:   "test-" + fact,
		ExpiresAt:  expiresAt,
	}
	if err := config.DB.Create(intel).Error; err != nil {
		t.Fatalf("create intel: %v", err)
	}
	return intel
}

func TestLocalIntelExcludesExpiredFacts(t *testing.T) {
	if config.DB == nil {
		t.Skip("integration DB unavailable")
	}
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	user := makeLocalIntelUser(t)
	past := time.Now().UTC().Add(-time.Hour)
	future := time.Now().UTC().Add(time.Hour)
	makeLocalIntel(t, "Live fact about the market", &future)
	makeLocalIntel(t, "Expired fact about the market", &past)

	ctxStr := callerContext(user)
	if !containsString(ctxStr, "Live fact about the market") {
		t.Fatalf("expected prompt context to include live fact, got:\n%s", ctxStr)
	}
	if containsString(ctxStr, "Expired fact about the market") {
		t.Fatalf("expected prompt context to exclude expired fact, got:\n%s", ctxStr)
	}

	ctx, recorder := commJSONContext(t, "GET", "/local-intel", nil, user)
	GetLocalIntel(ctx)

	if recorder.Code != 200 {
		t.Fatalf("want 200, got %d %s", recorder.Code, recorder.Body.String())
	}
	var body struct {
		Items []localIntelItem `json:"items"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Items) != 1 {
		t.Fatalf("want only the live fact, got %d items: %+v", len(body.Items), body.Items)
	}
	if body.Items[0].Fact != "Live fact about the market" {
		t.Fatalf("unexpected fact returned: %q", body.Items[0].Fact)
	}
}

func TestLocalIntelUpvoteIsIdempotent(t *testing.T) {
	if config.DB == nil {
		t.Skip("integration DB unavailable")
	}
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	user := makeLocalIntelUser(t)
	future := time.Now().UTC().Add(time.Hour)
	intel := makeLocalIntel(t, "Fact worth upvoting", &future)

	upvote := func(u *models.User) (int, bool) {
		t.Helper()
		ctx, recorder := commJSONContext(t, "POST", "/local-intel/"+intel.ID.String()+"/upvote", nil, u)
		ctx.Params = gin.Params{{Key: "id", Value: intel.ID.String()}}
		UpvoteLocalIntel(ctx)
		if recorder.Code != 200 {
			t.Fatalf("want 200, got %d %s", recorder.Code, recorder.Body.String())
		}
		var body struct {
			Upvotes    int  `json:"upvotes"`
			HasUpvoted bool `json:"hasUpvoted"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		return body.Upvotes, body.HasUpvoted
	}

	if count, voted := upvote(user); count != 1 || !voted {
		t.Fatalf("first upvote: want 1/true, got %d/%v", count, voted)
	}
	if count, voted := upvote(user); count != 1 || !voted {
		t.Fatalf("repeat upvote must not increment: want 1/true, got %d/%v", count, voted)
	}

	var votes int64
	if err := config.DB.Model(&models.LocalIntelligenceVote{}).
		Where("intel_id = ? AND user_id = ?", intel.ID, user.ID).
		Count(&votes).Error; err != nil {
		t.Fatalf("count votes: %v", err)
	}
	if votes != 1 {
		t.Fatalf("want exactly one vote row, got %d", votes)
	}

	other := testutil.MakeUser(t, "citizen")
	if count, _ := upvote(other); count != 2 {
		t.Fatalf("second user should raise the count to 2, got %d", count)
	}
}

func TestLocalIntelRejectsUpvoteForExpiredFact(t *testing.T) {
	if config.DB == nil {
		t.Skip("integration DB unavailable")
	}
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	user := makeLocalIntelUser(t)
	past := time.Now().UTC().Add(-time.Hour)
	intel := makeLocalIntel(t, "Already stale", &past)

	ctx, recorder := commJSONContext(t, "POST", "/local-intel/"+intel.ID.String()+"/upvote", nil, user)
	ctx.Params = gin.Params{{Key: "id", Value: intel.ID.String()}}
	UpvoteLocalIntel(ctx)

	if recorder.Code != 404 {
		t.Fatalf("want 404 for expired fact, got %d %s", recorder.Code, recorder.Body.String())
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
