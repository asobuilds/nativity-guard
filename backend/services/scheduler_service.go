package services

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"log"
	"sync/atomic"
	"time"

	"security-solution/config"
	"security-solution/models"
)

// schedulerLastRunNanos records the wall-clock time of the most recent
// successful RunOnce completion. Health checks read it to determine
// whether the scheduler is stale.
var schedulerLastRunNanos atomic.Int64

// SchedulerLastRunAt returns the last successful RunOnce completion time.
// Returns the zero time if the scheduler has never completed a run.
func SchedulerLastRunAt() time.Time {
	ns := schedulerLastRunNanos.Load()
	if ns == 0 {
		return time.Time{}
	}
	return time.Unix(0, ns)
}

type SchedulerService struct {
	Election   *ElectionService
	Revocation *RevocationService
}

// Event status constants for CommunityEvent.
const (
	EventStatusUpcoming  = "upcoming"
	EventStatusCompleted = "completed"
	EventReminderType    = "event_reminder"
)

func NewSchedulerService() *SchedulerService {
	return &SchedulerService{
		Election:   NewElectionService(),
		Revocation: NewRevocationService(),
	}
}

// RunOnce acquires a Postgres advisory lock so only one backend instance
// runs scheduled jobs at a time, then performs one pass of all jobs.
// If the lock is held elsewhere, this call is a no-op.
func (s *SchedulerService) RunOnce() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	sqlDB, err := config.DB.DB()
	if err != nil {
		log.Printf("scheduler: cannot get sql.DB: %v", err)
		return
	}

	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		log.Printf("scheduler: cannot acquire connection: %v", err)
		return
	}
	defer conn.Close()

	// Advisory lock ID: any fixed int64. Pick one and never change it.
	const schedulerLockID int64 = 8420194710293847

	var got bool
	if err := conn.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", schedulerLockID).Scan(&got); err != nil {
		log.Printf("scheduler: advisory lock query failed: %v", err)
		return
	}
	if !got {
		// Another instance is running the scheduler right now.
		return
	}
	defer func() {
		_, _ = conn.ExecContext(context.Background(), "SELECT pg_advisory_unlock($1)", schedulerLockID)
	}()

	s.CloseExpiredElections()
	s.ExpireStaleInvites()
	s.UpdateEventStatuses()
	s.SendEventReminders()
	s.CleanupExpiredSessions()
	s.CloseFinishedFinancialYears()
	s.RecomputeRankings()
	s.AnnounceOfficerOfTheWeek()
	s.EscalateStaleAppeals()
	s.RecomputeMinorStatuses()
	s.PurgeDeletedAccounts()
	s.ExpireLocalIntelligence()
	s.EscalateStaleSOS()

	// Record the last successful completion so the health check can
	// detect a stalled scheduler.
	schedulerLastRunNanos.Store(time.Now().UnixNano())
}

// CloseExpiredElections finalizes any open UnitAdminElection whose voting window has passed.
func (s *SchedulerService) CloseExpiredElections() {
	now := time.Now().UTC()

	var elections []models.UnitAdminElection
	if err := config.DB.
		Where("status = ? AND voting_ends_at IS NOT NULL AND voting_ends_at <= ?", "open", now).
		Find(&elections).Error; err != nil {
		log.Printf("scheduler: failed to load expired elections: %v", err)
		return
	}

	for _, e := range elections {
		var err error
		if e.ElectionType == "head_admin" {
			err = s.Election.FinalizeHeadAdminElection(e.ID)
		} else {
			err = s.Election.CloseAdminElection(e.ID)
		}
		if err != nil {
			log.Printf("scheduler: failed to close election %s: %v", e.ID, err)
		}
	}
}

// ExpireStaleInvites marks pending invites as expired once their expiry date passes.
func (s *SchedulerService) ExpireStaleInvites() {
	now := time.Now().UTC()

	if err := config.DB.
		Model(&models.UnitInvite{}).
		Where("status = ? AND expires_at IS NOT NULL AND expires_at <= ?", "pending", now).
		Update("status", "expired").Error; err != nil {
		log.Printf("scheduler: failed to expire invites: %v", err)
	}
}

