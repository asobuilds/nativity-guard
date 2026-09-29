package services

import (
    "fmt"
    "math"
    "time"

    "github.com/google/uuid"

    "security-solution/config"
    "security-solution/models"
)

// SosCandidate is a unit that should be notified for an SOS.
type SosCandidate struct {
    UnitID   uuid.UUID
    Distance float64 // km, 0 if unknown
    Reason   string  // "nearest" | "active" | "rated"
}

// SosDispatchService selects and notifies units for an SOS alert.
// Stateless — safe to construct per-request.
type SosDispatchService struct{}

func NewSosDispatchService() *SosDispatchService { return &SosDispatchService{} }

// haversineKm returns the great-circle distance in km between two points.
// Duplicated from handlers/unit_handler.go:618 so the service has no
// dependency on the handler package.
func haversineKm(lat1, lon1, lat2, lon2 float64) float64 {
    const R = 6371.0
    rad := math.Pi / 180
    dLat := (lat2 - lat1) * rad
    dLon := (lon2 - lon1) * rad
    a := math.Sin(dLat/2)*math.Sin(dLat/2) +
        math.Cos(lat1*rad)*math.Cos(lat2*rad)*
            math.Sin(dLon/2)*math.Sin(dLon/2)
    c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
    return R * c
}

// SelectCandidates returns the union of:
//   - top 5 nearest active units within 100 km
//   - top 5 units by case count in the last 30 days
//   - top 5 units by UnitScore.CompositeScore
// Deduplicated by unit ID.
func (s *SosDispatchService) SelectCandidates(lat, lng float64) []SosCandidate {
    seen := map[uuid.UUID]bool{}
    var out []SosCandidate

    add := func(c SosCandidate) {
        if c.UnitID == uuid.Nil || seen[c.UnitID] {
            return
        }
        seen[c.UnitID] = true
        out = append(out, c)
    }

    // --- 1. Nearest 5 (<=100km) ---
    var units []models.SecurityUnit
    config.DB.Where("status = ?", "active").Find(&units)

    type near struct {
        id  uuid.UUID
        km  float64
    }
    var within []near
    for _, u := range units {
        if u.Latitude == 0 && u.Longitude == 0 {
            continue
        }
        d := haversineKm(lat, lng, u.Latitude, u.Longitude)
        if d <= 100 {
            within = append(within, near{u.ID, d})
        }
    }
    // bubble 5 nearest
    for i := 0; i < len(within) && i < 5; i++ {
        for j := i + 1; j < len(within); j++ {
            if within[j].km < within[i].km {
                within[i], within[j] = within[j], within[i]
            }
        }
    }
    nearestCount := len(within)
    if nearestCount > 5 {
        nearestCount = 5
    }
    for i := 0; i < nearestCount; i++ {
        add(SosCandidate{UnitID: within[i].id, Distance: within[i].km, Reason: "nearest"})
    }

    // --- 2. Top 5 most-active (case count last 30 days) ---
    type active struct {
        id    uuid.UUID
        count int64
    }
    var actives []active
    since := time.Now().AddDate(0, 0, -30)
    for _, u := range units {
        var c int64
        config.DB.Model(&models.Case{}).
            Where("unit_id = ? AND created_at > ?", u.ID, since).
            Count(&c)
        if c > 0 {
            actives = append(actives, active{u.ID, c})
        }
    }
    for i := 0; i < len(actives); i++ {
        for j := i + 1; j < len(actives); j++ {
            if actives[j].count > actives[i].count {
                actives[i], actives[j] = actives[j], actives[i]
            }
        }
    }
    actCount := len(actives)
    if actCount > 5 {
        actCount = 5
    }
    for i := 0; i < actCount; i++ {
        add(SosCandidate{UnitID: actives[i].id, Reason: "active"})
    }

    // --- 3. Top 5 by UnitScore.CompositeScore ---
    var scores []models.UnitScore
    config.DB.Order("composite_score desc").Limit(10).Find(&scores)
    ratedCount := 0
    for _, sc := range scores {
        if ratedCount >= 5 {
            break
        }
        add(SosCandidate{UnitID: sc.UnitID, Reason: "rated"})
        ratedCount++
    }

    return out
}

// NotifyCandidates sends an in-app notification to every unit_admin
// (and super_admin bound to a unit) belonging to each candidate unit.
func (s *SosDispatchService) NotifyCandidates(sos models.SOSAlert, candidates []SosCandidate) {
    for _, c := range candidates {
        var admins []models.User
        config.DB.Where("unit_id = ? AND role IN (?)", c.UnitID, []string{"unit_admin"}).Find(&admins)

        title := "🚨 SOS Alert Nearby"
        msg := fmt.Sprintf("SOS at %.4f, %.4f — %s. Open to respond.", sos.Latitude, sos.Longitude, c.Reason)
        if c.Distance > 0 {
            msg = fmt.Sprintf("SOS %.1f km away — %s. Open to respond.", c.Distance, c.Reason)
        }

        for _, admin := range admins {
            config.DB.Create(&models.Notification{
                UserID:     admin.ID,
                Title:      title,
                Message:    msg,
                Type:       "sos_alert",
                Status:     "unread",
                EntityType: "sos",
                EntityID:   sos.ID,
                LinkTo:     "/sos/" + sos.ID.String(),
            })
        }
    }
}

// NotifySuperAdmins notifies every user with role='super_admin'.
// Called at SOS creation AND at 5-minute escalation.
func (s *SosDispatchService) NotifySuperAdmins(sos models.SOSAlert, reason string) {
    var admins []models.User
    config.DB.Where("role = ?", "super_admin").Find(&admins)
    for _, admin := range admins {
        config.DB.Create(&models.Notification{
            UserID:     admin.ID,
            Title:      "🚨 SOS — Super Admin",
            Message:    fmt.Sprintf("SOS %s (%s) — assign or observe.", sos.ID.String()[:8], reason),
            Type:       "sos_escalation",
            Status:     "unread",
            EntityType: "sos",
            EntityID:   sos.ID,
            LinkTo:     "/sos/" + sos.ID.String(),
        })
    }
}

// NotifyJurisdictionHeads notifies users whose unit's State/LGA match
// the SOS jurisdiction (head_admin role). No-op if jurisdiction unset.
func (s *SosDispatchService) NotifyJurisdictionHeads(sos models.SOSAlert) {
    if sos.JurisdictionState == "" && sos.JurisdictionLGA == "" {
        return
    }
    var units []models.SecurityUnit
    q := config.DB.Where("status = ?", "active")
    if sos.JurisdictionState != "" {
        q = q.Where("state = ?", sos.JurisdictionState)
    }
    if sos.JurisdictionLGA != "" {
        q = q.Where("lga = ?", sos.JurisdictionLGA)
    }
    q.Find(&units)

    for _, u := range units {
        var heads []models.User
        config.DB.Where("unit_id = ? AND role = ?", u.ID, "unit_admin").Find(&heads)
        for _, h := range heads {
            config.DB.Create(&models.Notification{
                UserID:     h.ID,
                Title:      "🚨 SOS in your jurisdiction",
                Message:    fmt.Sprintf("SOS at %s/%s. Open to review.", sos.JurisdictionLGA, sos.JurisdictionState),
                Type:       "sos_alert",
                Status:     "unread",
                EntityType: "sos",
                EntityID:   sos.ID,
                LinkTo:     "/sos/" + sos.ID.String(),
            })
        }
    }
}