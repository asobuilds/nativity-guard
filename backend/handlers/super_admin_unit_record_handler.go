package handlers

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"

    "security-solution/config"
    "security-solution/models"
)

// GetSuperAdminUnitRecord — GET /admin/units/:id/record
//
// One aggregate payload for a super admin inspecting a unit from the
// platform console. Everything a super admin could otherwise fetch across
// six endpoints, in one round trip. Access is enforced by the surrounding
// superAdmin middleware group.
func GetSuperAdminUnitRecord(c *gin.Context) {
    unitID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid unit id"})
        return
    }

    var unit models.SecurityUnit
    if err := config.DB.First(&unit, "id = ?", unitID).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "unit not found"})
        return
    }

    // ---- Memberships (full roster, ordered active → pending) -------------
    var memberships []models.UnitMembership
    config.DB.Where("unit_id = ? AND status IN ?", unitID, []string{"active", "pending"}).
        Find(&memberships)

    memberUserIDs := make([]uuid.UUID, 0, len(memberships))
    for _, m := range memberships {
        memberUserIDs = append(memberUserIDs, m.UserID)
    }
    var memberUsers []models.User
    if len(memberUserIDs) > 0 {
        config.DB.Where("id IN ?", memberUserIDs).Find(&memberUsers)
    }
    userByID := map[uuid.UUID]models.User{}
    for _, u := range memberUsers {
        userByID[u.ID] = u
    }

    members := make([]gin.H, 0, len(memberships))
    counts := gin.H{"active": 0, "pending": 0, "admins": 0, "officers": 0, "headAdmin": 0}
    for _, m := range memberships {
        u := userByID[m.UserID]
        members = append(members, gin.H{
            "membershipId":    m.ID,
            "userId":          m.UserID,
            "firstName":       u.FirstName,
            "lastName":        u.LastName,
            "email":           u.Email,
            "avatarPath":      u.AvatarPath,
            "role":            m.Role,
            "status":          m.Status,
            "isHeadAdmin":     m.IsHeadAdmin,
            "joinedViaInvite": m.JoinedViaInvite,
            "acceptedAt":      m.AcceptedAt,
            "electedAt":       m.ElectedAt,
            "termStartAt":     m.TermStartAt,
            "termEndAt":       m.TermEndAt,
            "verifiedAt":      m.VerifiedAt,
            "createdAt":       m.CreatedAt,
        })
        if m.Status == "active" {
            counts["active"] = counts["active"].(int) + 1
            if m.IsHeadAdmin {
                counts["headAdmin"] = counts["headAdmin"].(int) + 1
            }
            if m.Role == "unit_admin" || m.IsHeadAdmin {
                counts["admins"] = counts["admins"].(int) + 1
            } else if m.Role == "officer" {
                counts["officers"] = counts["officers"].(int) + 1
            }
        } else if m.Status == "pending" {
            counts["pending"] = counts["pending"].(int) + 1
        }
    }

    // ---- Seats ------------------------------------------------------------
    var seats []models.UnitAdminSeat
    config.DB.Where("unit_id = ?", unitID).Order("seat_number asc").Find(&seats)

    // ---- Elections --------------------------------------------------------
    var openElections []models.UnitAdminElection
    config.DB.Where("unit_id = ? AND status = ?", unitID, "open").
        Order("created_at desc").Find(&openElections)

    var recentElections []models.UnitAdminElection
    config.DB.Where("unit_id = ? AND status IN ?", unitID,
        []string{"finalized", "finalized_low_turnout"}).
        Order("result_finalized_at desc").Limit(10).Find(&recentElections)

    // ---- Revocations -----------------------------------------------------
    var revocations []models.RevocationCycle
    config.DB.Where("unit_id = ?", unitID).
        Order("opened_at desc").Limit(20).Find(&revocations)

    // ---- Case counts -----------------------------------------------------
    var openCases, totalCases int64
    config.DB.Model(&models.Case{}).
        Where("unit_id = ? AND status != ?", unitID, "closed").Count(&openCases)
    config.DB.Model(&models.Case{}).Where("unit_id = ?", unitID).Count(&totalCases)

    // ---- Audit log (last 40 entries for this unit) -----------------------
    var auditLogs []models.AuditLog
    config.DB.Preload("User").
        Where("entity_type = ? AND entity_id = ?", "security_unit", unitID.String()).
        Order("timestamp desc").Limit(40).Find(&auditLogs)

    c.JSON(http.StatusOK, gin.H{
        "unit":            unit,
        "counts":          counts,
        "members":         members,
        "seats":           seats,
        "openElections":   openElections,
        "recentElections": recentElections,
        "revocations":     revocations,
        "cases":           gin.H{"open": openCases, "total": totalCases},
        "auditLog":        auditLogs,
    })
}
