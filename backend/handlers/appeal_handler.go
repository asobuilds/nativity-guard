package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"security-solution/models"
	"security-solution/services"
)

// FileAppeal — the removed member files an appeal against a completed
// revocation cycle. Auth gate + eligibility live in the service.
func FileAppeal(c *gin.Context) {
	userValue, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}
	userObj := userValue.(*models.User)

	var input struct {
		RevocationCycleID string `json:"revocationCycleId" binding:"required"`
		Reason            string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cycleID, err := uuid.Parse(input.RevocationCycleID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid revocation cycle ID"})
		return
	}

	svc := services.NewAppealService()
	appeal, err := svc.FileAppeal(services.FileAppealInput{
		RevocationCycleID: cycleID,
		AppellantUserID:   userObj.ID,
		Reason:            input.Reason,
	})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			// 404 for missing cycle/membership
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":                appeal.ID,
		"revocationCycleId": appeal.RevocationCycleID,
		"status":            appeal.Status,
		"filedAt":           appeal.FiledAt,
	})
}

// GetAppeal returns a single appeal. Accessible to super admin or to the
// appellant themselves (reporter-style exclusion is inherent: the reporter is
// neither super admin nor the appellant).
func GetAppeal(c *gin.Context) {
	requestID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid appeal ID"})
		return
	}
	userValue, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}
	userObj := userValue.(*models.User)

	svc := services.NewAppealService()
	appeal, err := svc.GetAppeal(requestID, userObj.ID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, appealResponse(appeal))
}

// ListAppeals lists appeals for super admins only. Optional ?status filter.
func ListAppeals(c *gin.Context) {
	userValue, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}
	userObj := userValue.(*models.User)
	if !userObj.IsSuperAdmin && userObj.Role != "super_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only super admin may list appeals"})
		return
	}

	status := c.Query("status")
	svc := services.NewAppealService()
	appeals, err := svc.ListAppeals(status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rows := make([]gin.H, 0, len(appeals))
	for _, a := range appeals {
		rows = append(rows, appealResponse(&a))
	}
	c.JSON(http.StatusOK, gin.H{"appeals": rows})
}

// DecideAppeal lets a super admin uphold or overturn a pending appeal.
func DecideAppeal(c *gin.Context) {
	requestID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid appeal ID"})
		return
	}
	userValue, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}
	userObj := userValue.(*models.User)

	var input struct {
		Decision string `json:"decision" binding:"required"`
		Reason   string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	svc := services.NewAppealService()
	appeal, err := svc.DecideAppeal(requestID, userObj.ID, services.DecideAppealInput{
		Decision: input.Decision,
		Reason:   input.Reason,
	})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if strings.Contains(err.Error(), "only a super admin") {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp := appealResponse(appeal)
	if appeal.Status == "overturned" {
		resp["restored"] = true
	} else {
		resp["restored"] = false
	}
	c.JSON(http.StatusOK, resp)
}

func appealResponse(a *models.Appeal) gin.H {
	return gin.H{
		"id":                a.ID,
		"revocationCycleId": a.RevocationCycleID,
		"appellantUserId":   a.AppellantUserID,
		"status":            a.Status,
		"reason":            a.Reason,
		"filedAt":           a.FiledAt,
		"decisionReason":    a.DecisionReason,
		"decidedBy":         a.DecidedBy,
		"decidedAt":         a.DecidedAt,
		"escalatedAt":       a.EscalatedAt,
		"createdAt":         a.CreatedAt,
		"updatedAt":         a.UpdatedAt,
	}
}
