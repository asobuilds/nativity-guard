package handlers

import (
    "errors"
    "net/http"
    "strings"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "gorm.io/gorm"

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
//
// isHeadAdmin reports whether the caller holds the head-admin right on their
// membership. Reading the channel needs only eligibility; posting needs
// isHeadAdmin (RES-012, decision D-015).
func callerLGAContext(c *gin.Context) (lga, state string, unitID uuid.UUID, unitName, userName string, isHeadAdmin, ok bool) {
    userVal, exists := c.Get("user")
    if !exists {
        return "", "", uuid.Nil, "", "", false, false
    }
    user, cast := userVal.(*models.User)
    if !cast || user == nil {
        return "", "", uuid.Nil, "", "", false, false
    }

    var m models.UnitMembership
    err := config.DB.
        Where("user_id = ? AND status = ?", user.ID, "active").
        Order("created_at ASC").
        First(&m).Error
    if err != nil {
        return "", "", uuid.Nil, "", "", false, false
    }

    var unit models.SecurityUnit
    if err := config.DB.First(&unit, "id = ?", m.UnitID).Error; err != nil {
        return "", "", uuid.Nil, "", "", false, false
    }
    if !unit.IsVerified {
        return "", "", uuid.Nil, "", "", false, false
    }
    if strings.TrimSpace(unit.LGA) == "" || strings.TrimSpace(unit.State) == "" {
        return "", "", uuid.Nil, "", "", false, false
    }

    fullName := strings.TrimSpace(user.FirstName + " " + user.LastName)
    if fullName == "" {
        fullName = user.Email
    }
    return unit.LGA, unit.State, unit.ID, unit.Name, fullName, membershipIsHeadAdmin(m), true
}

// membershipIsHeadAdmin mirrors the existing head-admin rule used across the
// unit handlers (e.g. officers_handler.go): the flag, or the unit-admin role.
func membershipIsHeadAdmin(m models.UnitMembership) bool {
    return m.IsHeadAdmin || m.Role == models.UnitRoleAdmin
}

// GetLGAChannel — GET /lga/channel
//
// Read is open to every verified unit member in the LGA. `canPost` tells the
// client whether to offer the composer (head admins only).
func GetLGAChannel(c *gin.Context) {
    lga, state, _, _, _, isHeadAdmin, ok := callerLGAContext(c)
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
        "canPost":  isHeadAdmin,
    })
}

// PostLGAChannel — POST /lga/channel
//
// The LGA and state are taken from the caller's unit — never the request.
// **Only head admins may post** (RES-012, decision D-015): every verified unit
// in the LGA reads, but the group page has a single, accountable voice for
// creating posts. Rate limit: 5 posts per hour per user.
func PostLGAChannel(c *gin.Context) {
    lga, state, unitID, unitName, authorName, isHeadAdmin, ok := callerLGAContext(c)
    if !ok {
        c.JSON(http.StatusForbidden, gin.H{
            "error": "LGA channel is available to verified security unit members only",
        })
        return
    }
    if !isHeadAdmin {
        c.JSON(http.StatusForbidden, gin.H{
            "error": "only head admins can post to the LGA group",
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

// lgaMemberResponse shapes a group membership for a client.
func lgaMemberResponse(m models.LGAGroupMember) gin.H {
    return gin.H{
        "id":        m.ID,
        "unitId":    m.UnitID,
        "unitName":  m.UnitName,
        "joinedAt":  m.CreatedAt,
    }
}

// JoinLGAGroup — POST /community/lga-group/join
//
// The caller's verified unit joins the convergence group for the LGA it is
// registered in. The LGA and state come from the unit, never the request, so a
// unit can only ever join its own area. Joining grants the roster, not the
// posting right.
func JoinLGAGroup(c *gin.Context) {
    lga, state, unitID, unitName, _, _, ok := callerLGAContext(c)
    if !ok {
        c.JSON(http.StatusForbidden, gin.H{
            "error": "only verified security unit members can join the LGA group",
        })
        return
    }

    userVal, _ := c.Get("user")
    user := userVal.(*models.User)

    var existing models.LGAGroupMember
    err := config.DB.Unscoped().
        Where("state = ? AND lga = ? AND unit_id = ?", state, lga, unitID).
        First(&existing).Error

    switch {
    case err == nil && !existing.DeletedAt.Valid:
        c.JSON(http.StatusOK, gin.H{"message": "already joined", "member": lgaMemberResponse(existing)})
        return
    case err == nil:
        // The unit left earlier; re-activate the same row rather than risk a
        // partial-unique violation on the live index.
        if err := config.DB.Unscoped().Model(&models.LGAGroupMember{}).
            Where("id = ?", existing.ID).
            Updates(map[string]any{
                "deleted_at":     nil,
                "joined_by_user": user.ID,
                "unit_name":      unitName,
            }).Error; err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to join group"})
            return
        }
        existing.DeletedAt = gorm.DeletedAt{}
        existing.JoinedByUser = user.ID
        existing.UnitName = unitName
        c.JSON(http.StatusOK, gin.H{"message": "rejoined", "member": lgaMemberResponse(existing)})
        return
    case !errors.Is(err, gorm.ErrRecordNotFound):
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to join group"})
        return
    }

    member := models.LGAGroupMember{
        State:        state,
        LGA:          lga,
        UnitID:       unitID,
        UnitName:     unitName,
        JoinedByUser: user.ID,
    }
    if err := config.DB.Create(&member).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to join group"})
        return
    }
    c.JSON(http.StatusCreated, gin.H{"message": "joined", "member": lgaMemberResponse(member)})
}

// LeaveLGAGroup — POST /community/lga-group/leave
func LeaveLGAGroup(c *gin.Context) {
    lga, state, unitID, _, _, _, ok := callerLGAContext(c)
    if !ok {
        c.JSON(http.StatusForbidden, gin.H{
            "error": "only verified security unit members can leave the LGA group",
        })
        return
    }

    res := config.DB.
        Where("state = ? AND lga = ? AND unit_id = ?", state, lga, unitID).
        Delete(&models.LGAGroupMember{})
    if res.Error != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to leave group"})
        return
    }
    if res.RowsAffected == 0 {
        c.JSON(http.StatusOK, gin.H{"message": "not a member"})
        return
    }
    c.JSON(http.StatusOK, gin.H{"message": "left"})
}

// GetLGAGroupMembers — GET /community/lga-group/members
//
// The roster is readable by any eligible member of the LGA (not only the ones
// who joined) together with the caller's own rights, so the UI can render the
// composer for head admins and a join prompt for everyone else.
func GetLGAGroupMembers(c *gin.Context) {
    lga, state, unitID, _, _, isHeadAdmin, ok := callerLGAContext(c)
    if !ok {
        c.JSON(http.StatusForbidden, gin.H{
            "error": "the LGA group is available to verified security unit members only",
        })
        return
    }

    var members []models.LGAGroupMember
    if err := config.DB.
        Where("state = ? AND lga = ?", state, lga).
        Order("created_at ASC").
        Find(&members).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load members"})
        return
    }

    results := make([]gin.H, 0, len(members))
    joined := false
    for _, m := range members {
        if m.UnitID == unitID {
            joined = true
        }
        results = append(results, lgaMemberResponse(m))
    }

    c.JSON(http.StatusOK, gin.H{
        "group":   gin.H{"state": state, "lga": lga},
        "members": results,
        "count":   len(results),
        "joined":  joined,
        "canPost": isHeadAdmin,
    })
}
