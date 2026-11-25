package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"security-solution/config"
	"security-solution/models"
)

// GetLocalNews returns the 50 most recently published local news articles.
func GetLocalNews(c *gin.Context) {
	articles := make([]models.LocalNewsArticle, 0, 50)
	if err := config.DB.
		Order("published_at DESC").
		Limit(50).
		Find(&articles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch local news"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"articles": articles})
}
