package handlers

import (
	"errors"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"security-solution/config"
	"security-solution/models"
)

const observerDescriptionMaxLen = 500

var (
	observerEmailRE       = regexp.MustCompile(`(?i)[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}`)
	observerCoordinateRE  = regexp.MustCompile(`-?\d{1,3}\.\d{3,}\s*,\s*-?\d{1,3}\.\d{3,}`)
	observerURLRE         = regexp.MustCompile(`(?i)\b(?:https?://|www\.)\S+`)
	observerPhoneRE       = regexp.MustCompile(`(?:\+?234|0)[789][01]\d{8}|(?:\+?\d[\d\s().-]{7,}\d)`)
	observerCapitalizedRE = regexp.MustCompile(`\b[A-Z][a-z]{2,}\b`)
	observerAllCapsRE     = regexp.MustCompile(`\b[A-Z]{2,}\b`)
)

// observerSafeWords are common sentence/proper words that are not treated as
// suspected personal names by the conservative free-text check.
var observerSafeWords = map[string]struct{}{
	"The": {}, "This": {}, "There": {}, "These": {}, "Those": {}, "When": {},
	"While": {}, "After": {}, "Before": {}, "During": {}, "Near": {}, "Along": {},
	"Between": {}, "Around": {}, "Several": {}, "Multiple": {}, "Unknown": {},
	"Reported": {}, "Suspected": {}, "Fire": {}, "Police": {}, "Road": {},
	"Accident": {}, "Emergency": {}, "Market": {}, "School": {}, "Hospital": {},
	"Church": {}, "Mosque": {}, "Government": {}, "Military": {}, "Community": {},
	"Local": {}, "Public": {}, "Building": {}, "Vehicle": {}, "Rescue": {},
	"Medical": {}, "Security": {}, "Water": {}, "Power": {}, "Bridge": {},
	"Flood": {}, "Land": {}, "Traffic": {}, "Ward": {}, "State": {}, "Nigeria": {},
	"Nigerian": {}, "NSCDC": {}, "FRSC": {}, "SOS": {}, "LGA": {}, "Urgent": {},
	"Help": {}, "Need": {}, "Assistance": {}, "Immediate": {}, "Attention": {},
	"Danger": {}, "Dangerous": {}, "Smoke": {}, "Explosion": {}, "Crash": {},
	"Robbery": {}, "Theft": {}, "Attack": {}, "Vandalism": {}, "Disaster": {},
	"Infrastructure": {}, "Collision": {}, "Injury": {}, "Injured": {}, "People": {},
	"Person": {}, "Man": {}, "Woman": {}, "Child": {}, "Children": {}, "Area": {},
	"Street": {}, "Block": {}, "House": {}, "Shop": {}, "Station": {},
}

// ObserverLocation is the only location shape exposed to agency observers.
// It intentionally contains no coordinates or free-form address text.
type ObserverLocation struct {
	State string `json:"state"`
	LGA   string `json:"lga"`
	Ward  string `json:"ward,omitempty"`
}

// ObserverIncident is the public-safe incident projection for observers.
type ObserverIncident struct {
	Time           time.Time        `json:"time"`
	Type           string           `json:"type"`
	Location       ObserverLocation `json:"location"`
	RespondingUnit string           `json:"respondingUnit"`
	Status         string           `json:"status"`
	Description    string           `json:"description"`
}

// ObserverUnitSummary is an activity-derived summary for units responding
// inside the observer's LGA. It exposes no contact, officer, or unit PII.
type ObserverUnitSummary struct {
	Time           time.Time        `json:"time"`
	Type           string           `json:"type"`
	Location       ObserverLocation `json:"location"`
	RespondingUnit string           `json:"respondingUnit"`
	Status         string           `json:"status"`
	ActiveSOS      int              `json:"activeSOS"`
	OpenCases      int              `json:"openCases"`
}

type observerSOSRow struct {
	ID                uuid.UUID  `gorm:"column:id"`
	Category          string     `gorm:"column:category"`
	Status            string     `gorm:"column:status"`
	CreatedAt         time.Time  `gorm:"column:created_at"`
	JurisdictionState string     `gorm:"column:jurisdiction_state"`
	JurisdictionLGA   string     `gorm:"column:jurisdiction_lga"`
	Description       string     `gorm:"column:description"`
	UnitID            *uuid.UUID `gorm:"column:unit_id"`
	AcceptedByUnitID  *uuid.UUID `gorm:"column:accepted_by_unit_id"`
}

