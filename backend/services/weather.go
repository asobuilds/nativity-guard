package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// WeatherService talks to Open-Meteo. No API key, no cost, no account.
// Docs: https://open-meteo.com/en/docs
type WeatherService struct {
	client *http.Client
}

func NewWeatherService() *WeatherService {
	return &WeatherService{
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// CurrentWeather is the shape returned to the frontend.
type CurrentWeather struct {
	Temperature  float64 `json:"temperature"`
	FeelsLike    float64 `json:"apparentTemperature"`
	WeatherCode  int     `json:"weatherCode"`
	WeatherLabel string  `json:"weatherLabel"`
	Icon         string  `json:"icon"`
	WindKph      float64 `json:"windKph"`
	Humidity     float64 `json:"humidity"`
	IsDay        bool    `json:"isDay"`
	ObservedAt   string  `json:"observedAt"`
}

// HourlyPoint is one entry in the 12-hour forecast strip.
type HourlyPoint struct {
	Time                     string  `json:"time"`
	Temperature              float64 `json:"temperature"`
	WeatherCode              int     `json:"weatherCode"`
	WeatherLabel             string  `json:"weatherLabel"`
	Icon                     string  `json:"icon"`
	PrecipitationProbability float64 `json:"precipitationProbability"`
}

// DailyPoint is one entry in the 7-day strip.
type DailyPoint struct {
	Date                     string  `json:"date"`
	TempMax                  float64 `json:"tempMax"`
	TempMin                  float64 `json:"tempMin"`
	WeatherCode              int     `json:"weatherCode"`
	WeatherLabel             string  `json:"weatherLabel"`
	Icon                     string  `json:"icon"`
	PrecipitationProbability float64 `json:"precipitationProbability"`
}

// WeatherBundle is the whole response.
type WeatherBundle struct {
	Location string         `json:"location"`
	Timezone string         `json:"timezone"`
	Current  CurrentWeather `json:"current"`
	Hourly   []HourlyPoint  `json:"hourly"`
	Daily    []DailyPoint   `json:"daily"`
	Advisory string         `json:"advisory,omitempty"`
	Alert    string         `json:"alert,omitempty"`
	Approx   bool           `json:"approximate"`
}

type openMeteoResponse struct {
	Timezone string `json:"timezone"`
	Current  struct {
		Time                string  `json:"time"`
		Temperature         float64 `json:"temperature_2m"`
		ApparentTemperature float64 `json:"apparent_temperature"`
		WeatherCode         int     `json:"weather_code"`
		WindSpeed           float64 `json:"wind_speed_10m"`
		Humidity            float64 `json:"relative_humidity_2m"`
		IsDay               int     `json:"is_day"`
	} `json:"current"`
	Hourly struct {
		Time                     []string  `json:"time"`
		Temperature              []float64 `json:"temperature_2m"`
		WeatherCode              []int     `json:"weather_code"`
		PrecipitationProbability []float64 `json:"precipitation_probability"`
	} `json:"hourly"`
	Daily struct {
		Time                     []string  `json:"time"`
		WeatherCode              []int     `json:"weather_code"`
		TempMax                  []float64 `json:"temperature_2m_max"`
		TempMin                  []float64 `json:"temperature_2m_min"`
		PrecipitationProbability []float64 `json:"precipitation_probability_max"`
	} `json:"daily"`
}

type weatherCacheEntry struct {
	bundle  WeatherBundle
	expires time.Time
}

var (
	weatherCacheMu sync.Mutex
	weatherCache   = map[string]weatherCacheEntry{}
)

func weatherCacheKey(lat, lng float64) string {
	// Round to 2 decimal places (~1.1km) so nearby users share cache entries.
	return fmt.Sprintf("%.2f,%.2f", lat, lng)
}

// GetWeather fetches current + hourly + daily conditions for a point.
func (s *WeatherService) GetWeather(lat, lng float64, locationLabel string, approximate bool) (WeatherBundle, error) {
	key := weatherCacheKey(lat, lng)

	weatherCacheMu.Lock()
	if entry, ok := weatherCache[key]; ok && time.Now().Before(entry.expires) {
		weatherCacheMu.Unlock()
		entry.bundle.Location = locationLabel
		entry.bundle.Approx = approximate
		return entry.bundle, nil
	}
	weatherCacheMu.Unlock()

	q := url.Values{}
	q.Set("latitude", fmt.Sprintf("%.4f", lat))
	q.Set("longitude", fmt.Sprintf("%.4f", lng))
	q.Set("current", "temperature_2m,apparent_temperature,weather_code,wind_speed_10m,relative_humidity_2m,is_day")
	q.Set("hourly", "temperature_2m,weather_code,precipitation_probability")
	q.Set("daily", "weather_code,temperature_2m_max,temperature_2m_min,precipitation_probability_max")
	q.Set("timezone", "auto")
	q.Set("forecast_days", "7")

	endpoint := "https://api.open-meteo.com/v1/forecast?" + q.Encode()

	resp, err := s.client.Get(endpoint)
	if err != nil {
		return WeatherBundle{}, fmt.Errorf("weather: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		snippet := string(body)
		if len(snippet) > 200 {
			snippet = snippet[:200]
		}
		return WeatherBundle{}, fmt.Errorf("weather: status %d: %s", resp.StatusCode, snippet)
	}

	var parsed openMeteoResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return WeatherBundle{}, fmt.Errorf("weather: decode: %w", err)
	}

	bundle := WeatherBundle{
		Location: locationLabel,
		Timezone: parsed.Timezone,
		Approx:   approximate,
		Current: CurrentWeather{
			Temperature:  parsed.Current.Temperature,
			FeelsLike:    parsed.Current.ApparentTemperature,
			WeatherCode:  parsed.Current.WeatherCode,
			WeatherLabel: wmoLabel(parsed.Current.WeatherCode),
			Icon:         wmoIcon(parsed.Current.WeatherCode),
			WindKph:      parsed.Current.WindSpeed,
			Humidity:     parsed.Current.Humidity,
			IsDay:        parsed.Current.IsDay == 1,
			ObservedAt:   parsed.Current.Time,
		},
	}

	// Next 12 hours
	now := time.Now()
	for i := 0; i < len(parsed.Hourly.Time) && len(bundle.Hourly) < 12; i++ {
		t, err := time.Parse("2006-01-02T15:04", parsed.Hourly.Time[i])
		if err != nil || t.Before(now.Add(-1*time.Hour)) {
			continue
		}
		var precip float64
		if i < len(parsed.Hourly.PrecipitationProbability) {
			precip = parsed.Hourly.PrecipitationProbability[i]
		}
		bundle.Hourly = append(bundle.Hourly, HourlyPoint{
			Time:                     parsed.Hourly.Time[i],
			Temperature:              parsed.Hourly.Temperature[i],
			WeatherCode:              parsed.Hourly.WeatherCode[i],
			WeatherLabel:             wmoLabel(parsed.Hourly.WeatherCode[i]),
			Icon:                     wmoIcon(parsed.Hourly.WeatherCode[i]),
			PrecipitationProbability: precip,
		})
	}

	// 7 days
	for i := 0; i < len(parsed.Daily.Time) && i < 7; i++ {
		var precip float64
		if i < len(parsed.Daily.PrecipitationProbability) {
			precip = parsed.Daily.PrecipitationProbability[i]
		}
		bundle.Daily = append(bundle.Daily, DailyPoint{
			Date:                     parsed.Daily.Time[i],
			TempMax:                  parsed.Daily.TempMax[i],
			TempMin:                  parsed.Daily.TempMin[i],
			WeatherCode:              parsed.Daily.WeatherCode[i],
			WeatherLabel:             wmoLabel(parsed.Daily.WeatherCode[i]),
			Icon:                     wmoIcon(parsed.Daily.WeatherCode[i]),
			PrecipitationProbability: precip,
		})
	}

	// LLM advisory — one paragraph, plain English.
	if advisory, alert, err := s.generateAdvisory(bundle); err == nil {
		bundle.Advisory = advisory
		bundle.Alert = alert
	}

	weatherCacheMu.Lock()
	weatherCache[key] = weatherCacheEntry{bundle: bundle, expires: time.Now().Add(15 * time.Minute)}
	weatherCacheMu.Unlock()

	return bundle, nil
}

// generateAdvisory asks the LLM for a 2–3 sentence summary of what the
// weather means for the user, and flags any severe condition as an alert.
func (s *WeatherService) generateAdvisory(w WeatherBundle) (string, string, error) {
	hourly := ""
	for i, h := range w.Hourly {
		if i >= 6 {
			break
		}
		hourly += fmt.Sprintf("%s: %s, %.0f°C, %.0f%% rain\n", h.Time, h.WeatherLabel, h.Temperature, h.PrecipitationProbability)
	}

	prompt := fmt.Sprintf(`Current weather in %s: %s, %.0f°C (feels like %.0f°C), wind %.0f km/h, humidity %.0f%%.
Next few hours:
%s

Write a 2-3 sentence plain-English advisory for a resident. Be practical: what to wear, whether to carry an umbrella, whether road conditions are a concern. Do not add a greeting or a sign-off.

If any condition is severe (thunderstorm, wind over 50 km/h, temperature over 40°C, or heavy rain imminent), put a one-sentence alert in the second line of your reply prefixed with ALERT:. Otherwise leave the second line empty.`,
		w.Location, w.Current.WeatherLabel, w.Current.Temperature, w.Current.FeelsLike,
		w.Current.WindKph, w.Current.Humidity, hourly)

	ai := NewAIService()
	reply, err := ai.Chat([]ChatMessage{
		{Role: "system", Content: "You write short, practical weather advisories. No preamble, no greetings, no emojis."},
		{Role: "user", Content: prompt},
	})
	if err != nil {
		return "", "", err
	}

	advisory := reply
	alert := ""
	for _, line := range splitLines(reply) {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "ALERT:") {
			alert = strings.TrimSpace(strings.TrimPrefix(trimmed, "ALERT:"))
			advisory = strings.Replace(reply, line, "", 1)
			break
		}
	}
	return strings.TrimSpace(advisory), alert, nil
}

func splitLines(s string) []string {
	return strings.Split(s, "\n")
}

// wmoLabel maps a WMO weather code to a human label.
func wmoLabel(code int) string {
	switch code {
	case 0:
		return "Clear sky"
	case 1:
		return "Mainly clear"
	case 2:
		return "Partly cloudy"
	case 3:
		return "Overcast"
	case 45, 48:
		return "Fog"
	case 51:
		return "Light drizzle"
	case 53:
		return "Moderate drizzle"
	case 55:
		return "Dense drizzle"
	case 56, 57:
		return "Freezing drizzle"
	case 61:
		return "Light rain"
	case 63:
		return "Moderate rain"
	case 65:
		return "Heavy rain"
	case 66, 67:
		return "Freezing rain"
	case 71, 73, 75:
		return "Snow"
	case 77:
		return "Snow grains"
	case 80, 81:
		return "Rain showers"
	case 82:
		return "Violent rain showers"
	case 85, 86:
		return "Snow showers"
	case 95:
		return "Thunderstorm"
	case 96, 99:
		return "Thunderstorm with hail"
	}
	return "Unknown"
}

// wmoIcon returns one of a fixed set of icon names the frontend maps.
func wmoIcon(code int) string {
	switch code {
	case 0:
		return "sun"
	case 1, 2:
		return "partly-cloudy"
	case 3:
		return "cloud"
	case 45, 48:
		return "fog"
	case 51, 53, 55, 56, 57:
		return "drizzle"
	case 61, 63, 65, 66, 67, 80, 81, 82:
		return "rain"
	case 71, 73, 75, 77, 85, 86:
		return "snow"
	case 95, 96, 99:
		return "thunderstorm"
	}
	return "cloud"
}