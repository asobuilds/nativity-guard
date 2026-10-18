//go:build integration

package handlers

import (
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

// RES-004 proof tests. These run only with `-tags integration` and a
// TEST_DATABASE_URL; they exercise the real Postgres row lock and the partial
// unique index idx_event_attendee_active.

// rsvpContext builds the gin.Context that routes.go hands to RSVPToEvent for
// POST /community/events/:id/rsvp (event id in Params, acting user in context).
func rsvpContext(eventID uuid.UUID, user *models.User) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest("POST", "/community/events/"+eventID.String()+"/rsvp", nil)
	ctx.Params = gin.Params{{Key: "id", Value: eventID.String()}}
	ctx.Set("user", user)
	return ctx, recorder
}

func makeRSVPEvent(t *testing.T, author *models.User, maxAttendees int) *models.CommunityEvent {
	t.Helper()
	event := &models.CommunityEvent{
		Title:        "Safety briefing",
		Description:  "Community safety meeting",
		Location:     "HQ",
		EventDate:    time.Now().UTC().Add(48 * time.Hour),
		Type:         "meeting",
		Status:       "upcoming",
		MaxAttendees: maxAttendees,
		AuthorID:     author.ID,
	}
	if err := config.DB.Create(event).Error; err != nil {
		t.Fatalf("create event: %v", err)
	}
	return event
}

// TestRSVPConcurrentRespectsCapacity fires more RSVPs than there are seats, at
// the same instant. Exactly `capacity` must succeed, the rest must be rejected
// as full, the attendee rows must equal the capacity, and the denormalised
// counter must not drift.
func TestRSVPConcurrentRespectsCapacity(t *testing.T) {
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	const (
		capacity = 5
		callers  = 40
	)

	author := testutil.MakeUser(t, "unit_admin")
	event := makeRSVPEvent(t, author, capacity)

	users := make([]*models.User, callers)
	for i := range users {
		users[i] = testutil.MakeUser(t, "citizen")
	}

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		ok      int
		full    int
		other   int
		startCh = make(chan struct{})
	)

	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func(u *models.User) {
			defer wg.Done()
			<-startCh // barrier: all callers contend at once
			ctx, rec := rsvpContext(event.ID, u)
			RSVPToEvent(ctx)
			mu.Lock()
			switch rec.Code {
			case 200:
				ok++
			case 400:
				full++
			default:
				other++
			}
			mu.Unlock()
		}(users[i])
	}

	close(startCh)
	wg.Wait()

	if other != 0 {
		t.Fatalf("unexpected non-200/400 responses: %d", other)
	}
	if ok != capacity {
		t.Fatalf("capacity race: expected %d successful RSVPs, got %d (full=%d)", capacity, ok, full)
	}
	if full != callers-capacity {
		t.Fatalf("expected %d 'event full' rejections, got %d", callers-capacity, full)
	}

	var rows int64
	if err := config.DB.Model(&models.EventAttendee{}).
		Where("event_id = ?", event.ID).
		Count(&rows).Error; err != nil {
		t.Fatalf("count attendees: %v", err)
	}
	if rows != capacity {
		t.Fatalf("expected %d attendee rows, got %d", capacity, rows)
	}

	var stored models.CommunityEvent
	if err := config.DB.First(&stored, "id = ?", event.ID).Error; err != nil {
		t.Fatalf("reload event: %v", err)
	}
	if stored.AttendeeCount != capacity {
		t.Fatalf("denormalised attendee_count drifted: expected %d, got %d", capacity, stored.AttendeeCount)
	}
}

// TestRSVPConcurrentDuplicateUserSingleRow has one user RSVP many times at
// once. Exactly one insert may survive; the partial unique index and the row
// lock must both hold, and the counter must stay at one.
func TestRSVPConcurrentDuplicateUserSingleRow(t *testing.T) {
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	const attempts = 30

	author := testutil.MakeUser(t, "unit_admin")
	event := makeRSVPEvent(t, author, 0) // 0 == unlimited capacity
	user := testutil.MakeUser(t, "citizen")

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		ok      int
		dupe    int
		other   int
		startCh = make(chan struct{})
	)

	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-startCh
			ctx, rec := rsvpContext(event.ID, user)
			RSVPToEvent(ctx)
			mu.Lock()
			switch rec.Code {
			case 200:
				ok++
			case 400:
				dupe++
			default:
				other++
			}
			mu.Unlock()
		}()
	}

	close(startCh)
	wg.Wait()

	if other != 0 {
		t.Fatalf("unexpected non-200/400 responses: %d", other)
	}
	if ok != 1 {
		t.Fatalf("expected exactly 1 successful RSVP for a single user, got %d (dupe=%d)", ok, dupe)
	}
	if dupe != attempts-1 {
		t.Fatalf("expected %d duplicate rejections, got %d", attempts-1, dupe)
	}

	var rows int64
	if err := config.DB.Model(&models.EventAttendee{}).
		Where("event_id = ? AND user_id = ?", event.ID, user.ID).
		Count(&rows).Error; err != nil {
		t.Fatalf("count attendees: %v", err)
	}
	if rows != 1 {
		t.Fatalf("expected exactly 1 attendee row, got %d", rows)
	}

	var stored models.CommunityEvent
	if err := config.DB.First(&stored, "id = ?", event.ID).Error; err != nil {
		t.Fatalf("reload event: %v", err)
	}
	if stored.AttendeeCount != 1 {
		t.Fatalf("expected attendee_count 1, got %d", stored.AttendeeCount)
	}
}