// Start runs the scheduler loop every interval until stopped. Call from main.
func (s *SchedulerService) Start(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			s.RunOnce()
		}
	}()
}

// FetchLocalNewsJob pulls the configured Nigerian news feeds into the
// local_news_articles table.
func (s *SchedulerService) FetchLocalNewsJob() {
	if err := FetchNigerianNews(); err != nil {
		log.Printf("scheduler: local news fetch failed: %v", err)
	}
}

// StartLocalNewsFetcher runs the local news job on its own cadence so it does
// not change the interval used by the existing scheduler jobs.
func (s *SchedulerService) StartLocalNewsFetcher(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			s.FetchLocalNewsJob()
		}
	}()
}

func (s *SchedulerService) CleanupExpiredSessions() {
	if err := NewSessionService().CleanupExpired(); err != nil {
		log.Printf("scheduler: session cleanup failed: %v", err)
	}
	if err := NewRefreshTokenService().CleanupExpired(); err != nil {
		log.Printf("scheduler: refresh-token cleanup failed: %v", err)
	}
	if err := NewTokenService().CleanupExpired(); err != nil {
		log.Printf("scheduler: token cleanup failed: %v", err)
	}
	_ = config.DB.Where("expires_at < ?", time.Now().UTC()).Delete(&models.IdempotencyRecord{}).Error
}

// CloseFinishedFinancialYears closes the previous calendar year for every
// unit that has ledger activity but no UnitFinancialYear record yet.
// Runs daily; idempotent — running twice in one day is harmless.
func (s *SchedulerService) CloseFinishedFinancialYears() {
	currentYear := time.Now().UTC().Year()
	prevYear := currentYear - 1

	type unitRow struct {
		UnitID uuid.UUID
	}
	var units []unitRow
	config.DB.Model(&models.FinancialLedger{}).
		Select("DISTINCT unit_id").
		Where("year = ?", prevYear).
		Scan(&units)

	ledgerSvc := NewLedgerService()
	for _, u := range units {
		var existing models.UnitFinancialYear
		if err := config.DB.Where("unit_id = ? AND year = ?", u.UnitID, prevYear).
			First(&existing).Error; err == nil {
			continue
		}
		_, _ = ledgerSvc.CloseYear(u.UnitID, prevYear, uuid.Nil)
	}
}

// RecomputeRankings refreshes officer and unit ranking scores from current ratings.
func (s *SchedulerService) RecomputeRankings() {
	if err := NewRankingService().RecomputeAll(); err != nil {
		log.Printf("scheduler: ranking recompute failed: %v", err)
	}
}

// EscalateStaleAppeals flags unresolved appeals past the 30-day threshold and
// notifies all super admins. Delegates to the appeal service.
func (s *SchedulerService) EscalateStaleAppeals() {
	NewAppealService().EscalateStaleAppeals()
}

