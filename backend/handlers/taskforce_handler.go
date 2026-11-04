package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"security-solution/config"
	"security-solution/models"
	"security-solution/services"
	ws "security-solution/websocket"
)

const (
	defaultTaskForceInviteHours = 72
	maxTaskForceInviteHours     = 720
	maxTaskForceEnlistBatch     = 100
)

type taskForceEnlistResult struct {
	OfficerID uuid.UUID `json:"officerId"`
	Status    string    `json:"status"`
	Message   string    `json:"message,omitempty"`
}

type taskForceInviteSummary struct {
	ID        uuid.UUID  `json:"id"`
	UnitID    uuid.UUID  `json:"unitId"`
	Status    string     `json:"status"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
}

type taskForceSummary struct {
	TaskForce        models.TaskForce `json:"taskForce"`
	Role             string           `json:"role"`
	MembershipStatus string           `json:"membershipStatus"`
}

func currentUser(c *gin.Context) *models.User {
	value, exists := c.Get("user")
	if !exists {
		return nil
	}
	user, ok := value.(*models.User)
	if !ok {
		return nil
	}
	return user
}

func userIsHeadAdminOf(user *models.User, unitID uuid.UUID) bool {
	if user == nil {
		return false
	}
	if user.IsSuperAdmin || user.Role == models.RoleSuperAdmin {
		return true
	}
	var membership models.UnitMembership
	err := config.DB.
		Where("user_id = ? AND unit_id = ? AND status = ?", user.ID, unitID, models.MembershipActive).
		First(&membership).Error
	if err != nil {
		return false
	}
	return membership.IsHeadAdmin
}

func callerIsHeadAdminOf(c *gin.Context, unitID uuid.UUID) bool {
	return userIsHeadAdminOf(currentUser(c), unitID)
}

func taskForceRoomKey(taskForceID uuid.UUID) string {
	return "taskforce:" + taskForceID.String()
}

func loadTaskForce(id uuid.UUID) (*models.TaskForce, error) {
	var taskForce models.TaskForce
	if err := config.DB.Preload("HostUnit").First(&taskForce, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &taskForce, nil
}

func taskForceForRoom(roomID uuid.UUID) (*models.TaskForce, error) {
	var link models.TaskForceChannel
	if err := config.DB.Where("room_id = ?", roomID).First(&link).Error; err != nil {
		return nil, err
	}
	var taskForce models.TaskForce
	if err := config.DB.First(&taskForce, "id = ?", link.TaskForceID).Error; err != nil {
		return nil, err
	}
	return &taskForce, nil
}

func getTaskForceChannel(taskForceID uuid.UUID) (models.CommunicationRoom, error) {
	var link models.TaskForceChannel
	if err := config.DB.Preload("Room").Where("task_force_id = ?", taskForceID).First(&link).Error; err != nil {
		return models.CommunicationRoom{}, err
	}
	return link.Room, nil
}

func ensureTaskForceChannel(taskForce *models.TaskForce, createdBy uuid.UUID) (models.CommunicationRoom, error) {
	if taskForce == nil || taskForce.Status != models.TaskForceStatusActive {
		return models.CommunicationRoom{}, errTaskForceNotActive
	}
	if room, err := getTaskForceChannel(taskForce.ID); err == nil {
		return room, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return models.CommunicationRoom{}, err
	}

	var room models.CommunicationRoom
	err := config.DB.Transaction(func(tx *gorm.DB) error {
		room = models.CommunicationRoom{
			UnitID:      taskForce.HostUnitID,
			Name:        taskForce.Name,
			Type:        models.TaskForceRoomType,
			Status:      "active",
			IsEncrypted: true,
			Priority:    "normal",
			CreatedBy:   createdBy,
		}
		if err := tx.Create(&room).Error; err != nil {
			return err
		}
		return tx.Create(&models.TaskForceChannel{
			TaskForceID: taskForce.ID,
			RoomID:      room.ID,
		}).Error
	})
	if err != nil {
		if existing, findErr := getTaskForceChannel(taskForce.ID); findErr == nil {
			return existing, nil
		}
		return models.CommunicationRoom{}, err
	}
	return room, nil
}

func createTaskForceChannelTx(tx *gorm.DB, taskForce *models.TaskForce, createdBy uuid.UUID) error {
	room := models.CommunicationRoom{
		UnitID:      taskForce.HostUnitID,
		Name:        taskForce.Name,
		Type:        models.TaskForceRoomType,
		Status:      "active",
		IsEncrypted: true,
		Priority:    "normal",
		CreatedBy:   createdBy,
	}
	if err := tx.Create(&room).Error; err != nil {
		return err
	}
	return tx.Create(&models.TaskForceChannel{
		TaskForceID: taskForce.ID,
		RoomID:      room.ID,
	}).Error
}

func activeTaskForceUnitIDs(taskForceID uuid.UUID) ([]uuid.UUID, map[uuid.UUID]bool, error) {
	var members []models.TaskForceMember
	if err := config.DB.
		Where("task_force_id = ? AND status = ?", taskForceID, models.TaskForceMemberStatusActive).
		Find(&members).Error; err != nil {
		return nil, nil, err
	}
	ids := make([]uuid.UUID, 0, len(members))
	set := make(map[uuid.UUID]bool, len(members))
	for _, member := range members {
		ids = append(ids, member.UnitID)
		set[member.UnitID] = true
	}
	return ids, set, nil
}

func taskForceHasActiveMember(taskForceID, unitID uuid.UUID) bool {
	var count int64
	config.DB.Model(&models.TaskForceMember{}).
		Where("task_force_id = ? AND unit_id = ? AND status = ?", taskForceID, unitID, models.TaskForceMemberStatusActive).
		Count(&count)
	return count > 0
}

func taskForceMemberStatusForUnit(taskForceID, unitID uuid.UUID) (string, bool) {
	var member models.TaskForceMember
	if err := config.DB.
		Where("task_force_id = ? AND unit_id = ?", taskForceID, unitID).
		First(&member).Error; err != nil {
		return "", false
	}
	return member.Status, true
}

func canViewTaskForce(user *models.User, taskForce *models.TaskForce) bool {
	if user == nil || taskForce == nil {
		return false
	}
	if user.IsSuperAdmin || user.Role == models.RoleSuperAdmin {
		return true
	}
	if user.UnitID == nil {
		return false
	}
	if user.Role != models.RoleOfficer && user.Role != models.RoleUnitAdmin {
		return false
	}
	status, found := taskForceMemberStatusForUnit(taskForce.ID, *user.UnitID)
	if !found {
		return false
	}
	if status == models.TaskForceMemberStatusActive {
		return true
	}
	// An invited unit's head admin may open the task force to enter the invite
	// code. Walkie/PTT access still requires an active membership.
	return status == models.TaskForceMemberStatusInvited && userIsHeadAdminOf(user, *user.UnitID)
}

func canAccessTaskForceChannel(user *models.User, taskForce *models.TaskForce) bool {
	if user == nil || taskForce == nil || taskForce.Status != models.TaskForceStatusActive {
		return false
	}
	if user.IsSuperAdmin || user.Role == models.RoleSuperAdmin {
		return true
	}
	return canTransmitOnTaskForce(user, taskForce)
}

func canTransmitOnTaskForce(user *models.User, taskForce *models.TaskForce) bool {
	if user == nil || taskForce == nil || taskForce.Status != models.TaskForceStatusActive {
		return false
	}
	if user.IsSuperAdmin || user.Role == models.RoleSuperAdmin {
		return false
	}
	if user.UnitID == nil {
		return false
	}
	if user.Role != models.RoleOfficer && user.Role != models.RoleUnitAdmin {
		return false
	}
	return taskForceHasActiveMember(taskForce.ID, *user.UnitID)
}

func taskForceInviteHash(code string) string {
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:])
}

func newTaskForceInviteCode() (string, string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", err
	}
	code := hex.EncodeToString(raw)
	return code, taskForceInviteHash(code), nil
}

func CreateTaskForce(c *gin.Context) {
	user, ok := getAuthUser(c)
	if !ok {
		return
	}

	var input struct {
		Name       string `json:"name" binding:"required"`
		HostUnitID string `json:"hostUnitId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	name := strings.TrimSpace(input.Name)
	if len(name) < 2 || len(name) > 120 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name must be between 2 and 120 characters"})
		return
	}

	hostUnitID, err := uuid.Parse(input.HostUnitID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid host unit ID"})
		return
	}
	if !userIsHeadAdminOf(user, hostUnitID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only the host unit's head admin can create a task force"})
		return
	}

	var unit models.SecurityUnit
	if err := config.DB.First(&unit, "id = ?", hostUnitID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Host unit not found"})
		return
	}
	if unit.Status != "active" || !unit.IsVerified {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Host unit must be active and verified"})
		return
	}

	taskForce := models.TaskForce{
		Name:       name,
		HostUnitID: hostUnitID,
		Status:     models.TaskForceStatusActive,
		CreatedBy:  user.ID,
	}
	now := time.Now().UTC()

	err = config.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&taskForce).Error; err != nil {
			return err
		}
		hostMember := models.TaskForceMember{
			TaskForceID: taskForce.ID,
			UnitID:      hostUnitID,
			Role:        models.TaskForceMemberRoleHost,
			Status:      models.TaskForceMemberStatusActive,
			AcceptedAt:  &now,
		}
		if err := tx.Create(&hostMember).Error; err != nil {
			return err
		}
		return createTaskForceChannelTx(tx, &taskForce, user.ID)
	})
	if err != nil {
		log.Printf("taskforce: create failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create task force"})
		return
	}

	_ = services.NewAuditService().LogAction(
		user.ID,
		"taskforce.created",
		"task_force",
		taskForce.ID.String(),
		nil,
		map[string]any{"name": taskForce.Name, "hostUnitId": hostUnitID.String()},
		c.ClientIP(),
		c.Request.UserAgent(),
	)

	c.JSON(http.StatusCreated, gin.H{
		"message":   "Task force created",
		"taskForce": taskForce,
	})
}

