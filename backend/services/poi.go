package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// POIService queries the Overpass API (OpenStreetMap) for points of interest
// around a location. No API key. Free.
//
// Overpass is stateful and rate-limited globally, so we cache aggressively
// (1 hour per rounded coordinate + category set). Queries go through our
// backend so the whole platform shares one cache.
type POIService struct {
	client *http.Client
}

func NewPOIService() *POIService {
	return &POIService{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// POI is one point of interest.
type POI struct {
	ID        string            `json:"id"`
	Category  string            `json:"category"`
	Name      string            `json:"name"`
	Latitude  float64           `json:"latitude"`
	Longitude float64           `json:"longitude"`
	Address   string            `json:"address,omitempty"`
	Phone     string            `json:"phone,omitempty"`
	Extra     map[string]string `json:"extra,omitempty"`
}

// AllowedCategories is the fixed set the frontend can ask for.
var AllowedCategories = map[string][]string{
	// category -> list of (key=value) OSM tags to search
	"hospital":   {"amenity=hospital", "amenity=clinic"},
	"police":     {"amenity=police"},
	"fire":       {"amenity=fire_station"},
	"pharmacy":   {"amenity=pharmacy"},
	"bank":       {"amenity=bank", "amenity=atm"},
	"school":     {"amenity=school", "amenity=university", "amenity=college"},
	"church":     {"amenity=place_of_worship,religion=christian"},
	"mosque":     {"amenity=place_of_worship,religion=muslim"},
	"landmark":   {"tourism=attraction", "historic=monument", "historic=memorial"},
}

type poiCacheEntry struct {
	items   []POI
	expires time.Time
}

var (
	poiCacheMu sync.Mutex
	poiCache   = map[string]poiCacheEntry{}
)

type overpassResponse struct {
	Elements []struct {
		Type   string             `json:"type"`
		ID     int64              `json:"id"`
		Lat    float64            `json:"lat"`
		Lon    float64            `json:"lon"`
		Center *struct {
			Lat float64 `json:"lat"`
			Lon float64 `json:"lon"`
		} `json:"center"`
		Tags map[string]string `json:"tags"`
	} `json:"elements"`
}

// GetPOIs returns all POIs within `radiusMeters` of the given point, for the
// requested categories. Result is deduplicated and capped at 200 items.
func (s *POIService) GetPOIs(lat, lng float64, radiusMeters int, categories []string) ([]POI, error) {
	if radiusMeters < 200 {
		radiusMeters = 200
	}
	if radiusMeters > 10000 {
		radiusMeters = 10000
	}
	if len(categories) == 0 {
		categories = []string{"hospital", "police", "bank", "school", "church"}
	}

	// Filter to allowed categories only.
	valid := []string{}
	for _, c := range categories {
		if _, ok := AllowedCategories[c]; ok {
			valid = append(valid, c)
		}
	}
	if len(valid) == 0 {
		return nil, fmt.Errorf("poi: no valid categories requested")
	}
	sortStrings(valid)

	key := fmt.Sprintf("%.3f,%.3f,%d,%s", lat, lng, radiusMeters, strings.Join(valid, "|"))

	poiCacheMu.Lock()
	if entry, ok := poiCache[key]; ok && time.Now().Before(entry.expires) {
		poiCacheMu.Unlock()
		return entry.items, nil
	}
	poiCacheMu.Unlock()

	// Build Overpass QL query.
	var qb strings.Builder
	qb.WriteString("[out:json][timeout:25];\n(\n")
	for _, c := range valid {
		for _, tag := range AllowedCategories[c] {
			parts := strings.SplitN(tag, ",", 2)
			selector := `["` + parts[0] + `"]`
			if len(parts) == 2 {
				// additional constraint like religion=christian
				kv := strings.SplitN(parts[1], "=", 2)
				if len(kv) == 2 {
					selector = `["` + kv[0] + `"="` + kv[1] + `"]` + selector
				}
			}
			qb.WriteString(fmt.Sprintf(
				"  nwr%s(around:%d,%.6f,%.6f);\n",
				selector, radiusMeters, lat, lng,
			))
		}
	}
	qb.WriteString(");\nout center tags;\n")

	body := []byte(qb.String())
	req, err := http.NewRequest("POST", "https://overpass-api.de/api/interpreter", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("poi: build request: %w", err)
	}
	req.Header.Set("Content-Type", "text/plain")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("poi: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		snippet := strings.TrimSpace(string(raw))
		if len(snippet) > 200 {
			snippet = snippet[:200]
		}
		return nil, fmt.Errorf("poi: status %d: %s", resp.StatusCode, snippet)
	}

	var parsed overpassResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("poi: decode: %w", err)
	}

	items := make([]POI, 0, len(parsed.Elements))
	seen := map[string]bool{}
	for _, e := range parsed.Elements {
		if e.Tags == nil {
			continue
		}
		name := e.Tags["name"]
		if name == "" {
			continue
		}

		lat, lon := e.Lat, e.Lon
		if e.Center != nil {
			lat, lon = e.Center.Lat, e.Center.Lon
		}
		if lat == 0 && lon == 0 {
			continue
		}

		category := classifyPOI(e.Tags, valid)
		if category == "" {
			continue
		}

		dedupKey := fmt.Sprintf("%s|%.5f|%.5f", category, lat, lon)
		if seen[dedupKey] {
			continue
		}
		seen[dedupKey] = true

		items = append(items, POI{
			ID:        fmt.Sprintf("%s-%d", e.Type, e.ID),
			Category:  category,
			Name:      name,
			Latitude:  lat,
			Longitude: lon,
			Address:   buildAddress(e.Tags),
			Phone:     e.Tags["phone"],
		})

		if len(items) >= 200 {
			break
		}
	}

	poiCacheMu.Lock()
	poiCache[key] = poiCacheEntry{items: items, expires: time.Now().Add(1 * time.Hour)}
	poiCacheMu.Unlock()

	return items, nil
}

// classifyPOI returns the requested category that best matches the tags.
func classifyPOI(tags map[string]string, requested []string) string {
	amenity := tags["amenity"]
	tourism := tags["tourism"]
	historic := tags["historic"]

	for _, c := range requested {
		switch c {
		case "hospital":
			if amenity == "hospital" || amenity == "clinic" {
				return c
			}
		case "police":
			if amenity == "police" {
				return c
			}
		case "fire":
			if amenity == "fire_station" {
				return c
			}
		case "pharmacy":
			if amenity == "pharmacy" {
				return c
			}
		case "bank":
			if amenity == "bank" || amenity == "atm" {
				return c
			}
		case "school":
			if amenity == "school" || amenity == "university" || amenity == "college" {
				return c
			}
		case "church":
			if amenity == "place_of_worship" && tags["religion"] == "christian" {
				return c
			}
		case "mosque":
			if amenity == "place_of_worship" && tags["religion"] == "muslim" {
				return c
			}
		case "landmark":
			if tourism == "attraction" || historic == "monument" || historic == "memorial" {
				return c
			}
		}
	}
	return ""
}

// buildAddress assembles a one-line address from standard OSM address tags.
func buildAddress(tags map[string]string) string {
	parts := []string{}
	if v := tags["addr:housenumber"]; v != "" {
		parts = append(parts, v)
	}
	if v := tags["addr:street"]; v != "" {
		parts = append(parts, v)
	}
	if v := tags["addr:suburb"]; v != "" {
		parts = append(parts, v)
	}
	if v := tags["addr:city"]; v != "" {
		parts = append(parts, v)
	}
	return strings.Join(parts, ", ")
}
