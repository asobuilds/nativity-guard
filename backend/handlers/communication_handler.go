package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"security-solution/config"
	"security-solution/models"
)

// CreateRoom creates a communication room
func CreateRoom(c *gin.Context) {
	var input struct {
		UnitID      string `json:"unitId" binding:"required"`
		Name        string `json:"name" binding:"required"`
		Type        string `json:"type"`
		IsEncrypted bool   `json:"isEncrypted"`
		Priority    string `json:"priority"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj := user.(*models.User)

	if userObj.Role != "super_admin" && userObj.Role != "unit_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only unit admins can create communication rooms"})
		return
	}

	unitID, err := uuid.Parse(input.UnitID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid unit ID"})
		return
	}

	if userObj.Role != "super_admin" && (userObj.UnitID == nil || *userObj.UnitID != unitID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only create rooms for your unit"})
		return
	}

	if input.Type == "" {
		input.Type = "unit"
	}
	if input.Priority == "" {
		input.Priority = "normal"
	}

	room := models.CommunicationRoom{
		UnitID:      unitID,
		Name:        input.Name,
		Type:        input.Type,
		Status:      "active",
		IsEncrypted: input.IsEncrypted,
		Priority:    input.Priority,
		CreatedBy:   userObj.ID,
	}

	if err := config.DB.Create(&room).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create room"})
		return
	}

	// The creator becomes the first member of the room.
	now := time.Now()
	member := models.CommunicationRoomMember{
		RoomID:   room.ID,
		UserID:   userObj.ID,
		Role:     "admin",
		Status:   "active",
		JoinedAt: &now,
	}
	config.DB.Create(&member)

	c.JSON(http.StatusCreated, gin.H{
		"message": "Room created successfully",
		"room":    room,
	})
}

// GetRooms gets all rooms for a unit
func GetRooms(c *gin.Context) {
	unitID := c.Param("id")
	id, err := uuid.Parse(unitID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid unit ID"})
		return
	}

	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj := user.(*models.User)

	if userObj.Role != "super_admin" && userObj.Role != "unit_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to view rooms"})
		return
	}

	if userObj.Role != "super_admin" && (userObj.UnitID == nil || *userObj.UnitID != id) {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have access to this unit"})
		return
	}

	var rooms []models.CommunicationRoom
	if err := config.DB.Where("unit_id = ?", id).Order("created_at desc").Find(&rooms).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch rooms"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"rooms": rooms,
	})
}

// SendMessage sends a message to a room with offline support
func SendMessage(c *gin.Context) {
	var input struct {
		RoomID      string `json:"roomId" binding:"required"`
		Message     string `json:"message" binding:"required"`
		MessageType string `json:"messageType"`
		IsBroadcast bool   `json:"isBroadcast"`
		IsEmergency bool   `json:"isEmergency"`
		Priority    string `json:"priority"`
		OfflineID   string `json:"offlineId"` // For offline sync
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj := user.(*models.User)

	roomID, err := uuid.Parse(input.RoomID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid room ID"})
		return
	}

	if input.MessageType == "" {
		input.MessageType = "text"
	}
	if input.Priority == "" {
		input.Priority = "normal"
	}
	if input.IsEmergency {
		input.Priority = "emergency"
	}

	message := models.CommunicationMessage{
		RoomID:      roomID,
		SenderID:    userObj.ID,
		Message:     input.Message,
		MessageType: input.MessageType,
		Status:      "sent",
		IsBroadcast: input.IsBroadcast,
		IsEmergency: input.IsEmergency,
		Priority:    input.Priority,
	}

	if !roomAccessAllowed(c, userObj, roomID) {
		return
	}

	if err := config.DB.Create(&message).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send message"})
		return
	}

	// If emergency, create alert
	if input.IsEmergency {
		go createEmergencyAlert(roomID, message)
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Message sent successfully",
		"data":    message,
	})
}

// GetMessages gets messages for a room with offline sync support
func GetMessages(c *gin.Context) {
	roomID := c.Param("roomId")
	id, err := uuid.Parse(roomID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid room ID"})
		return
	}

	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj := user.(*models.User)

	if !roomAccessAllowed(c, userObj, id) {
		return
	}

	// Get last sync time from query
	lastSync := c.Query("lastSync")
	var lastSyncTime time.Time
	if lastSync != "" {
		parsed, err := time.Parse(time.RFC3339, lastSync)
		if err == nil {
			lastSyncTime = parsed
		}
	}

	// Get messages since last sync
	var messages []models.CommunicationMessage
	query := config.DB.Preload("Sender").Where("room_id = ?", id)

	if !lastSyncTime.IsZero() {
		query = query.Where("created_at > ?", lastSyncTime)
	}

	if err := query.Order("created_at asc").Find(&messages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch messages"})
		return
	}

	// Update sync status
	sync := models.CommunicationSync{
		UserID:       userObj.ID,
		RoomID:       id,
		LastSyncAt:   time.Now(),
		SyncStatus:   "completed",
		PendingCount: 0,
	}
	config.DB.Create(&sync)

	c.JSON(http.StatusOK, gin.H{
		"messages": messages,
		"syncTime": time.Now().Format(time.RFC3339),
		"hasMore":  len(messages) > 0,
	})
}

// InitiateCall initiates a voice call
func InitiateCall(c *gin.Context) {
	var input struct {
		RoomID     string `json:"roomId" binding:"required"`
		ReceiverID string `json:"receiverId"`
		CallType   string `json:"callType"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj := user.(*models.User)

	roomID, err := uuid.Parse(input.RoomID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid room ID"})
		return
	}

	if !roomAccessAllowed(c, userObj, roomID) {
		return
	}

	if input.CallType == "" {
		input.CallType = "group"
	}

	// Group calls have no single receiver, so the field stays NULL.
	var receiverID *uuid.UUID
	if input.ReceiverID != "" {
		parsed, err := uuid.Parse(input.ReceiverID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid receiver ID"})
			return
		}
		receiverID = &parsed
	}

	now := time.Now()
	call := models.VoiceCall{
		RoomID:      roomID,
		CallerID:    userObj.ID,
		ReceiverID:  receiverID,
		CallType:    input.CallType,
		Status:      "initiated",
		StartedAt:   &now,
		IsEncrypted: true,
		Quality:     "good",
	}

	if err := config.DB.Create(&call).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initiate call"})
		return
	}

	// Broadcast call notification via WebSocket
	go broadcastCallNotification(call)

	c.JSON(http.StatusCreated, gin.H{
		"message": "Call initiated",
		"call":    call,
	})
}

