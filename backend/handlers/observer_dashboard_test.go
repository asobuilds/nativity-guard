//go:build integration

package handlers_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"security-solution/handlers"
	"security-solution/internal/testutil"
	"security-solution/models"
	"security-solution/utils"
)

const (
	observerSecretName    = "Adaeze Obi"
	observerSecretPhone   = "08031234567"
	observerSecretEmail   = "ada.secret@example.test"
	observerSecretAddress = "12 Secret Close, Ikeja"
	observerSecretLat     = "9.0765"
	observerSecretLng     = "7.3986"
)

type observerFixture struct {
	tenant     *models.AgencyTenant
	policeUnit *models.SecurityUnit
	fireUnit   *models.SecurityUnit
	otherUnit  *models.SecurityUnit
	policeUser *models.User
	fireUser   *models.User
	nscdcUser  *models.User
	frscUser   *models.User
}

type observerIncidentsResponse struct {
	Incidents []handlers.ObserverIncident `json:"incidents"`
	Total     int                         `json:"total"`
}

type observerUnitsResponse struct {
	Units []handlers.ObserverUnitSummary `json:"units"`
}

func observerTenantForTest(t *testing.T, agencyType, state, lga string) *models.AgencyTenant {
	t.Helper()

	tenant := &models.AgencyTenant{
		AgencyType:   agencyType,
		State:        state,
		LGA:          lga,
		DivisionName: agencyType + " " + lga + " Division",
	}
	if err := testutil.GetDB().Create(tenant).Error; err != nil {
		t.Fatalf("create agency tenant: %v", err)
	}
	return tenant
}

func observerUserForTest(t *testing.T, tenantID uuid.UUID) *models.User {
	t.Helper()

	user := testutil.MakeUser(t, models.RoleAgencyObserver)
	observer := &models.AgencyObserver{
		UserID:   user.ID,
		TenantID: tenantID,
		Rank:     "Observer",
	}
	if err := testutil.GetDB().Create(observer).Error; err != nil {
		t.Fatalf("create agency observer: %v", err)
	}
	return user
}

func observerUnitForTest(t *testing.T, unitType, state, lga, ward string) *models.SecurityUnit {
	t.Helper()

	unit := testutil.MakeUnit(t, nil)
	unit.Type = unitType
	unit.State = state
	unit.LGA = lga
	unit.Ward = ward
	unit.City = "Secret City"
	unit.ContactPerson = observerSecretName
	unit.ContactPhone = observerSecretPhone
	unit.ContactEmail = observerSecretEmail
	unit.CommanderName = observerSecretName
	unit.RegistrationNumber = "REG-OBSERVER-SECRET-" + uuid.NewString()[:8]
	if err := testutil.GetDB().Save(unit).Error; err != nil {
		t.Fatalf("save observer unit: %v", err)
	}
	return unit
}

func observerCaseForTest(t *testing.T, reporter *models.User, unit *models.SecurityUnit, category, status, description string, at time.Time) *models.Case {
	t.Helper()

	caseObj := &models.Case{
		TrackingID:   "CS-OBS-" + uuid.NewString()[:8],
		UnitID:       unit.ID,
		ReportedBy:   reporter.ID,
		Title:        "Observer test case",
		Description:  description,
		Category:     category,
		Status:       status,
		Priority:     "medium",
		IsPublic:     true,
		Location:     observerSecretAddress,
		Latitude:     9.0765,
		Longitude:    7.3986,
		IncidentDate: at,
		CreatedAt:    at,
		UpdatedAt:    at,
	}
	if err := testutil.GetDB().Create(caseObj).Error; err != nil {
		t.Fatalf("create observer case: %v", err)
	}
	return caseObj
}

func observerSOSForTest(t *testing.T, reporter *models.User, unit *models.SecurityUnit, category, status, description, state, lga string, at time.Time) *models.SOSAlert {
	t.Helper()

	unitID := unit.ID
	sos := &models.SOSAlert{
		UserID:            reporter.ID,
		UnitID:            &unitID,
		Category:          category,
		Status:            status,
		Priority:          "high",
		Severity:          "normal",
		DispatchState:     "open",
		JurisdictionState: state,
		JurisdictionLGA:   lga,
		Description:       description,
		Latitude:          9.0765,
		Longitude:         7.3986,
		CreatedAt:         at,
		UpdatedAt:         at,
	}
	if status == "resolved" {
		sos.DispatchState = "resolved"
	}
	if err := testutil.GetDB().Create(sos).Error; err != nil {
		t.Fatalf("create observer SOS: %v", err)
	}
	return sos
}

