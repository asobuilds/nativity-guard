package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"security-solution/config"
	"security-solution/models"
)

type CaseCloseRequest struct {
	FinalReport string `json:"finalReport" binding:"required"`
}

func getAuthenticatedOfficer(c *gin.Context) (*models.Officer, uuid.UUID, bool) {
	userIDValue, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "authenticated user not found",
		})
		return nil, uuid.Nil, false
	}

	userID, ok := userIDValue.(string)
	if !ok || userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid authenticated user",
		})
		return nil, uuid.Nil, false
	}

	officerID, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid authenticated user",
		})
		return nil, uuid.Nil, false
	}

	var officer models.Officer
	if err := config.DB.First(&officer, "id = ?", officerID).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "authenticated user is not an officer",
		})
		return nil, uuid.Nil, false
	}

	return &officer, officerID, true
}

func authorizeCaseOfficer(
	c *gin.Context,
	caseRecord *models.Case,
	requireAssignment bool,
) (*models.Officer, uuid.UUID, bool) {
	officer, officerID, ok := getAuthenticatedOfficer(c)
	if !ok {
		return nil, uuid.Nil, false
	}

	if officer.UnitID != caseRecord.UnitID {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "officer is not assigned to this case unit",
		})
		return nil, uuid.Nil, false
	}

	if requireAssignment {
		if caseRecord.AssignedTo == nil {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "case has not been assigned to an officer",
			})
			return nil, uuid.Nil, false
		}

		if *caseRecord.AssignedTo != officerID {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "officer is not assigned to this case",
			})
			return nil, uuid.Nil, false
		}
	}

	return officer, officerID, true
}

// DispatchCase dispatches the assigned officer to the case.
func DispatchCase(c *gin.Context) {
	caseID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid case id",
		})
		return
	}

	var caseRecord models.Case
	if err := config.DB.First(&caseRecord, "id = ?", caseID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "case not found",
		})
		return
	}

	_, _, ok := authorizeCaseOfficer(c, &caseRecord, true)
	if !ok {
		return
	}

	if caseRecord.Status != "pending" && caseRecord.Status != "assigned" {
		c.JSON(http.StatusConflict, gin.H{
			"error":  "case cannot be dispatched from its current status",
			"status": caseRecord.Status,
		})
		return
	}

	now := time.Now()

	caseRecord.Status = "dispatched"
	caseRecord.DispatchedAt = &now

	if err := config.DB.Save(&caseRecord).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to dispatch case",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "case dispatched successfully",
		"case":    caseRecord,
	})
}

// ArriveAtCase records the assigned officer's arrival at the case location.
// The officer's current GPS position must be within 200 m of the case
// coordinates; otherwise the request is rejected with 400.
func ArriveAtCase(c *gin.Context) {
	caseID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid case id",
		})
		return
	}

	var input struct {
		Latitude  float64 `json:"latitude" binding:"required"`
		Longitude float64 `json:"longitude" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "latitude and longitude are required",
		})
		return
	}

	var caseRecord models.Case
	if err := config.DB.First(&caseRecord, "id = ?", caseID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "case not found",
		})
		return
	}

	_, _, ok := authorizeCaseOfficer(c, &caseRecord, true)
	if !ok {
		return
	}

	if caseRecord.Status != "dispatched" {
		c.JSON(http.StatusConflict, gin.H{
			"error":  "case cannot be marked on-scene from its current status",
			"status": caseRecord.Status,
		})
		return
	}

	// 200 m geofence: the officer's reported position must be close enough
	// to the case location for the arrival to be considered genuine.
	if caseRecord.Latitude != 0 || caseRecord.Longitude != 0 {
		distanceKm := haversine(input.Latitude, input.Longitude, caseRecord.Latitude, caseRecord.Longitude)
		if distanceKm*1000 > 200 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "you are not within 200 m of the case location",
			})
			return
		}
	}

	now := time.Now()

	caseRecord.Status = "on_scene"
	caseRecord.ArrivedAt = &now

	if err := config.DB.Save(&caseRecord).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to record arrival",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "arrival recorded successfully",
		"case":    caseRecord,
	})
}

// CloseCase closes the case after the assigned officer submits the final report.
func CloseCase(c *gin.Context) {
	caseID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid case id",
		})
		return
	}

	var req CaseCloseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "finalReport is required",
		})
		return
	}

	var caseRecord models.Case
	if err := config.DB.First(&caseRecord, "id = ?", caseID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "case not found",
		})
		return
	}

	if caseRecord.Status == "closed" {
		c.JSON(http.StatusConflict, gin.H{
			"error": "case is already closed",
		})
		return
	}

	accessLevel := c.GetString("case_access_level")
	if accessLevel != "super_admin" && accessLevel != "head_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "only admins may close a case directly; officers must submit for review"})
		c.Abort()
		return
	}

	_, officerID, ok := authorizeCaseOfficer(c, &caseRecord, true)
	if !ok {
		return
	}

	if caseRecord.Status != "on_scene" {
		c.JSON(http.StatusConflict, gin.H{
			"error":  "case cannot be closed before arrival",
			"status": caseRecord.Status,
		})
		return
	}

	now := time.Now()

	caseRecord.Status = "closed"
	caseRecord.ClosedAt = &now
	caseRecord.ClosedBy = &officerID
	caseRecord.FinalReport = req.FinalReport

	if err := config.DB.Save(&caseRecord).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to close case",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "case closed successfully",
		"case":    caseRecord,
	})
}
