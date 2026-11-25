package models

import (
	"time"

	"github.com/google/uuid"
)

// LocalNewsArticle is an article pulled from a Nigerian news publisher feed.
type LocalNewsArticle struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Title       string    `gorm:"not null" json:"title"`
	Description string    `gorm:"type:text" json:"description"`
	URL         string    `gorm:"not null;uniqueIndex" json:"url"`
	SourceName  string    `json:"sourceName"`
	Category    string    `gorm:"not null;default:General" json:"category"`
	PublishedAt time.Time `gorm:"index" json:"publishedAt"`
	CreatedAt   time.Time `json:"createdAt"`
}

func (LocalNewsArticle) TableName() string {
	return "local_news_articles"
}