// EscalateStaleSOS escalates SOS alerts that have been pending for more than
// 3 minutes and notifies the LGA head admin plus super admins.
func (s *SchedulerService) EscalateStaleSOS() {
	threshold := time.Now().UTC().Add(-3 * time.Minute)

	var sos []models.SOSAlert
	if err := config.DB.
		Where("status = ? AND created_at < ?", "pending", threshold).
		Find(&sos).Error; err != nil {
		log.Printf("scheduler: failed to load stale SOS alerts: %v", err)
		return
	}

	if len(sos) == 0 {
		return
	}

	for _, sos := range sos {
		// Mark the alert as escalated and bump priority to urgent.
		if err := config.DB.Model(&models.SOSAlert{}).
			Where("id = ?", sos.ID).
			Updates(map[string]interface{}{
				"status":   "escalated",
				"priority": "urgent",
			}).Error; err != nil {
			log.Printf("scheduler: failed to escalate SOS %s: %v", sos.ID, err)
			continue
		}

		notifTitle := "🚨 SOS Escalated"
		notifMsg := fmt.Sprintf("SOS %s escalated to urgent — %s/%s",
			sos.ID.String(), sos.JurisdictionLGA, sos.JurisdictionState)

		// Notify LGA head admins of units matching the SOS jurisdiction.
		if sos.JurisdictionState != "" && sos.JurisdictionLGA != "" {
			var units []models.SecurityUnit
			config.DB.Where("status = ? AND state = ? AND lga = ?",
				"active", sos.JurisdictionState, sos.JurisdictionLGA).
				Find(&units)

			for _, u := range units {
				var heads []models.User
				config.DB.Where("unit_id = ? AND role = ? AND is_head_admin = ?",
					u.ID, models.UnitRoleAdmin, true).
					Find(&heads)
				for _, h := range heads {
					config.DB.Create(&models.Notification{
						UserID:     h.ID,
						Title:      notifTitle,
						Message:    notifMsg,
						Type:       "sos_escalation",
						Status:     "unread",
						EntityType: "sos",
						EntityID:   sos.ID,
						LinkTo:     "/sos/" + sos.ID.String(),
					})
					if h.Phone != "" {
						smsText := fmt.Sprintf("URGENT SOS in %s: %s. View: https://nativity-guard.vercel.app/sos/%s", sos.JurisdictionLGA, sos.Description, sos.ID.String())
						go SendSMS(h.Phone, smsText)
					}
				}
			}
		}

		// Also notify every super admin.
		var admins []models.User
		config.DB.Where("role = ?", "super_admin").Find(&admins)
		for _, a := range admins {
			config.DB.Create(&models.Notification{
				UserID:     a.ID,
				Title:      notifTitle,
				Message:    notifMsg,
				Type:       "sos_escalation",
				Status:     "unread",
				EntityType: "sos",
				EntityID:   sos.ID,
				LinkTo:     "/sos/" + sos.ID.String(),
			})
		}
	}
}

// AnnounceOfficerOfTheWeek picks the top-rated officer of the past 7 days
// and creates a news post. Runs at most once per week — guarded by checking
// whether an OOW post already exists in the last 7 days.
// Only fires on Mondays after 09:00 UTC.
func (s *SchedulerService) AnnounceOfficerOfTheWeek() {
	now := time.Now().UTC()
	if now.Weekday() != time.Monday || now.Hour() < 9 {
		return
	}

	// Idempotency guard — has an OOW post gone out in the last 7 days?
	var recentCount int64
	config.DB.Model(&models.News{}).
		Where("category = ? AND created_at > ?", "officer_of_the_week", now.AddDate(0, 0, -7)).
		Count(&recentCount)
	if recentCount > 0 {
		return
	}

	// Count ratings for each officer in the last 7 days; require >= 3 new.
	type row struct {
		TargetID uuid.UUID
		Sum      float64
		Count    int64
	}
	var rows []row
	config.DB.Model(&models.Rating{}).
		Select("target_id, COALESCE(SUM(rating), 0) as sum, COUNT(*) as count").
		Where("target_type = ? AND status = ? AND created_at > ?",
			"officer", "active", now.AddDate(0, 0, -7)).
		Group("target_id").
		Having("COUNT(*) >= 3").
		Scan(&rows)

	if len(rows) == 0 {
		return
	}

	// Pick the highest weekly Bayesian score
	var winner row
	var bestScore float64 = -1
	for _, r := range rows {
		bayes := BayesianScore(r.Sum, r.Count)
		if bayes > bestScore {
			bestScore = bayes
			winner = r
		}
	}

	var officer models.Officer
	if err := config.DB.First(&officer, "id = ?", winner.TargetID).Error; err != nil {
		return
	}

	var unit models.SecurityUnit
	config.DB.First(&unit, "id = ?", officer.UnitID)

	title := "Officer of the Week: " + officer.Name
	content := fmt.Sprintf(
		"Congratulations to %s of %s.\n\nWeekly rating: %.2f★ from %d verified ratings this week.\n\nThank you for your service.",
		officer.Name, unit.Name, bestScore, winner.Count,
	)

	news := models.News{
		Title:       title,
		Content:     content,
		Source:      "Nativity Guard",
		Author:      "Nativity Guard",
		Category:    "officer_of_the_week",
		Location:    unit.Name,
		Sentiment:   "positive",
		ThreatLevel: "low",
		Status:      "published",
		PublishedAt: now,
		CreatedBy:   uuid.Nil,
	}
	config.DB.Create(&news)

	// Notify the officer
	notification := models.Notification{
		UserID:  officer.ID,
		Title:   "You are Officer of the Week!",
		Message: fmt.Sprintf("Recognized platform-wide for %d verified ratings this week (%.2f★).", winner.Count, bestScore),
		Type:    "officer_of_the_week",
		Status:  "unread",
	}
	config.DB.Create(&notification)
}