func InviteTaskForceUnit(c *gin.Context) {
	user, ok := getAuthUser(c)
	if !ok {
		return
	}

	taskForceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task force ID"})
		return
	}
	taskForce, err := loadTaskForce(taskForceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task force not found"})
		return
	}
	if taskForce.Status != models.TaskForceStatusActive {
		c.JSON(http.StatusConflict, gin.H{"error": "Task force is not active"})
		return
	}
	if !userIsHeadAdminOf(user, taskForce.HostUnitID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only the host unit's head admin can invite units"})
		return
	}

	var input struct {
		UnitID         string `json:"unitId" binding:"required"`
		ExpiresInHours int    `json:"expiresInHours"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	targetUnitID, err := uuid.Parse(input.UnitID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid unit ID"})
		return
	}
	if targetUnitID == taskForce.HostUnitID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "The host unit is already a member"})
		return
	}

	var targetUnit models.SecurityUnit
	if err := config.DB.First(&targetUnit, "id = ?", targetUnitID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Target unit not found"})
		return
	}
	if targetUnit.Status != "active" || !targetUnit.IsVerified {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Target unit must be active and verified"})
		return
	}

	var existingMember models.TaskForceMember
	memberErr := config.DB.
		Where("task_force_id = ? AND unit_id = ?", taskForce.ID, targetUnitID).
		First(&existingMember).Error
	if memberErr == nil && existingMember.Status == models.TaskForceMemberStatusActive {
		c.JSON(http.StatusConflict, gin.H{"error": "Unit is already a member of this task force"})
		return
	}
	if memberErr != nil && !errors.Is(memberErr, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check membership"})
		return
	}

	code, codeHash, err := newTaskForceInviteCode()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate invite code"})
		return
	}

	hours := input.ExpiresInHours
	if hours <= 0 {
		hours = defaultTaskForceInviteHours
	}
	if hours > maxTaskForceInviteHours {
		hours = maxTaskForceInviteHours
	}
	expiresAt := time.Now().UTC().Add(time.Duration(hours) * time.Hour)
	now := time.Now().UTC()

	invite := models.TaskForceInvite{
		TaskForceID: taskForce.ID,
		UnitID:      targetUnitID,
		CodeHash:    codeHash,
		Status:      models.TaskForceInviteStatusPending,
		ExpiresAt:   &expiresAt,
		MaxUses:     1,
		CreatedBy:   user.ID,
	}

	err = config.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.TaskForceInvite{}).
			Where("task_force_id = ? AND unit_id = ? AND status = ?",
				taskForce.ID, targetUnitID, models.TaskForceInviteStatusPending).
			Update("status", models.TaskForceInviteStatusRevoked).Error; err != nil {
			return err
		}
		if err := tx.Create(&invite).Error; err != nil {
			return err
		}
		if memberErr == nil {
			existingMember.Role = models.TaskForceMemberRoleMember
			existingMember.Status = models.TaskForceMemberStatusInvited
			existingMember.AcceptedAt = nil
			return tx.Save(&existingMember).Error
		}
		return tx.Create(&models.TaskForceMember{
			TaskForceID: taskForce.ID,
			UnitID:      targetUnitID,
			Role:        models.TaskForceMemberRoleMember,
			Status:      models.TaskForceMemberStatusInvited,
		}).Error
	})
	if err != nil {
		log.Printf("taskforce: invite failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create invite"})
		return
	}

	_ = services.NewAuditService().LogAction(
		user.ID,
		"taskforce.unit_invited",
		"task_force",
		taskForce.ID.String(),
		nil,
		map[string]any{"unitId": targetUnitID.String(), "inviteId": invite.ID.String(), "expiresAt": expiresAt},
		c.ClientIP(),
		c.Request.UserAgent(),
	)

	c.JSON(http.StatusCreated, gin.H{
		"invite": gin.H{
			"id":        invite.ID,
			"unitId":    targetUnitID,
			"status":    invite.Status,
			"expiresAt": invite.ExpiresAt,
			"maxUses":   invite.MaxUses,
		},
		"code":    code,
		"warning": "Share this code with the target unit's head admin. It cannot be retrieved again.",
		"now":     now,
	})
}

func JoinTaskForce(c *gin.Context) {
	user, ok := getAuthUser(c)
	if !ok {
		return
	}

	taskForceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task force ID"})
		return
	}
	taskForce, err := loadTaskForce(taskForceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task force not found"})
		return
	}
	if taskForce.Status != models.TaskForceStatusActive {
		c.JSON(http.StatusConflict, gin.H{"error": "Task force is not active"})
		return
	}

	var input struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code is required"})
		return
	}

	codeHash := taskForceInviteHash(strings.TrimSpace(input.Code))
	now := time.Now().UTC()

	var joinedMember models.TaskForceMember
	err = config.DB.Transaction(func(tx *gorm.DB) error {
		var invite models.TaskForceInvite
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("code_hash = ? AND task_force_id = ?", codeHash, taskForce.ID).
			First(&invite).Error; err != nil {
			return err
		}
		if invite.Status != models.TaskForceInviteStatusPending {
			return errTaskForceInviteInactive
		}
		if invite.ExpiresAt != nil && invite.ExpiresAt.Before(now) {
			return errTaskForceInviteExpired
		}
		if invite.UseCount >= invite.MaxUses {
			return errTaskForceInviteInactive
		}
		if !userIsHeadAdminOf(user, invite.UnitID) {
			return errTaskForceNotHeadAdmin
		}

		var member models.TaskForceMember
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("task_force_id = ? AND unit_id = ?", taskForce.ID, invite.UnitID).
			First(&member).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			member = models.TaskForceMember{
				TaskForceID: taskForce.ID,
				UnitID:      invite.UnitID,
				Role:        models.TaskForceMemberRoleMember,
			}
		} else if member.Status == models.TaskForceMemberStatusActive {
			return errTaskForceAlreadyMember
		}

		member.Role = models.TaskForceMemberRoleMember
		member.Status = models.TaskForceMemberStatusActive
		member.AcceptedAt = &now
		if member.ID == uuid.Nil {
			if err := tx.Create(&member).Error; err != nil {
				return err
			}
		} else if err := tx.Save(&member).Error; err != nil {
			return err
		}

		invite.Status = models.TaskForceInviteStatusAccepted
		invite.UseCount++
		invite.AcceptedBy = &user.ID
		invite.AcceptedAt = &now
		if err := tx.Save(&invite).Error; err != nil {
			return err
		}

		joinedMember = member
		return nil
	})

	switch {
	case errors.Is(err, errTaskForceInviteExpired):
		c.JSON(http.StatusGone, gin.H{"error": "Invite has expired"})
		return
	case errors.Is(err, errTaskForceInviteInactive):
		c.JSON(http.StatusGone, gin.H{"error": "Invite is no longer valid"})
		return
	case errors.Is(err, errTaskForceNotHeadAdmin):
		c.JSON(http.StatusForbidden, gin.H{"error": "Only the invited unit's head admin can join"})
		return
	case errors.Is(err, errTaskForceAlreadyMember):
		c.JSON(http.StatusConflict, gin.H{"error": "Unit is already a member of this task force"})
		return
	case errors.Is(err, gorm.ErrRecordNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "Invite not found"})
		return
	case err != nil:
		log.Printf("taskforce: join failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to join task force"})
		return
	}

	_ = services.NewAuditService().LogAction(
		user.ID,
		"taskforce.unit_joined",
		"task_force",
		taskForce.ID.String(),
		nil,
		map[string]any{"unitId": joinedMember.UnitID.String()},
		c.ClientIP(),
		c.Request.UserAgent(),
	)

	c.JSON(http.StatusOK, gin.H{
		"message": "Unit joined the task force",
		"member":  joinedMember,
	})
}

var (
	errTaskForceInviteInactive = errors.New("task force invite inactive")
	errTaskForceInviteExpired  = errors.New("task force invite expired")
	errTaskForceNotHeadAdmin   = errors.New("caller is not head admin")
	errTaskForceAlreadyMember  = errors.New("unit is already a member")
	errTaskForceNotActive      = errors.New("task force is not active")
)

func EnlistTaskForceOfficers(c *gin.Context) {
	user, ok := getAuthUser(c)
	if !ok {
		return
	}

	taskForceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task force ID"})
		return
	}
	taskForce, err := loadTaskForce(taskForceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task force not found"})
		return
	}
	if taskForce.Status != models.TaskForceStatusActive {
		c.JSON(http.StatusConflict, gin.H{"error": "Task force is not active"})
		return
	}
	if !userIsHeadAdminOf(user, taskForce.HostUnitID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only the host unit's head admin can enlist officers"})
		return
	}

	var input struct {
		OfficerIDs []string `json:"officerIds" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "officerIds is required"})
		return
	}
	if len(input.OfficerIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "officerIds is required"})
		return
	}
	if len(input.OfficerIDs) > maxTaskForceEnlistBatch {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("at most %d officers per request", maxTaskForceEnlistBatch)})
		return
	}

	_, memberUnits, err := activeTaskForceUnitIDs(taskForce.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load task force units"})
		return
	}

	uniqueIDs := make([]uuid.UUID, 0, len(input.OfficerIDs))
	seen := make(map[uuid.UUID]bool, len(input.OfficerIDs))
	results := make([]taskForceEnlistResult, 0, len(input.OfficerIDs))

	for _, raw := range input.OfficerIDs {
		officerID, parseErr := uuid.Parse(strings.TrimSpace(raw))
		if parseErr != nil {
			results = append(results, taskForceEnlistResult{Status: "invalid_id", Message: "Invalid officer ID"})
			continue
		}
		if seen[officerID] {
			continue
		}
		seen[officerID] = true
		uniqueIDs = append(uniqueIDs, officerID)
	}

	var officers []models.Officer
	if len(uniqueIDs) > 0 {
		if err := config.DB.Where("id IN ?", uniqueIDs).Find(&officers).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load officers"})
			return
		}
	}
	officerByID := make(map[uuid.UUID]models.Officer, len(officers))
	for _, officer := range officers {
		officerByID[officer.ID] = officer
	}

	var activeRows []models.TaskForceOfficer
	if len(uniqueIDs) > 0 {
		if err := config.DB.
			Where("officer_id IN ? AND status = ?", uniqueIDs, models.TaskForceOfficerStatusActive).
			Find(&activeRows).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check enlistments"})
			return
		}
	}
	activeByOfficer := make(map[uuid.UUID]models.TaskForceOfficer, len(activeRows))
	for _, row := range activeRows {
		activeByOfficer[row.OfficerID] = row
	}

	var priorRows []models.TaskForceOfficer
	if len(uniqueIDs) > 0 {
		if err := config.DB.
			Where("task_force_id = ? AND officer_id IN ?", taskForce.ID, uniqueIDs).
			Find(&priorRows).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check prior enlistments"})
			return
		}
	}
	priorByOfficer := make(map[uuid.UUID]models.TaskForceOfficer, len(priorRows))
	for _, row := range priorRows {
		priorByOfficer[row.OfficerID] = row
	}

	now := time.Now().UTC()
	enlisted := make([]uuid.UUID, 0, len(uniqueIDs))

	for _, officerID := range uniqueIDs {
		officer, found := officerByID[officerID]
		if !found {
			results = append(results, taskForceEnlistResult{OfficerID: officerID, Status: "not_found"})
			continue
		}
		if officer.Status != "active" {
			results = append(results, taskForceEnlistResult{OfficerID: officerID, Status: "inactive"})
			continue
		}
		if !memberUnits[officer.UnitID] {
			results = append(results, taskForceEnlistResult{OfficerID: officerID, Status: "unit_not_member"})
			continue
		}
		if active, exists := activeByOfficer[officerID]; exists {
			if active.TaskForceID == taskForce.ID {
				results = append(results, taskForceEnlistResult{OfficerID: officerID, Status: "already_active"})
			} else {
				results = append(results, taskForceEnlistResult{OfficerID: officerID, Status: "conflict", Message: "Officer is already in another active task force"})
			}
			continue
		}

		if prior, exists := priorByOfficer[officerID]; exists {
			prior.Status = models.TaskForceOfficerStatusActive
			prior.UnitID = officer.UnitID
			prior.EnlistedAt = now
			prior.EnlistedBy = &user.ID
			prior.RemovedAt = nil
			if err := config.DB.Save(&prior).Error; err != nil {
				results = append(results, taskForceEnlistResult{OfficerID: officerID, Status: "conflict", Message: "Officer could not be enlisted"})
				continue
			}
			results = append(results, taskForceEnlistResult{OfficerID: officerID, Status: "enlisted"})
			enlisted = append(enlisted, officerID)
			continue
		}

		row := models.TaskForceOfficer{
			TaskForceID: taskForce.ID,
			OfficerID:   officer.ID,
			UnitID:      officer.UnitID,
			Status:      models.TaskForceOfficerStatusActive,
			EnlistedAt:  now,
			EnlistedBy:  &user.ID,
		}
		if err := config.DB.Create(&row).Error; err != nil {
			var active models.TaskForceOfficer
			if findErr := config.DB.
				Where("officer_id = ? AND status = ?", officerID, models.TaskForceOfficerStatusActive).
				First(&active).Error; findErr == nil {
				if active.TaskForceID == taskForce.ID {
					results = append(results, taskForceEnlistResult{OfficerID: officerID, Status: "already_active"})
				} else {
					results = append(results, taskForceEnlistResult{OfficerID: officerID, Status: "conflict", Message: "Officer is already in another active task force"})
				}
			} else {
				results = append(results, taskForceEnlistResult{OfficerID: officerID, Status: "error", Message: "Failed to enlist officer"})
			}
			continue
		}
		results = append(results, taskForceEnlistResult{OfficerID: officerID, Status: "enlisted"})
		enlisted = append(enlisted, officerID)
	}

	if len(enlisted) > 0 {
		_ = services.NewAuditService().LogAction(
			user.ID,
			"taskforce.officers_enlisted",
			"task_force",
			taskForce.ID.String(),
			nil,
			map[string]any{"officerIds": enlisted},
			c.ClientIP(),
			c.Request.UserAgent(),
		)
	}

	c.JSON(http.StatusOK, gin.H{
		"results":  results,
		"enlisted": len(enlisted),
	})
}

