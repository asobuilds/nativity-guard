//go:build integration

package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"security-solution/config"
	"security-solution/internal/testutil"
	"security-solution/models"
)

func bookingContext(
	t *testing.T,
	method, target string,
	body map[string]any,
	user *models.User,
	params gin.Params,
) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
	}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(method, target, bytes.NewReader(payload))
	ctx.Request.Header.Set("Content-Type", "application/json")
	if user != nil {
		ctx.Set("user", user)
	}
	if params != nil {
		ctx.Params = params
	}
	return ctx, recorder
}

func bookingIDParams(id uuid.UUID) gin.Params {
	return gin.Params{{Key: "id", Value: id.String()}}
}

func decodeBookingResponse(t *testing.T, recorder *httptest.ResponseRecorder, target any) {
	t.Helper()
	if err := json.Unmarshal(recorder.Body.Bytes(), target); err != nil {
		t.Fatalf("decode response %q: %v", recorder.Body.String(), err)
	}
}

func makeBookingFixture(t *testing.T) (hostUser, memberUser *models.User, hostUnit, memberUnit *models.SecurityUnit, booking *models.BookingRequest) {
	t.Helper()

	hostUser = testutil.MakeUser(t, models.RoleUnitAdmin)
	hostUnit = testutil.MakeUnit(t, hostUser)
	setUserUnit(t, hostUser, hostUnit.ID)
	hostUnit.BaseRate = 50000
	if err := config.DB.Save(hostUnit).Error; err != nil { t.Fatalf("save host unit rate: %v", err) }

	memberUser = testutil.MakeUser(t, models.RoleUnitAdmin)
	memberUnit = testutil.MakeUnit(t, memberUser)
	setUserUnit(t, memberUser, memberUnit.ID)
	memberUnit.BaseRate = 30000
	if err := config.DB.Save(memberUnit).Error; err != nil { t.Fatalf("save member unit rate: %v", err) }

	booking = &models.BookingRequest{
		UserID:        memberUser.ID,
		UnitID:        hostUnit.ID,
		EventType:     "wedding",
		EventDate:     time.Now().UTC().Add(48 * time.Hour),
		EventLocation: "Lagos",
		DurationHours: 4,
		GuardsNeeded:  3,
		Status:        models.BookingStatusPending,
		Notes:         "VIP event",
	}
	if err := config.DB.Create(booking).Error; err != nil {
		t.Fatalf("create booking: %v", err)
	}
	return hostUser, memberUser, hostUnit, memberUnit, booking
}

func TestBookingCreateSetsPendingAndComputesQuotation(t *testing.T) {
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	_, memberUser, hostUnit, _, existingBooking := makeBookingFixture(t)
	if err := config.DB.Unscoped().Delete(&models.BookingRequest{}, "id = ?", existingBooking.ID).Error; err != nil {
		t.Fatalf("remove fixture booking: %v", err)
	}

	ctx, recorder := bookingContext(t, http.MethodPost, "/bookings", map[string]any{
		"unitId":        hostUnit.ID.String(),
		"eventType":     "funeral",
		"eventDate":     time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339),
		"eventLocation": "Abuja",
		"durationHours": 2,
		"guardsNeeded":  2,
	}, memberUser, nil)
	CreateBooking(ctx)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("create booking: want 201, got %d %s", recorder.Code, recorder.Body.String())
	}

	var payload struct {
		Booking models.BookingRequest `json:"booking"`
	}
	decodeBookingResponse(t, recorder, &payload)

	b := payload.Booking
	if b.Status != models.BookingStatusPending {
		t.Fatalf("status = %q, want pending", b.Status)
	}
	if b.EventType != "funeral" {
		t.Fatalf("eventType = %q", b.EventType)
	}
	// Quotation: 2h * 2 guards * 50000 baseRate = 200000
	if b.QuotationAmount != 200000 {
		t.Fatalf("quotationAmount = %v, want 200000", b.QuotationAmount)
	}
	// Platform fee = 5% of 200000 = 10000
	if b.PlatformFee != 10000 {
		t.Fatalf("platformFee = %v, want 10000", b.PlatformFee)
	}
}