// RecomputeMinorStatuses rolls over minors who have aged out (and adults who
// somehow drifted into "minor") based on the stored DateOfBirth. Skipped for
// users with a super-admin-granted MinorExceptionGranted to avoid surprises.
func (s *SchedulerService) RecomputeMinorStatuses() {
	NewAgeService().RecomputeMinorStatuses()
}

// ExpireLocalIntelligence hard-deletes facts whose TTL window has closed,
// along with their upvotes. LocalIntelligence is soft-deleted by default, so
// Unscoped() is required to actually remove the row. Idempotent.
func (s *SchedulerService) ExpireLocalIntelligence() {
	now := time.Now().UTC()

	expired := config.DB.Model(&models.LocalIntelligence{}).Select("id").
		Where("expires_at IS NOT NULL AND expires_at <= ?", now)
	if err := config.DB.
		Where("intel_id IN (?)", expired).
		Delete(&models.LocalIntelligenceVote{}).Error; err != nil {
		log.Printf("scheduler: failed to delete expired intelligence votes: %v", err)
	}

	if err := config.DB.
		Unscoped().
		Where("expires_at IS NOT NULL AND expires_at <= ?", now).
		Delete(&models.LocalIntelligence{}).Error; err != nil {
		log.Printf("scheduler: failed to expire local intelligence: %v", err)
	}
}

// PurgeDeletedAccounts permanently removes accounts whose deletion grace
// period (30 days) has elapsed. Before hard-deleting the user row, all
// personally identifiable fields are anonymized in place so dependent
// rows (audit logs, financial ledgers, case assignments, etc.) keep their
// foreign-key references intact without leaking PII.
//
// Anonymization map:
//
//	Email            -> "deleted_<uuid>@deleted.local"
//	Phone            -> ""
//	FirstName/LastName -> "Deleted"
//	MedicalInfo      -> ""
//	AvatarPath/CoverPath -> ""
//	Password         -> bcrypt hash of random 32 bytes (unusable)
//
// The user row is then hard-deleted with Unscoped().
func (s *SchedulerService) PurgeDeletedAccounts() {
	cutoff := time.Now().UTC().AddDate(0, 0, -30)

	var users []models.User
	if err := config.DB.
		Unscoped().
		Where("deletion_requested_at IS NOT NULL AND deletion_requested_at <= ?", cutoff).
		Find(&users).Error; err != nil {
		log.Printf("scheduler: failed to load accounts for purge: %v", err)
		return
	}

	if len(users) == 0 {
		return
	}

	for _, u := range users {
		// Anonymize PII in place so dependent rows keep valid FKs.
		fakeHash, err := bcrypt.GenerateFromPassword(
			[]byte(uuid.NewString()),
			bcrypt.DefaultCost,
		)
		if err != nil {
			fakeHash = []byte("$2y$10$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA") //nolint
		}

		if err := config.DB.Unscoped().Model(&models.User{}).
			Where("id = ?", u.ID).
			Updates(map[string]interface{}{
				"email":        "deleted_" + u.ID.String() + "@deleted.local",
				"phone":        "",
				"first_name":   "Deleted",
				"last_name":    "Deleted",
				"medical_info": "",
				"avatar_path":  "",
				"cover_path":   "",
				"password":     string(fakeHash),
				"last_login":   nil,
			}).Error; err != nil {
			log.Printf("scheduler: failed to anonymize user %s: %v", u.ID, err)
			continue
		}

		// Hard-delete the user row. GORM soft-delete is bypassed via Unscoped().
		if err := config.DB.Unscoped().Delete(&models.User{}, u.ID).Error; err != nil {
			log.Printf("scheduler: failed to hard-delete user %s: %v", u.ID, err)
		}
	}

	log.Printf("scheduler: purged %d deleted accounts", len(users))
}

