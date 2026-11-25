package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"security-solution/config"
	"security-solution/models"
	"security-solution/services"
)

// OpenElectionRequest — body for POST /units/:unitId/elections
type OpenElectionRequest struct {
	RotationGroup string `json:"rotationGroup" binding:"required"` // "A" or "B"
}

// OpenAdminElection creates a new admin election for the unit.
// Only head admin or unit admin may open an election.
func OpenAdminElection(c *gin.Context) {
	unitID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid unit id"})
		return
	}

	userValue, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}
	userObj := userValue.(*models.User)

	var membership models.UnitMembership
	if err := config.DB.
		Where("unit_id = ? AND user_id = ? AND status = ?", unitID, userObj.ID, models.MembershipActive).
		First(&membership).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not a member of this unit"})
		return
	}
	if !membership.IsHeadAdmin && membership.Role != models.UnitRoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only head admin or admins can open elections"})
		return
	}

	var req OpenElectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "rotationGroup is required"})
		return
	}

	svc := services.NewElectionService()
	election, err := svc.OpenAdminElection(unitID, userObj.ID, req.RotationGroup)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"election": election})
}

// CastAdminVoteRequest — body for POST /elections/:id/vote
type CastAdminVoteRequest struct {
	CandidateID uuid.UUID `json:"candidateId" binding:"required"`
}

// CastAdminVote records a verified member's vote.
func CastAdminVote(c *gin.Context) {
	electionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid election id"})
		return
	}

	userValue, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}
	userObj := userValue.(*models.User)

	var req CastAdminVoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "candidateId is required"})
		return
	}

	svc := services.NewElectionService()
	if err := svc.CastAdminVote(electionID, userObj.ID, req.CandidateID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "vote recorded"})
}

// CloseAdminElection finalizes an election and seats the winners.
func CloseAdminElection(c *gin.Context) {
	electionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid election id"})
		return
	}

	userValue, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}
	userObj := userValue.(*models.User)

	var election models.UnitAdminElection
	if err := config.DB.First(&election, "id = ?", electionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "election not found"})
		return
	}

	var membership models.UnitMembership
	if err := config.DB.
		Where("unit_id = ? AND user_id = ? AND status = ?", election.UnitID, userObj.ID, models.MembershipActive).
		First(&membership).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not a member of this unit"})
		return
	}
	if !membership.IsHeadAdmin && membership.Role != models.UnitRoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only head admin or admins can close elections"})
		return
	}

	svc := services.NewElectionService()
	if err := svc.CloseAdminElection(electionID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "election finalized"})
}

// GetElectionResults returns the election and its resulting seats.
func GetElectionResults(c *gin.Context) {
	electionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid election id"})
		return
	}

	var election models.UnitAdminElection
	if err := config.DB.First(&election, "id = ?", electionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "election not found"})
		return
	}

	userValue, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}
	userObj, ok := userValue.(*models.User)
	if !ok || userObj == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user"})
		return
	}

	if !userObj.IsSuperAdmin && userObj.Role != "super_admin" {
		var membership models.UnitMembership
		if err := config.DB.
			Where("unit_id = ? AND user_id = ? AND status = ?", election.UnitID, userObj.ID, models.MembershipActive).
			First(&membership).Error; err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "not a member of this unit"})
			return
		}

		isUnitAdmin := membership.IsHeadAdmin || membership.Role == models.UnitRoleAdmin
		isFinalized := election.Status == "finalized" ||
			election.Status == "finalized_low_turnout" ||
			election.ResultFinalizedAt != nil

		if !isUnitAdmin && !isFinalized {
			c.JSON(http.StatusForbidden, gin.H{"error": "election results are not available until the election is finalized"})
			return
		}
	}

	var seats []models.UnitAdminSeat
	config.DB.Where("election_id = ?", electionID).Order("seat_number ASC").Find(&seats)

	c.JSON(http.StatusOK, gin.H{
		"election": election,
		"seats":    seats,
	})
}

// OpenHeadAdminElection starts a head-admin election among current admins.
func OpenHeadAdminElection(c *gin.Context) {
	unitID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid unit id"})
		return
	}

	userValue, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}
	userObj := userValue.(*models.User)

	var membership models.UnitMembership
	if err := config.DB.
		Where("unit_id = ? AND user_id = ? AND status = ?", unitID, userObj.ID, models.MembershipActive).
		First(&membership).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not a member of this unit"})
		return
	}
	if !membership.IsHeadAdmin && membership.Role != models.UnitRoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only head admin or admins can open head-admin elections"})
		return
	}

	svc := services.NewElectionService()
	if err := svc.RunHeadAdminElection(unitID, userObj.ID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "head-admin election opened"})
}