func TestBookingListOwnVsUnitBookings(t *testing.T) {
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	hostUser, memberUser, _, _, booking := makeBookingFixture(t)

	ctx, recorder := bookingContext(t, http.MethodGet, "/bookings", nil, memberUser, nil)
	ListBookings(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("list as member: want 200, got %d %s", recorder.Code, recorder.Body.String())
	}
	var memberPayload struct {
		Bookings []models.BookingRequest `json:"bookings"`
	}
	decodeBookingResponse(t, recorder, &memberPayload)
	if len(memberPayload.Bookings) != 1 {
		t.Fatalf("member bookings = %d, want 1", len(memberPayload.Bookings))
	}
	if memberPayload.Bookings[0].ID != booking.ID {
		t.Fatalf("member saw wrong booking")
	}

	ctx, recorder = bookingContext(t, http.MethodGet, "/bookings", nil, hostUser, nil)
	ListBookings(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("list as admin: want 200, got %d %s", recorder.Code, recorder.Body.String())
	}
	var hostPayload struct {
		Bookings []models.BookingRequest `json:"bookings"`
	}
	decodeBookingResponse(t, recorder, &hostPayload)
	if len(hostPayload.Bookings) != 1 {
		t.Fatalf("host bookings = %d, want 1", len(hostPayload.Bookings))
	}
	if hostPayload.Bookings[0].ID != booking.ID {
		t.Fatalf("host saw wrong booking")
	}
}

func TestBookingAcceptGeneratesQuotationWithPlatformFee(t *testing.T) {
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	hostUser, _, _, _, booking := makeBookingFixture(t)

	ctx, recorder := bookingContext(t, http.MethodPost, "/bookings/"+booking.ID.String()+"/accept",
		nil, hostUser, bookingIDParams(booking.ID))
	AcceptBooking(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("accept: want 200, got %d %s", recorder.Code, recorder.Body.String())
	}

	var payload struct {
		Booking models.BookingRequest `json:"booking"`
	}
	decodeBookingResponse(t, recorder, &payload)
	b := payload.Booking
	if b.Status != models.BookingStatusAccepted {
		t.Fatalf("status = %q, want accepted", b.Status)
	}
	// Recomputed quotation: 4h * 3 guards * 50000 = 600000
	if b.QuotationAmount != 600000 {
		t.Fatalf("quotationAmount = %v, want 600000", b.QuotationAmount)
	}
	// Platform fee = 5% of 600000 = 30000
	if b.PlatformFee != 30000 {
		t.Fatalf("platformFee = %v, want 30000", b.PlatformFee)
	}
}

func TestBookingDeclineSetsDeclinedStatus(t *testing.T) {
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	hostUser, _, _, _, booking := makeBookingFixture(t)

	ctx, recorder := bookingContext(t, http.MethodPost, "/bookings/"+booking.ID.String()+"/decline",
		nil, hostUser, bookingIDParams(booking.ID))
	DeclineBooking(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("decline: want 200, got %d %s", recorder.Code, recorder.Body.String())
	}

	var payload struct {
		Booking models.BookingRequest `json:"booking"`
	}
	decodeBookingResponse(t, recorder, &payload)
	if payload.Booking.Status != models.BookingStatusDeclined {
		t.Fatalf("status = %q, want declined", payload.Booking.Status)
	}
}

func TestBookingMarkPaidSetsPaidStatus(t *testing.T) {
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	hostUser, _, _, _, booking := makeBookingFixture(t)

	// First accept
	ctx, recorder := bookingContext(t, http.MethodPost, "/bookings/"+booking.ID.String()+"/accept",
		nil, hostUser, bookingIDParams(booking.ID))
	AcceptBooking(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("accept: %d %s", recorder.Code, recorder.Body.String())
	}

	// Then mark paid
	ctx, recorder = bookingContext(t, http.MethodPost, "/bookings/"+booking.ID.String()+"/mark-paid",
		nil, hostUser, bookingIDParams(booking.ID))
	MarkBookingPaid(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("mark paid: want 200, got %d %s", recorder.Code, recorder.Body.String())
	}

	var payload struct {
		Booking models.BookingRequest `json:"booking"`
	}
	decodeBookingResponse(t, recorder, &payload)
	if payload.Booking.Status != models.BookingStatusPaid {
		t.Fatalf("status = %q, want paid", payload.Booking.Status)
	}
}