func DisbandTaskForce(c *gin.Context) {
	user, ok := getAuthUser(c)
	if !ok {
		return
	}

	taskForceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task force ID"})
		return
	}
	taskForce, err := loadTaskForce(taskForceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task force not found"})
		return
	}
	if !userIsHeadAdminOf(user, taskForce.HostUnitID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only the host unit's head admin can disband this task force"})
		return
	}
	if taskForce.Status != models.TaskForceStatusActive {
		c.JSON(http.StatusConflict, gin.H{"error": "Task force is already disbanded"})
		return
	}

	now := time.Now().UTC()
	err = config.DB.Transaction(func(tx *gorm.DB) error {
		var locked models.TaskForce
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&locked, "id = ?", taskForce.ID).Error; err != nil {
			return err
		}
		if locked.Status != models.TaskForceStatusActive {
			return errTaskForceAlreadyMember
		}
		if err := tx.Model(&locked).Update("status", models.TaskForceStatusDisbanded).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.TaskForceMember{}).
			Where("task_force_id = ? AND status IN ?", taskForce.ID,
				[]string{models.TaskForceMemberStatusActive, models.TaskForceMemberStatusInvited}).
			Update("status", models.TaskForceMemberStatusLeft).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.TaskForceOfficer{}).
			Where("task_force_id = ? AND status = ?", taskForce.ID, models.TaskForceOfficerStatusActive).
			Updates(map[string]any{
				"status":     models.TaskForceOfficerStatusRemoved,
				"removed_at": now,
			}).Error; err != nil {
			return err
		}
		var link models.TaskForceChannel
		if err := tx.Where("task_force_id = ?", taskForce.ID).First(&link).Error; err == nil {
			if err := tx.Model(&models.CommunicationRoom{}).
				Where("id = ?", link.RoomID).
				Update("status", "closed").Error; err != nil {
				return err
			}
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, errTaskForceAlreadyMember) {
			c.JSON(http.StatusConflict, gin.H{"error": "Task force is already disbanded"})
			return
		}
		log.Printf("taskforce: disband failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to disband task force"})
		return
	}

	_ = services.NewAuditService().LogAction(
		user.ID,
		"taskforce.disbanded",
		"task_force",
		taskForce.ID.String(),
		map[string]any{"status": models.TaskForceStatusActive},
		map[string]any{"status": models.TaskForceStatusDisbanded},
		c.ClientIP(),
		c.Request.UserAgent(),
	)
	c.JSON(http.StatusOK, gin.H{"message": "Task force disbanded"})
}

