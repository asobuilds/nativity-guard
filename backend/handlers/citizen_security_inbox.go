package handlers

import (
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"security-solution/config"
	"security-solution/models"
)

func OpenCitizenSecurityConversation(c *gin.Context) {
	user, ok := getAuthUser(c)
	if !ok { return }
	if user.Role != "citizen" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only citizens can open a citizen-to-security conversation"})
		return
	}
	var input struct {
		UnitID string `json:"unitId" binding:"required"`
		Message string `json:"message" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	unitID, err := uuid.Parse(input.UnitID)
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid security unit"}); return }
	content := strings.TrimSpace(input.Message)
	if content == "" || !utf8.ValidString(content) || utf8.RuneCountInString(content) > 4000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Message must contain 1 to 4000 valid characters"})
		return
	}
	var unit models.SecurityUnit
	if err := config.DB.Where("id = ? AND is_verified = ? AND status = ?", unitID, true, "active").First(&unit).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Verified active security unit not found"})
		return
	}
	var recipientIDs []uuid.UUID
	if err := config.DB.Table("unit_memberships").
		Select("unit_memberships.user_id").
		Joins("JOIN users ON users.id = unit_memberships.user_id").
		Where("unit_memberships.unit_id = ? AND unit_memberships.status = ? AND unit_memberships.role IN ? AND users.role IN ? AND users.status = ? AND users.deleted_at IS NULL",
			unit.ID, models.MembershipActive, []string{models.UnitRoleAdmin, models.UnitRoleOfficer}, []string{"unit_admin", "officer"}, "active").
		Scan(&recipientIDs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not resolve security recipients"})
		return
	}
	seen := map[uuid.UUID]bool{user.ID: true}
	filtered := make([]uuid.UUID, 0, len(recipientIDs)+1)
	for _, recipientID := range recipientIDs {
		if !seen[recipientID] { filtered = append(filtered, recipientID); seen[recipientID] = true }
	}
	recipientIDs = filtered
	if unit.HostUserID != nil && !seen[*unit.HostUserID] {
		var host models.User
		if err := config.DB.Where("id = ? AND role IN ? AND status = ? AND deleted_at IS NULL", *unit.HostUserID, []string{"unit_admin", "officer"}, "active").First(&host).Error; err == nil {
			recipientIDs = append(recipientIDs, host.ID)
		}
	}
	if len(recipientIDs) == 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "This security unit has no active messaging recipients yet"})
		return
	}

	var room models.CommunicationRoom
	var message models.CommunicationMessage
	now := time.Now()
	err = config.DB.Transaction(func(tx *gorm.DB) error {
		room = models.CommunicationRoom{
			UnitID: unit.ID, Name: "Citizen support", Type: "private",
			Status: "active", IsEncrypted: false, Priority: "normal", CreatedBy: user.ID,
		}
		if err := tx.Create(&room).Error; err != nil { return err }
		participants := append([]uuid.UUID{user.ID}, recipientIDs...)
		for _, participantID := range participants {
			member := models.CommunicationRoomMember{RoomID: room.ID, UserID: participantID, Role: "member", Status: "active", JoinedAt: &now}
			if err := tx.Create(&member).Error; err != nil { return err }
		}
		message = models.CommunicationMessage{
			RoomID: room.ID, SenderID: user.ID, Message: content,
			MessageType: "text", Status: "sent", Priority: "normal",
		}
		return tx.Create(&message).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not open the private conversation"})
		return
	}
	for _, recipientID := range recipientIDs {
		n := models.Notification{
			UserID: recipientID, Title: "New citizen support request",
			Message: "A citizen has opened a private conversation with your security unit.",
			Type: "communication", Status: "unread",
			EntityType: "communication", EntityID: room.ID, LinkTo: "/messages?roomId=" + room.ID.String(),
		}
		if config.DB.Create(&n).Error == nil { go dispatchWebPush(n.UserID, n.Title, n.Message) }
	}
	c.JSON(http.StatusCreated, gin.H{
		"room": gin.H{"id": room.ID, "unitId": unit.ID, "unitName": unit.Name, "type": room.Type},
		"message": gin.H{"id": message.ID, "roomId": message.RoomID, "senderId": message.SenderID, "message": message.Message, "createdAt": message.CreatedAt},
	})
}

func GetMyCommunicationRooms(c *gin.Context) {
	user, ok := getAuthUser(c)
	if !ok { return }
	var rooms []models.CommunicationRoom
	if err := config.DB.
		Joins("JOIN communication_room_members m ON m.room_id = communication_rooms.id").
		Where("m.user_id = ? AND m.status = ? AND communication_rooms.status = ?", user.ID, "active", "active").
		Preload("Unit").
		Order("communication_rooms.updated_at DESC").
		Find(&rooms).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not load your conversations"})
		return
	}
	out := make([]gin.H, 0, len(rooms))
	for _, room := range rooms {
		if room.Type == "private" {
			out = append(out, gin.H{"id": room.ID, "name": room.Name, "type": room.Type, "unitId": room.UnitID, "unitName": room.Unit.Name})
		} else if user.Role == "officer" || user.Role == "unit_admin" || user.Role == "super_admin" {
			out = append(out, gin.H{"id": room.ID, "name": room.Name, "type": room.Type, "unitId": room.UnitID, "unitName": room.Unit.Name})
		}
	}
	if user.Role == "officer" || user.Role == "unit_admin" || user.Role == "super_admin" {
		if user.UnitID != nil {
			var unitRooms []models.CommunicationRoom
			if err := config.DB.Preload("Unit").Where("unit_id = ? AND status = ? AND type NOT IN ?", *user.UnitID, "active", []string{"private", models.TaskForceRoomType}).Order("created_at DESC").Find(&unitRooms).Error; err == nil {
				for _, room := range unitRooms {
					out = append(out, gin.H{"id": room.ID, "name": room.Name, "type": room.Type, "unitId": room.UnitID, "unitName": room.Unit.Name})
				}
			}
			var taskForceMembers []models.TaskForceMember
			if err := config.DB.Where("unit_id = ? AND status = ?", *user.UnitID, models.TaskForceMemberStatusActive).Find(&taskForceMembers).Error; err == nil {
				for _, member := range taskForceMembers {
					var channel models.TaskForceChannel
					if err := config.DB.Where("task_force_id = ?", member.TaskForceID).First(&channel).Error; err != nil { continue }
					var taskForce models.TaskForce
					if err := config.DB.First(&taskForce, "id = ?", member.TaskForceID).Error; err != nil { continue }
					if !canTransmitOnTaskForce(user, &taskForce) { continue }
					var room models.CommunicationRoom
					if err := config.DB.Preload("Unit").First(&room, "id = ?", channel.RoomID).Error; err != nil { continue }
					out = append(out, gin.H{"id": room.ID, "name": taskForce.Name, "type": models.TaskForceRoomType, "unitId": room.UnitID, "unitName": room.Unit.Name})
				}
			}
		}
	}
	if user.Role == "super_admin" {
		var channels []models.TaskForceChannel
		if err := config.DB.Find(&channels).Error; err == nil {
			for _, channel := range channels {
				var taskForce models.TaskForce
				if err := config.DB.First(&taskForce, "id = ? AND status = ?", channel.TaskForceID, models.TaskForceStatusActive).Error; err != nil { continue }
				var room models.CommunicationRoom
				if err := config.DB.Preload("Unit").First(&room, "id = ?", channel.RoomID).Error; err != nil { continue }
				out = append(out, gin.H{"id": room.ID, "name": taskForce.Name, "type": models.TaskForceRoomType, "unitId": room.UnitID, "unitName": room.Unit.Name})
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{"rooms": out})
}

func notifyCommunicationRoomMembers(roomID, senderID uuid.UUID) {
	var room models.CommunicationRoom
	if err := config.DB.First(&room, "id = ?", roomID).Error; err != nil { return }
	recipients := make(map[uuid.UUID]bool)
	add := func(id uuid.UUID) {
		if id != uuid.Nil && id != senderID { recipients[id] = true }
	}
	if room.Type == "private" {
		var members []models.CommunicationRoomMember
		if err := config.DB.Where("room_id = ? AND status = ?", roomID, "active").Find(&members).Error; err != nil { return }
		for _, member := range members { add(member.UserID) }
	} else if room.Type == models.TaskForceRoomType {
		taskForce, err := taskForceForRoom(roomID)
		if err != nil { return }
		var unitMembers []models.TaskForceMember
		if err := config.DB.Where("task_force_id = ? AND status = ?", taskForce.ID, models.TaskForceMemberStatusActive).Find(&unitMembers).Error; err != nil { return }
		for _, unitMember := range unitMembers {
			var staff []models.UnitMembership
			if err := config.DB.Where("unit_id = ? AND status = ? AND role IN ?", unitMember.UnitID, models.MembershipActive, []string{models.UnitRoleAdmin, models.UnitRoleOfficer}).Find(&staff).Error; err == nil {
				for _, member := range staff { add(member.UserID) }
			}
		}
	} else {
		var staff []models.UnitMembership
		if err := config.DB.Where("unit_id = ? AND status = ? AND role IN ?", room.UnitID, models.MembershipActive, []string{models.UnitRoleAdmin, models.UnitRoleOfficer}).Find(&staff).Error; err == nil {
			for _, member := range staff { add(member.UserID) }
		}
		var explicit []models.CommunicationRoomMember
		if err := config.DB.Where("room_id = ? AND status = ?", roomID, "active").Find(&explicit).Error; err == nil {
			for _, member := range explicit { add(member.UserID) }
		}
	}
	for recipientID := range recipients {
		n := models.Notification{
			UserID: recipientID, Title: "New secure message",
			Message: "You have a new message in an authorised security conversation.",
			Type: "communication", Status: "unread",
			EntityType: "communication", EntityID: roomID,
			LinkTo: "/messages?roomId=" + roomID.String(),
		}
		if config.DB.Create(&n).Error == nil { go dispatchWebPush(n.UserID, n.Title, n.Message) }
	}
}
