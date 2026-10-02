package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// DirectionsService wraps OSRM's public demo server. No API key. Free.
// Coverage is global via OpenStreetMap.
type DirectionsService struct {
	client *http.Client
}

func NewDirectionsService() *DirectionsService {
	return &DirectionsService{
		client: &http.Client{Timeout: 20 * time.Second},
	}
}

// RouteStep is one turn-by-turn instruction.
type RouteStep struct {
	Instruction    string  `json:"instruction"`
	DistanceMeters float64 `json:"distanceMeters"`
}

// Route is the shape returned to the frontend.
type Route struct {
	DistanceMeters  float64     `json:"distanceMeters"`
	DurationSeconds float64     `json:"durationSeconds"`
	Geometry        string      `json:"geometry"` // encoded polyline
	Steps           []RouteStep `json:"steps"`
	Summary         string      `json:"summary"`
	Profile         string      `json:"profile"`
}

type osrmResponse struct {
	Code   string `json:"code"`
	Routes []struct {
		Distance float64 `json:"distance"`
		Duration float64 `json:"duration"`
		Geometry string  `json:"geometry"`
		Legs     []struct {
			Steps []struct {
				Distance  float64 `json:"distance"`
				Name      string  `json:"name"`
				Maneuver  struct {
					Type     string `json:"type"`
					Modifier string `json:"modifier"`
				} `json:"maneuver"`
			} `json:"steps"`
		} `json:"legs"`
	} `json:"routes"`
	Message string `json:"message"`
}

// GetRoute computes a route between two points using the given profile
// ("driving", "cycling" or "walking"). Defaults to driving.
func (s *DirectionsService) GetRoute(fromLat, fromLng, toLat, toLng float64, profile string) (Route, error) {
	switch profile {
	case "driving", "cycling", "walking":
	default:
		profile = "driving"
	}

	endpoint := fmt.Sprintf(
		"https://router.project-osrm.org/route/v1/%s/%.6f,%.6f;%.6f,%.6f?overview=full&steps=true&geometries=polyline",
		profile, fromLng, fromLat, toLng, toLat,
	)

	resp, err := s.client.Get(endpoint)
	if err != nil {
		return Route{}, fmt.Errorf("directions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		snippet := strings.TrimSpace(string(body))
		if len(snippet) > 200 {
			snippet = snippet[:200]
		}
		return Route{}, fmt.Errorf("directions: status %d: %s", resp.StatusCode, snippet)
	}

	var parsed osrmResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return Route{}, fmt.Errorf("directions: decode: %w", err)
	}

	if parsed.Code != "Ok" || len(parsed.Routes) == 0 {
		msg := parsed.Message
		if msg == "" {
			msg = parsed.Code
		}
		return Route{}, fmt.Errorf("directions: %s", msg)
	}

	r := parsed.Routes[0]
	route := Route{
		DistanceMeters:  r.Distance,
		DurationSeconds: r.Duration,
		Geometry:        r.Geometry,
		Profile:         profile,
	}

	if len(r.Legs) > 0 {
		for _, step := range r.Legs[0].Steps {
			instr := humanStep(step.Maneuver.Type, step.Maneuver.Modifier, step.Name)
			if instr == "" {
				continue
			}
			route.Steps = append(route.Steps, RouteStep{
				Instruction:    instr,
				DistanceMeters: step.Distance,
			})
		}
	}

	route.Summary = fmt.Sprintf("%s · about %s %s",
		formatDistance(r.Distance),
		formatDuration(r.Duration),
		profile,
	)

	return route, nil
}

// humanStep converts OSRM's maneuver codes into a natural phrase.
func humanStep(maneuverType, modifier, roadName string) string {
	road := ""
	if roadName != "" {
		road = " onto " + roadName
	}
	switch maneuverType {
	case "depart":
		if roadName != "" {
			return "Head out on " + roadName
		}
		return "Start the route"
	case "arrive":
		return "Arrive at your destination"
	case "turn":
		switch modifier {
		case "left":
			return "Turn left" + road
		case "right":
			return "Turn right" + road
		case "slight left":
			return "Bear left" + road
		case "slight right":
			return "Bear right" + road
		case "sharp left":
			return "Sharp left" + road
		case "sharp right":
			return "Sharp right" + road
		case "uturn":
			return "Make a U-turn"
		default:
			return "Continue" + road
		}
	case "roundabout", "rotary":
		return "Take the roundabout" + road
	case "merge":
		return "Merge" + road
	case "fork":
		return "Keep " + modifier + road
	case "continue":
		return "Continue" + road
	case "new name":
		return "Continue" + road
	case "end of road":
		return "Turn " + modifier + road
	case "on ramp":
		return "Take the ramp" + road
	case "off ramp":
		return "Take the exit" + road
	}
	if roadName != "" {
		return "Continue on " + roadName
	}
	return ""
}

func formatDistance(meters float64) string {
	if meters < 1000 {
		return fmt.Sprintf("%.0f m", meters)
	}
	return fmt.Sprintf("%.1f km", meters/1000)
}

func formatDuration(seconds float64) string {
	mins := int(seconds / 60)
	if mins < 1 {
		return "less than a minute"
	}
	if mins < 60 {
		return fmt.Sprintf("%d min", mins)
	}
	h := mins / 60
	m := mins % 60
	if m == 0 {
		return fmt.Sprintf("%d h", h)
	}
	return fmt.Sprintf("%d h %d min", h, m)
}