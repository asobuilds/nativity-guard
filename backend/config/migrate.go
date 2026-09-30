package config

import (
	"log"

	"security-solution/models"
)

// AutoMigrateAll runs AutoMigrate on every model in the schema.
// Called on every boot so new columns and tables land in production
// without a manual migration step.
func AutoMigrateAll() error {
	log.Println("running auto-migration...")
	err := DB.AutoMigrate(
		&models.User{},
		&models.SecurityUnit{},
		&models.Officer{},
		&models.GovernmentIDVerification{},

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
		// SOSResponder references SOSAlert and SecurityUnit, so it is
		// registered after both are migrated.
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
	log.Println("auto-migration complete")
	return nil
}