func LeaveTaskForce(c *gin.Context) {
	user, ok := getAuthUser(c)
	if !ok {
		return
	}

	taskForceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task force ID"})
		return
	}
	taskForce, err := loadTaskForce(taskForceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task force not found"})
		return
	}
	if taskForce.Status != models.TaskForceStatusActive {
		c.JSON(http.StatusConflict, gin.H{"error": "Task force is not active"})
		return
	}

	var input struct {
		UnitID string `json:"unitId"`
	}
	_ = c.ShouldBindJSON(&input)

	var unitID uuid.UUID
	if strings.TrimSpace(input.UnitID) != "" {
		unitID, err = uuid.Parse(input.UnitID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid unit ID"})
			return
		}
	} else if user.UnitID != nil {
		unitID = *user.UnitID
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unitId is required"})
		return
	}

	if !userIsHeadAdminOf(user, unitID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only the unit's head admin can leave a task force"})
		return
	}

	var member models.TaskForceMember
	if err := config.DB.
		Where("task_force_id = ? AND unit_id = ?", taskForce.ID, unitID).
		First(&member).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Unit is not a member of this task force"})
		return
	}
	if member.Role == models.TaskForceMemberRoleHost {
		c.JSON(http.StatusConflict, gin.H{"error": "The host unit cannot leave; disband the task force instead"})
		return
	}
	if member.Status != models.TaskForceMemberStatusActive {
		c.JSON(http.StatusConflict, gin.H{"error": "Unit is not an active member"})
		return
	}

	now := time.Now().UTC()
	err = config.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&member).Update("status", models.TaskForceMemberStatusLeft).Error; err != nil {
			return err
		}
		return tx.Model(&models.TaskForceOfficer{}).
			Where("task_force_id = ? AND unit_id = ? AND status = ?",
				taskForce.ID, unitID, models.TaskForceOfficerStatusActive).
			Updates(map[string]any{
				"status":     models.TaskForceOfficerStatusRemoved,
				"removed_at": now,
			}).Error
	})
	if err != nil {
		log.Printf("taskforce: leave failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to leave task force"})
		return
	}

	_ = services.NewAuditService().LogAction(
		user.ID,
		"taskforce.unit_left",
		"task_force",
		taskForce.ID.String(),
		map[string]any{"status": models.TaskForceMemberStatusActive},
		map[string]any{"unitId": unitID.String(), "status": models.TaskForceMemberStatusLeft},
		c.ClientIP(),
		c.Request.UserAgent(),
	)
	c.JSON(http.StatusOK, gin.H{"message": "Unit left the task force"})
}

