package services

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"security-solution/config"
	"security-solution/models"
)

// SuspectService owns the cross-case lifecycle rules for suspects: when a
// case reaches a terminal status, every suspect linked to it should drop out
// of active tracking (status -> "archived") unless they are still linked to
// another open case.
type SuspectService struct{}

func NewSuspectService() *SuspectService {
	return &SuspectService{}
}

// ArchiveSuspectsForCase marks every suspect linked to the given case as
// archived, but only if they have no other linked case that is still open.
//
// The caller passes a *gorm.DB (either config.DB or an open transaction) so
// the archive rows and the audit rows commit atomically with the case status
// change.
//
// actorID is the user who caused the case to close; it is recorded on the
// suspect.archive audit rows. It may be uuid.Nil when the archive is a
// background cleanup with no direct actor.
func (s *SuspectService) ArchiveSuspectsForCase(db *gorm.DB, caseID, actorID uuid.UUID) error {
	var links []models.SuspectCase
	if err := db.Where("case_id = ?", caseID).Find(&links).Error; err != nil {
		return err
	}
	if len(links) == 0 {
		return nil
	}

	auditSvc := NewAuditService()

	for _, link := range links {
		var suspect models.Suspect
		if err := db.First(&suspect, "id = ?", link.SuspectID).Error; err != nil {
			// Suspect already deleted (soft); skip.
			continue
		}

		// Only archive if this is the last open case that references them.
		var otherOpen int64
		if err := db.Model(&models.SuspectCase{}).
			Joins("JOIN cases ON cases.id = suspect_cases.case_id").
			Where("suspect_cases.suspect_id = ? AND suspect_cases.case_id <> ? AND cases.status NOT IN ?",
				link.SuspectID, caseID,
				[]string{models.CaseStatusClosed, "resolved", "cancelled"}).
			Count(&otherOpen).Error; err != nil {
			return err
		}
		if otherOpen > 0 {
			continue
		}

		if suspect.Status == models.SuspectStatusArchived || suspect.Status == models.SuspectStatusExpunged {
			// Already terminal; don't overwrite or duplicate audit rows.
			continue
		}

		oldStatus := suspect.Status
		suspect.Status = models.SuspectStatusArchived
		if err := db.Save(&suspect).Error; err != nil {
			return err
		}

		_ = auditSvc.LogAction(
			actorID,
			"suspect.archive",
			"suspect",
			suspect.ID.String(),
			map[string]string{"status": oldStatus},
			map[string]string{"status": models.SuspectStatusArchived, "trigger": "case_closed", "caseId": caseID.String()},
			"",
			"",
		)
	}

	return nil
}

// ArchiveSuspectsForCase is the config.DB variant used by close paths that do
// not already hold a transaction.
func ArchiveSuspectsForCase(caseID, actorID uuid.UUID) error {
	return NewSuspectService().ArchiveSuspectsForCase(config.DB, caseID, actorID)
}
