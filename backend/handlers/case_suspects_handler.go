package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"security-solution/config"
	"security-solution/models"
)

// CaseSuspectView is the suspect summary returned by GET /cases/:id/suspects.
// PII fields are omitted on purpose: the caller has read access to the case,
// and the case-scoped check (CanAccessCase) already ensures they are one of
// the assigned officer, a unit/super admin of the case's unit, or the
// reporter — everyone else is blocked before this handler runs.
type CaseSuspectView struct {
	SuspectID   string `json:"suspectId"`
	FirstName   string `json:"firstName"`
	LastName    string `json:"lastName"`
	Alias       string `json:"alias,omitempty"`
	Gender      string `json:"gender,omitempty"`
	Category    string `json:"category,omitempty"`
	DangerLevel string `json:"dangerLevel,omitempty"`
	Status      string `json:"status"`
	Wanted      bool   `json:"wanted"`
	RoleInCase  string `json:"roleInCase"`
	Notes       string `json:"notes,omitempty"`
}

// GetCaseSuspects returns the suspects linked to a case, gated to the case's
// unit (assigned officer / unit admin / super admin). The CanAccessCase
// middleware runs first and sets case_access_level; the handler blocks the
// reporter tier (a reporter of the case has no cross-suspect read rights)
// and lets officer / unit_admin / super_admin tiers through.
func GetCaseSuspects(c *gin.Context) {
	caseID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid case id"})
		return
	}

	var caseRecord models.Case
	if err := config.DB.First(&caseRecord, "id = ?", caseID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "case not found"})
		return
	}

	// Block the reporter tier explicitly; officers / unit admins / super
	// admins are allowed.
	if level := c.GetString("case_access_level"); level == "reporter" {
		c.JSON(http.StatusForbidden, gin.H{"error": "reporters may not view case suspects"})
		return
	}

	var links []models.SuspectCase
	if err := config.DB.Preload("Suspect").Where("case_id = ?", caseID).Find(&links).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch case suspects"})
		return
	}

	views := make([]CaseSuspectView, 0, len(links))
	for _, link := range links {
		s := link.Suspect
		views = append(views, CaseSuspectView{
			SuspectID:   s.ID.String(),
			FirstName:   s.FirstName,
			LastName:    s.LastName,
			Alias:       s.Alias,
			Gender:      s.Gender,
			Category:    s.Category,
			DangerLevel: s.DangerLevel,
			Status:      s.Status,
			Wanted:      s.Wanted,
			RoleInCase:  link.Role,
			Notes:       link.Notes,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"caseId":   caseID.String(),
		"suspects": views,
	})
}
