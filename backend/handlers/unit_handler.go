package handlers

import (
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"security-solution/config"
	"security-solution/data"
	"security-solution/models"
	"security-solution/services"
)

// validateGeography checks a submitted state/LGA pair against the
// embedded Nigeria dataset. It returns (errorMessage, true) when the
// pair is acceptable.
//
// An empty state is always accepted: older rows and older clients may
// legitimately carry no geography, and rejecting them would break
// backwards compatibility. A non-empty state must exist, and a non-empty
// LGA must exist within that state.
func validateGeography(state, lga string) (string, bool) {
	if state == "" {
		return "", true
	}
	st, ok := data.FindState(state)
	if !ok {
		return "unknown state: " + state, false
	}
	if lga != "" && !data.IsValidLGA(st.Name, lga) {
		return "unknown LGA " + lga + " for state " + st.Name, false
	}
	return "", true
}

// parseFormationDate reads a caller-supplied "YYYY-MM-DD" date.
//
// It returns (nil, true) for an absent value, so an omitted field is never
// confused with a malformed one. The pointer type on the request structs is
// what makes that distinction possible: JSON `null`/absent decodes to nil,
// while `"formationDate": ""` decodes to a pointer to the empty string, which
// is how a client clears the value on update.
func parseFormationDate(raw string) (*time.Time, error) {
	if raw == "" {
		return nil, nil
	}
	parsed, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

// derefFormationDate unwraps the optional JSON field to the raw string,
// treating an absent key as an empty one. On update the caller checks the
// pointer separately; on create both cases mean "unset".
func derefFormationDate(raw *string) string {
	if raw == nil {
		return ""
	}
	return *raw
}

// validateShiftPattern checks the registration form's shift enum. An empty
// value is accepted: the field is optional, and an omitted key must not block
// a submission. Returns (errorMessage, true) when the value is acceptable.
func validateShiftPattern(pattern string) (string, bool) {
	switch pattern {
	case "", "day", "night", "24h":
		return "", true
	default:
		return "shiftPattern must be one of day, night, 24h", false
	}
}

// derefBool unwraps an optional JSON bool. A nil pointer means the caller
// omitted the field on create, which defaults to false.
func derefBool(raw *bool) bool {
	if raw == nil {
		return false
	}
	return *raw
}

// GetNearbyUnits returns units near a location
func GetNearbyUnits(c *gin.Context) {
	latStr := c.Query("lat")
	lngStr := c.Query("lng")
	radiusStr := c.Query("radius")
	stateStr := c.Query("state")
	cityStr := c.Query("city")

	if latStr == "" || lngStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Latitude and longitude required"})
		return
	}

	lat, err := parseFloat(latStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid latitude"})
		return
	}

	lng, err := parseFloat(lngStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid longitude"})
		return
	}

	if lat < -90 || lat > 90 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Latitude out of range (-90 to 90)"})
		return
	}
	if lng < -180 || lng > 180 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Longitude out of range (-180 to 180)"})
		return
	}

	radius := 20.0
	if radiusStr != "" {
		r, err := parseFloat(radiusStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid radius"})
			return
		}
		radius = r
	}
	if radius <= 0 {
		radius = 20
	}
	if radius > 500 {
		radius = 500
	}

	var units []models.SecurityUnit
	query := config.DB.Where("status = ?", "active")

	if stateStr != "" {
		query = query.Where("state = ? OR state ILIKE ?", stateStr, "%"+stateStr+"%")
	}

	if cityStr != "" {
		query = query.Where("city = ? OR city ILIKE ?", cityStr, "%"+cityStr+"%")
	}

	if err := query.Find(&units).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch units"})
		return
	}

	type UnitWithDistance struct {
		models.SecurityUnit
		Distance  float64 `json:"distance"`
		IsInRange bool    `json:"isInRange"`
	}

	var result []UnitWithDistance
	for _, unit := range units {
		if unit.Latitude == 0 || unit.Longitude == 0 {
			continue
		}
		distance := haversine(lat, lng, unit.Latitude, unit.Longitude)
		isInRange := distance <= unit.OperationalRadius

		if distance <= radius || isInRange {
			result = append(result, UnitWithDistance{
				SecurityUnit: unit,
				Distance:     distance,
				IsInRange:    isInRange,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"units": result,
	})
}

// GetUnitsByLocation returns units by state/city
func GetUnitsByLocation(c *gin.Context) {
	state := c.Query("state")
	lga := c.Query("lga")
	city := c.Query("city")

	var units []models.SecurityUnit
	query := config.DB.Where("status = ?", "active")

	if state != "" {
		query = query.Where("state ILIKE ?", "%"+state+"%")
	}
	if lga != "" {
		query = query.Where("lga ILIKE ?", "%"+lga+"%")
	}
	if city != "" {
		query = query.Where("city ILIKE ?", "%"+city+"%")
	}

	if err := query.Find(&units).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch units"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"units": units,
	})
}

