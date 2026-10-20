//go:build integration

package services_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"security-solution/config"
	"security-solution/internal/testutil"
	"security-solution/models"
	"security-solution/services"
)

func TestUpdateEventStatuses_MarksPastEventsCompleted(t *testing.T) {
	s := services.NewSchedulerService()

	// Clear all events first
	testutil.TruncateAll(t)

	// Create a user and unit as author for events
	author := testutil.MakeUser(t, "unit_admin")
	unit := testutil.MakeUnit(t, author)

	pastEvent := models.CommunityEvent{
		ID:        uuid.New(),
		UnitID:    &unit.ID,
		Status:    services.EventStatusUpcoming,
		EventDate: time.Now().UTC().Add(-2 * 24 * time.Hour),
		EndDate:   ptrTime(time.Now().UTC().Add(-24 * time.Hour)),
		AuthorID:  author.ID,
	}
	if err := config.DB.Create(&pastEvent).Error; err != nil {
		t.Fatalf("create past event: %v", err)
	}

	pastEventNoEnd := models.CommunityEvent{
		ID:        uuid.New(),
		UnitID:    &unit.ID,
		Status:    services.EventStatusUpcoming,
		EventDate: time.Now().UTC().Add(-2 * 24 * time.Hour),
		EndDate:   nil,
		AuthorID:  author.ID,
	}
	if err := config.DB.Create(&pastEventNoEnd).Error; err != nil {
		t.Fatalf("create past event (no end): %v", err)
	}

	futureEvent := models.CommunityEvent{
		ID:        uuid.New(),
		UnitID:    &unit.ID,
		Status:    services.EventStatusUpcoming,
		EventDate: time.Now().UTC().Add(48 * time.Hour),
		EndDate:   ptrTime(time.Now().UTC().Add(49 * time.Hour)),
		AuthorID:  author.ID,
	}
	if err := config.DB.Create(&futureEvent).Error; err != nil {
		t.Fatalf("create future event: %v", err)
	}

	preparingEvent := models.CommunityEvent{
		ID:        uuid.New(),
		UnitID:    &unit.ID,
		Status:    services.EventStatusUpcoming,
		EventDate: time.Now().UTC().Add(-1 * time.Hour), // hasn't ended yet
		EndDate:   ptrTime(time.Now().UTC().Add(23 * time.Hour)),
		AuthorID:  author.ID,
	}
	if err := config.DB.Create(&preparingEvent).Error; err != nil {
		t.Fatalf("create preparing event: %v", err)
	}

	// Before update
	var beforeCount int64
	config.DB.Model(&models.CommunityEvent{}).Where("status = ?", services.EventStatusCompleted).Count(&beforeCount)
	if beforeCount != 0 {
		t.Fatalf("expected 0 completed events before update, got %d", beforeCount)
	}

	s.UpdateEventStatuses()

	// After update: pastEvent and pastEventNoEnd should be completed; future and preparing stay upcoming
	var completed []models.CommunityEvent
	if err := config.DB.Where("status = ?", services.EventStatusCompleted).Find(&completed).Error; err != nil {
		t.Fatalf("load completed events: %v", err)
	}
	if len(completed) != 2 {
		t.Errorf("expected 2 completed events, got %d", len(completed))
	}
	completedIDs := make(map[uuid.UUID]bool)
	for _, e := range completed {
		completedIDs[e.ID] = true
	}
	if !completedIDs[pastEvent.ID] {
		t.Error("past event should be completed")
	}
	if !completedIDs[pastEventNoEnd.ID] {
		t.Error("past event (no end date) should be completed")
	}

	var stillUpcoming []models.CommunityEvent
	if err := config.DB.Where("status = ?", services.EventStatusUpcoming).Find(&stillUpcoming).Error; err != nil {
		t.Fatalf("load upcoming events: %v", err)
	}
	if len(stillUpcoming) != 2 {
		t.Errorf("expected 2 events still upcoming, got %d", len(stillUpcoming))
	}
}