// EndCall ends a voice call
func EndCall(c *gin.Context) {
	callID := c.Param("id")
	id, err := uuid.Parse(callID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid call ID"})
		return
	}

	var input struct {
		Quality string `json:"quality"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj := user.(*models.User)

	var call models.VoiceCall
	if err := config.DB.First(&call, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Call not found"})
		return
	}

	now := time.Now()
	if call.CallerID != userObj.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only the call initiator can end the call"})
		return
	}
	call.Status = "ended"
	call.EndedAt = &now
	if call.StartedAt != nil {
		call.Duration = int(now.Sub(*call.StartedAt).Seconds())
	}
	if input.Quality != "" {
		call.Quality = input.Quality
	}

	if err := config.DB.Save(&call).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to end call"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Call ended",
		"call":    call,
	})
}

// SyncMessages syncs offline messages for rural areas
func SyncMessages(c *gin.Context) {
	var input struct {
		RoomID          string `json:"roomId" binding:"required"`
		LastSync        string `json:"lastSync"`
		OfflineMessages []struct {
			ID          string `json:"id"`
			Message     string `json:"message"`
			MessageType string `json:"messageType"`
			CreatedAt   string `json:"createdAt"`
		} `json:"offlineMessages"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj := user.(*models.User)

	roomID, err := uuid.Parse(input.RoomID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid room ID"})
		return
	}

	if !roomAccessAllowed(c, userObj, roomID) {
		return
	}

	// Process offline messages
	var syncedCount int
	for _, offlineMsg := range input.OfflineMessages {
		msgTime, _ := time.Parse(time.RFC3339, offlineMsg.CreatedAt)

		message := models.CommunicationMessage{
			RoomID:      roomID,
			SenderID:    userObj.ID,
			Message:     offlineMsg.Message,
			MessageType: offlineMsg.MessageType,
			Status:      "sent",
			CreatedAt:   msgTime,
		}

		if err := config.DB.Create(&message).Error; err == nil {
			syncedCount++
		}
	}

	// Update sync record
	sync := models.CommunicationSync{
		UserID:       userObj.ID,
		RoomID:       roomID,
		LastSyncAt:   time.Now(),
		SyncStatus:   "completed",
		PendingCount: 0,
	}
	config.DB.Create(&sync)

	// Get new messages since last sync
	var newMessages []models.CommunicationMessage
	var lastSyncTime time.Time
	if input.LastSync != "" {
		parsed, _ := time.Parse(time.RFC3339, input.LastSync)
		lastSyncTime = parsed
	}

	query := config.DB.Preload("Sender").Where("room_id = ?", roomID)
	if !lastSyncTime.IsZero() {
		query = query.Where("created_at > ?", lastSyncTime)
	}
	query.Order("created_at asc").Find(&newMessages)

	c.JSON(http.StatusOK, gin.H{
		"message":     "Sync completed",
		"syncedCount": syncedCount,
		"newMessages": newMessages,
		"syncTime":    time.Now().Format(time.RFC3339),
	})
}

// GetSyncStatus gets sync status for offline users
func GetSyncStatus(c *gin.Context) {
	roomID := c.Param("roomId")
	id, err := uuid.Parse(roomID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid room ID"})
		return
	}

	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj := user.(*models.User)

	if !roomAccessAllowed(c, userObj, id) {
		return
	}

	var sync models.CommunicationSync
	if err := config.DB.Where("user_id = ? AND room_id = ?", userObj.ID, id).Order("created_at desc").First(&sync).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{
			"hasSynced": false,
			"message":   "No sync record found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"hasSynced":    true,
		"lastSyncAt":   sync.LastSyncAt,
		"syncStatus":   sync.SyncStatus,
		"pendingCount": sync.PendingCount,
	})
}

