//go:build integration

package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"security-solution/handlers"
	"security-solution/internal/testutil"
)

func getPublicEndpoint(t *testing.T, path string) []byte {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, path, nil)
	w := httptest.NewRecorder()
	FreshServer(t).ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET %s status = %d, want 200. body=%s", path, w.Code, w.Body.String())
	}
	return w.Body.Bytes()
}

func rawObjects(t *testing.T, body []byte, key string) []map[string]json.RawMessage {
	t.Helper()

	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}
	raw, ok := envelope[key]
	if !ok {
		t.Fatalf("response missing %q key: %s", key, body)
	}
	var objects []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &objects); err != nil {
		t.Fatalf("unmarshal %q: %v", key, err)
	}
	return objects
}

func assertKeysAbsent(t *testing.T, object map[string]json.RawMessage, keys []string) {
	t.Helper()

	for _, key := range keys {
		if _, ok := object[key]; ok {
			t.Errorf("public payload must not contain %q", key)
		}
	}
}

func assertKeysPresent(t *testing.T, object map[string]json.RawMessage, keys []string) {
	t.Helper()

	for _, key := range keys {
		if _, ok := object[key]; !ok {
			t.Errorf("public payload missing %q", key)
		}
	}
}

func TestPublicUnits_ExcludesSensitiveFields(t *testing.T) {
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	unit := testutil.MakeUnit(t, nil)
	unit.City = "City Sentinel"
	unit.CoverageArea = "Coverage Area Sentinel"
	unit.ContactPerson = "Contact Person Sentinel"
	unit.ContactPhone = "08030000001"
	unit.ContactEmail = "unit-secret@example.test"
	unit.RegistrationNumber = "REG-SENTINEL-999"
	unit.OperationalRadius = 12.5
	unit.TotalMembers = 48
	unit.CommanderName = "Commander Sentinel"
	unit.CommanderNIN = "NIN-SENTINEL-001"
	unit.CommanderPhoneAlt = "08030000002"
	unit.CommanderOccupation = "Occupation Sentinel"
	unit.CommanderPriorExperience = "Experience Sentinel"
	unit.HasUniform = true
	unit.UniformDescription = "Uniform Sentinel"
	unit.ShiftPattern = "24h"
	unit.PermittedTools = `["Sentinel Tool"]`
	unit.WeaponsRegistered = true
	unit.KindredHeadName = "Kindred Sentinel"
	unit.KindredHeadPhone = "08030000003"
	unit.WardHeadName = "Ward Sentinel"
	unit.WardHeadPhone = "08030000004"
	unit.VerificationNotes = "Verification Sentinel"
	if err := testutil.GetDB().Save(unit).Error; err != nil {
		t.Fatalf("save unit: %v", err)
	}

	body := getPublicEndpoint(t, "/api/v1/public/units")
	bodyText := string(body)
	for _, secret := range []string{
		"City Sentinel",
		"Contact Person Sentinel",
		"08030000001",
		"unit-secret@example.test",
		"REG-SENTINEL-999",
		"Commander Sentinel",
		"NIN-SENTINEL-001",
		"08030000002",
		"Occupation Sentinel",
		"Experience Sentinel",
		"Uniform Sentinel",
		"Sentinel Tool",
		"Kindred Sentinel",
		"08030000003",
		"Ward Sentinel",
		"08030000004",
		"Verification Sentinel",
	} {
		if strings.Contains(bodyText, secret) {
			t.Errorf("public units response leaked %q: %s", secret, body)
		}
	}

	rawUnits := rawObjects(t, body, "units")
	if len(rawUnits) != 1 {
		t.Fatalf("units count = %d, want 1", len(rawUnits))
	}
	assertKeysAbsent(t, rawUnits[0], []string{
		"contactPerson",
		"contactPhone",
		"contactEmail",
		"registrationNumber",
		"commanderName",
		"commanderNin",
		"commanderPhoneAlt",
		"commanderOccupation",
		"commanderPriorExperience",
		"hasUniform",
		"uniformDescription",
		"shiftPattern",
		"permittedTools",
		"weaponsRegistered",
		"kindredHeadName",
		"kindredHeadPhone",
		"wardHeadName",
		"wardHeadPhone",
		"verificationStatus",
		"verifiedBy",
		"verificationNotes",
		"hostUserId",
		"adminCount",
		"memberCount",
		"city",
		"coverageArea",
	})
	assertKeysPresent(t, rawUnits[0], []string{
		"id",
		"name",
		"type",
		"state",
		"lga",
		"description",
		"status",
		"latitude",
		"longitude",
		"totalMembers",
		"operationalRadius",
		"isVerified",
	})

	var response struct {
		Units []handlers.PublicUnitDTO `json:"units"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("unmarshal units: %v", err)
	}
	got := response.Units[0]
	if got.ID != unit.ID {
		t.Errorf("id = %s, want %s", got.ID, unit.ID)
	}
	if got.Name != unit.Name {
		t.Errorf("name = %q, want %q", got.Name, unit.Name)
	}
	if got.Description != unit.CoverageArea {
		t.Errorf("description = %q, want %q", got.Description, unit.CoverageArea)
	}
	if got.OperationalRadius != unit.OperationalRadius {
		t.Errorf("operationalRadius = %v, want %v", got.OperationalRadius, unit.OperationalRadius)
	}
	if got.TotalMembers != unit.TotalMembers {
		t.Errorf("totalMembers = %d, want %d", got.TotalMembers, unit.TotalMembers)
	}
	if !got.IsVerified {
		t.Error("isVerified = false, want true")
	}
}

func TestPublicUnits_OmitsEmptyDescription(t *testing.T) {
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	unit := testutil.MakeUnit(t, nil)
	unit.CoverageArea = ""
	if err := testutil.GetDB().Save(unit).Error; err != nil {
		t.Fatalf("save unit: %v", err)
	}

	body := getPublicEndpoint(t, "/api/v1/public/units")
	rawUnits := rawObjects(t, body, "units")
	if len(rawUnits) != 1 {
		t.Fatalf("units count = %d, want 1", len(rawUnits))
	}
	if _, ok := rawUnits[0]["description"]; ok {
		t.Fatalf("description must be omitted when coverageArea is empty: %s", body)
	}
}

func TestPublicCases_ExcludesSensitiveFields(t *testing.T) {
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	admin := testutil.MakeUser(t, "unit_admin")
	reporter := testutil.MakeUser(t, "citizen")
	unit := testutil.MakeUnit(t, admin)
	publicCase := testutil.MakeCase(t, reporter, unit)
	publicCase.Title = "Public Case Title"
	publicCase.Description = "Public Case Description"
	publicCase.Location = "Public Case Location"
	publicCase.Latitude = 9.123
	publicCase.Longitude = 7.456
	publicCase.GISLatitude = 9.124
	publicCase.GISLongitude = 7.457
	publicCase.LocationGeohash = "s1k3m"
	publicCase.Status = "pending"
	publicCase.Priority = "high"
	publicCase.PriorityLevel = "P2"
	publicCase.IncidentDate = time.Now().Add(-time.Hour)
	publicCase.TransferDetails = "TRANSFER-SENTINEL"
	publicCase.FinalReport = "FINAL-REPORT-SENTINEL"
	publicCase.AssignedTo = &admin.ID
	publicCase.ClosedBy = &admin.ID
	publicCase.ApprovedBy = &admin.ID
	publicCase.ClosedViaPlatform = true
	if err := testutil.GetDB().Save(publicCase).Error; err != nil {
		t.Fatalf("save case: %v", err)
	}

	body := getPublicEndpoint(t, "/api/v1/public/cases")
	bodyText := string(body)
	for _, secret := range []string{
		"TRANSFER-SENTINEL",
		"FINAL-REPORT-SENTINEL",
		reporter.ID.String(),
		admin.ID.String(),
	} {
		if strings.Contains(bodyText, secret) {
			t.Errorf("public cases response leaked %q: %s", secret, body)
		}
	}

	rawCases := rawObjects(t, body, "cases")
	if len(rawCases) != 1 {
		t.Fatalf("cases count = %d, want 1", len(rawCases))
	}
	assertKeysAbsent(t, rawCases[0], []string{
		"reportedBy",
		"assignedTo",
		"closedBy",
		"approvedBy",
		"transferDetails",
		"finalReport",
		"closedViaPlatform",
		"incidentDate",
		"priority",
		"assignedAt",
		"dispatchedAt",
		"arrivedAt",
		"closedAt",
		"evidence",
		"progress",
	})
	assertKeysPresent(t, rawCases[0], []string{
		"id",
		"unitId",
		"trackingId",
		"title",
		"description",
		"location",
		"latitude",
		"longitude",
		"gisLatitude",
		"gisLongitude",
		"locationGeohash",
		"status",
		"priorityLevel",
		"isPublic",
		"isAnonymous",
		"createdAt",
		"updatedAt",
	})

	var response struct {
		Cases []handlers.PublicCaseDTO `json:"cases"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("unmarshal cases: %v", err)
	}
	got := response.Cases[0]
	if got.ID != publicCase.ID {
		t.Errorf("id = %s, want %s", got.ID, publicCase.ID)
	}
	if got.UnitID != unit.ID {
		t.Errorf("unitId = %s, want %s", got.UnitID, unit.ID)
	}
	if got.TrackingID != publicCase.TrackingID {
		t.Errorf("trackingId = %q, want %q", got.TrackingID, publicCase.TrackingID)
	}
	if got.Title != publicCase.Title {
		t.Errorf("title = %q, want %q", got.Title, publicCase.Title)
	}
	if got.Description != publicCase.Description {
		t.Errorf("description = %q, want %q", got.Description, publicCase.Description)
	}
	if got.Location != publicCase.Location {
		t.Errorf("location = %q, want %q", got.Location, publicCase.Location)
	}
	if got.PriorityLevel != publicCase.PriorityLevel {
		t.Errorf("priorityLevel = %q, want %q", got.PriorityLevel, publicCase.PriorityLevel)
	}
	if got.Latitude != publicCase.Latitude || got.Longitude != publicCase.Longitude {
		t.Errorf("coordinates = (%v, %v), want (%v, %v)", got.Latitude, got.Longitude, publicCase.Latitude, publicCase.Longitude)
	}
	if got.GISLatitude != publicCase.GISLatitude || got.GISLongitude != publicCase.GISLongitude {
		t.Errorf("GIS coordinates = (%v, %v), want (%v, %v)", got.GISLatitude, got.GISLongitude, publicCase.GISLatitude, publicCase.GISLongitude)
	}
	if !got.IsPublic {
		t.Error("isPublic = false, want true")
	}
}

