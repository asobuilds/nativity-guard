package handlers

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"security-solution/config"
	"security-solution/models"
)

// Citizen chat is intentionally separate from the forum and from security
// communication rooms. Geographic access is recalculated on every request.
const citizenChatMessageLimit = 4000

func validCitizenChatScope(scope string) bool {
	scope = strings.ToLower(strings.TrimSpace(scope))
	return scope == ScopeLocality || scope == ScopeLGA || scope == ScopeState
}

func encodeCitizenChatCursor(at time.Time, id uuid.UUID) string {
	return base64.RawURLEncoding.EncodeToString([]byte(at.Format(time.RFC3339Nano) + "|" + id.String()))
}

func decodeCitizenChatCursor(raw string) (time.Time, uuid.UUID, error) {
	var zero uuid.UUID
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil { return time.Time{}, zero, err }
	parts := strings.Split(string(decoded), "|")
	if len(parts) != 2 { return time.Time{}, zero, errors.New("invalid cursor fields") }
	at, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil { return time.Time{}, zero, err }
	id, err := uuid.Parse(parts[1])
	if err != nil { return time.Time{}, zero, err }
	return at, id, nil
}

func citizenChatMessageResponse(m models.CitizenChatMessage) gin.H {
	return gin.H{
		"id": m.ID, "roomId": m.RoomID, "senderId": m.SenderID,
		"sender": strings.TrimSpace(m.Sender.FirstName + " " + m.Sender.LastName),
		"content": m.Content, "createdAt": m.CreatedAt,
	}
}

func citizenChatRoomAccess(c *gin.Context, user *models.User, roomID uuid.UUID) (*models.CitizenChatRoom, bool) {
	var room models.CitizenChatRoom
	if err := config.DB.First(&room, "id = ? AND status = ?", roomID, "active").Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chat room not found"})
		return nil, false
	}
	place := resolveViewerPlace(user, nil, nil)
	if !placeVisibleToViewer(room.GeoPlace, place) {
		c.JSON(http.StatusForbidden, gin.H{"error": "This chat is outside your resolved area"})
		return nil, false
	}
	var member models.CitizenChatRoomMember
	if err := config.DB.Where("room_id = ? AND user_id = ? AND status = ?", room.ID, user.ID, "active").First(&member).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Join this local chat before accessing its messages"})
		return nil, false
	}
	return &room, true
}

