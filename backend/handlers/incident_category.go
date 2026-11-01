package handlers

import "strings"

var validIncidentCategories = map[string]struct{}{
	"infrastructure": {},
	"vandalism":      {},
	"disaster":       {},
	"fire":           {},
	"road_accident":  {},
	"traffic":        {},
}

// normalizeIncidentCategory lowercases and trims a caller-supplied category.
// An empty category is allowed so existing clients keep working; any other
// value must be one of the supported incident categories.
func normalizeIncidentCategory(raw string) (string, bool) {
	category := strings.ToLower(strings.TrimSpace(raw))
	if category == "" {
		return "", true
	}
	if _, ok := validIncidentCategories[category]; !ok {
		return "", false
	}
	return category, true
}
