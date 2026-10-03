package handlers

import (
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"

    "security-solution/config"
    "security-solution/models"
)

// governanceAdmin is one current admin entry in the governance response.
type governanceAdmin struct {
    MembershipID     uuid.UUID  `json:"membershipId"`
    UserID           uuid.UUID  `json:"userId"`
    FirstName        string     `json:"firstName"`
    LastName         string     `json:"lastName"`
    Email            string     `json:"email"`
    AvatarPath       string     `json:"avatarPath,omitempty"`
    IsHeadAdmin      bool       `json:"isHeadAdmin"`
    ElectedAt        *time.Time `json:"electedAt,omitempty"`
    TermStartAt      *time.Time `json:"termStartAt,omitempty"`
    TermEndAt        *time.Time `json:"termEndAt,omitempty"`
    ConsecutiveTerms int        `json:"consecutiveTerms"`
    CoolingOffUntil  *time.Time `json:"coolingOffUntil,omitempty"`
}

// GetUnitGovernance — GET /units/:id/governance
//
// Aggregated governance state for a unit: current admins, seats, open and
// recent elections, open revocations. Member-only — a citizen who is not in
// the unit cannot see who runs it.
func GetUnitGovernance(c *gin.Context) {
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

    // Member-only access (super admins exempt).
    var caller models.User
    config.DB.First(&caller, "id = ?", callerID)
    isSuper := caller.IsSuperAdmin || caller.Role == "super_admin"

    if !isSuper {
        var m models.UnitMembership
        if err := config.DB.
            Where("unit_id = ? AND user_id = ? AND status = ?", unitID, callerID, "active").
            First(&m).Error; err != nil {
            c.JSON(http.StatusForbidden, gin.H{"error": "not a member of this unit"})
            return
        }
    }

    // ---- Current admins ------------------------------------------------
    var admins []models.UnitMembership
    config.DB.
        Where("unit_id = ? AND status = ? AND role = ?", unitID, "active", "unit_admin").
        Find(&admins)

    adminUserIDs := make([]uuid.UUID, 0, len(admins))
    for _, m := range admins {
        adminUserIDs = append(adminUserIDs, m.UserID)
    }
    var adminUsers []models.User
    if len(adminUserIDs) > 0 {
        config.DB.Where("id IN ?", adminUserIDs).Find(&adminUsers)
    }
    adminUserByID := map[uuid.UUID]models.User{}
    for _, u := range adminUsers {
        adminUserByID[u.ID] = u
    }

    adminsOut := make([]governanceAdmin, 0, len(admins))
    headAdminCount := 0
    for _, m := range admins {
        u := adminUserByID[m.UserID]
        if m.IsHeadAdmin {
            headAdminCount++
        }
        adminsOut = append(adminsOut, governanceAdmin{
            MembershipID:     m.ID,
            UserID:           m.UserID,
            FirstName:        u.FirstName,
            LastName:         u.LastName,
            Email:            u.Email,
            AvatarPath:       u.AvatarPath,
            IsHeadAdmin:      m.IsHeadAdmin,
            ElectedAt:        m.ElectedAt,
            TermStartAt:      m.TermStartAt,
            TermEndAt:        m.TermEndAt,
            ConsecutiveTerms: m.ConsecutiveTerms,
            CoolingOffUntil:  m.CoolingOffUntil,
        })
    }

    // ---- Seats ---------------------------------------------------------
    var seats []models.UnitAdminSeat
    config.DB.
        Where("unit_id = ? AND status IN ?", unitID, []string{"active", "vacant"}).
        Order("seat_number asc").
        Find(&seats)

    // ---- Open elections -------------------------------------------------
    var openElections []models.UnitAdminElection
    config.DB.
        Where("unit_id = ? AND status = ?", unitID, "open").
        Order("created_at desc").
        Find(&openElections)

    // ---- Recent finalized elections ------------------------------------
    var recentElections []models.UnitAdminElection
    config.DB.
        Where("unit_id = ? AND status IN ?", unitID, []string{"finalized", "finalized_low_turnout"}).
        Order("result_finalized_at desc").
        Limit(5).
        Find(&recentElections)

    // ---- Open revocations ----------------------------------------------
    var openRevocations []models.RevocationCycle
    config.DB.
        Where("unit_id = ? AND status = ?", unitID, "open").
        Order("opened_at desc").
        Find(&openRevocations)

    c.JSON(http.StatusOK, gin.H{
        "unitId":           unitID,
        "admins":           adminsOut,
        "adminCount":       len(adminsOut),
        "headAdminCount":   headAdminCount,
        "seats":            seats,
        "openElections":    openElections,
        "recentElections":  recentElections,
        "openRevocations":  openRevocations,
    })
}
