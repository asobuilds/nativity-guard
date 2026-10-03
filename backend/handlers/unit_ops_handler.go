package handlers

import (
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"

    "security-solution/config"
    "security-solution/models"
)

// callerIsUnitAdminOf reports whether the authenticated caller is an active
// unit admin (or head admin) of the given unit. Super admins are also allowed.
func callerIsUnitAdminOf(c *gin.Context, unitID uuid.UUID) bool {
    value, exists := c.Get("user")
    if !exists {
        return false
    }
    user, ok := value.(*models.User)
    if !ok || user == nil {
        return false
    }
    if user.IsSuperAdmin || user.Role == "super_admin" {
        return true
    }
    var m models.UnitMembership
    err := config.DB.
        Where("user_id = ? AND unit_id = ? AND status = ?", user.ID, unitID, "active").
        First(&m).Error
    if err != nil {
        return false
    }
    return m.Role == "unit_admin" || m.IsHeadAdmin
}

// GetUnitInbox — GET /units/:id/inbox
//
// Pending cases, unrouted SOS alerts, and membership applications for the
// unit. Read-only aggregate view — the writes happen elsewhere.
func GetUnitInbox(c *gin.Context) {
    unitID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid unit id"})
        return
    }
    if !callerIsUnitAdminOf(c, unitID) {
        c.JSON(http.StatusForbidden, gin.H{"error": "unit admin access required"})
        return
    }

    var pendingCases []models.Case
    config.DB.
        Where("unit_id = ? AND status IN ?", unitID, []string{"pending", "assigned"}).
        Order("created_at asc").
        Limit(20).
        Find(&pendingCases)

    var sosAlerts []models.SOSAlert
    config.DB.
        Where("unit_id = ? AND status = ? AND accepted_by_unit_id IS NULL", unitID, "pending").
        Order("created_at asc").
        Limit(20).
        Find(&sosAlerts)

    var applications []models.UnitMembership
    config.DB.
        Where("unit_id = ? AND status = ?", unitID, "pending").
        Order("created_at asc").
        Limit(20).
        Find(&applications)

    userIDs := make([]uuid.UUID, 0, len(applications))
    for _, a := range applications {
        userIDs = append(userIDs, a.UserID)
    }
    var users []models.User
    if len(userIDs) > 0 {
        config.DB.Where("id IN ?", userIDs).Find(&users)
    }
    userByID := map[uuid.UUID]models.User{}
    for _, u := range users {
        userByID[u.ID] = u
    }

    appsOut := make([]gin.H, 0, len(applications))
    for _, a := range applications {
        u := userByID[a.UserID]
        appsOut = append(appsOut, gin.H{
            "membershipId": a.ID,
            "userId":       a.UserID,
            "firstName":    u.FirstName,
            "lastName":     u.LastName,
            "email":        u.Email,
            "avatarPath":   u.AvatarPath,
            "role":         a.Role,
            "createdAt":    a.CreatedAt,
        })
    }

    c.JSON(http.StatusOK, gin.H{
        "pendingCases":     pendingCases,
        "pendingCaseCount": len(pendingCases),
        "sosAlerts":        sosAlerts,
        "sosAlertCount":    len(sosAlerts),
        "applications":     appsOut,
        "applicationCount": len(appsOut),
    })
}

// GetUnitCompliance — GET /units/:id/compliance
//
// A per-officer matrix: how many weekly updates each active officer filed
// in each of the last 8 weeks. Weeks start Monday 00:00 UTC.
func GetUnitCompliance(c *gin.Context) {
    unitID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid unit id"})
        return
    }
    if !callerIsUnitAdminOf(c, unitID) {
        c.JSON(http.StatusForbidden, gin.H{"error": "unit admin access required"})
        return
    }

    var memberships []models.UnitMembership
    config.DB.
        Where("unit_id = ? AND status = ? AND role = ?", unitID, "active", "officer").
        Find(&memberships)

    officerIDs := make([]uuid.UUID, 0, len(memberships))
    for _, m := range memberships {
        officerIDs = append(officerIDs, m.UserID)
    }
    var users []models.User
    if len(officerIDs) > 0 {
        config.DB.Where("id IN ?", officerIDs).Find(&users)
    }
    userByID := map[uuid.UUID]models.User{}
    for _, u := range users {
        userByID[u.ID] = u
    }

    type weekBucket struct {
        start time.Time
        end   time.Time
    }
    now := time.Now().UTC()
    dayOffset := (int(now.Weekday()) + 6) % 7
    thisMonday := time.Date(now.Year(), now.Month(), now.Day()-dayOffset, 0, 0, 0, 0, time.UTC)
    weeks := make([]weekBucket, 0, 8)
    for i := 7; i >= 0; i-- {
        start := thisMonday.AddDate(0, 0, -7*i)
        end := start.AddDate(0, 0, 7)
        weeks = append(weeks, weekBucket{start: start, end: end})
    }

    officers := make([]gin.H, 0, len(memberships))
    for _, m := range memberships {
        u := userByID[m.UserID]
        weekCounts := make([]int64, len(weeks))
        for i, w := range weeks {
            var n int64
            config.DB.Model(&models.CaseWeeklyUpdate{}).
                Where("officer_id = ? AND week_start >= ? AND week_start < ?", m.UserID, w.start, w.end).
                Count(&n)
            weekCounts[i] = n
        }
        filed := 0
        for _, c := range weekCounts {
            if c > 0 {
                filed++
            }
        }
        rate := 0.0
        if len(weeks) > 0 {
            rate = float64(filed) / float64(len(weeks)) * 100.0
        }
        officers = append(officers, gin.H{
            "membershipId":   m.ID,
            "userId":         m.UserID,
            "firstName":      u.FirstName,
            "lastName":       u.LastName,
            "email":          u.Email,
            "avatarPath":     u.AvatarPath,
            "weeksFiled":     filed,
            "weeksTotal":     len(weeks),
            "complianceRate": rate,
            "weekCounts":     weekCounts,
        })
    }

    weekLabels := make([]string, len(weeks))
    for i, w := range weeks {
        weekLabels[i] = w.start.Format("2006-01-02")
    }

    c.JSON(http.StatusOK, gin.H{
        "weeks":    weekLabels,
        "officers": officers,
    })
}
