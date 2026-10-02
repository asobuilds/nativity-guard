package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// NewsService queries The Guardian Open Platform. Free, no rate limit
// issues at our scale. Key is read from GUARDIAN_API_KEY.
type NewsService struct {
	apiKey string
	client *http.Client
}

func NewNewsService() *NewsService {
	return &NewsService{
		apiKey: strings.TrimSpace(os.Getenv("GUARDIAN_API_KEY")),
		client: &http.Client{Timeout: 20 * time.Second},
	}
}

// NewsItem is one article returned to the frontend.
type NewsItem struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Summary     string    `json:"summary"`
	URL         string    `json:"url"`
	Section     string    `json:"section"`
	PublishedAt time.Time `json:"publishedAt"`
	Thumbnail   string    `json:"thumbnail,omitempty"`
	Category    string    `json:"category"`
}

type guardianResponse struct {
	Response struct {
		Status  string `json:"status"`
		Results []struct {
			ID                 string `json:"id"`
			WebTitle           string `json:"webTitle"`
			WebURL             string `json:"webUrl"`
			SectionName        string `json:"sectionName"`
			WebPublicationDate string `json:"webPublicationDate"`
			Fields             struct {
				TrailText string `json:"trailText"`
				Thumbnail string `json:"thumbnail"`
			} `json:"fields"`
		} `json:"results"`
	} `json:"response"`
}

type newsCacheEntry struct {
	items   []NewsItem
	expires time.Time
}

var (
	newsCacheMu sync.Mutex
	newsCache   = map[string]newsCacheEntry{}
)

func categoryQuery(category string) string {
	switch category {
	case "security":
		return `(crime OR robbery OR kidnapping OR banditry OR "armed attack")`
	case "community":
		return `(community OR "public safety" OR "local government")`
	case "weather":
		return `(flood OR storm OR "heavy rain" OR "weather warning")`
	case "general":
		return `(Nigeria OR Lagos OR Abuja)`
	}
	return `(Nigeria)`
}

// GetForCategories returns up to `limit` articles matching any of the given
// categories, newest first, cached for 15 minutes per category set.
func (s *NewsService) GetForCategories(categories []string, limit int) ([]NewsItem, error) {
	if s.apiKey == "" {
		return nil, fmt.Errorf("news: GUARDIAN_API_KEY not set")
	}
	if limit <= 0 || limit > 30 {
		limit = 10
	}
	if len(categories) == 0 {
		categories = []string{"general"}
	}

	sorted := append([]string{}, categories...)
	sortStrings(sorted)
	key := strings.Join(sorted, ",")

	newsCacheMu.Lock()
	if entry, ok := newsCache[key]; ok && time.Now().Before(entry.expires) {
		newsCacheMu.Unlock()
		if len(entry.items) > limit {
			return entry.items[:limit], nil
		}
		return entry.items, nil
	}
	newsCacheMu.Unlock()

	parts := []string{}
	for _, c := range sorted {
		parts = append(parts, categoryQuery(c))
	}
	combined := strings.Join(parts, " OR ")

	q := url.Values{}
	q.Set("q", combined)
	q.Set("tag", "world/nigeria")
	q.Set("order-by", "newest")
	q.Set("page-size", fmt.Sprintf("%d", limit*2))
	q.Set("show-fields", "trailText,thumbnail")
	q.Set("api-key", s.apiKey)

	endpoint := "https://content.guardianapis.com/search?" + q.Encode()

	resp, err := s.client.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("news: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		snippet := strings.TrimSpace(string(body))
		if len(snippet) > 200 {
			snippet = snippet[:200]
		}
		return nil, fmt.Errorf("news: status %d: %s", resp.StatusCode, snippet)
	}

	var parsed guardianResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("news: decode: %w", err)
	}

	items := make([]NewsItem, 0, len(parsed.Response.Results))
	seen := map[string]bool{}
	for _, r := range parsed.Response.Results {
		if seen[r.ID] {
			continue
		}
		seen[r.ID] = true
		pub, _ := time.Parse(time.RFC3339, r.WebPublicationDate)

		// bestCategory needs the inner anonymous struct's fields; extract them.
		text := strings.ToLower(r.WebTitle + " " + r.SectionName + " " + r.Fields.TrailText)
		category := pickCategory(text, sorted)

		items = append(items, NewsItem{
			ID:          r.ID,
			Title:       r.WebTitle,
			Summary:     stripHTML(r.Fields.TrailText),
			URL:         r.WebURL,
			Section:     r.SectionName,
			PublishedAt: pub,
			Thumbnail:   r.Fields.Thumbnail,
			Category:    category,
		})
	}

	newsCacheMu.Lock()
	newsCache[key] = newsCacheEntry{items: items, expires: time.Now().Add(15 * time.Minute)}
	newsCacheMu.Unlock()

	if len(items) > limit {
		return items[:limit], nil
	}
	return items, nil
}

// pickCategory picks the category that best matches an article's text.
func pickCategory(text string, available []string) string {
	for _, c := range available {
		switch c {
		case "security":
			if strings.Contains(text, "crime") || strings.Contains(text, "robber") ||
				strings.Contains(text, "kidnap") || strings.Contains(text, "bandit") ||
				strings.Contains(text, "attack") || strings.Contains(text, "police") {
				return c
			}
		case "community":
			if strings.Contains(text, "community") || strings.Contains(text, "safety") ||
				strings.Contains(text, "government") {
				return c
			}
		case "weather":
			if strings.Contains(text, "flood") || strings.Contains(text, "storm") ||
				strings.Contains(text, "rain") || strings.Contains(text, "weather") {
				return c
			}
		}
	}
	if len(available) > 0 {
		return available[0]
	}
	return "general"
}

// stripHTML removes simple HTML tags from Guardian summary text.
func stripHTML(s string) string {
	out := strings.Builder{}
	in := false
	for _, r := range s {
		if r == '<' {
			in = true
			continue
		}
		if r == '>' {
			in = false
			continue
		}
		if !in {
			out.WriteRune(r)
		}
	}
	return strings.TrimSpace(out.String())
}