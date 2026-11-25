package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm/clause"

	"security-solution/config"
	"security-solution/models"
)

const (
	localNewsEndpoint         = "https://freenewsapi.ai/v1/search"
	localNewsPageSize         = 20
	localNewsDefaultCategory  = "General"
	localNewsRequestTimeout   = 20 * time.Second
	localNewsResponseSnippets = 200
)

type localNewsSource struct {
	Host string
	Name string
}

var localNewsSources = []localNewsSource{
	{Host: "nnu.ng", Name: "Nigeria News Update"},
	{Host: "newscarte.com.ng", Name: "Newscarte"},
}

type localNewsPayload struct {
	URL         string    `json:"url"`
	Title       string    `json:"title"`
	Description *string   `json:"description"`
	PublishedAt time.Time `json:"published_at"`
	Sitename    string    `json:"sitename"`
	Categories  []string  `json:"categories"`
}

type localNewsResponse struct {
	Results []localNewsPayload `json:"results"`
}

// FetchNigerianNews pulls the two configured Nigerian publisher feeds and
// stores any new articles. It attempts both sources even if one fails.
func FetchNigerianNews() error {
	client := &http.Client{Timeout: localNewsRequestTimeout}
	var errs []error

	for _, source := range localNewsSources {
		if err := fetchAndStoreLocalNews(client, localNewsEndpoint, source); err != nil {
			errs = append(errs, fmt.Errorf("local news %s: %w", source.Name, err))
		}
	}

	return errors.Join(errs...)
}

func fetchAndStoreLocalNews(client *http.Client, endpoint string, source localNewsSource) error {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return fmt.Errorf("parse endpoint: %w", err)
	}

	query := parsed.Query()
	query.Set("host", source.Host)
	query.Set("size", strconv.Itoa(localNewsPageSize))
	parsed.RawQuery = query.Encode()

	req, err := http.NewRequest(http.MethodGet, parsed.String(), nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "NativityGuard/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, localNewsResponseSnippets))
		return fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	articles, err := decodeLocalNewsArticles(resp.Body, source.Name)
	if err != nil {
		return err
	}

	inserted, skipped, storeErrs := storeLocalNewsArticles(articles)
	log.Printf(
		"local news: source=%s fetched=%d inserted=%d skipped=%d",
		source.Name, len(articles), inserted, skipped,
	)

	return errors.Join(storeErrs...)
}

func decodeLocalNewsArticles(r io.Reader, fallbackSource string) ([]models.LocalNewsArticle, error) {
	var payload localNewsResponse
	if err := json.NewDecoder(r).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	articles := make([]models.LocalNewsArticle, 0, len(payload.Results))
	seenURLs := make(map[string]struct{}, len(payload.Results))

	for _, item := range payload.Results {
		articleURL := strings.TrimSpace(item.URL)
		title := strings.TrimSpace(item.Title)
		if articleURL == "" || title == "" {
			continue
		}
		if _, exists := seenURLs[articleURL]; exists {
			continue
		}
		seenURLs[articleURL] = struct{}{}

		description := ""
		if item.Description != nil {
			description = strings.TrimSpace(*item.Description)
		}

		category := localNewsDefaultCategory
		for _, candidate := range item.Categories {
			if candidate = strings.TrimSpace(candidate); candidate != "" {
				category = candidate
				break
			}
		}

		sourceName := strings.TrimSpace(item.Sitename)
		if sourceName == "" {
			sourceName = strings.TrimSpace(fallbackSource)
		}

		publishedAt := item.PublishedAt
		if !publishedAt.IsZero() {
			publishedAt = publishedAt.UTC()
		}

		articles = append(articles, models.LocalNewsArticle{
			Title:       title,
			Description: description,
			URL:         articleURL,
			SourceName:  sourceName,
			Category:    category,
			PublishedAt: publishedAt,
		})
	}

	return articles, nil
}

func storeLocalNewsArticles(articles []models.LocalNewsArticle) (inserted int, skipped int, errs []error) {
	for i := range articles {
		article := articles[i]

		var existing int64
		if err := config.DB.Model(&models.LocalNewsArticle{}).
			Where("url = ?", article.URL).
			Count(&existing).Error; err != nil {
			errs = append(errs, fmt.Errorf("check duplicate %s: %w", article.URL, err))
			continue
		}
		if existing > 0 {
			skipped++
			continue
		}

		result := config.DB.
			Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "url"}},
				DoNothing: true,
			}).
			Create(&article)
		if result.Error != nil {
			errs = append(errs, fmt.Errorf("store %s: %w", article.URL, result.Error))
			continue
		}
		if result.RowsAffected == 0 {
			skipped++
			continue
		}
		inserted++
	}

	return inserted, skipped, errs
}
