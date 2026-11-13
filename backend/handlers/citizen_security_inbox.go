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
	var memberships []models.UnitMembership
	if err := config.DB.Where("unit_id = ? AND status = ? AND role IN ?", unit.ID, models.MembershipActive, []string{models.UnitRoleAdmin, models.UnitRoleOfficer}).Find(&memberships).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not resolve security recipients"})
		return
	}
	recipientIDs := make([]uuid.UUID, 0, len(memberships)+1)
	seen := map[uuid.UUID]bool{user.ID: true}
	for _, membership := range memberships {
		if !seen[membership.UserID] { recipientIDs = append(recipientIDs, membership.UserID); seen[membership.UserID] = true }
	}
	if unit.HostUserID != nil && !seen[*unit.HostUserID] {
		recipientIDs = append(recipientIDs, *unit.HostUserID)
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
		}
	}
	c.JSON(http.StatusOK, gin.H{"rooms": out})
}
