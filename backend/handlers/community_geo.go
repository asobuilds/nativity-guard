package handlers

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"security-solution/config"
	"security-solution/models"
	"security-solution/services"
	"security-solution/utils"
)

// Community geographic scopes, most specific first (decision D-011).
const (
	ScopeLocality = "locality"
	ScopeLGA      = "lga"
	ScopeState    = "state"
	ScopeCountry  = "country"
	ScopeWorld    = "world"

	// defaultCountry is used when a resolved place has no country name. The
	// deployment is national, so an unspecified country means Nigeria.
	defaultCountry = "Nigeria"

	// geohashPrecision keeps a coarse, ~4.9km cell for display/analytics. Match
	// is never done on the geohash — it is a privacy-preserving hint, not a key.
	geohashPrecision = 5
)

var scopeRanks = map[string]int{
	ScopeLocality: 0,
	ScopeLGA:      1,
	ScopeState:    2,
	ScopeCountry:  3,
	ScopeWorld:    4,
}

// scopeRank returns the breadth rank of a scope and whether it is valid.
func scopeRank(scope string) (int, bool) {
	rank, ok := scopeRanks[strings.ToLower(strings.TrimSpace(scope))]
	return rank, ok
}

func validScope(scope string) bool {
	_, ok := scopeRank(scope)
	return ok
}

// viewerPlace is a caller's resolved geographic position. It is always built
// server-side from a verified unit or reverse-geocoded coordinates and is never
// populated from client-supplied place names (decision D-012).
type viewerPlace struct {
	Locality string
	LGA      string
	State    string
	Country  string
}

// resolveViewerPlace answers "where is this caller?" for feed scoping and
// participation gating.
//
// Precedence: verified active unit membership (staff), then explicit
// coordinates supplied with the request (a citizen who shared their position),
// then the caller's most recent stored UserLocation. A caller with no resolvable
// place gets the zero place and can therefore see and reply to only world-scoped
// content — the honest default.
func resolveViewerPlace(user *models.User, lat, lng *float64) viewerPlace {
	if user == nil || config.DB == nil {
		return viewerPlace{}
	}

	if place, ok := placeFromVerifiedUnit(user.ID); ok {
		return place
	}
	if lat != nil && lng != nil {
		if place, ok := placeFromCoordinates(*lat, *lng); ok {
			return place
		}
	}
	if place, ok := placeFromLatestLocation(user.ID); ok {
		return place
	}
	return viewerPlace{}
}

func placeFromVerifiedUnit(userID uuid.UUID) (viewerPlace, bool) {
	var m models.UnitMembership
	if err := config.DB.Where("user_id = ? AND status = ?", userID, "active").
		Order("created_at ASC").First(&m).Error; err != nil {
		return viewerPlace{}, false
	}

	var unit models.SecurityUnit
	if err := config.DB.First(&unit, "id = ?", m.UnitID).Error; err != nil {
		return viewerPlace{}, false
	}
	if !unit.IsVerified {
		return viewerPlace{}, false
	}

	place := viewerPlace{
		Locality: strings.TrimSpace(unit.Ward),
		LGA:      strings.TrimSpace(unit.LGA),
		State:    strings.TrimSpace(unit.State),
		Country:  defaultCountry,
	}
	if place.State == "" && place.LGA == "" {
		return viewerPlace{}, false
	}
	return place, true
}

func placeFromLatestLocation(userID uuid.UUID) (viewerPlace, bool) {
	var loc models.UserLocation
	if err := config.DB.Where("user_id = ?", userID).
		Order("recorded_at DESC").First(&loc).Error; err != nil {
		return viewerPlace{}, false
	}
	return placeFromCoordinates(loc.Latitude, loc.Longitude)
}

// placeFromCoordinates reverse-geocodes through the cached geocoder. The cache
// is keyed by geohash precision 6 (~1.2km), so repeated feeds in the same cell
// do not hit Nominatim.
func placeFromCoordinates(lat, lng float64) (viewerPlace, bool) {
	rec, err := services.NewGeocodingService().ReverseGeocode(lat, lng)
	if err != nil || rec == nil {
		return viewerPlace{}, false
	}
	place := viewerPlace{
		Locality: strings.TrimSpace(firstNonEmptyValue(rec.Landmark, rec.Suburb)),
		LGA:      strings.TrimSpace(rec.LGA),
		State:    strings.TrimSpace(rec.State),
		Country:  strings.TrimSpace(firstNonEmptyValue(rec.Country, defaultCountry)),
	}
	if place.State == "" && place.LGA == "" && place.Locality == "" {
		return viewerPlace{}, false
	}
	return place, true
}

// placeVisibleToViewer is the single rule that governs both reading and
// participating (decision D-014): if a post is not visible to you, it is not
// openable and not replyable. Every branch pins the post's place to the
// viewer's own place, so widening a feed can never reach another jurisdiction.
func placeVisibleToViewer(p models.GeoPlace, v viewerPlace) bool {
	switch strings.ToLower(strings.TrimSpace(p.Scope)) {
	case "", ScopeWorld:
		return true
	case ScopeCountry:
		return samePlace(p.Country, v.Country, "")
	case ScopeState:
		return v.State != "" && samePlace(p.State, v.State, "")
	case ScopeLGA:
		// LGA names repeat across states, so an LGA is only ever matched
		// together with its state.
		return v.State != "" && v.LGA != "" &&
			samePlace(p.State, v.State, "") && samePlace(p.LGA, v.LGA, "")
	case ScopeLocality:
		return v.State != "" && v.LGA != "" && v.Locality != "" &&
			samePlace(p.State, v.State, "") &&
			samePlace(p.LGA, v.LGA, "") &&
			samePlace(p.Locality, v.Locality, "")
	default:
		return false
	}
}