func setupObserverFixture(t *testing.T) *observerFixture {
	t.Helper()

	tenant := observerTenantForTest(t, "police", "Lagos", "Ikeja")
	fireTenant := observerTenantForTest(t, "fire", "Lagos", "Ikeja")
	nscdcTenant := observerTenantForTest(t, "nscdc", "Lagos", "Ikeja")
	frscTenant := observerTenantForTest(t, "frsc", "Lagos", "Ikeja")

	policeUnit := observerUnitForTest(t, "police", "Lagos", "Ikeja", "Ward A")
	fireUnit := observerUnitForTest(t, "fire", "Lagos", "Ikeja", "Ward B")
	otherUnit := observerUnitForTest(t, "fire", "Lagos", "Oshodi", "Ward C")
	reporter := testutil.MakeUser(t, models.RoleCitizen)

	now := time.Now().UTC()

	observerCaseForTest(t, reporter, policeUnit, "road_accident", "assigned", "collision on the expressway", now.Add(-4*time.Hour))
	observerCaseForTest(t, reporter, policeUnit, "vandalism", "investigating", "broken fence near the school", now.Add(-5*time.Hour))
	observerCaseForTest(t, reporter, policeUnit, "disaster", "pending", "flood water across the road", now.Add(-6*time.Hour))
	observerCaseForTest(t, reporter, policeUnit, "infrastructure", "pending", "damaged bridge on the road", now.Add(-7*time.Hour))
	observerCaseForTest(t, reporter, policeUnit, "", "pending", "general disturbance reported", now.Add(-8*time.Hour))
	observerCaseForTest(t, reporter, fireUnit, "fire", "pending", "smoke near the market", now.Add(-3*time.Hour))
	observerCaseForTest(t, reporter, fireUnit, "fire", "closed", "old fire", now.Add(-9*time.Hour))
	observerCaseForTest(t, reporter, otherUnit, "fire", "pending", "fire in another area", now.Add(-2*time.Hour))

	observerSOSForTest(t, reporter, fireUnit, "fire", "pending", "", "Lagos", "Ikeja", now.Add(-1*time.Hour))
	observerSOSForTest(t, reporter, policeUnit, "vandalism", "dispatched",
		"Call "+observerSecretName+" on "+observerSecretPhone+" or "+observerSecretEmail+
			" near "+observerSecretLat+", "+observerSecretLng,
		"Lagos", "Ikeja", now.Add(-2*time.Hour))
	observerSOSForTest(t, reporter, policeUnit, "road_accident", "escalated", "crash on the highway", "Lagos", "Ikeja", now.Add(-3*time.Hour))
	observerSOSForTest(t, reporter, fireUnit, "fire", "resolved", "resolved fire", "Lagos", "Ikeja", now.Add(-4*time.Hour))
	observerSOSForTest(t, reporter, otherUnit, "fire", "pending", "fire elsewhere", "Lagos", "Oshodi", now.Add(-2*time.Hour))

	return &observerFixture{
		tenant:     tenant,
		policeUnit: policeUnit,
		fireUnit:   fireUnit,
		otherUnit:  otherUnit,
		policeUser: observerUserForTest(t, tenant.ID),
		fireUser:   observerUserForTest(t, fireTenant.ID),
		nscdcUser:  observerUserForTest(t, nscdcTenant.ID),
		frscUser:   observerUserForTest(t, frscTenant.ID),
	}
}

func decodeObserverIncidents(t *testing.T, body []byte) observerIncidentsResponse {
	t.Helper()

	var response observerIncidentsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("decode observer incidents: %v. body=%s", err, body)
	}
	return response
}

func assertObserverResponseHasNoPII(t *testing.T, body []byte) {
	t.Helper()

	text := string(body)
	for _, secret := range []string{
		observerSecretName,
		observerSecretPhone,
		observerSecretEmail,
		observerSecretAddress,
		observerSecretLat,
		observerSecretLng,
		"Secret City",
		"REG-OBSERVER-SECRET",
	} {
		if strings.Contains(text, secret) {
			t.Errorf("observer response leaked %q: %s", secret, text)
		}
	}
}

