package handlers

import (
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"

    "security-solution/config"
    "security-solution/models"
)

// GetUnitPublicSummary — GET /public/units/:id/summary
func GetUnitPublicSummary(c *gin.Context) {
    id, err := uuid.Parse(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid unit id"})
        return
    }

    var unit models.SecurityUnit
    if err := config.DB.First(&unit, "id = ?", id).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "unit not found"})
        return
    }

    now := time.Now().UTC()
    thirtyDaysAgo := now.AddDate(0, 0, -30)

    var openCases int64
    config.DB.Model(&models.Case{}).
        Where("unit_id = ? AND status != ?", id, "closed").
        Count(&openCases)

    var totalCases int64
    config.DB.Model(&models.Case{}).Where("unit_id = ?", id).Count(&totalCases)

    var recentCases int64
    config.DB.Model(&models.Case{}).
        Where("unit_id = ? AND created_at > ?", id, thirtyDaysAgo).
        Count(&recentCases)

    var activeMembers int64
    config.DB.Model(&models.UnitMembership{}).
        Where("unit_id = ? AND status = ?", id, "active").
        Count(&activeMembers)

    var officers int64
    config.DB.Model(&models.Officer{}).
        Where("unit_id = ? AND status = ?", id, "active").
        Count(&officers)

    var verifiedDays int
    if unit.VerifiedAt != nil {
        verifiedDays = int(now.Sub(*unit.VerifiedAt).Hours() / 24)
    }

    c.JSON(http.StatusOK, gin.H{
        "unitId":            unit.ID,
        "name":              unit.Name,
        "type":              unit.Type,
        "verified":          unit.IsVerified,
        "verifiedAt":        unit.VerifiedAt,
        "verifiedDays":      verifiedDays,
        "formationDate":     unit.FormationDate,
        "operationalRadius": unit.OperationalRadius,
        "openCases":         openCases,
        "totalCases":        totalCases,
        "recentCases":       recentCases,
        "activeMembers":     activeMembers,
        "officers":          officers,
    })
}