// scopeWindowCondition builds the feed's WHERE fragment for "posts inside the
// viewer's jurisdiction, no broader than maxRank". It is the SQL mirror of
// placeVisibleToViewer and must stay in step with it.
//
// Rows with no scope (written before geography existed, or an unknown place) are
// treated as world so they are never presented as local.
func scopeWindowCondition(v viewerPlace, maxRank int) (string, []any) {
	clauses := []string{"(scope IS NULL OR scope = '')"}
	args := []any{}

	if maxRank >= scopeRanks[ScopeLocality] && v.State != "" && v.LGA != "" && v.Locality != "" {
		clauses = append(clauses, "(scope = 'locality' AND state = ? AND lga = ? AND locality = ?)")
		args = append(args, v.State, v.LGA, v.Locality)
	}
	if maxRank >= scopeRanks[ScopeLGA] && v.State != "" && v.LGA != "" {
		clauses = append(clauses, "(scope = 'lga' AND state = ? AND lga = ?)")
		args = append(args, v.State, v.LGA)
	}
	if maxRank >= scopeRanks[ScopeState] && v.State != "" {
		clauses = append(clauses, "(scope = 'state' AND state = ?)")
		args = append(args, v.State)
	}
	if maxRank >= scopeRanks[ScopeCountry] && v.Country != "" {
		clauses = append(clauses, "(scope = 'country' AND country = ?)")
		args = append(args, v.Country)
	}
	if maxRank >= scopeRanks[ScopeWorld] {
		clauses = append(clauses, "(scope = 'world')")
	}

	return "(" + strings.Join(clauses, " OR ") + ")", args
}

// scopeSatisfiedBy reports whether the author's resolved place is detailed
// enough to actually address the scope they chose. Publishing to `lga` without
// an LGA is a bug, not a broad post, so it is rejected at the edge.
func scopeSatisfiedBy(scope string, v viewerPlace) bool {
	switch strings.ToLower(strings.TrimSpace(scope)) {
	case ScopeLocality:
		return v.State != "" && v.LGA != "" && v.Locality != ""
	case ScopeLGA:
		return v.State != "" && v.LGA != ""
	case ScopeState:
		return v.State != ""
	case ScopeCountry:
		return firstNonEmptyValue(v.Country, defaultCountry) != ""
	case ScopeWorld:
		return true
	default:
		return false
	}
}

// newPlace builds the stored GeoPlace for a publication. Coordinates are only
// recorded when the client actually supplied them.
func newPlace(scope string, v viewerPlace, lat, lng *float64) models.GeoPlace {
	place := models.GeoPlace{
		Scope:    strings.ToLower(strings.TrimSpace(scope)),
		Locality: v.Locality,
		LGA:      v.LGA,
		State:    v.State,
		Country:  firstNonEmptyValue(v.Country, defaultCountry),
	}
	if lat != nil && lng != nil {
		place.Latitude = *lat
		place.Longitude = *lng
		place.Geohash = utils.EncodeGeohash(*lat, *lng, geohashPrecision)
	}
	return place
}

// geoPlaceResponse renders the place for a client. Coordinates are included
// because they are already public on the content that carries them.
func geoPlaceResponse(p models.GeoPlace) map[string]any {
	return map[string]any{
		"scope":    p.Scope,
		"locality": p.Locality,
		"lga":      p.LGA,
		"state":    p.State,
		"country":  p.Country,
	}
}

func samePlace(a, b, fallback string) bool {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	if a == "" {
		a = fallback
	}
	if b == "" {
		b = fallback
	}
	if a == "" || b == "" {
		return false
	}
	return strings.EqualFold(a, b)
}

func firstNonEmptyValue(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// postVisibleTo is the read/participate gate for a single post. Geography is the
// boundary (D-014); a unit-tagged post stays visible to its own unit, and a
// super admin sees everything.
func postVisibleTo(user *models.User, post models.ForumPost, lat, lng *float64) bool {
	if user == nil {
		return false
	}
	if user.Role == "super_admin" {
		return true
	}
	if post.UnitID != nil && user.UnitID != nil && *user.UnitID == *post.UnitID {
		return true
	}
	return placeVisibleToViewer(post.GeoPlace, resolveViewerPlace(user, lat, lng))
}

// defaultScopeFor picks the audience when the author did not choose one: the
// LGA is the most reliable administrative unit we hold for Nigeria, so it is the
// default, falling back to state and then to world for a caller with no place.
func defaultScopeFor(v viewerPlace) string {
	switch {
	case v.LGA != "" && v.State != "":
		return ScopeLGA
	case v.State != "":
		return ScopeState
	default:
		return ScopeWorld
	}
}

// parseCoords reads optional `lat`/`lng` query parameters. Both must be present
// and numerically valid, otherwise the position is treated as absent.
func parseCoords(c *gin.Context) (lat, lng *float64) {
	latRaw := strings.TrimSpace(c.Query("lat"))
	lngRaw := strings.TrimSpace(c.Query("lng"))
	if latRaw == "" || lngRaw == "" {
		return nil, nil
	}
	latVal, errLat := strconv.ParseFloat(latRaw, 64)
	lngVal, errLng := strconv.ParseFloat(lngRaw, 64)
	if errLat != nil || errLng != nil {
		return nil, nil
	}
	return &latVal, &lngVal
}