func assertObserverIncidentShape(t *testing.T, body []byte) {
	t.Helper()

	objects := rawObjects(t, body, "incidents")
	for _, object := range objects {
		assertKeysPresent(t, object, []string{"time", "type", "location", "respondingUnit", "status", "description"})
		assertKeysAbsent(t, object, []string{
			"id", "trackingId", "userId", "reportedBy", "unitId", "assignedTo",
			"latitude", "longitude", "gisLatitude", "gisLongitude", "locationGeohash",
			"email", "phone", "firstName", "lastName", "title", "address",
		})

		var location map[string]json.RawMessage
		if err := json.Unmarshal(object["location"], &location); err != nil {
			t.Fatalf("decode observer location: %v", err)
		}
		assertKeysPresent(t, location, []string{"state", "lga"})
		assertKeysAbsent(t, location, []string{"latitude", "longitude", "lat", "lng", "address"})
	}
}

func TestObserverDashboard_IncidentsScopingAndPrivacy(t *testing.T) {
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	server := FreshServer(t)
	fixture := setupObserverFixture(t)

	t.Run("requires authentication", func(t *testing.T) {
		w := agencyJSONRequest(t, server, http.MethodGet, "/api/v1/observer/incidents", "", nil)
		requireAgencyStatus(t, w, http.StatusUnauthorized)
	})

	t.Run("rejects non-observer roles", func(t *testing.T) {
		citizen := testutil.MakeUser(t, models.RoleCitizen)
		w := agencyJSONRequest(t, server, http.MethodGet, "/api/v1/observer/incidents", testutil.AuthHeader(t, citizen), nil)
		requireAgencyStatus(t, w, http.StatusForbidden)
	})

	t.Run("rejects observer without a tenant assignment", func(t *testing.T) {
		unassigned := testutil.MakeUser(t, models.RoleAgencyObserver)
		w := agencyJSONRequest(t, server, http.MethodGet, "/api/v1/observer/incidents", testutil.AuthHeader(t, unassigned), nil)
		requireAgencyStatus(t, w, http.StatusForbidden)
	})

	t.Run("rejects observer whose tenant no longer exists", func(t *testing.T) {
		user := testutil.MakeUser(t, models.RoleAgencyObserver)
		observer := &models.AgencyObserver{UserID: user.ID, TenantID: uuid.New(), Rank: "Observer"}
		if err := testutil.GetDB().Create(observer).Error; err != nil {
			t.Fatalf("create orphan observer: %v", err)
		}
		w := agencyJSONRequest(t, server, http.MethodGet, "/api/v1/observer/incidents", testutil.AuthHeader(t, user), nil)
		requireAgencyStatus(t, w, http.StatusForbidden)
	})

	t.Run("police observer sees every active or open incident in the LGA", func(t *testing.T) {
		token := testutil.AuthHeader(t, fixture.policeUser)
		w := agencyJSONRequest(t, server, http.MethodGet, "/api/v1/observer/incidents", token, nil)
		requireAgencyStatus(t, w, http.StatusOK)

		response := decodeObserverIncidents(t, w.Body.Bytes())
		if response.Total != 9 {
			t.Fatalf("police incident total = %d, want 9. body=%s", response.Total, w.Body.String())
		}
		if len(response.Incidents) != 9 {
			t.Fatalf("police incident count = %d, want 9", len(response.Incidents))
		}
		for _, incident := range response.Incidents {
			if incident.Location.State != "Lagos" || incident.Location.LGA != "Ikeja" {
				t.Errorf("police observer received out-of-LGA incident: %+v", incident)
			}
			if incident.Status == "resolved" || incident.Status == "closed" {
				t.Errorf("police observer received inactive incident: %+v", incident)
			}
		}

		assertObserverResponseHasNoPII(t, w.Body.Bytes())
		assertObserverIncidentShape(t, w.Body.Bytes())
	})

	t.Run("fire observer sees only fire incidents", func(t *testing.T) {
		token := testutil.AuthHeader(t, fixture.fireUser)
		w := agencyJSONRequest(t, server, http.MethodGet, "/api/v1/observer/incidents", token, nil)
		requireAgencyStatus(t, w, http.StatusOK)

		response := decodeObserverIncidents(t, w.Body.Bytes())
		if response.Total != 2 {
			t.Fatalf("fire incident total = %d, want 2. body=%s", response.Total, w.Body.String())
		}
		for _, incident := range response.Incidents {
			if incident.Type != "fire" {
				t.Errorf("fire observer received non-fire incident type %q", incident.Type)
			}
		}
		assertObserverResponseHasNoPII(t, w.Body.Bytes())
	})

	t.Run("nscdc observer sees only its allowed categories", func(t *testing.T) {
		token := testutil.AuthHeader(t, fixture.nscdcUser)
		w := agencyJSONRequest(t, server, http.MethodGet, "/api/v1/observer/incidents", token, nil)
		requireAgencyStatus(t, w, http.StatusOK)

		response := decodeObserverIncidents(t, w.Body.Bytes())
		if response.Total != 4 {
			t.Fatalf("nscdc incident total = %d, want 4. body=%s", response.Total, w.Body.String())
		}
		for _, incident := range response.Incidents {
			switch incident.Type {
			case "infrastructure", "vandalism", "disaster":
			default:
				t.Errorf("nscdc observer received disallowed category %q", incident.Type)
			}
		}
	})

	t.Run("frsc observer sees only road and traffic categories", func(t *testing.T) {
		token := testutil.AuthHeader(t, fixture.frscUser)
		w := agencyJSONRequest(t, server, http.MethodGet, "/api/v1/observer/incidents", token, nil)
		requireAgencyStatus(t, w, http.StatusOK)

		response := decodeObserverIncidents(t, w.Body.Bytes())
		if response.Total != 2 {
			t.Fatalf("frsc incident total = %d, want 2. body=%s", response.Total, w.Body.String())
		}
		for _, incident := range response.Incidents {
			switch incident.Type {
			case "road_accident", "traffic":
			default:
				t.Errorf("frsc observer received disallowed category %q", incident.Type)
			}
		}
	})
}