func TestSendEventReminders_SendsOnlyOncePerAttendee(t *testing.T) {
	s := services.NewSchedulerService()

	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	// Find a valid unit ID for the event
	author := testutil.MakeUser(t, "unit_admin")
	unit := testutil.MakeUnit(t, author)

	// Create an event starting in 12 hours
	eventDate := time.Now().UTC().Add(12 * time.Hour)
	event := models.CommunityEvent{
		ID:        uuid.New(),
		UnitID:    &unit.ID,
		Status:    services.EventStatusUpcoming,
		EventDate: eventDate,
		AuthorID:  author.ID,
		Title:     "Test Event",
		Location:  "Test Location",
	}
	if err := config.DB.Create(&event).Error; err != nil {
		t.Fatalf("create event: %v", err)
	}

	// Create attendees
	attendee1 := testutil.MakeUser(t, "citizen")
	attendee2 := testutil.MakeUser(t, "citizen")
	for _, a := range []*models.User{attendee1, attendee2} {
		if err := config.DB.Create(&models.EventAttendee{
			EventID: event.ID,
			UserID:  a.ID,
			Status:  "registered",
		}).Error; err != nil {
			t.Fatalf("create attendee: %v", err)
		}
	}

	// First run - should create 2 notifications
	s.SendEventReminders()

	var count int64
	config.DB.Model(&models.Notification{}).
		Where("type = ? AND entity_id = ?", services.EventReminderType, event.ID).Count(&count)
	if count != 2 {
		t.Fatalf("expected 2 notifications after first run, got %d", count)
	}

	// Second run - should be idempotent (no new notifications)
	s.SendEventReminders()

	config.DB.Model(&models.Notification{}).
		Where("type = ? AND entity_id = ?", services.EventReminderType, event.ID).Count(&count)
	if count != 2 {
		t.Errorf("expected 2 notifications after second run (idempotent), got %d", count)
	}
}

func TestSendEventReminders_NoReminderForEventsOutside24hWindow(t *testing.T) {
	s := services.NewSchedulerService()

	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	author := testutil.MakeUser(t, "unit_admin")
	unit := testutil.MakeUnit(t, author)

	// Event starting in 48 hours - outside the 24h window
	event := models.CommunityEvent{
		ID:        uuid.New(),
		UnitID:    &unit.ID,
		Status:    services.EventStatusUpcoming,
		EventDate: time.Now().UTC().Add(48 * time.Hour),
		AuthorID:  author.ID,
		Title:     "Far Future Event",
	}
	if err := config.DB.Create(&event).Error; err != nil {
		t.Fatalf("create event: %v", err)
	}

	attendee := testutil.MakeUser(t, "citizen")
	if err := config.DB.Create(&models.EventAttendee{
		EventID: event.ID,
		UserID:  attendee.ID,
		Status:  "registered",
	}).Error; err != nil {
		t.Fatalf("create attendee: %v", err)
	}

	s.SendEventReminders()

	var count int64
	config.DB.Model(&models.Notification{}).
		Where("type = ? AND entity_id = ?", services.EventReminderType, event.ID).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 notifications for event outside 24h window, got %d", count)
	}
}

func TestSendEventReminders_NoReminderForCompletedEvents(t *testing.T) {
	s := services.NewSchedulerService()

	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	author := testutil.MakeUser(t, "unit_admin")
	unit := testutil.MakeUnit(t, author)

	// Event that has already passed its end time
	event := models.CommunityEvent{
		ID:        uuid.New(),
		UnitID:    &unit.ID,
		Status:    services.EventStatusCompleted,
		EventDate: time.Now().UTC().Add(-48 * time.Hour),
		EndDate:   ptrTime(time.Now().UTC().Add(-24 * time.Hour)),
		AuthorID:  author.ID,
		Title:     "Past Event",
	}
	if err := config.DB.Create(&event).Error; err != nil {
		t.Fatalf("create event: %v", err)
	}

	attendee := testutil.MakeUser(t, "citizen")
	if err := config.DB.Create(&models.EventAttendee{
		EventID: event.ID,
		UserID:  attendee.ID,
		Status:  "registered",
	}).Error; err != nil {
		t.Fatalf("create attendee: %v", err)
	}

	s.SendEventReminders()

	var count int64
	config.DB.Model(&models.Notification{}).
		Where("type = ? AND entity_id = ?", services.EventReminderType, event.ID).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 notifications for completed event, got %d", count)
	}
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
