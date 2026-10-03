package handlers

import (
    "net/http"
    "strings"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"

    "security-solution/config"
    "security-solution/models"
)

var validLGACategories = map[string]bool{
    "intel":        true,
    "alert":        true,
    "coordination": true,
    "request":      true,
    "general":      true,
}

// callerLGAContext resolves the caller's LGA channel by walking their
// verified membership to a verified unit. Returns ok=false when the caller
// is not eligible — the normal, expected answer for citizens and for staff
// whose unit is not yet verified.
func callerLGAContext(c *gin.Context) (lga, state string, unitID uuid.UUID, unitName, userName string, ok bool) {
    userVal, exists := c.Get("user")
    if !exists {
        return "", "", uuid.Nil, "", "", false
    }
    user, cast := userVal.(*models.User)
    if !cast || user == nil {
        return "", "", uuid.Nil, "", "", false
    }

    var m models.UnitMembership
    err := config.DB.
        Where("user_id = ? AND status = ?", user.ID, "active").
        Order("created_at ASC").
        First(&m).Error
    if err != nil {
        return "", "", uuid.Nil, "", "", false
    }

    var unit models.SecurityUnit
    if err := config.DB.First(&unit, "id = ?", m.UnitID).Error; err != nil {
        return "", "", uuid.Nil, "", "", false
    }
    if !unit.IsVerified {
        return "", "", uuid.Nil, "", "", false
    }
    if strings.TrimSpace(unit.LGA) == "" || strings.TrimSpace(unit.State) == "" {
        return "", "", uuid.Nil, "", "", false
    }

    fullName := strings.TrimSpace(user.FirstName + " " + user.LastName)
    if fullName == "" {
        fullName = user.Email
    }
    return unit.LGA, unit.State, unit.ID, unit.Name, fullName, true
}

// GetLGAChannel — GET /lga/channel
func GetLGAChannel(c *gin.Context) {
    lga, state, _, _, _, ok := callerLGAContext(c)
    if !ok {
        c.JSON(http.StatusForbidden, gin.H{
            "error": "LGA channel is available to verified security unit members only",
        })
        return
    }

    var msgs []models.LGAMessage
    now := time.Now().UTC()
    if err := config.DB.
        Where("state = ? AND lga = ?", state, lga).
        Where("expires_at IS NULL OR expires_at > ?", now).
        Order("created_at desc").
        Limit(100).
        Find(&msgs).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load messages"})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "channel": gin.H{
            "lga":   lga,
            "state": state,
        },
        "messages": msgs,
        "count":    len(msgs),
    })
}

// PostLGAChannel — POST /lga/channel
//
// The LGA and state are taken from the caller's unit — never the request.
// Rate limit: 5 posts per hour per user.
func PostLGAChannel(c *gin.Context) {
    lga, state, unitID, unitName, authorName, ok := callerLGAContext(c)
    if !ok {
        c.JSON(http.StatusForbidden, gin.H{
            "error": "LGA channel is available to verified security unit members only",
        })
        return
    }

    var input struct {
        Category       string `json:"category" binding:"required"`
        Priority       string `json:"priority"`
        Title          string `json:"title" binding:"required"`
        Body           string `json:"body" binding:"required"`
        ExpiresInHours int    `json:"expiresInHours"`
    }
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "category, title and body are required"})
        return
    }

    input.Category = strings.ToLower(strings.TrimSpace(input.Category))
    if !validLGACategories[input.Category] {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "category must be one of: intel, alert, coordination, request, general",
        })
        return
    }

    input.Priority = strings.ToLower(strings.TrimSpace(input.Priority))
    if input.Priority == "" {
        input.Priority = "normal"
    }
    if input.Priority != "normal" && input.Priority != "urgent" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "priority must be normal or urgent"})
        return
    }

    input.Title = strings.TrimSpace(input.Title)
    input.Body = strings.TrimSpace(input.Body)
    if input.Title == "" || input.Body == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "title and body cannot be empty"})
        return
    }
    if len(input.Title) > 200 {
        input.Title = input.Title[:200]
    }
    if len(input.Body) > 4000 {
        input.Body = input.Body[:4000]
    }

    userVal, _ := c.Get("user")
    user := userVal.(*models.User)

    // Rate limit: 5 posts per user per hour, globally.
    var recentCount int64
    config.DB.Model(&models.LGAMessage{}).
        Where("author_user_id = ? AND created_at > ?", user.ID, time.Now().UTC().Add(-1*time.Hour)).
        Count(&recentCount)
    if recentCount >= 5 {
        c.JSON(http.StatusTooManyRequests, gin.H{
            "error": "you have reached the posting limit for this hour",
        })
        return
    }

    var expiresAt *time.Time
    if input.ExpiresInHours > 0 {
        t := time.Now().UTC().Add(time.Duration(input.ExpiresInHours) * time.Hour)
        expiresAt = &t
    }

    msg := models.LGAMessage{
        State:        state,
        LGA:          lga,
        Category:     input.Category,
        Priority:     input.Priority,
        Title:        input.Title,
        Body:         input.Body,
        AuthorUserID: user.ID,
        AuthorUnitID: unitID,
        AuthorName:   authorName,
        AuthorUnit:   unitName,
        ExpiresAt:    expiresAt,
    }
    if err := config.DB.Create(&msg).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to post message"})
        return
    }

    c.JSON(http.StatusCreated, gin.H{"message": msg})
}
