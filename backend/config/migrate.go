package config

import (
	"log"

	"security-solution/models"
)

func AutoMigrateAll() error {
	log.Println("running auto-migration...")
	err := DB.AutoMigrate(
		&models.User{},
		&models.UserPreferences{},
		&models.SecurityUnit{},
		&models.Officer{},
		&models.GovernmentIDVerification{},
		&models.IdentityVerification{},

		&models.Feedback{},
		&models.LGAMessage{},

		&models.Case{},
		&models.CaseAccountabilityEvent{},
		&models.Evidence{},
		&models.Progress{},
		&models.CaseFeedback{},
		&models.CaseOfficer{},
		&models.CaseProgress{},
		&models.CaseTemplate{},
		&models.CaseTimeline{},
		&models.CaseTransfer{},
		&models.TransferRequest{},
		&models.TransferApproval{},

		&models.Rating{},
		&models.Notification{},
		&models.OTP{},
		&models.PasswordReset{},
		&models.PushSubscription{},
		&models.PushToken{},

		&models.AIAnalysis{},
		&models.Alert{},
		&models.Announcement{},
		&models.News{},
		&models.NewsAlert{},

		&models.SOSAlert{},
		&models.SOSResponder{},
		&models.BankAccount{},
		&models.Donation{},
		&models.Transaction{},
		&models.TransactionApproval{},
		&models.Budget{},
		&models.FinancialReport{},

		&models.ForumPost{},
		&models.ForumReply{},
		&models.CommunityAnnouncement{},
		&models.CommunityEvent{},
		&models.EventAttendee{},
		&models.CommunityAlert{},
		&models.AlertSubscription{},

		&models.AuditLog{},
		&models.SystemHealth{},
		&models.ActivityLog{},
		&models.NotificationLog{},

		&models.DataExport{},
		&models.EmailTemplate{},
		&models.SystemSettings{},
		&models.SystemBackup{},
		&models.UserOnboarding{},

		&models.PeaceCommittee{},
		&models.CommitteeMember{},
		&models.ConflictResolution{},
		&models.CommunityTrustScore{},
		&models.PeaceMetric{},

		&models.Suspect{},
		&models.SuspectAssociation{},
		&models.SuspectSighting{},
		&models.SuspectCase{},

		&models.Camera{},
		&models.VideoAlert{},
		&models.SocialMediaPost{},

		&models.CommunicationRoom{},
		&models.CommunicationMessage{},
		&models.VoiceCall{},
		&models.CommunicationSync{},

		&models.ChatMessage{},
		&models.Report{},
		&models.CaseReview{},
		&models.CaseWeeklyUpdate{},
		&models.UnitMembership{},
		&models.UnitAuth{},
		&models.UnitInvite{},
		&models.UnitAdminSeat{},
		&models.UnitAdminElection{},
		&models.AdminVote{},
		&models.HeadAdminVote{},
		&models.CaseAdminAssignment{},
		&models.RevocationCycle{},
		&models.RevocationVote{},

		&models.RevokedToken{},
		&models.TermsDocument{},
		&models.TermsAcceptance{},
		&models.RefreshToken{},
		&models.UserSession{},
		&models.IdempotencyRecord{},

		&models.FinancialLedger{},
		&models.LedgerSequence{},
		&models.UnitFinancialYear{},
		&models.PlatformDonation{},

		&models.OfficerScore{},
		&models.UnitScore{},
		&models.CounterStatement{},
		&models.Appeal{},
		&models.AgeAudit{},
		&models.GeocodeCache{},
	)
	if err != nil {
		return err
	}

	// EventAttendee is soft-deleted, so RSVP uniqueness must cover only live
	// rows. The model tag declares this index; this is the explicit, idempotent
	// migration form (RES-004). It is intentionally NOT tolerant of existing
	// duplicates: AutoMigrate/this statement fails loudly so the drift is
	// fixed by an approved data migration rather than silently.
	if err := DB.Exec(
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_event_attendee_active ON event_attendees (event_id, user_id) WHERE deleted_at IS NULL`,
	).Error; err != nil {
		return err
	}

	log.Println("auto-migration complete")
	return nil
}