func TestPublicCases_RedactsAnonymousAndCoarseCoordinates(t *testing.T) {
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	admin := testutil.MakeUser(t, "unit_admin")
	reporter := testutil.MakeUser(t, "citizen")
	unit := testutil.MakeUnit(t, admin)

	anonymous := testutil.MakeCase(t, reporter, unit)
	anonymous.IsAnonymous = true
	anonymous.Latitude = 6.1
	anonymous.Longitude = 5.1
	anonymous.GISLatitude = 6.11
	anonymous.GISLongitude = 5.11
	anonymous.LocationGeohash = "anon-geo"
	if err := testutil.GetDB().Save(anonymous).Error; err != nil {
		t.Fatalf("save anonymous case: %v", err)
	}

	coarse := testutil.MakeCase(t, reporter, unit)
	coarse.Latitude = 0
	coarse.Longitude = 5.2
	coarse.GISLatitude = 6.21
	coarse.GISLongitude = 5.21
	coarse.LocationGeohash = "coarse-geo"
	if err := testutil.GetDB().Save(coarse).Error; err != nil {
		t.Fatalf("save coarse case: %v", err)
	}

	precise := testutil.MakeCase(t, reporter, unit)
	precise.Latitude = 6.3
	precise.Longitude = 5.3
	precise.GISLatitude = 6.31
	precise.GISLongitude = 5.31
	precise.LocationGeohash = "precise-geo"
	if err := testutil.GetDB().Save(precise).Error; err != nil {
		t.Fatalf("save precise case: %v", err)
	}

	body := getPublicEndpoint(t, "/api/v1/public/cases")
	var response struct {
		Cases []handlers.PublicCaseDTO `json:"cases"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("unmarshal cases: %v", err)
	}

	byTrackingID := make(map[string]handlers.PublicCaseDTO, len(response.Cases))
	for _, publicCase := range response.Cases {
		byTrackingID[publicCase.TrackingID] = publicCase
	}

	anonymousOut, ok := byTrackingID[anonymous.TrackingID]
	if !ok {
		t.Fatalf("anonymous case missing from response: %s", body)
	}
	if anonymousOut.Latitude != 0 || anonymousOut.Longitude != 0 ||
		anonymousOut.GISLatitude != 0 || anonymousOut.GISLongitude != 0 {
		t.Errorf("anonymous coordinates were not redacted: %+v", anonymousOut)
	}
	if anonymousOut.LocationGeohash != anonymous.LocationGeohash {
		t.Errorf("anonymous geohash = %q, want %q", anonymousOut.LocationGeohash, anonymous.LocationGeohash)
	}

	coarseOut, ok := byTrackingID[coarse.TrackingID]
	if !ok {
		t.Fatalf("coarse case missing from response: %s", body)
	}
	if coarseOut.Latitude != 0 || coarseOut.Longitude != 0 ||
		coarseOut.GISLatitude != 0 || coarseOut.GISLongitude != 0 {
		t.Errorf("coarse coordinates were not redacted: %+v", coarseOut)
	}
	if coarseOut.LocationGeohash != coarse.LocationGeohash {
		t.Errorf("coarse geohash = %q, want %q", coarseOut.LocationGeohash, coarse.LocationGeohash)
	}

	preciseOut, ok := byTrackingID[precise.TrackingID]
	if !ok {
		t.Fatalf("precise case missing from response: %s", body)
	}
	if preciseOut.Latitude != precise.Latitude || preciseOut.Longitude != precise.Longitude ||
		preciseOut.GISLatitude != precise.GISLatitude || preciseOut.GISLongitude != precise.GISLongitude {
		t.Errorf("precise coordinates were altered: %+v", preciseOut)
	}
}
