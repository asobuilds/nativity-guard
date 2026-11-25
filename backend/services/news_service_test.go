package services

import (
	"strings"
	"testing"
	"time"
)

func TestDecodeLocalNewsArticlesNormalizesAndDeduplicates(t *testing.T) {
	const payload = `{
		"results": [
			{
				"url": "https://example.com/one",
				"title": " First article ",
				"description": null,
				"published_at": "2026-10-26T11:57:45Z",
				"sitename": "Example News",
				"categories": null
			},
			{
				"url": "https://example.com/two",
				"title": "Second article",
				"description": " Some description ",
				"published_at": "2026-10-26T11:58:45Z",
				"sitename": "",
				"categories": ["", "politics", "crime"]
			},
			{
				"url": "https://example.com/one",
				"title": "Duplicate article",
				"description": "should be ignored",
				"published_at": "2026-10-26T11:59:45Z",
				"sitename": "Duplicate",
				"categories": ["crime"]
			},
			{
				"url": "",
				"title": "Missing URL",
				"description": "should be ignored",
				"published_at": "2026-10-26T12:00:45Z",
				"sitename": "Example News",
				"categories": ["crime"]
			}
		]
	}`

	articles, err := decodeLocalNewsArticles(strings.NewReader(payload), "Fallback Source")
	if err != nil {
		t.Fatalf("decodeLocalNewsArticles: %v", err)
	}
	if len(articles) != 2 {
		t.Fatalf("expected 2 articles, got %d", len(articles))
	}

	first := articles[0]
	if first.Title != "First article" {
		t.Errorf("title = %q, want %q", first.Title, "First article")
	}
	if first.Description != "" {
		t.Errorf("description = %q, want empty", first.Description)
	}
	if first.Category != localNewsDefaultCategory {
		t.Errorf("category = %q, want %q", first.Category, localNewsDefaultCategory)
	}
	if first.SourceName != "Example News" {
		t.Errorf("source = %q, want %q", first.SourceName, "Example News")
	}
	wantPublished := time.Date(2026, 10, 26, 11, 57, 45, 0, time.UTC)
	if !first.PublishedAt.Equal(wantPublished) {
		t.Errorf("publishedAt = %v, want %v", first.PublishedAt, wantPublished)
	}

	second := articles[1]
	if second.Category != "politics" {
		t.Errorf("category = %q, want first non-empty category %q", second.Category, "politics")
	}
	if second.SourceName != "Fallback Source" {
		t.Errorf("source = %q, want fallback %q", second.SourceName, "Fallback Source")
	}
	if second.Description != "Some description" {
		t.Errorf("description = %q, want trimmed description", second.Description)
	}
}

func TestDecodeLocalNewsArticlesRejectsInvalidJSON(t *testing.T) {
	if _, err := decodeLocalNewsArticles(strings.NewReader(`{"results":`), "Fallback"); err == nil {
		t.Fatal("expected invalid JSON to return an error")
	}
}