// GetAllUnits returns all units
func GetAllUnits(c *gin.Context) {
	var units []models.SecurityUnit
	if err := config.DB.Where("status = ?", "active").Find(&units).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch units"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"units": units,
	})
}

// GetUnitByID returns a specific unit
func GetUnitByID(c *gin.Context) {
	id := c.Param("id")
	unitID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid unit ID"})
		return
	}

	var unit models.SecurityUnit
	if err := config.DB.First(&unit, "id = ?", unitID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Unit not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"unit": unit,
	})
}

// CreateUnit creates a new unit
func CreateUnit(c *gin.Context) {
	var input struct {
		Name               string  `json:"name" binding:"required"`
		Type               string  `json:"type" binding:"required"`
		Latitude           float64 `json:"latitude"`
		Longitude          float64 `json:"longitude"`
		OperationalRadius  float64 `json:"operationalRadius"`
		State              string  `json:"state"`
		LGA                string  `json:"lga"`
		City               string  `json:"city"`
		CoverageArea       string  `json:"coverageArea"`
		ContactPerson      string  `json:"contactPerson"`
		ContactPhone       string  `json:"contactPhone"`
		ContactEmail       string  `json:"contactEmail"`
		RegistrationNumber string  `json:"registrationNumber"`
		// Pointer so an absent `formationDate` is distinguishable from an
		// explicit `""`; both are legal on create and mean "unset".
		FormationDate *string `json:"formationDate"`
		Ward          string  `json:"ward"`
		TotalMembers  int     `json:"totalMembers"`

		// Section B — commander.
		CommanderName            string `json:"commanderName"`
		CommanderNIN             string `json:"commanderNin"`
		CommanderPhoneAlt        string `json:"commanderPhoneAlt"`
		CommanderOccupation      string `json:"commanderOccupation"`
		CommanderPriorExperience string `json:"commanderPriorExperience"`

		// Section C — operational profile.
		HasUniform         *bool  `json:"hasUniform"`
		UniformDescription string `json:"uniformDescription"`
		ShiftPattern       string `json:"shiftPattern"`
		PermittedTools     string `json:"permittedTools"`
		WeaponsRegistered  *bool  `json:"weaponsRegistered"`

		// Section D — traditional endorsement.
		KindredHeadName  string `json:"kindredHeadName"`
		KindredHeadPhone string `json:"kindredHeadPhone"`
		WardHeadName     string `json:"wardHeadName"`
		WardHeadPhone    string `json:"wardHeadPhone"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if msg, ok := validateShiftPattern(input.ShiftPattern); !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	// Wave 10.2a — reject unknown states/LGAs, but never reject empty
	// values (backwards compatibility with rows created before the
	// Nigeria dataset existed).
	if msg, ok := validateGeography(input.State, input.LGA); !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj := user.(*models.User)

	if !userObj.IsSuperAdmin && userObj.Role != "super_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only super admins can create units"})
		return
	}

	if input.OperationalRadius == 0 {
		input.OperationalRadius = 10
	}

	formationDate, err := parseFormationDate(derefFormationDate(input.FormationDate))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid formationDate (expected YYYY-MM-DD)"})
		return
	}

	unit := models.SecurityUnit{
		Name:               input.Name,
		Type:               input.Type,
		Latitude:           input.Latitude,
		Longitude:          input.Longitude,
		OperationalRadius:  input.OperationalRadius,
		State:              input.State,
		LGA:                input.LGA,
		City:               input.City,
		CoverageArea:       input.CoverageArea,
		ContactPerson:      input.ContactPerson,
		ContactPhone:       input.ContactPhone,
		ContactEmail:       input.ContactEmail,
		RegistrationNumber: input.RegistrationNumber,
		Ward:               input.Ward,
		FormationDate:      formationDate,
		TotalMembers:       input.TotalMembers,
		Status:             "active",
		IsVerified:         false,

		CommanderName:            input.CommanderName,
		CommanderNIN:             input.CommanderNIN,
		CommanderPhoneAlt:        input.CommanderPhoneAlt,
		CommanderOccupation:      input.CommanderOccupation,
		CommanderPriorExperience: input.CommanderPriorExperience,

		HasUniform:         derefBool(input.HasUniform),
		UniformDescription: input.UniformDescription,
		ShiftPattern:       input.ShiftPattern,
		PermittedTools:     input.PermittedTools,
		WeaponsRegistered:  derefBool(input.WeaponsRegistered),

		KindredHeadName:  input.KindredHeadName,
		KindredHeadPhone: input.KindredHeadPhone,
		WardHeadName:     input.WardHeadName,
		WardHeadPhone:    input.WardHeadPhone,
	}

	// `RegistrationNumber` carries a unique index with no default, so two
	// units created without one collide and the loser gets a bare 500.
	// Generate a readable placeholder instead; the field stays editable.
	if unit.RegistrationNumber == "" {
		unit.RegistrationNumber = "REG-" + strings.ToUpper(uuid.NewString()[:8])
	}

	if err := config.DB.Create(&unit).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create unit"})
		return
	}

	auditSvc := services.NewAuditService()
	_ = auditSvc.LogAction(
		userObj.ID,
		"unit.create",
		"unit",
		unit.ID.String(),
		nil,
		unit,
		c.ClientIP(),
		c.Request.UserAgent(),
	)

	c.JSON(http.StatusCreated, gin.H{
		"message": "Unit created successfully",
		"unit":    unit,
	})
}

// UpdateUnit updates a unit
func UpdateUnit(c *gin.Context) {
	id := c.Param("id")
	unitID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid unit ID"})
		return
	}

	var input struct {
		Name              string  `json:"name"`
		Type              string  `json:"type"`
		Latitude          float64 `json:"latitude"`
		Longitude         float64 `json:"longitude"`
		OperationalRadius float64 `json:"operationalRadius"`
		State             string  `json:"state"`
		LGA               string  `json:"lga"`
		City              string  `json:"city"`
		CoverageArea      string  `json:"coverageArea"`
		ContactPerson     string  `json:"contactPerson"`
		ContactPhone      string  `json:"contactPhone"`
		ContactEmail      string  `json:"contactEmail"`
		Status            string  `json:"status"`
		// Pointer for the same reason as CreateUnit: nil is "leave alone",
		// `""` is "clear this field".
		FormationDate *string `json:"formationDate"`
		Ward          string  `json:"ward"`
		TotalMembers  int     `json:"totalMembers"`

		// Section B — commander.
		CommanderName            string `json:"commanderName"`
		CommanderNIN             string `json:"commanderNin"`
		CommanderPhoneAlt        string `json:"commanderPhoneAlt"`
		CommanderOccupation      string `json:"commanderOccupation"`
		CommanderPriorExperience string `json:"commanderPriorExperience"`

		// Section C — operational profile.
		HasUniform         *bool  `json:"hasUniform"`
		UniformDescription string `json:"uniformDescription"`
		ShiftPattern       string `json:"shiftPattern"`
		PermittedTools     string `json:"permittedTools"`
		WeaponsRegistered  *bool  `json:"weaponsRegistered"`

		// Section D — traditional endorsement.
		KindredHeadName  string `json:"kindredHeadName"`
		KindredHeadPhone string `json:"kindredHeadPhone"`
		WardHeadName     string `json:"wardHeadName"`
		WardHeadPhone    string `json:"wardHeadPhone"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if msg, ok := validateShiftPattern(input.ShiftPattern); !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	// Wave 10.2a — same geography validation as CreateUnit.
	if msg, ok := validateGeography(input.State, input.LGA); !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj := user.(*models.User)

	if !callerIsHeadAdminOfUnit(userObj, unitID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only the unit's head admin can update this unit"})
		return
	}

	var unit models.SecurityUnit
	if err := config.DB.First(&unit, "id = ?", unitID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Unit not found"})
		return
	}

	// Snapshot pre-update values for the audit trail.
	oldUnit := unit

	if input.Name != "" {
		unit.Name = input.Name
	}
	if input.Type != "" {
		unit.Type = input.Type
	}
	if input.Latitude != 0 {
		unit.Latitude = input.Latitude
	}
	if input.Longitude != 0 {
		unit.Longitude = input.Longitude
	}
	if input.OperationalRadius != 0 {
		unit.OperationalRadius = input.OperationalRadius
	}
	if input.State != "" {
		unit.State = input.State
	}
	if input.LGA != "" {
		unit.LGA = input.LGA
	}
	if input.City != "" {
		unit.City = input.City
	}
	if input.CoverageArea != "" {
		unit.CoverageArea = input.CoverageArea
	}
	if input.ContactPerson != "" {
		unit.ContactPerson = input.ContactPerson
	}
	if input.ContactPhone != "" {
		unit.ContactPhone = input.ContactPhone
	}
	if input.ContactEmail != "" {
		unit.ContactEmail = input.ContactEmail
	}
	if input.Status != "" {
		unit.Status = input.Status
	}
	if input.Ward != "" {
		unit.Ward = input.Ward
	}
	if input.TotalMembers != 0 {
		unit.TotalMembers = input.TotalMembers
	}

	// Registration form fields. Strings follow the same "non-zero means set"
	// guard used above; the bools are pointers so an absent key leaves the
	// stored value alone, and only an explicit `false` clears it.
	if input.CommanderName != "" {
		unit.CommanderName = input.CommanderName
	}
	if input.CommanderNIN != "" {
		unit.CommanderNIN = input.CommanderNIN
	}
	if input.CommanderPhoneAlt != "" {
		unit.CommanderPhoneAlt = input.CommanderPhoneAlt
	}
	if input.CommanderOccupation != "" {
		unit.CommanderOccupation = input.CommanderOccupation
	}
	if input.CommanderPriorExperience != "" {
		unit.CommanderPriorExperience = input.CommanderPriorExperience
	}
	if input.HasUniform != nil {
		unit.HasUniform = *input.HasUniform
	}
	if input.UniformDescription != "" {
		unit.UniformDescription = input.UniformDescription
	}
	if input.ShiftPattern != "" {
		unit.ShiftPattern = input.ShiftPattern
	}
	if input.PermittedTools != "" {
		unit.PermittedTools = input.PermittedTools
	}
	if input.WeaponsRegistered != nil {
		unit.WeaponsRegistered = *input.WeaponsRegistered
	}
	if input.KindredHeadName != "" {
		unit.KindredHeadName = input.KindredHeadName
	}
	if input.KindredHeadPhone != "" {
		unit.KindredHeadPhone = input.KindredHeadPhone
	}
	if input.WardHeadName != "" {
		unit.WardHeadName = input.WardHeadName
	}
	if input.WardHeadPhone != "" {
		unit.WardHeadPhone = input.WardHeadPhone
	}
	// Only touch the date when the caller sent the key at all. Every other
	// field above uses the same "non-zero means set" guard, but a date has a
	// meaningful zero: `""` is how a client clears it, and an update that
	// omits `formationDate` entirely must not wipe a date it never mentioned.
	if input.FormationDate != nil {
		parsed, err := parseFormationDate(*input.FormationDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid formationDate (expected YYYY-MM-DD)"})
			return
		}
		unit.FormationDate = parsed
	}

	if err := config.DB.Save(&unit).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update unit"})
		return
	}

	auditSvc := services.NewAuditService()
	_ = auditSvc.LogAction(
		userObj.ID,
		"unit.update",
		"unit",
		unitID.String(),
		oldUnit,
		unit,
		c.ClientIP(),
		c.Request.UserAgent(),
	)

	c.JSON(http.StatusOK, gin.H{
		"message": "Unit updated successfully",
		"unit":    unit,
	})
}

// callerIsHeadAdminOfUnit reports whether the caller may act for the given
// unit. Super admins bypass the membership lookup; everyone else must be an
// active, verified head admin of that exact unit.
func callerIsHeadAdminOfUnit(user *models.User, unitID uuid.UUID) bool {
	if user == nil {
		return false
	}
	if user.IsSuperAdmin || user.Role == "super_admin" {
		return true
	}

	var membership models.UnitMembership
	err := config.DB.
		Where("unit_id = ? AND user_id = ? AND status = ? AND verified_at IS NOT NULL",
			unitID, user.ID, models.MembershipActive).
		First(&membership).Error
	if err != nil {
		return false
	}
	return membership.IsHeadAdmin
}

// Helper functions
func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscan(s, &f)
	return f, err
}

// haversine - kept only here, removed from case_handler.go
func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371
	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLon := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}