func TestBookingUnauthorizedAcceptRejects(t *testing.T) {
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	_, memberUser, _, _, booking := makeBookingFixture(t)

	ctx, recorder := bookingContext(t, http.MethodPost, "/bookings/"+booking.ID.String()+"/accept",
		nil, memberUser, bookingIDParams(booking.ID))
	AcceptBooking(ctx)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("wrong-unit accept: want 403, got %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestBookingDuplicatePendingIsRejected(t *testing.T) {
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	_, memberUser, hostUnit, _, booking := makeBookingFixture(t)

	// Try to create another pending booking for the same unit by the same user
	ctx, recorder := bookingContext(t, http.MethodPost, "/bookings", map[string]any{
		"unitId":        hostUnit.ID.String(),
		"eventType":     "concert",
		"eventDate":     time.Now().UTC().Add(72 * time.Hour).Format(time.RFC3339),
		"eventLocation": "Lagos",
		"durationHours": 2,
		"guardsNeeded":  1,
	}, memberUser, nil)
	CreateBooking(ctx)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("duplicate booking: want 409, got %d %s", recorder.Code, recorder.Body.String())
	}

	// The original booking should still be pending
	var updated models.BookingRequest
	if err := config.DB.First(&updated, "id = ?", booking.ID).Error; err != nil {
		t.Fatalf("reload booking: %v", err)
	}
	if updated.Status != models.BookingStatusPending {
		t.Fatalf("original booking status changed to %q", updated.Status)
	}
}


func TestBookingActionsRequireUnitAdminRoleEvenWithMatchingUnit(t *testing.T) {
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	_, citizen, hostUnit, _, booking := makeBookingFixture(t)
	citizen.Role = models.RoleCitizen
	citizen.UnitID = &hostUnit.ID

	actions := []struct {
		name    string
		handler gin.HandlerFunc
		status  string
	}{
		{name: "accept", handler: AcceptBooking, status: models.BookingStatusPending},
		{name: "decline", handler: DeclineBooking, status: models.BookingStatusPending},
		{name: "mark-paid", handler: MarkBookingPaid, status: models.BookingStatusAccepted},
	}
	for _, action := range actions {
		t.Run(action.name, func(t *testing.T) {
			if action.status == models.BookingStatusAccepted {
				if err := config.DB.Model(&models.BookingRequest{}).Where("id = ?", booking.ID).
					Update("status", models.BookingStatusAccepted).Error; err != nil {
					t.Fatalf("prepare accepted booking: %v", err)
				}
			}
			ctx, recorder := bookingContext(t, http.MethodPost,
				"/bookings/"+booking.ID.String()+"/"+action.name, nil, citizen, bookingIDParams(booking.ID))
			action.handler(ctx)
			if recorder.Code != http.StatusForbidden {
				t.Fatalf("%s: want 403, got %d %s", action.name, recorder.Code, recorder.Body.String())
			}
			var current models.BookingRequest
			if err := config.DB.First(&current, "id = ?", booking.ID).Error; err != nil {
				t.Fatalf("reload booking: %v", err)
			}
			if current.Status != action.status {
				t.Fatalf("%s changed status to %q, want %q", action.name, current.Status, action.status)
			}
		})
	}
}

func TestBookingConcurrentAcceptOnlyTransitionsOnce(t *testing.T) {
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	hostUser, _, _, _, booking := makeBookingFixture(t)
	type result struct {
		status int
		body   string
	}
	results := make(chan result, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, recorder := bookingContext(t, http.MethodPost,
				"/bookings/"+booking.ID.String()+"/accept", nil, hostUser, bookingIDParams(booking.ID))
			AcceptBooking(ctx)
			results <- result{status: recorder.Code, body: recorder.Body.String()}
		}()
	}
	wg.Wait()
	close(results)

	successes, conflicts := 0, 0
	for result := range results {
		switch result.status {
		case http.StatusOK:
			successes++
		case http.StatusConflict:
			conflicts++
		default:
			t.Fatalf("unexpected concurrent accept response: %d %s", result.status, result.body)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("got %d successful and %d conflicting transitions; want exactly one of each", successes, conflicts)
	}
}
