package handlers

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"security-solution/models"
)

// The moderation queue and the keyset cursor are the two new pieces of logic in
// RES-014. Both are pinned here without a database: the cursor is a pure
// encode/decode pair, and canModeratePost is a pure authorization function. The
// handler/DB paths run in CI's integration-race job against Postgres 16.

func TestFeedCursorRoundTrip(t *testing.T) {
	original := feedCursor{
		Pinned:    true,
		CreatedAt: time.Date(2026, 10, 18, 13, 38, 0, 123456789, time.UTC),
		ID:        uuid.MustParse("6f1d1cf7-0e0c-4a2e-9d70-8f0d0f3b2a11"),
	}

	decoded, ok := decodeFeedCursor(encodeFeedCursor(original))
	if !ok {
		t.Fatal("a cursor produced by encodeFeedCursor must decode")
	}
	if decoded.Pinned != original.Pinned {
		t.Errorf("pinned: got %v, want %v", decoded.Pinned, original.Pinned)
	}
	if !decoded.CreatedAt.Equal(original.CreatedAt) {
		t.Errorf("createdAt: got %s, want %s", decoded.CreatedAt, original.CreatedAt)
	}
	if decoded.ID != original.ID {
		t.Errorf("id: got %s, want %s", decoded.ID, original.ID)
	}
}

func TestFeedCursorIsURLSafe(t *testing.T) {
	// The cursor travels in a query string. RawURLEncoding must not emit '+' or
	// '/' or '=' — a padded standard-base64 cursor would be mangled in transit.
	var sawUnpadded bool
	for i := 0; i < 64; i++ {
		encoded := encodeFeedCursor(feedCursor{
			Pinned:    i%2 == 0,
			CreatedAt: time.Now().UTC().Add(time.Duration(i) * time.Millisecond),
			ID:        uuid.New(),
		})
		if strings.ContainsAny(encoded, "+/=") {
			t.Fatalf("cursor %q contains characters that are unsafe in a query string", encoded)
		}
		if !strings.HasSuffix(encoded, "=") {
			sawUnpadded = true
		}
		if _, ok := decodeFeedCursor(encoded); !ok {
			t.Fatalf("cursor %q did not round-trip", encoded)
		}
	}
	if !sawUnpadded {
		t.Skip("all generated cursors happened to align; nothing asserted")
	}
}

func TestDecodeFeedCursorRejectsGarbage(t *testing.T) {
	enc := func(raw string) string { return base64.RawURLEncoding.EncodeToString([]byte(raw)) }
	validID := uuid.New().String()
	validTime := time.Now().UTC().Format(time.RFC3339Nano)

	cases := map[string]string{
		"empty":                 "",
		"not base64":            "!!!! not base64 !!!!",
		"too few fields":        enc("true|" + validTime),
		"not a bool":            enc("yes|" + validTime + "|" + validID),
		"not a timestamp":       enc("true|yesterday|" + validID),
		"not a uuid":            enc("true|" + validTime + "|not-a-uuid"),
		"padded standard b64":   "YWJjZGU=",
		"standard-alphabet b64": "YWJj+2RlL3g",
		"plus and slash":        "a+b/c",
	}

	for name, value := range cases {
		if _, ok := decodeFeedCursor(value); ok {
			t.Errorf("%s: expected rejection, but it decoded", name)
		}
	}

	// Padding is not part of the format, so appending it must not be accepted
	// even though the leading bytes are a real cursor.
	valid := encodeFeedCursor(feedCursor{Pinned: true, CreatedAt: time.Now().UTC(), ID: uuid.New()})
	if _, ok := decodeFeedCursor(valid + "="); ok {
		t.Error("trailing padding must be rejected")
	}
	if _, ok := decodeFeedCursor(strings.TrimSpace(valid) + "\n"); !ok {
		t.Error("surrounding whitespace should be tolerated by TrimSpace")
	}
}