func GetTaskForce(c *gin.Context) {
	user, ok := getAuthUser(c)
	if !ok {
		return
	}

	taskForceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task force ID"})
		return
	}
	taskForce, err := loadTaskForce(taskForceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task force not found"})
		return
	}
	if !canViewTaskForce(user, taskForce) {
		c.JSON(http.StatusForbidden, gin.H{"error": "You do not have access to this task force"})
		return
	}

	var members []models.TaskForceMember
	if err := config.DB.Preload("Unit").
		Where("task_force_id = ? AND status IN ?", taskForce.ID,
			[]string{models.TaskForceMemberStatusActive, models.TaskForceMemberStatusInvited}).
		Order("created_at asc").Find(&members).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load task force members"})
		return
	}

	var officers []models.TaskForceOfficer
	if err := config.DB.Preload("Officer").Preload("Unit").
		Where("task_force_id = ? AND status = ?", taskForce.ID, models.TaskForceOfficerStatusActive).
		Order("enlisted_at asc").Find(&officers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load enlisted officers"})
		return
	}

	var invites []models.TaskForceInvite
	if err := config.DB.
		Where("task_force_id = ? AND status = ?", taskForce.ID, models.TaskForceInviteStatusPending).
		Order("created_at asc").Find(&invites).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load invites"})
		return
	}
	inviteSummaries := make([]taskForceInviteSummary, 0, len(invites))
	for _, invite := range invites {
		inviteSummaries = append(inviteSummaries, taskForceInviteSummary{
			ID:        invite.ID,
			UnitID:    invite.UnitID,
			Status:    invite.Status,
			ExpiresAt: invite.ExpiresAt,
		})
	}

	var channel *models.CommunicationRoom
	if taskForce.Status == models.TaskForceStatusActive {
		room, channelErr := ensureTaskForceChannel(taskForce, user.ID)
		if channelErr != nil {
			log.Printf("taskforce: ensure channel failed: %v", channelErr)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to resolve task force channel"})
			return
		}
		channel = &room
	} else if room, channelErr := getTaskForceChannel(taskForce.ID); channelErr == nil {
		channel = &room
	}

	roomKey := taskForceRoomKey(taskForce.ID)
	response := gin.H{
		"taskForce":   taskForce,
		"members":     members,
		"officers":    officers,
		"invites":     inviteSummaries,
		"canManage":   taskForce.Status == models.TaskForceStatusActive && userIsHeadAdminOf(user, taskForce.HostUnitID),
		"canTransmit": canTransmitOnTaskForce(user, taskForce),
		"ptt":         ws.GetPTTManager().Snapshot(roomKey),
		"online":      ws.GetHub().RoomSize(roomKey),
	}
	if channel != nil {
		response["channel"] = channel
	}

	c.JSON(http.StatusOK, response)
}