// GetRoomMembers lists the members of a room.
func GetRoomMembers(c *gin.Context) {
	roomIDStr := c.Param("roomId")
	roomID, err := uuid.Parse(roomIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid room ID"})
		return
	}

	userObj, ok := getAuthUser(c)
	if !ok {
		return
	}

	if !roomAccessAllowed(c, userObj, roomID) {
		return
	}

	var members []models.CommunicationRoomMember
	if err := config.DB.Preload("User").Where("room_id = ?", roomID).Order("created_at asc").Find(&members).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch room members"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"members": members})
}

// AddRoomMember adds a user to a room.
func AddRoomMember(c *gin.Context) {
	roomIDStr := c.Param("roomId")
	roomID, err := uuid.Parse(roomIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid room ID"})
		return
	}

	var input struct {
		UserID string `json:"userId" binding:"required"`
		Role   string `json:"role"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	targetID, err := uuid.Parse(input.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	userObj, ok := getAuthUser(c)
	if !ok {
		return
	}

	if !canManageRoom(c, userObj, roomID) {
		return
	}

	// The member must belong to the room's unit.
	var room models.CommunicationRoom
	if err := config.DB.First(&room, "id = ?", roomID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Room not found"})
		return
	}

	var target models.User
	if err := config.DB.First(&target, "id = ?", targetID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if target.UnitID == nil || *target.UnitID != room.UnitID {
		c.JSON(http.StatusForbidden, gin.H{"error": "User does not belong to this room's unit"})
		return
	}

	role := input.Role
	if role == "" {
		role = "member"
	}
	if role != "admin" && role != "member" && role != "observer" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid member role"})
		return
	}

	// Re-adding an existing member reactivates the row.
	var existing models.CommunicationRoomMember
	if err := config.DB.Where("room_id = ? AND user_id = ?", roomID, targetID).First(&existing).Error; err == nil {
		now := time.Now()
		existing.Status = "active"
		existing.Role = role
		existing.LeftAt = nil
		existing.JoinedAt = &now
		if err := config.DB.Save(&existing).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add room member"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Member added to room", "member": existing})
		return
	}

	now := time.Now()
	member := models.CommunicationRoomMember{
		RoomID:   roomID,
		UserID:   targetID,
		Role:     role,
		Status:   "active",
		JoinedAt: &now,
	}
	if err := config.DB.Create(&member).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add room member"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Member added to room", "member": member})
}

// RemoveRoomMember removes a user from a room.
func RemoveRoomMember(c *gin.Context) {
	roomIDStr := c.Param("roomId")
	roomID, err := uuid.Parse(roomIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid room ID"})
		return
	}

	targetID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	userObj, ok := getAuthUser(c)
	if !ok {
		return
	}

	if !canManageRoom(c, userObj, roomID) {
		return
	}

	var member models.CommunicationRoomMember
	if err := config.DB.Where("room_id = ? AND user_id = ?", roomID, targetID).First(&member).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Member not found"})
		return
	}

	now := time.Now()
	member.Status = "removed"
	member.LeftAt = &now
	if err := config.DB.Save(&member).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove room member"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Member removed from room", "member": member})
}

// Helper functions

// createEmergencyAlert raises a unit-scoped emergency alert for the room the
// message was sent in. The alert inherits the room's unit so it is never
// broadcast across unit boundaries.
func createEmergencyAlert(roomID uuid.UUID, message models.CommunicationMessage) {
	var room models.CommunicationRoom
	if err := config.DB.First(&room, "id = ?", roomID).Error; err != nil {
		return
	}

	alert := models.CommunityAlert{
		Title:     "🚨 Emergency Communication",
		Content:   message.Message,
		Type:      "emergency",
		Severity:  "critical",
		UnitID:    &room.UnitID,
		Location:  room.Name,
		Status:    "active",
		CreatedBy: message.SenderID,
	}
	config.DB.Create(&alert)
}

func broadcastCallNotification(call models.VoiceCall) {
	// Broadcast via WebSocket
	// This will be implemented with the WebSocket hub
}