func TestFeedCursorConditionMatchesFeedOrdering(t *testing.T) {
	cursor := feedCursor{
		Pinned:    false,
		CreatedAt: time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC),
		ID:        uuid.New(),
	}
	condition, args := feedCursorCondition(cursor)

	for _, column := range []string{"is_pinned", "created_at", "id"} {
		if !strings.Contains(condition, column) {
			t.Errorf("condition must constrain %q: %s", column, condition)
		}
	}
	// The feed is ordered DESC on all three keys, so walking forward is strictly
	// less-than on each. One '<' per column.
	if got := strings.Count(condition, "<"); got != 3 {
		t.Errorf("expected 3 forward comparisons, got %d: %s", got, condition)
	}
	if len(args) != strings.Count(condition, "?") {
		t.Errorf("%d args for %d placeholders: %s", len(args), strings.Count(condition, "?"), condition)
	}
	// Cursor values are bound, never inlined.
	if strings.Contains(condition, cursor.ID.String()) {
		t.Errorf("cursor id must be parameterised: %s", condition)
	}
}

func TestIsCommunityModerator(t *testing.T) {
	allowed := []string{"unit_admin", "super_admin"}
	for _, role := range allowed {
		if !isCommunityModerator(&models.User{Role: role}) {
			t.Errorf("%s must be able to open the moderation queue", role)
		}
	}
	for _, role := range []string{"citizen", "officer", "", "unit_admin "} {
		if isCommunityModerator(&models.User{Role: role}) {
			t.Errorf("%q must not open the moderation queue", role)
		}
	}
	if isCommunityModerator(nil) {
		t.Error("nil user must not open the moderation queue")
	}
}

func TestCanModeratePost(t *testing.T) {
	unit := uuid.New()
	otherUnit := uuid.New()
	place := viewerPlace{Locality: "Garki", LGA: "Abuja Municipal", State: "FCT", Country: "Nigeria"}

	unitAdmin := &models.User{ID: uuid.New(), Role: "unit_admin", UnitID: &unit}
	citizen := &models.User{ID: uuid.New(), Role: "citizen"}
	superAdmin := &models.User{ID: uuid.New(), Role: "super_admin"}

	localLGA := models.ForumPost{GeoPlace: models.GeoPlace{Scope: ScopeLGA, State: "FCT", LGA: "Abuja Municipal"}}
	localState := models.ForumPost{GeoPlace: models.GeoPlace{Scope: ScopeState, State: "FCT"}}
	national := models.ForumPost{GeoPlace: models.GeoPlace{Scope: ScopeCountry, Country: "Nigeria"}}
	world := models.ForumPost{GeoPlace: models.GeoPlace{Scope: ScopeWorld}}
	unknownPlace := models.ForumPost{GeoPlace: models.GeoPlace{Scope: ""}}
	otherState := models.ForumPost{GeoPlace: models.GeoPlace{Scope: ScopeLGA, State: "Lagos", LGA: "Ikeja"}}
	ownUnitNational := models.ForumPost{UnitID: &unit, GeoPlace: models.GeoPlace{Scope: ScopeWorld}}
	foreignUnitNational := models.ForumPost{UnitID: &otherUnit, GeoPlace: models.GeoPlace{Scope: ScopeWorld}}

	cases := []struct {
		name string
		user *models.User
		post models.ForumPost
		want bool
	}{
		{"super admin moderates anything", superAdmin, world, true},
		{"super admin moderates other states", superAdmin, otherState, true},
		{"unit admin moderates their own LGA", unitAdmin, localLGA, true},
		{"unit admin moderates their own state", unitAdmin, localState, true},
		{"unit admin moderates their own unit's post anywhere", unitAdmin, ownUnitNational, true},
		{"unit admin does NOT moderate a national post", unitAdmin, national, false},
		{"unit admin does NOT moderate a world post", unitAdmin, world, false},
		{"unit admin does NOT moderate a post of unknown place", unitAdmin, unknownPlace, false},
		{"unit admin does NOT moderate another state", unitAdmin, otherState, false},
		{"unit admin does NOT moderate another unit's national post", unitAdmin, foreignUnitNational, false},
		{"citizen never moderates", citizen, localLGA, false},
	}

	for _, tc := range cases {
		if got := canModeratePost(tc.user, tc.post, place); got != tc.want {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}

	if canModeratePost(nil, localLGA, place) {
		t.Error("nil user must never moderate")
	}
	// A unit admin with no resolvable place must not reach local posts either.
	if got := canModeratePost(unitAdmin, localLGA, viewerPlace{}); got {
		t.Error("a unit admin with no resolved place must not moderate local posts")
	}
}