func GetTaskForceChannel(c *gin.Context) {
	user, ok := getAuthUser(c)
	if !ok {
		return
	}

	taskForceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task force ID"})
		return
	}
	taskForce, err := loadTaskForce(taskForceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task force not found"})
		return
	}
	if !canAccessTaskForceChannel(user, taskForce) {
		c.JSON(http.StatusForbidden, gin.H{"error": "You do not have access to this task force channel"})
		return
	}

	room, err := ensureTaskForceChannel(taskForce, user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to resolve task force channel"})
		return
	}

	roomKey := taskForceRoomKey(taskForce.ID)
	c.JSON(http.StatusOK, gin.H{
		"channel":     room,
		"ptt":         ws.GetPTTManager().Snapshot(roomKey),
		"online":      ws.GetHub().RoomSize(roomKey),
		"canTransmit": canTransmitOnTaskForce(user, taskForce),
	})
}

func GetTaskForceOfficers(c *gin.Context) {
	user, ok := getAuthUser(c)
	if !ok {
		return
	}

	taskForceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task force ID"})
		return
	}
	taskForce, err := loadTaskForce(taskForceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task force not found"})
		return
	}
	if !userIsHeadAdminOf(user, taskForce.HostUnitID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only the host unit's head admin can view member rosters"})
		return
	}

	unitIDs, memberSet, err := activeTaskForceUnitIDs(taskForce.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load task force units"})
		return
	}

	if raw := strings.TrimSpace(c.Query("unitId")); raw != "" {
		filterID, parseErr := uuid.Parse(raw)
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid unit ID"})
			return
		}
		if !memberSet[filterID] {
			c.JSON(http.StatusForbidden, gin.H{"error": "Unit is not a member of this task force"})
			return
		}
		unitIDs = []uuid.UUID{filterID}
	}

	var officers []models.Officer
	if len(unitIDs) > 0 {
		if err := config.DB.Preload("Unit").
			Where("unit_id IN ? AND status = ?", unitIDs, "active").
			Order("name asc").Find(&officers).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load officers"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"officers": officers})
}