// UpdateEventStatuses marks community events whose end time has passed as
// completed.
//
// The event list only shows "upcoming" rows, so an event that has run its
// course would otherwise sit in the feed forever. An event with an explicit
// end date completes when that end date passes; an event without one falls
// back to its start date, because there is no other point at which it can be
// said to be over.
//
// Idempotent: the WHERE clause only matches rows still marked "upcoming", so
// running the job twice in one cycle changes nothing the second time.
func (s *SchedulerService) UpdateEventStatuses() {
	now := time.Now().UTC()

	if err := config.DB.Model(&models.CommunityEvent{}).
		Where("status = ? AND COALESCE(end_date, event_date) <= ?", EventStatusUpcoming, now).
		Update("status", EventStatusCompleted).Error; err != nil {
		log.Printf("scheduler: failed to complete past events: %v", err)
	}
}

// SendEventReminders notifies the attendees of community events that start
// within the next 24 hours.
//
// The window is [now, now+24h]. Because the scheduler runs hourly, an event
// in that window is seen by roughly 24 hourly runs, so the job must not send
// the same reminder twice. It does not: before writing a notification it
// checks whether that attendee already has an "event_reminder" for that event
// inside a slightly wider window (25h), and skips if so.
//
// An event that has just been created (say 2 hours before it starts) is also
// picked up on the very next run, so a reminder is not lost.
func (s *SchedulerService) SendEventReminders() {
	now := time.Now().UTC()
	windowEnd := now.Add(24 * time.Hour)
	dedupeSince := now.Add(-25 * time.Hour)

	var events []models.CommunityEvent
	if err := config.DB.
		Where("status = ? AND event_date > ? AND event_date <= ?", EventStatusUpcoming, now, windowEnd).
		Find(&events).Error; err != nil {
		log.Printf("scheduler: failed to load events starting soon: %v", err)
		return
	}

	for _, event := range events {
		var attendees []models.EventAttendee
		if err := config.DB.Where("event_id = ? AND status = ?", event.ID, "registered").Find(&attendees).Error; err != nil {
			log.Printf("scheduler: failed to load attendees of event %s: %v", event.ID, err)
			continue
		}

		for _, attendee := range attendees {
			var existing int64
			if err := config.DB.Model(&models.Notification{}).
				Where("user_id = ? AND type = ? AND entity_id = ? AND created_at >= ?",
					attendee.UserID, EventReminderType, event.ID, dedupeSince).
				Count(&existing).Error; err != nil {
				log.Printf("scheduler: failed to check existing reminders for event %s: %v", event.ID, err)
				continue
			}
			if existing > 0 {
				continue
			}

			notification := models.Notification{
				UserID:     attendee.UserID,
				Title:      "Reminder: \"" + event.Title + "\" starts soon",
				Message:    fmt.Sprintf("The event \"%s\" begins at %s UTC.", event.Title, event.EventDate.Format(time.RFC3339)),
				Type:       EventReminderType,
				Status:     "unread",
				EntityType: "event",
				EntityID:   event.ID,
				LinkTo:     "/community/events/" + event.ID.String(),
			}
			if err := config.DB.Create(&notification).Error; err != nil {
				log.Printf("scheduler: failed to send event reminder for event %s: %v", event.ID, err)
			}
		}
	}
}
