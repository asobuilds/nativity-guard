package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"security-solution/config"
	"security-solution/models"
)

type localIntelItem struct {
	ID         uuid.UUID  `json:"id"`
	Fact       string     `json:"fact"`
	Category   string     `json:"category"`
	Confidence string     `json:"confidence"`
	State      string     `json:"state,omitempty"`
	LGA        string     `json:"lga,omitempty"`
	Upvotes    int        `json:"upvotes"`
	HasUpvoted bool       `json:"hasUpvoted"`
	ExpiresAt  *time.Time `json:"expiresAt,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
}

// GetLocalIntel — GET /local-intel
// Returns the caller's unexpired local facts, most-upvoted first.
func GetLocalIntel(c *gin.Context) {
	userValue, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	user, ok := userValue.(*models.User)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	now := time.Now().UTC()
	state, lga := callerArea(user)

	query := config.DB.Model(&models.LocalIntelligence{}).
		Where("(expires_at IS NULL OR expires_at > ?)", now)
	if state != "" && lga != "" {
		query = query.Where("((state = ? AND lga = ?) OR state = ? OR lga = '' OR state = '')", state, lga, state)
	} else if state != "" {
		query = query.Where("(state = ? OR state = '')", state)
	}

	var intel []models.LocalIntelligence
	if err := query.Order("upvotes desc, created_at desc").Limit(20).Find(&intel).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load local intelligence"})
		return
	}

	voted := make(map[uuid.UUID]bool, len(intel))
	if len(intel) > 0 {
		ids := make([]uuid.UUID, 0, len(intel))
		for _, item := range intel {
			ids = append(ids, item.ID)
		}
		var votes []models.LocalIntelligenceVote
		if err := config.DB.
			Where("user_id = ? AND intel_id IN ?", user.ID, ids).
			Find(&votes).Error; err == nil {
			for _, vote := range votes {
				voted[vote.IntelID] = true
			}
		}
	}

	items := make([]localIntelItem, 0, len(intel))
	for _, item := range intel {
		items = append(items, localIntelItem{
			ID:         item.ID,
			Fact:       item.Fact,
			Category:   item.Category,
			Confidence: item.Confidence,
			State:      item.State,
			LGA:        item.LGA,
			Upvotes:    item.Upvotes,
			HasUpvoted: voted[item.ID],
			ExpiresAt:  item.ExpiresAt,
			CreatedAt:  item.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"items": items})
}

// UpvoteLocalIntel — POST /local-intel/:id/upvote
// One vote per user per fact; a repeat tap is a no-op that still returns
// the current count so the UI stays in sync.
func UpvoteLocalIntel(c *gin.Context) {
	userValue, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	user, ok := userValue.(*models.User)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	intelID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid intelligence id"})
		return
	}

	now := time.Now().UTC()
	var intel models.LocalIntelligence
	if err := config.DB.
		Where("id = ? AND (expires_at IS NULL OR expires_at > ?)", intelID, now).
		First(&intel).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "intelligence not found"})
		return
	}

	vote := models.LocalIntelligenceVote{IntelID: intelID, UserID: user.ID}
	result := config.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&vote)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record upvote"})
		return
	}

	if result.RowsAffected > 0 {
		if err := config.DB.Model(&models.LocalIntelligence{}).
			Where("id = ?", intelID).
			Updates(map[string]any{
				"upvotes":         gorm.Expr("upvotes + 1"),
				"last_upvoted_at": now,
			}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record upvote"})
			return
		}
	}

	if err := config.DB.First(&intel, "id = ?", intelID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load intelligence"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":         intel.ID,
		"upvotes":    intel.Upvotes,
		"hasUpvoted": true,
	})
}
