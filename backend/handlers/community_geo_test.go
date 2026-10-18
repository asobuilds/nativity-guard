package handlers

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"security-solution/models"
)

// These tests cover the geography rules directly. They need no database: the
// scope ladder, the visibility predicate and the SQL window builder are pure
// functions, so the logic that decides who sees what is pinned without a
// Postgres instance (the handler/DB paths run in CI's integration-race job).

func TestScopeRankOrdering(t *testing.T) {
	order := []string{ScopeLocality, ScopeLGA, ScopeState, ScopeCountry, ScopeWorld}
	for i := 1; i < len(order); i++ {
		prev, _ := scopeRank(order[i-1])
		cur, _ := scopeRank(order[i])
		if !(prev < cur) {
			t.Fatalf("expected %s (rank %d) to be narrower than %s (rank %d)", order[i-1], prev, order[i], cur)
		}
	}
	if _, ok := scopeRank("galaxy"); ok {
		t.Fatal("unknown scope must not be valid")
	}
	if _, ok := scopeRank("  STATE "); !ok {
		t.Fatal("scope lookup must trim and lowercase")
	}
}

func TestPlaceVisibleToViewer(t *testing.T) {
	viewer := viewerPlace{Locality: "Garki", LGA: "Abuja Municipal", State: "FCT", Country: "Nigeria"}
	empty := viewerPlace{}

	cases := []struct {
		name  string
		place models.GeoPlace
		v     viewerPlace
		want  bool
	}{
		{"world is visible to anyone", models.GeoPlace{Scope: ScopeWorld}, empty, true},
		{"legacy row with no scope is world", models.GeoPlace{Scope: ""}, empty, true},
		{"national post visible when country matches", models.GeoPlace{Scope: ScopeCountry, Country: "Nigeria"}, viewer, true},
		{"national post needs a resolved country", models.GeoPlace{Scope: ScopeCountry, Country: "Nigeria"}, empty, false},
		{"same state visible", models.GeoPlace{Scope: ScopeState, State: "FCT"}, viewer, true},
		{"other state not visible", models.GeoPlace{Scope: ScopeState, State: "Lagos"}, viewer, false},
		{"same lga visible", models.GeoPlace{Scope: ScopeLGA, State: "FCT", LGA: "Abuja Municipal"}, viewer, true},
		{"same lga name, different state, not visible", models.GeoPlace{Scope: ScopeLGA, State: "Lagos", LGA: "Abuja Municipal"}, viewer, false},
		{"same locality visible", models.GeoPlace{Scope: ScopeLocality, State: "FCT", LGA: "Abuja Municipal", Locality: "Garki"}, viewer, true},
		{"other locality not visible", models.GeoPlace{Scope: ScopeLocality, State: "FCT", LGA: "Abuja Municipal", Locality: "Wuse"}, viewer, false},
		{"case-insensitive place match", models.GeoPlace{Scope: ScopeState, State: "fct"}, viewer, true},
		{"unknown scope is never visible", models.GeoPlace{Scope: "continent", State: "FCT"}, viewer, false},
	}

	for _, tc := range cases {
		if got := placeVisibleToViewer(tc.place, tc.v); got != tc.want {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestScopeWindowConditionWidensButNeverMoves(t *testing.T) {
	viewer := viewerPlace{Locality: "Garki", LGA: "Abuja Municipal", State: "FCT", Country: "Nigeria"}

	// Place-placeholder counts: locality 3, +lga 2, +state 1, +country 1.
	placeholderCounts := map[string]int{
		ScopeLocality: 3,
		ScopeLGA:      5,
		ScopeState:    6,
		ScopeCountry:  7,
		ScopeWorld:    7,
	}

	for scope, wantPlaceholders := range placeholderCounts {
		rank, _ := scopeRank(scope)
		condition, args := scopeWindowCondition(viewer, rank)

		if got := strings.Count(condition, "?"); got != wantPlaceholders {
			t.Errorf("scope=%s: expected %d placeholders, got %d (%s)", scope, wantPlaceholders, got, condition)
		}
		if len(args) != strings.Count(condition, "?") {
			t.Errorf("scope=%s: %d args for %d placeholders", scope, len(args), strings.Count(condition, "?"))
		}
		// Widening can never reach another place.
		if strings.Contains(condition, "Lagos") {
			t.Errorf("scope=%s: condition must not contain other places: %s", scope, condition)
		}

		// A narrower scope must not leak a broader clause.
		if rank < scopeRanks[ScopeWorld] && strings.Contains(condition, "scope = 'world'") {
			t.Errorf("scope=%s: world clause must not appear below world", scope)
		}
		if rank < scopeRanks[ScopeCountry] && strings.Contains(condition, "scope = 'country'") {
			t.Errorf("scope=%s: country clause must not appear below country", scope)
		}
		if rank < scopeRanks[ScopeState] && strings.Contains(condition, "scope = 'state'") {
			t.Errorf("scope=%s: state clause must not appear below state", scope)
		}
		if rank < scopeRanks[ScopeLGA] && strings.Contains(condition, "scope = 'lga'") {
			t.Errorf("scope=%s: lga clause must not appear below lga", scope)
		}

		// The viewer's own place is always a bound parameter, never inlined.
		if strings.Contains(condition, "Abuja Municipal") || strings.Contains(condition, "Garki") {
			t.Errorf("scope=%s: viewer place must be parameterised, not inlined: %s", scope, condition)
		}
	}
}

func TestScopeWindowConditionForResolvedEmptyViewer(t *testing.T) {
	// A caller with no resolvable place sees only legacy/world rows: the window
	// collapses rather than falling open.
	rank, _ := scopeRank(ScopeWorld)
	condition, args := scopeWindowCondition(viewerPlace{}, rank)
	if len(args) != 0 {
		t.Fatalf("empty viewer must produce no bound args, got %v", args)
	}
	if !strings.Contains(condition, "scope = 'world'") {
		t.Fatalf("world clause missing: %s", condition)
	}
	if strings.Contains(condition, "state = ?") || strings.Contains(condition, "lga = ?") {
		t.Fatalf("empty viewer must not match any place: %s", condition)
	}
}

func TestScopeSatisfiedBy(t *testing.T) {
	full := viewerPlace{Locality: "Garki", LGA: "Abuja Municipal", State: "FCT", Country: "Nigeria"}
	stateOnly := viewerPlace{State: "FCT", Country: "Nigeria"}

	cases := []struct {
		scope string
		v     viewerPlace
		want  bool
	}{
		{ScopeLocality, full, true},
		{ScopeLocality, stateOnly, false},
		{ScopeLGA, full, true},
		{ScopeLGA, stateOnly, false},
		{ScopeState, stateOnly, true},
		{ScopeState, viewerPlace{}, false},
		{ScopeCountry, viewerPlace{}, true}, // defaults to Nigeria
		{ScopeWorld, viewerPlace{}, true},
		{"nonsense", full, false},
	}
	for _, tc := range cases {
		if got := scopeSatisfiedBy(tc.scope, tc.v); got != tc.want {
			t.Errorf("scope=%s place=%+v: got %v, want %v", tc.scope, tc.v, got, tc.want)
		}
	}
}

func TestDefaultScopeFor(t *testing.T) {
	if got := defaultScopeFor(viewerPlace{LGA: "Abuja Municipal", State: "FCT"}); got != ScopeLGA {
		t.Errorf("expected lga default, got %s", got)
	}
	if got := defaultScopeFor(viewerPlace{State: "FCT"}); got != ScopeState {
		t.Errorf("expected state default, got %s", got)
	}
	if got := defaultScopeFor(viewerPlace{}); got != ScopeWorld {
		t.Errorf("expected world default, got %s", got)
	}
}

func TestNewPlaceRecordsCoordinatesOnlyWhenGiven(t *testing.T) {
	viewer := viewerPlace{Locality: "Garki", LGA: "Abuja Municipal", State: "FCT", Country: "Nigeria"}

	without := newPlace(ScopeLGA, viewer, nil, nil)
	if without.Geohash != "" || without.Latitude != 0 {
		t.Fatalf("no coordinates supplied must leave the place bare, got %+v", without)
	}

	lat, lng := 9.0579, 7.4951
	with := newPlace(ScopeLGA, viewer, &lat, &lng)
	if with.Geohash == "" {
		t.Fatal("a supplied coordinate must record a geohash")
	}
	if with.Latitude != lat || with.Longitude != lng {
		t.Fatalf("coordinates not recorded: %+v", with)
	}
	if with.Scope != ScopeLGA {
		t.Fatalf("scope not recorded: %+v", with)
	}
}

func TestPostVisibleTo(t *testing.T) {
	unit := uuid.New()
	other := uuid.New()

	author := &models.User{ID: uuid.New(), Role: "unit_admin", UnitID: &unit}
	foreign := &models.User{ID: uuid.New(), Role: "citizen"}

	unitPost := models.ForumPost{UnitID: &unit, GeoPlace: models.GeoPlace{Scope: ScopeWorld}}
	foreignPost := models.ForumPost{UnitID: &other, GeoPlace: models.GeoPlace{Scope: ScopeLGA, State: "Lagos", LGA: "Ikeja"}}
	distantWorldPost := models.ForumPost{GeoPlace: models.GeoPlace{Scope: ScopeWorld}}

	if !postVisibleTo(author, unitPost, nil, nil) {
		t.Error("a member must see their own unit's post")
	}
	if !postVisibleTo(author, distantWorldPost, nil, nil) {
		t.Error("a world post must be visible to everyone")
	}
	if postVisibleTo(foreign, foreignPost, nil, nil) {
		t.Error("a citizen with no place must not see another state's post")
	}
	if !postVisibleTo(&models.User{Role: "super_admin"}, foreignPost, nil, nil) {
		t.Error("a super admin moderates the whole platform")
	}
}