func TestObserverDashboard_UnitsSummary(t *testing.T) {
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	server := FreshServer(t)
	fixture := setupObserverFixture(t)

	t.Run("requires observer role", func(t *testing.T) {
		w := agencyJSONRequest(t, server, http.MethodGet, "/api/v1/observer/units", "", nil)
		requireAgencyStatus(t, w, http.StatusUnauthorized)

		citizen := testutil.MakeUser(t, models.RoleCitizen)
		w = agencyJSONRequest(t, server, http.MethodGet, "/api/v1/observer/units", testutil.AuthHeader(t, citizen), nil)
		requireAgencyStatus(t, w, http.StatusForbidden)
	})

	t.Run("police summary counts active SOS and open cases per unit", func(t *testing.T) {
		token := testutil.AuthHeader(t, fixture.policeUser)
		w := agencyJSONRequest(t, server, http.MethodGet, "/api/v1/observer/units", token, nil)
		requireAgencyStatus(t, w, http.StatusOK)

		var response observerUnitsResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("decode unit summary: %v. body=%s", err, w.Body.String())
		}
		if len(response.Units) != 2 {
			t.Fatalf("unit summary count = %d, want 2. body=%s", len(response.Units), w.Body.String())
		}

		units := make(map[string]handlers.ObserverUnitSummary, len(response.Units))
		for _, unit := range response.Units {
			units[unit.RespondingUnit] = unit
			if unit.Location.State != "Lagos" || unit.Location.LGA != "Ikeja" {
				t.Errorf("unit summary contains out-of-LGA unit: %+v", unit)
			}
		}

		police := units[fixture.policeUnit.Name]
		if police.ActiveSOS != 2 || police.OpenCases != 5 {
			t.Errorf("police unit summary = %+v, want activeSOS=2 openCases=5", police)
		}
		fire := units[fixture.fireUnit.Name]
		if fire.ActiveSOS != 1 || fire.OpenCases != 1 {
			t.Errorf("fire unit summary = %+v, want activeSOS=1 openCases=1", fire)
		}

		assertObserverResponseHasNoPII(t, w.Body.Bytes())
	})

	t.Run("fire summary only includes fire activity", func(t *testing.T) {
		token := testutil.AuthHeader(t, fixture.fireUser)
		w := agencyJSONRequest(t, server, http.MethodGet, "/api/v1/observer/units", token, nil)
		requireAgencyStatus(t, w, http.StatusOK)

		var response observerUnitsResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("decode fire unit summary: %v. body=%s", err, w.Body.String())
		}
		if len(response.Units) != 1 {
			t.Fatalf("fire unit summary count = %d, want 1. body=%s", len(response.Units), w.Body.String())
		}
		unit := response.Units[0]
		if unit.RespondingUnit != fixture.fireUnit.Name || unit.ActiveSOS != 1 || unit.OpenCases != 1 {
			t.Fatalf("fire unit summary = %+v, want fire unit with activeSOS=1 openCases=1", unit)
		}
		assertObserverResponseHasNoPII(t, w.Body.Bytes())
	})
}