type observerCaseRow struct {
	ID           uuid.UUID `gorm:"column:id"`
	Category     string    `gorm:"column:category"`
	Status       string    `gorm:"column:status"`
	IncidentDate time.Time `gorm:"column:incident_date"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	Description  string    `gorm:"column:description"`
	UnitID       uuid.UUID `gorm:"column:unit_id"`
	UnitName     string    `gorm:"column:unit_name"`
	UnitType     string    `gorm:"column:unit_type"`
	UnitState    string    `gorm:"column:unit_state"`
	UnitLGA      string    `gorm:"column:unit_lga"`
	UnitWard     string    `gorm:"column:unit_ward"`
	UnitStatus   string    `gorm:"column:unit_status"`
}

type observerUnitInfo struct {
	ID     uuid.UUID `gorm:"column:id"`
	Name   string    `gorm:"column:name"`
	Type   string    `gorm:"column:type"`
	State  string    `gorm:"column:state"`
	LGA    string    `gorm:"column:lga"`
	Ward   string    `gorm:"column:ward"`
	Status string    `gorm:"column:status"`
}

type observerIncidentRecord struct {
	Incident ObserverIncident
	UnitID   *uuid.UUID
	Unit     observerUnitInfo
	IsSOS    bool
}

var observerClosedCaseStatuses = []string{
	"closed", "archived", "transferred", "resolved", "cancelled",
}

// GetObserverIncidents returns active SOS alerts and open cases inside the
// observer's tenant LGA, scoped by the tenant's agency type.
func GetObserverIncidents(c *gin.Context) {
	_, tenant, ok := observerTenantContext(c)
	if !ok {
		return
	}

	records, err := observerIncidentRecords(*tenant)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch observer incidents"})
		return
	}

	limit := observerQueryInt(c, "limit", 50, 100)
	page := observerQueryInt(c, "page", 1, 10000)
	total := len(records)

	start := (page - 1) * limit
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}

	incidents := make([]ObserverIncident, 0, end-start)
	for _, record := range records[start:end] {
		incidents = append(incidents, record.Incident)
	}

	c.JSON(http.StatusOK, gin.H{
		"incidents": incidents,
		"page":      page,
		"limit":     limit,
		"total":     total,
	})
}

// GetObserverUnits returns an activity-derived summary of units responding
// to incidents inside the observer's tenant LGA.
func GetObserverUnits(c *gin.Context) {
	_, tenant, ok := observerTenantContext(c)
	if !ok {
		return
	}

	records, err := observerIncidentRecords(*tenant)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch observer units"})
		return
	}

	type unitAggregate struct {
		Info      observerUnitInfo
		Last      time.Time
		ActiveSOS int
		OpenCases int
	}

	aggregates := make(map[uuid.UUID]*unitAggregate)
	for _, record := range records {
		if record.UnitID == nil {
			continue
		}
		aggregate, exists := aggregates[*record.UnitID]
		if !exists {
			aggregate = &unitAggregate{Info: record.Unit}
			aggregates[*record.UnitID] = aggregate
		}
		if record.Incident.Time.After(aggregate.Last) {
			aggregate.Last = record.Incident.Time
		}
		if record.IsSOS {
			aggregate.ActiveSOS++
		} else {
			aggregate.OpenCases++
		}
	}

	summaries := make([]ObserverUnitSummary, 0, len(aggregates))
	for _, aggregate := range aggregates {
		location := ObserverLocation{State: tenant.State, LGA: tenant.LGA}
		if observerUnitInTenant(aggregate.Info, *tenant) {
			location.Ward = aggregate.Info.Ward
		}
		summaries = append(summaries, ObserverUnitSummary{
			Time:           aggregate.Last,
			Type:           aggregate.Info.Type,
			Location:       location,
			RespondingUnit: aggregate.Info.Name,
			Status:         aggregate.Info.Status,
			ActiveSOS:      aggregate.ActiveSOS,
			OpenCases:      aggregate.OpenCases,
		})
	}

	sort.Slice(summaries, func(i, j int) bool {
		if summaries[i].Time.Equal(summaries[j].Time) {
			return summaries[i].RespondingUnit < summaries[j].RespondingUnit
		}
		return summaries[i].Time.After(summaries[j].Time)
	})

	c.JSON(http.StatusOK, gin.H{"units": summaries})
}

func observerTenantContext(c *gin.Context) (*models.AgencyObserver, *models.AgencyTenant, bool) {
	user, ok := getAuthUser(c)
	if !ok {
		return nil, nil, false
	}

	var observer models.AgencyObserver
	if err := config.DB.Where("user_id = ?", user.ID).First(&observer).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Observer is not assigned to an agency tenant"})
			return nil, nil, false
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load observer assignment"})
		return nil, nil, false
	}

	var tenant models.AgencyTenant
	if err := config.DB.First(&tenant, "id = ?", observer.TenantID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Observer agency tenant not found"})
			return nil, nil, false
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load observer agency tenant"})
		return nil, nil, false
	}

	return &observer, &tenant, true
}

func observerIncidentRecords(tenant models.AgencyTenant) ([]observerIncidentRecord, error) {
	allowed, ok := observerAllowedCategories(tenant.AgencyType)
	if !ok {
		return nil, nil
	}

	state := strings.ToLower(strings.TrimSpace(tenant.State))
	lga := strings.ToLower(strings.TrimSpace(tenant.LGA))

	sosQuery := config.DB.Model(&models.SOSAlert{}).
		Select("id, category, status, created_at, jurisdiction_state, jurisdiction_lga, description, unit_id, accepted_by_unit_id").
		Where("LOWER(TRIM(jurisdiction_state)) = ? AND LOWER(TRIM(jurisdiction_lga)) = ?", state, lga).
		Where("status NOT IN ?", []string{"resolved", "cancelled"}).
		Where("(dispatch_state IS NULL OR dispatch_state <> ?)", "resolved")
	sosQuery = observerApplyCategoryFilter(sosQuery, allowed)

	var sosRows []observerSOSRow
	if err := sosQuery.Find(&sosRows).Error; err != nil {
		return nil, err
	}

	caseQuery := config.DB.Table("cases AS c").
		Select(`c.id, c.category, c.status, c.incident_date, c.created_at, c.description,
			su.id AS unit_id, su.name AS unit_name, su.type AS unit_type,
			su.state AS unit_state, su.lga AS unit_lga, su.ward AS unit_ward,
			su.status AS unit_status`).
		Joins("JOIN security_units AS su ON su.id = c.unit_id").
		Where("c.deleted_at IS NULL AND su.deleted_at IS NULL").
		Where("LOWER(TRIM(su.state)) = ? AND LOWER(TRIM(su.lga)) = ?", state, lga).
		Where("c.status NOT IN ?", observerClosedCaseStatuses)
	caseQuery = observerApplyCategoryFilter(caseQuery, allowed)

	var caseRows []observerCaseRow
	if err := caseQuery.Scan(&caseRows).Error; err != nil {
		return nil, err
	}

	unitIDs := make([]uuid.UUID, 0, len(caseRows)+len(sosRows))
	seenUnits := make(map[uuid.UUID]struct{})
	addUnitID := func(id uuid.UUID) {
		if id == uuid.Nil {
			return
		}
		if _, exists := seenUnits[id]; !exists {
			seenUnits[id] = struct{}{}
			unitIDs = append(unitIDs, id)
		}
	}
	for _, row := range caseRows {
		addUnitID(row.UnitID)
	}
	for _, row := range sosRows {
		if row.AcceptedByUnitID != nil {
			addUnitID(*row.AcceptedByUnitID)
		} else if row.UnitID != nil {
			addUnitID(*row.UnitID)
		}
	}

	units, err := loadObserverUnits(unitIDs)
	if err != nil {
		return nil, err
	}

	records := make([]observerIncidentRecord, 0, len(sosRows)+len(caseRows))

	for _, row := range sosRows {
		var unitID *uuid.UUID
		if row.AcceptedByUnitID != nil {
			id := *row.AcceptedByUnitID
			unitID = &id
		} else if row.UnitID != nil {
			id := *row.UnitID
			unitID = &id
		}

		info := observerUnitInfo{}
		respondingUnit := "Unassigned"
		location := ObserverLocation{State: tenant.State, LGA: tenant.LGA}
		if unitID != nil {
			if loaded, exists := units[*unitID]; exists {
				info = loaded
				respondingUnit = loaded.Name
				if observerUnitInTenant(loaded, tenant) {
					location.Ward = loaded.Ward
				}
			} else {
				unitID = nil
			}
		}

		records = append(records, observerIncidentRecord{
			Incident: ObserverIncident{
				Time:           row.CreatedAt,
				Type:           observerIncidentType(row.Category),
				Location:       location,
				RespondingUnit: respondingUnit,
				Status:         row.Status,
				Description:    sanitizeObserverDescription(row.Description),
			},
			UnitID: unitID,
			Unit:   info,
			IsSOS:  true,
		})
	}

	for _, row := range caseRows {
		info := observerUnitInfo{
			ID:     row.UnitID,
			Name:   row.UnitName,
			Type:   row.UnitType,
			State:  row.UnitState,
			LGA:    row.UnitLGA,
			Ward:   row.UnitWard,
			Status: row.UnitStatus,
		}
		if loaded, exists := units[row.UnitID]; exists {
			info = loaded
		}

		incidentTime := row.IncidentDate
		if incidentTime.IsZero() {
			incidentTime = row.CreatedAt
		}

		location := ObserverLocation{State: tenant.State, LGA: tenant.LGA}
		if observerUnitInTenant(info, tenant) {
			location.Ward = info.Ward
		}

		unitID := row.UnitID
		records = append(records, observerIncidentRecord{
			Incident: ObserverIncident{
				Time:           incidentTime,
				Type:           observerIncidentType(row.Category),
				Location:       location,
				RespondingUnit: info.Name,
				Status:         row.Status,
				Description:    sanitizeObserverDescription(row.Description),
			},
			UnitID: &unitID,
			Unit:   info,
			IsSOS:  false,
		})
	}

	sort.Slice(records, func(i, j int) bool {
		if records[i].Incident.Time.Equal(records[j].Incident.Time) {
			return records[i].Incident.RespondingUnit < records[j].Incident.RespondingUnit
		}
		return records[i].Incident.Time.After(records[j].Incident.Time)
	})

	return records, nil
}

func observerAllowedCategories(agencyType string) ([]string, bool) {
	switch strings.ToLower(strings.TrimSpace(agencyType)) {
	case "police":
		return nil, true
	case "nscdc":
		return []string{"infrastructure", "vandalism", "disaster"}, true
	case "fire":
		return []string{"fire"}, true
	case "frsc":
		return []string{"road_accident", "traffic"}, true
	default:
		return nil, false
	}
}

func observerApplyCategoryFilter(query *gorm.DB, allowed []string) *gorm.DB {
	if allowed == nil {
		return query
	}
	if len(allowed) == 0 {
		return query.Where("1 = 0")
	}
	return query.Where("category IN ?", allowed)
}

func loadObserverUnits(ids []uuid.UUID) (map[uuid.UUID]observerUnitInfo, error) {
	out := make(map[uuid.UUID]observerUnitInfo, len(ids))
	if len(ids) == 0 {
		return out, nil
	}

	var rows []observerUnitInfo
	if err := config.DB.Model(&models.SecurityUnit{}).
		Select("id, name, type, state, lga, ward, status").
		Where("id IN ?", ids).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.ID] = row
	}
	return out, nil
}

func observerUnitInTenant(unit observerUnitInfo, tenant models.AgencyTenant) bool {
	return strings.EqualFold(strings.TrimSpace(unit.State), strings.TrimSpace(tenant.State)) &&
		strings.EqualFold(strings.TrimSpace(unit.LGA), strings.TrimSpace(tenant.LGA))
}

func observerIncidentType(category string) string {
	category = strings.TrimSpace(category)
	if category == "" {
		return "uncategorized"
	}
	return category
}

func observerQueryInt(c *gin.Context, key string, fallback, max int) int {
	value := strings.TrimSpace(c.Query(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	if parsed > max {
		return max
	}
	return parsed
}

// sanitizeObserverDescription redacts directly identifying patterns and then
// applies a conservative check for suspected personal names. Anything that
// cannot be safely reduced to non-identifying text is withheld entirely.
func sanitizeObserverDescription(raw string) string {
	text := strings.TrimSpace(raw)
	if text == "" {
		return ""
	}

	text = observerCoordinateRE.ReplaceAllString(text, "[redacted]")
	text = observerURLRE.ReplaceAllString(text, "[redacted]")
	text = observerEmailRE.ReplaceAllString(text, "[redacted]")
	text = observerPhoneRE.ReplaceAllString(text, "[redacted]")
	text = strings.Join(strings.Fields(text), " ")

	if observerContainsSuspectedName(text) {
		return "Incident details withheld"
	}

	runes := []rune(text)
	if len(runes) > observerDescriptionMaxLen {
		text = strings.TrimSpace(string(runes[:observerDescriptionMaxLen])) + "..."
	}
	return text
}

func observerContainsSuspectedName(text string) bool {
	for _, word := range observerCapitalizedRE.FindAllString(text, -1) {
		if _, safe := observerSafeWords[word]; !safe {
			return true
		}
	}
	for _, word := range observerAllCapsRE.FindAllString(text, -1) {
		if _, safe := observerSafeWords[word]; !safe {
			return true
		}
	}
	return false
}