func ListMyTaskForces(c *gin.Context) {
	user, ok := getAuthUser(c)
	if !ok {
		return
	}

	if user.IsSuperAdmin || user.Role == models.RoleSuperAdmin {
		var taskForces []models.TaskForce
		if err := config.DB.Preload("HostUnit").
			Where("status = ?", models.TaskForceStatusActive).
			Order("created_at desc").Find(&taskForces).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load task forces"})
			return
		}
		summaries := make([]taskForceSummary, 0, len(taskForces))
		for _, taskForce := range taskForces {
			summaries = append(summaries, taskForceSummary{
				TaskForce:        taskForce,
				Role:             "super_admin",
				MembershipStatus: models.TaskForceMemberStatusActive,
			})
		}
		c.JSON(http.StatusOK, gin.H{"taskForces": summaries})
		return
	}

	var memberships []models.UnitMembership
	if err := config.DB.
		Where("user_id = ? AND status = ?", user.ID, models.MembershipActive).
		Find(&memberships).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load unit memberships"})
		return
	}
	unitIDs := make([]uuid.UUID, 0, len(memberships)+1)
	seenUnits := make(map[uuid.UUID]bool, len(memberships)+1)
	for _, membership := range memberships {
		if !seenUnits[membership.UnitID] {
			seenUnits[membership.UnitID] = true
			unitIDs = append(unitIDs, membership.UnitID)
		}
	}
	if user.UnitID != nil && !seenUnits[*user.UnitID] {
		unitIDs = append(unitIDs, *user.UnitID)
	}
	if len(unitIDs) == 0 {
		c.JSON(http.StatusOK, gin.H{"taskForces": []taskForceSummary{}})
		return
	}

	var members []models.TaskForceMember
	if err := config.DB.Preload("TaskForce.HostUnit").
		Where("unit_id IN ? AND status IN ?", unitIDs,
			[]string{models.TaskForceMemberStatusActive, models.TaskForceMemberStatusInvited}).
		Order("created_at desc").Find(&members).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load task forces"})
		return
	}
	summaries := make([]taskForceSummary, 0, len(members))
	seenTaskForces := make(map[uuid.UUID]bool, len(members))
	for _, member := range members {
		if member.TaskForce.Status != models.TaskForceStatusActive {
			continue
		}
		if seenTaskForces[member.TaskForceID] {
			continue
		}
		seenTaskForces[member.TaskForceID] = true
		summaries = append(summaries, taskForceSummary{
			TaskForce:        member.TaskForce,
			Role:             member.Role,
			MembershipStatus: member.Status,
		})
	}
	c.JSON(http.StatusOK, gin.H{"taskForces": summaries})
}
