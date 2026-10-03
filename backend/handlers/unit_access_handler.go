package handlers

import (
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"

    "security-solution/config"
    "security-solution/models"
)

// GetUnitAccess — GET /units/:id/access
//
// Answers "who is the caller with respect to this unit?" so the frontend
// knows whether to render the member tabs. Never returns 404 for a real
// unit the caller isn't part of — it returns `isMember: false`, which is
// a normal answer, not an error.
func GetUnitAccess(c *gin.Context) {
    userID, ok := currentUserID(c)
    if !ok {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
        return
    }

    unitID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid unit id"})
        return
    }

    var membership models.UnitMembership
    err = config.DB.
        Where("user_id = ? AND unit_id = ?", userID, unitID).
        First(&membership).Error

    if err != nil {
        // Not a member — that is a normal, expected answer.
        c.JSON(http.StatusOK, gin.H{
            "isMember":     false,
            "role":         nil,
            "status":       nil,
            "isHeadAdmin":  false,
            "membershipId": nil,
        })
        return
    }

    active := membership.Status == "active"

    c.JSON(http.StatusOK, gin.H{
        "isMember":     active,
        "role":         membership.Role,
        "status":       membership.Status,
        "isHeadAdmin":  membership.IsHeadAdmin,
        "membershipId": membership.ID,
    })
}

// rosterMember is one entry in the roster response.
type rosterMember struct {
    MembershipID     uuid.UUID  `json:"membershipId"`
    UserID           uuid.UUID  `json:"userId"`
    FirstName        string     `json:"firstName"`
    LastName         string     `json:"lastName"`
    Email            string     `json:"email"`
    AvatarPath       string     `json:"avatarPath,omitempty"`
    Role             string     `json:"role"`
    Status           string     `json:"status"`
    IsHeadAdmin      bool       `json:"isHeadAdmin"`
    JoinedViaInvite  bool       `json:"joinedViaInvite"`
    AcceptedAt       *time.Time `json:"acceptedAt,omitempty"`
    ElectedAt        *time.Time `json:"electedAt,omitempty"`
    TermStartAt      *time.Time `json:"termStartAt,omitempty"`
    TermEndAt        *time.Time `json:"termEndAt,omitempty"`
    ConsecutiveTerms int        `json:"consecutiveTerms"`
    CreatedAt        time.Time  `json:"createdAt"`
}

// GetUnitRoster — GET /units/:id/roster
//
// Members of a unit, ordered head admin → admin → officer. Requires the
// caller to be an active member of the same unit — a citizen who is not in
// the unit cannot see who is.
func GetUnitRoster(c *gin.Context) {
    callerID, ok := currentUserID(c)
    if !ok {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
        return
    }

    unitID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid unit id"})
        return
    }

    // The caller must be an active member of this unit.
    var caller models.UnitMembership
    if err := config.DB.
        Where("user_id = ? AND unit_id = ? AND status = ?", callerID, unitID, "active").
        First(&caller).Error; err != nil {
        c.JSON(http.StatusForbidden, gin.H{"error": "not a member of this unit"})
        return
    }

    var memberships []models.UnitMembership
    if err := config.DB.
        Where("unit_id = ? AND status IN ?", unitID, []string{"active", "pending"}).
        Find(&memberships).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load roster"})
        return
    }

    // Collect user IDs for a single bulk fetch.
    userIDs := make([]uuid.UUID, 0, len(memberships))
    for _, m := range memberships {
        userIDs = append(userIDs, m.UserID)
    }

    var users []models.User
    if len(userIDs) > 0 {
        config.DB.Where("id IN ?", userIDs).Find(&users)
    }
    userByID := map[uuid.UUID]models.User{}
    for _, u := range users {
        userByID[u.ID] = u
    }

    roster := make([]rosterMember, 0, len(memberships))
    var admins, officers, pending int
    for _, m := range memberships {
        u := userByID[m.UserID]
        roster = append(roster, rosterMember{
            MembershipID:     m.ID,
            UserID:           m.UserID,
            FirstName:        u.FirstName,
            LastName:         u.LastName,
            Email:            u.Email,
            AvatarPath:       u.AvatarPath,
            Role:             m.Role,
            Status:           m.Status,
            IsHeadAdmin:      m.IsHeadAdmin,
            JoinedViaInvite:  m.JoinedViaInvite,
            AcceptedAt:       m.AcceptedAt,
            ElectedAt:        m.ElectedAt,
            TermStartAt:      m.TermStartAt,
            TermEndAt:        m.TermEndAt,
            ConsecutiveTerms: m.ConsecutiveTerms,
            CreatedAt:        m.CreatedAt,
        })
        if m.Status == "active" {
            if m.IsHeadAdmin || m.Role == "unit_admin" {
                admins++
            } else if m.Role == "officer" {
                officers++
            }
        } else if m.Status == "pending" {
            pending++
        }
    }

    c.JSON(http.StatusOK, gin.H{
        "members": roster,
        "counts": gin.H{
            "total":    len(roster),
            "admins":   admins,
            "officers": officers,
            "pending":  pending,
        },
    })
}