// JoinCitizenChatRoom creates or joins the caller's own locality/LGA/state room.
// Place names come only from resolveViewerPlace, never from request fields.
func JoinCitizenChatRoom(c *gin.Context) {
	user, ok := getAuthUser(c)
	if !ok { return }
	if user.Role != "citizen" && user.Role != "unit_admin" && user.Role != "super_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Community chat is available to citizens and authorised community moderators"})
		return
	}
	var input struct {
		Scope string `json:"scope" binding:"required"`
		Latitude *float64 `json:"latitude"`
		Longitude *float64 `json:"longitude"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	scope := strings.ToLower(strings.TrimSpace(input.Scope))
	if !validCitizenChatScope(scope) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chat scope must be locality, lga, or state"})
		return
	}
	if (input.Latitude == nil) != (input.Longitude == nil) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Latitude and longitude must be supplied together"})
		return
	}
	if input.Latitude != nil && (*input.Latitude < -90 || *input.Latitude > 90 || *input.Longitude < -180 || *input.Longitude > 180) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Coordinates are outside valid bounds"})
		return
	}
	place := resolveViewerPlace(user, input.Latitude, input.Longitude)
	if !scopeSatisfiedBy(scope, place) {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Your location does not contain enough verified detail for this chat scope"})
		return
	}
	geo := newPlace(scope, place, input.Latitude, input.Longitude)
	var room models.CitizenChatRoom
	query := config.DB.Where("scope = ? AND state = ? AND lga = ? AND locality = ? AND country = ?",
		geo.Scope, geo.State, geo.LGA, geo.Locality, geo.Country)
	if err := query.First(&room).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		room = models.CitizenChatRoom{GeoPlace: geo, Status: "active"}
		if createErr := config.DB.Create(&room).Error; createErr != nil {
			// A concurrent join may have created the same geographic room.
			if retryErr := query.First(&room).Error; retryErr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not open the local chat"})
				return
			}
		}
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not open the local chat"})
		return
	}
	member := models.CitizenChatRoomMember{RoomID: room.ID, UserID: user.ID, Status: "active", JoinedAt: time.Now()}
	if err := config.DB.Where("room_id = ? AND user_id = ?", room.ID, user.ID).FirstOrCreate(&member).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not join the local chat"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"room": gin.H{
		"id": room.ID, "scope": room.Scope, "locality": room.Locality,
		"lga": room.LGA, "state": room.State, "country": room.Country,
	}})
}

func GetCitizenChatMessages(c *gin.Context) {
	user, ok := getAuthUser(c)
	if !ok { return }
	roomID, err := uuid.Parse(c.Param("roomId"))
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid room ID"}); return }
	if _, ok := citizenChatRoomAccess(c, user, roomID); !ok { return }

	limit := 30
	if raw := c.Query("limit"); raw != "" {
		n, parseErr := strconv.Atoi(raw)
		if parseErr != nil || n < 1 || n > 50 { c.JSON(http.StatusBadRequest, gin.H{"error": "Limit must be between 1 and 50"}); return }
		limit = n
	}
	query := config.DB.Preload("Sender").Where("room_id = ? AND status = ?", roomID, "published")
	// A block is private to the blocker, but content from either side is hidden
	// from the other participant as soon as the block is active.
	query = query.Where("sender_id NOT IN (SELECT blocked_id FROM citizen_chat_blocks WHERE blocker_id = ? AND deleted_at IS NULL)", user.ID)
	query = query.Where("sender_id NOT IN (SELECT blocker_id FROM citizen_chat_blocks WHERE blocked_id = ? AND deleted_at IS NULL)", user.ID)
	if cursor := c.Query("before"); cursor != "" {
		at, id, decodeErr := decodeCitizenChatCursor(cursor)
		if decodeErr != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid message cursor"}); return }
		query = query.Where("(created_at, id) < (?, ?)", at, id)
	}
	var messages []models.CitizenChatMessage
	if err := query.Order("created_at DESC, id DESC").Limit(limit + 1).Find(&messages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not load chat history"})
		return
	}
	hasMore := len(messages) > limit
	if hasMore { messages = messages[:limit] }
	response := make([]gin.H, 0, len(messages))
	for _, message := range messages { response = append(response, citizenChatMessageResponse(message)) }
	next := ""
	if hasMore && len(messages) > 0 {
		last := messages[len(messages)-1]
		next = encodeCitizenChatCursor(last.CreatedAt, last.ID)
	}
	c.JSON(http.StatusOK, gin.H{"messages": response, "nextCursor": next, "hasMore": hasMore})
}

func SendCitizenChatMessage(c *gin.Context) {
	user, ok := getAuthUser(c)
	if !ok { return }
	roomID, err := uuid.Parse(c.Param("roomId"))
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid room ID"}); return }
	if _, ok := citizenChatRoomAccess(c, user, roomID); !ok { return }
	var input struct { Content string `json:"content" binding:"required"` }
	if err := c.ShouldBindJSON(&input); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	content := strings.TrimSpace(input.Content)
	if content == "" || !utf8.ValidString(content) || utf8.RuneCountInString(content) > citizenChatMessageLimit {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Message must contain 1 to %d valid characters", citizenChatMessageLimit)})
		return
	}
	message := models.CitizenChatMessage{RoomID: roomID, SenderID: user.ID, Content: content, Status: "published"}
	if err := config.DB.Create(&message).Error; err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not send message"}); return }

	var members []models.CitizenChatRoomMember
	if err := config.DB.Where("room_id = ? AND status = ? AND user_id <> ?", roomID, "active", user.ID).Find(&members).Error; err == nil {
		for _, member := range members {
			var blocked int64
			if err := config.DB.Model(&models.CitizenChatBlock{}).Where(
				"deleted_at IS NULL AND ((blocker_id = ? AND blocked_id = ?) OR (blocker_id = ? AND blocked_id = ?))",
				member.UserID, user.ID, user.ID, member.UserID).Count(&blocked).Error; err != nil || blocked > 0 {
				continue
			}
			n := models.Notification{UserID: member.UserID, Title: "New local chat message", Message: "There is a new message in your local community chat.", Type: "community_chat", Status: "unread", EntityType: "community_chat", EntityID: message.ID, LinkTo: "/community/chat"}
			if config.DB.Create(&n).Error == nil { go dispatchWebPush(n.UserID, n.Title, n.Message) }
		}
	}
	message.Sender = *user
	c.JSON(http.StatusCreated, gin.H{"message": citizenChatMessageResponse(message)})
}

func ReportCitizenChatMessage(c *gin.Context) {
	user, ok := getAuthUser(c)
	if !ok { return }
	messageID, err := uuid.Parse(c.Param("messageId"))
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid message ID"}); return }
	var message models.CitizenChatMessage
	if err := config.DB.First(&message, "id = ? AND status = ?", messageID, "published").Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Message not found"}); return
	}
	if _, ok := citizenChatRoomAccess(c, user, message.RoomID); !ok { return }
	if message.SenderID == user.ID { c.JSON(http.StatusBadRequest, gin.H{"error": "You cannot report your own message"}); return }
	var input struct { Reason string `json:"reason" binding:"required"` }
	if err := c.ShouldBindJSON(&input); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	reason := strings.TrimSpace(input.Reason)
	if reason == "" || utf8.RuneCountInString(reason) > 2000 { c.JSON(http.StatusBadRequest, gin.H{"error": "Report reason must be 1 to 2000 characters"}); return }
	report := models.CitizenChatReport{MessageID: message.ID, ReporterID: user.ID, Reason: reason, Status: "pending"}
	if err := config.DB.Create(&report).Error; err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") || strings.Contains(strings.ToLower(err.Error()), "duplicate key") { c.JSON(http.StatusConflict, gin.H{"error": "You have already reported this message"}); return }
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not submit report"}); return
	}
	c.JSON(http.StatusCreated, gin.H{"reportId": report.ID, "status": report.Status})
}

func BlockCitizenChatUser(c *gin.Context) {
	user, ok := getAuthUser(c)
	if !ok { return }
	var input struct { UserID string `json:"userId" binding:"required"` }
	if err := c.ShouldBindJSON(&input); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	target, err := uuid.Parse(input.UserID)
	if err != nil || target == user.ID { c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user to block"}); return }
	var exists int64
	if err := config.DB.Model(&models.User{}).Where("id = ?", target).Count(&exists).Error; err != nil || exists == 0 { c.JSON(http.StatusNotFound, gin.H{"error": "User not found"}); return }
	block := models.CitizenChatBlock{BlockerID: user.ID, BlockedID: target}
	if err := config.DB.Where("blocker_id = ? AND blocked_id = ?", user.ID, target).FirstOrCreate(&block).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not block user"}); return
	}
	c.JSON(http.StatusOK, gin.H{"status": "blocked"})
}

func UnblockCitizenChatUser(c *gin.Context) {
	user, ok := getAuthUser(c)
	if !ok { return }
	target, err := uuid.Parse(c.Param("userId"))
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"}); return }
	if err := config.DB.Where("blocker_id = ? AND blocked_id = ?", user.ID, target).Delete(&models.CitizenChatBlock{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not unblock user"}); return
	}
	c.JSON(http.StatusOK, gin.H{"status": "unblocked"})
}

func canModerateCitizenChatRoom(user *models.User, room models.CitizenChatRoom) bool {
	if user == nil { return false }
	if user.Role == "super_admin" || user.IsSuperAdmin { return true }
	if user.Role != "unit_admin" { return false }
	place := resolveViewerPlace(user, nil, nil)
	return place.State != "" && place.LGA != "" && placeVisibleToViewer(room.GeoPlace, place)
}

func GetCitizenChatReports(c *gin.Context) {
	user, ok := getAuthUser(c)
	if !ok { return }
	if user.Role != "unit_admin" && user.Role != "super_admin" { c.JSON(http.StatusForbidden, gin.H{"error": "Moderator access required"}); return }
	status := c.DefaultQuery("status", "pending")
	if status != "pending" && status != "reviewed" && status != "dismissed" { c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid report status"}); return }
	var reports []models.CitizenChatReport
	if err := config.DB.Where("status = ?", status).Order("created_at ASC").Limit(100).Find(&reports).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not load moderation queue"}); return
	}
	out := make([]gin.H, 0, len(reports))
	for _, report := range reports {
		var message models.CitizenChatMessage
		if err := config.DB.Preload("Sender").First(&message, "id = ?", report.MessageID).Error; err != nil { continue }
		var room models.CitizenChatRoom
		if err := config.DB.First(&room, "id = ?", message.RoomID).Error; err != nil || !canModerateCitizenChatRoom(user, room) { continue }
		out = append(out, gin.H{"id": report.ID, "reason": report.Reason, "status": report.Status, "createdAt": report.CreatedAt, "message": citizenChatMessageResponse(message)})
	}
	c.JSON(http.StatusOK, gin.H{"reports": out})
}

func ResolveCitizenChatReport(c *gin.Context) {
	user, ok := getAuthUser(c)
	if !ok { return }
	if user.Role != "unit_admin" && user.Role != "super_admin" { c.JSON(http.StatusForbidden, gin.H{"error": "Moderator access required"}); return }
	reportID, err := uuid.Parse(c.Param("reportId"))
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid report ID"}); return }
	var input struct {
		Status string `json:"status" binding:"required"`
		Action string `json:"action"`
		Note string `json:"note"`
	}
	if err := c.ShouldBindJSON(&input); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	if input.Status != "reviewed" && input.Status != "dismissed" { c.JSON(http.StatusBadRequest, gin.H{"error": "Status must be reviewed or dismissed"}); return }
	if input.Action != "" && input.Action != "hide" { c.JSON(http.StatusBadRequest, gin.H{"error": "Action must be hide or empty"}); return }
	var report models.CitizenChatReport
	if err := config.DB.First(&report, "id = ?", reportID).Error; err != nil { c.JSON(http.StatusNotFound, gin.H{"error": "Report not found"}); return }
	if report.Status != "pending" {
		c.JSON(http.StatusConflict, gin.H{"error": "This report has already been resolved"})
		return
	}
	var message models.CitizenChatMessage
	if err := config.DB.First(&message, "id = ?", report.MessageID).Error; err != nil { c.JSON(http.StatusNotFound, gin.H{"error": "Reported message not found"}); return }
	var room models.CitizenChatRoom
	if err := config.DB.First(&room, "id = ?", message.RoomID).Error; err != nil || !canModerateCitizenChatRoom(user, room) { c.JSON(http.StatusForbidden, gin.H{"error": "You cannot moderate this chat"}); return }
	now := time.Now()
	err = config.DB.Transaction(func(tx *gorm.DB) error {
		updates := map[string]any{"status": input.Status, "resolution": input.Action, "moderator_id": user.ID, "moderator_note": strings.TrimSpace(input.Note), "resolved_at": now}
		if err := tx.Model(&report).Updates(updates).Error; err != nil { return err }
		if input.Action == "hide" { return tx.Model(&message).Update("status", "hidden").Error }
		return nil
	})
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not resolve report"}); return }
	c.JSON(http.StatusOK, gin.H{"status": input.Status, "action": input.Action})
}
