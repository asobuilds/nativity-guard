package handlers

import (
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"security-solution/config"
	"security-solution/models"
	"security-solution/services"
)

const (
	defaultBookingBaseRate = 25000 // NGN per guard per shift
	platformFeePercent     = 0.05
)

var (
	errBookingNotFound      = errors.New("booking not found")
	errBookingNotPending    = errors.New("booking is not pending")
	errBookingAlreadyAction = errors.New("booking already actioned")
	errNotUnitAdmin         = errors.New("only the unit admin can perform this action")
	errUnitNotFound         = errors.New("unit not found")
	errUnitNotActive        = errors.New("unit is not active or verified")
)

type bookingCreateInput struct {
	UnitID        string  `json:"unitId" binding:"required"`
	EventType     string  `json:"eventType" binding:"required"`
	EventDate     string  `json:"eventDate" binding:"required"`
	EventLocation string  `json:"eventLocation" binding:"required"`
	DurationHours float64 `json:"durationHours" binding:"required"`
	GuardsNeeded  int     `json:"guardsNeeded" binding:"required"`
	Notes         string  `json:"notes"`
}

type bookingResponse struct {
	ID             uuid.UUID  `json:"id"`
	UserID         uuid.UUID  `json:"userId"`
	UnitID         uuid.UUID  `json:"unitId"`
	EventType      string     `json:"eventType"`
	EventDate      time.Time  `json:"eventDate"`
	EventLocation  string     `json:"eventLocation"`
	DurationHours  float64    `json:"durationHours"`
	GuardsNeeded   int        `json:"guardsNeeded"`
	Status         string     `json:"status"`
	Notes          string     `json:"notes"`
	QuotationAmount float64   `json:"quotationAmount"`
	PlatformFee     float64   `json:"platformFee"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

func toBookingResponse(b models.BookingRequest) bookingResponse {
	return bookingResponse{
		ID:             b.ID,
		UserID:         b.UserID,
		UnitID:         b.UnitID,
		EventType:      b.EventType,
		EventDate:      b.EventDate,
		EventLocation:  b.EventLocation,
		DurationHours:  b.DurationHours,
		GuardsNeeded:   b.GuardsNeeded,
		Status:         b.Status,
		Notes:          b.Notes,
		QuotationAmount: b.QuotationAmount,
		PlatformFee:     b.PlatformFee,
		CreatedAt:      b.CreatedAt,
		UpdatedAt:      b.UpdatedAt,
	}
}

func baseRateForUnit(unit *models.SecurityUnit) float64 {
	rate := unit.BaseRate
	if rate <= 0 {
		return defaultBookingBaseRate
	}
	return rate
}

func quotationAmount(durationHours float64, guardsNeeded int, baseRate float64) float64 {
	return durationHours * float64(guardsNeeded) * baseRate
}

func CreateBooking(c *gin.Context) {
	user, ok := getAuthUser(c)
	if !ok {
		return
	}

	var input bookingCreateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.DurationHours <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "durationHours must be greater than zero"})
		return
	}
	if input.GuardsNeeded <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "guardsNeeded must be greater than zero"})
		return
	}

	eventDate, err := time.Parse(time.RFC3339, input.EventDate)
	if err != nil {
		eventDate, err = time.Parse("2006-01-02", input.EventDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid eventDate (use RFC3339 or YYYY-MM-DD)"})
			return
		}
	}
	eventDate = eventDate.UTC()

	if time.Now().UTC().After(eventDate) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "eventDate must be in the future"})
		return
	}

	var unit models.SecurityUnit
	if err := config.DB.First(&unit, "id = ?", input.UnitID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": errUnitNotFound.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load unit"})
		return
	}
	if unit.Status != "active" || !unit.IsVerified {
		c.JSON(http.StatusBadRequest, gin.H{"error": errUnitNotActive.Error()})
		return
	}

	baseRate := baseRateForUnit(&unit)
	amount := quotationAmount(input.DurationHours, input.GuardsNeeded, baseRate)
	platformFee := amount * platformFeePercent

	booking := models.BookingRequest{
		UserID:         user.ID,
		UnitID:         unit.ID,
		EventType:      strings.TrimSpace(input.EventType),
		EventDate:      eventDate,
		EventLocation:  strings.TrimSpace(input.EventLocation),
		DurationHours:  input.DurationHours,
		GuardsNeeded:   input.GuardsNeeded,
		Status:         models.BookingStatusPending,
		Notes:          strings.TrimSpace(input.Notes),
		QuotationAmount: amount,
		PlatformFee:     platformFee,
	}

	if err := config.DB.Create(&booking).Error; err != nil {
		if isDuplicatePendingBooking(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "You already have a pending booking for this unit"})
			return
		}
		log.Printf("booking: create failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create booking"})
		return
	}

	_ = services.NewAuditService().LogAction(
		user.ID,
		"booking.created",
		"booking_request",
		booking.ID.String(),
		nil,
		map[string]any{"unitId": unit.ID.String(), "eventType": booking.EventType, "quotationAmount": booking.QuotationAmount},
		c.ClientIP(),
		c.Request.UserAgent(),
	)

	c.JSON(http.StatusCreated, gin.H{
		"message": "Booking request created",
		"booking": toBookingResponse(booking),
	})
}

func ListBookings(c *gin.Context) {
	user, ok := getAuthUser(c)
	if !ok {
		return
	}

	var bookings []models.BookingRequest
	query := config.DB

	isUnitAdmin := user.Role == models.RoleUnitAdmin || user.IsSuperAdmin
	if isUnitAdmin && user.UnitID != nil {
		// Unit admin sees all bookings for their unit
		query = query.Where("unit_id = ?", *user.UnitID)
	} else {
		// Regular user sees only their own bookings
		query = query.Where("user_id = ?", user.ID)
	}

	if err := query.Order("created_at desc").Find(&bookings).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load bookings"})
		return
	}

	responses := make([]bookingResponse, 0, len(bookings))
	for _, b := range bookings {
		responses = append(responses, toBookingResponse(b))
	}

	c.JSON(http.StatusOK, gin.H{
		"bookings": responses,
	})
}

// isDuplicatePendingBooking handles both GORM-translated duplicate errors and
// PostgreSQL's native unique-violation error. The constraint name check avoids
// misreporting unrelated uniqueness failures as duplicate active bookings.
func isDuplicatePendingBooking(err error) bool {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) &&
		pgErr.Code == "23505" &&
		pgErr.ConstraintName == "idx_booking_requests_active_pending"
}

func bookingActionError(c *gin.Context, err error, action string) {
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound), errors.Is(err, errBookingNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": errBookingNotFound.Error()})
	case errors.Is(err, errNotUnitAdmin):
		c.JSON(http.StatusForbidden, gin.H{"error": errNotUnitAdmin.Error()})
	case errors.Is(err, errBookingNotPending):
		c.JSON(http.StatusConflict, gin.H{"error": errBookingNotPending.Error()})
	case errors.Is(err, errBookingNotAccepted):
		c.JSON(http.StatusConflict, gin.H{"error": errBookingNotAccepted.Error()})
	default:
		log.Printf("booking: %s failed: %v", action, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to " + action + " booking"})
	}
}

var errBookingNotAccepted = errors.New("booking must be accepted before it can be marked paid")

// updateBookingStatus serializes state transitions. The row lock is acquired
// and held inside the same transaction that validates permissions/status and
// writes the new state, preventing concurrent requests from acting on stale data.
func updateBookingStatus(user *models.User, bookingID uuid.UUID, expectedStatus, nextStatus string, reprice bool) (models.BookingRequest, error) {
	var booking models.BookingRequest
	err := config.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&booking, "id = ?", bookingID).Error; err != nil {
			return err
		}

		isSuperAdmin := user.IsSuperAdmin || user.Role == models.RoleSuperAdmin
		isBookingUnitAdmin := user.Role == models.RoleUnitAdmin &&
			user.UnitID != nil && *user.UnitID == booking.UnitID
		if !isSuperAdmin && !isBookingUnitAdmin {
			return errNotUnitAdmin
		}
		if booking.Status != expectedStatus {
			if expectedStatus == models.BookingStatusAccepted {
				return errBookingNotAccepted
			}
			return errBookingNotPending
		}

		updates := map[string]any{
			"status":     nextStatus,
			"updated_at": time.Now().UTC(),
		}
		if reprice {
			var unit models.SecurityUnit
			if err := tx.First(&unit, "id = ?", booking.UnitID).Error; err != nil {
				return err
			}
			amount := quotationAmount(booking.DurationHours, booking.GuardsNeeded, baseRateForUnit(&unit))
			fee := amount * platformFeePercent
			updates["quotation_amount"] = amount
			updates["platform_fee"] = fee
			booking.QuotationAmount = amount
			booking.PlatformFee = fee
		}

		result := tx.Model(&models.BookingRequest{}).
			Where("id = ? AND status = ?", booking.ID, expectedStatus).
			Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errBookingAlreadyAction
		}
		booking.Status = nextStatus
		booking.UpdatedAt = updates["updated_at"].(time.Time)
		return nil
	})
	return booking, err
}

func AcceptBooking(c *gin.Context) {
	user, ok := getAuthUser(c)
	if !ok {
		return
	}
	bookingID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid booking ID"})
		return
	}

	booking, err := updateBookingStatus(user, bookingID, models.BookingStatusPending, models.BookingStatusAccepted, true)
	if err != nil {
		bookingActionError(c, err, "accept")
		return
	}
	_ = services.NewAuditService().LogAction(
		user.ID, "booking.accepted", "booking_request", booking.ID.String(),
		map[string]any{"status": models.BookingStatusPending},
		map[string]any{"status": models.BookingStatusAccepted, "quotationAmount": booking.QuotationAmount, "platformFee": booking.PlatformFee},
		c.ClientIP(), c.Request.UserAgent(),
	)
	c.JSON(http.StatusOK, gin.H{"message": "Booking accepted", "booking": toBookingResponse(booking)})
}

func DeclineBooking(c *gin.Context) {
	user, ok := getAuthUser(c)
	if !ok {
		return
	}
	bookingID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid booking ID"})
		return
	}

	booking, err := updateBookingStatus(user, bookingID, models.BookingStatusPending, models.BookingStatusDeclined, false)
	if err != nil {
		bookingActionError(c, err, "decline")
		return
	}
	_ = services.NewAuditService().LogAction(
		user.ID, "booking.declined", "booking_request", booking.ID.String(),
		map[string]any{"status": models.BookingStatusPending},
		map[string]any{"status": models.BookingStatusDeclined},
		c.ClientIP(), c.Request.UserAgent(),
	)
	c.JSON(http.StatusOK, gin.H{"message": "Booking declined", "booking": toBookingResponse(booking)})
}

func MarkBookingPaid(c *gin.Context) {
	user, ok := getAuthUser(c)
	if !ok {
		return
	}
	bookingID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid booking ID"})
		return
	}

	booking, err := updateBookingStatus(user, bookingID, models.BookingStatusAccepted, models.BookingStatusPaid, false)
	if err != nil {
		bookingActionError(c, err, "mark paid")
		return
	}
	_ = services.NewAuditService().LogAction(
		user.ID, "booking.marked_paid", "booking_request", booking.ID.String(),
		map[string]any{"status": models.BookingStatusAccepted},
		map[string]any{"status": models.BookingStatusPaid},
		c.ClientIP(), c.Request.UserAgent(),
	)
	c.JSON(http.StatusOK, gin.H{"message": "Booking marked as paid", "booking": toBookingResponse(booking)})
}