func TestIncidentCategoryValidation(t *testing.T) {
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	t.Setenv("TERMS_CONSENT_ENFORCED", "false")
	server := FreshServer(t)
	citizen := testutil.MakeUser(t, models.RoleCitizen)
	token := testutil.AuthHeader(t, citizen)
	unit := testutil.MakeUnit(t, nil)
	unit.State = "Lagos"
	unit.LGA = "Ikeja"
	if err := testutil.GetDB().Save(unit).Error; err != nil {
		t.Fatalf("save category test unit: %v", err)
	}

	casePayload := func(category string) map[string]any {
		return map[string]any{
			"unitId":      unit.ID.String(),
			"title":       "Category validation case",
			"description": "category validation",
			"category":    category,
			"latitude":    9.0765,
			"longitude":   7.3986,
			"location":    "Ikeja",
		}
	}

	t.Run("case persists an allowed category", func(t *testing.T) {
		w := agencyJSONRequest(t, server, http.MethodPost, "/api/v1/cases", token, casePayload("road_accident"))
		requireAgencyStatus(t, w, http.StatusCreated)

		var created models.Case
		if err := testutil.GetDB().Where("reported_by = ?", citizen.ID).Order("created_at desc").First(&created).Error; err != nil {
			t.Fatalf("load created case: %v", err)
		}
		if created.Category != "road_accident" {
			t.Fatalf("case category = %q, want road_accident", created.Category)
		}
	})

	t.Run("case rejects an unknown category without persisting", func(t *testing.T) {
		var before int64
		if err := testutil.GetDB().Model(&models.Case{}).Count(&before).Error; err != nil {
			t.Fatalf("count cases before: %v", err)
		}

		w := agencyJSONRequest(t, server, http.MethodPost, "/api/v1/cases", token, casePayload("theft"))
		requireAgencyStatus(t, w, http.StatusBadRequest)

		var after int64
		if err := testutil.GetDB().Model(&models.Case{}).Count(&after).Error; err != nil {
			t.Fatalf("count cases after: %v", err)
		}
		if after != before {
			t.Fatalf("invalid category created a case: before=%d after=%d", before, after)
		}
	})

	sosPayload := func(category string) map[string]any {
		return map[string]any{
			"latitude":    9.0765,
			"longitude":   7.3986,
			"description": "",
			"category":    category,
		}
	}

	geohash := utils.EncodeGeohash(9.0765, 7.3986, 6)
	cache := &models.GeocodeCache{
		Geohash:   geohash,
		Latitude:  9.0765,
		Longitude: 7.3986,
		State:     "Lagos",
		LGA:       "Ikeja",
		Source:    "nominatim",
		FetchedAt: time.Now().UTC(),
	}
	if err := testutil.GetDB().Create(cache).Error; err != nil {
		t.Fatalf("seed geocode cache: %v", err)
	}

	t.Run("SOS persists an allowed category", func(t *testing.T) {
		w := agencyJSONRequest(t, server, http.MethodPost, "/api/v1/sos/send", token, sosPayload("fire"))
		requireAgencyStatus(t, w, http.StatusCreated)

		var created models.SOSAlert
		if err := testutil.GetDB().Where("user_id = ?", citizen.ID).Order("created_at desc").First(&created).Error; err != nil {
			t.Fatalf("load created SOS: %v", err)
		}
		if created.Category != "fire" {
			t.Fatalf("SOS category = %q, want fire", created.Category)
		}
	})

	t.Run("SOS rejects an unknown category without persisting", func(t *testing.T) {
		var before int64
		if err := testutil.GetDB().Model(&models.SOSAlert{}).Count(&before).Error; err != nil {
			t.Fatalf("count SOS before: %v", err)
		}

		w := agencyJSONRequest(t, server, http.MethodPost, "/api/v1/sos/send", token, sosPayload("theft"))
		requireAgencyStatus(t, w, http.StatusBadRequest)

		var after int64
		if err := testutil.GetDB().Model(&models.SOSAlert{}).Count(&after).Error; err != nil {
			t.Fatalf("count SOS after: %v", err)
		}
		if after != before {
			t.Fatalf("invalid category created an SOS: before=%d after=%d", before, after)
		}
	})
}
