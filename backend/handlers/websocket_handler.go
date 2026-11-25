package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"security-solution/config"
	"security-solution/models"
	"security-solution/services"
	ws "security-solution/websocket"
)

const (
	maxWebSocketMessageSize = 64 * 1024
	maxAudioFrameSize       = 32 * 1024
)

var upgrader = websocket.Upgrader{
	CheckOrigin:     checkWebSocketOrigin,
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
}

// checkWebSocketOrigin rejects cross-site browser handshakes while still
// allowing same-origin clients, configured ALLOWED_ORIGINS, and local dev.
func checkWebSocketOrigin(r *http.Request) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return true // non-browser clients (tests, native apps) send no Origin
	}

	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}
	if strings.EqualFold(parsed.Host, r.Host) {
		return true
	}
	for _, allowed := range strings.Split(os.Getenv("ALLOWED_ORIGINS"), ",") {
		if strings.EqualFold(strings.TrimSpace(allowed), origin) {
			return true
		}
	}
	if gin.Mode() != gin.ReleaseMode {
		host := parsed.Hostname()
		if host == "localhost" || host == "127.0.0.1" || host == "::1" {
			return true
		}
	}
	return false
}

// HandleWebSocket upgrades an authenticated member into a unit or task-force
// walkie-talkie room. Authorization is based on the user's own UnitID and
// active task-force membership, never a client-supplied officer record.
func HandleWebSocket(c *gin.Context) {
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	user, ok := userInterface.(*models.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user data"})
		return
	}

	unitIDStr := strings.TrimSpace(c.Query("unitId"))
	taskForceIDStr := strings.TrimSpace(c.Query("taskforceId"))
	if taskForceIDStr == "" {
		taskForceIDStr = strings.TrimSpace(c.Query("taskForceId"))
	}
	if (unitIDStr == "") == (taskForceIDStr == "") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Provide exactly one of unitId or taskforceId"})
		return
	}

	var (
		roomKey     string
		unitIDValue string
		channelType string
		channelID   string
	)

	if unitIDStr != "" {
		unitID, err := uuid.Parse(unitIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Valid unit ID required"})
			return
		}

		var unit models.SecurityUnit
		if err := config.DB.First(&unit, "id = ?", unitID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Unit not found"})
			return
		}

		if !canJoinUnitChannel(user, unitID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission for this unit"})
			return
		}

		// Make sure the persisted unit channel exists. PTT itself is live-only; the
		// room row is the shared identity used by the REST helper and audits.
		if _, err := ensureUnitChannel(unitID, user.ID); err != nil {
			log.Printf("walkie-talkie: failed to ensure channel for unit %s: %v", unitID, err)
		}

		roomKey = unitID.String()
		unitIDValue = unitID.String()
		channelType = "unit"
		channelID = unitID.String()
	} else {
		taskForceID, err := uuid.Parse(taskForceIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Valid task force ID required"})
			return
		}
		taskForce, err := loadTaskForce(taskForceID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Task force not found"})
			return
		}
		if !canJoinTaskForceChannel(user, taskForce) {
			c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission for this task force"})
			return
		}
		if _, err := ensureTaskForceChannel(taskForce, user.ID); err != nil {
			log.Printf("walkie-talkie: failed to ensure channel for task force %s: %v", taskForceID, err)
		}

		roomKey = taskForceRoomKey(taskForce.ID)
		unitIDValue = taskForce.HostUnitID.String()
		channelType = models.TaskForceRoomType
		channelID = taskForce.ID.String()
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade WebSocket: %v", err)
		return
	}
	defer conn.Close()

	clientID := uuid.New().String()
	hub := ws.GetHub()
	ptt := ws.GetPTTManager()

	client := &ws.Client{
		ID:          clientID,
		Room:        roomKey,
		Conn:        conn,
		Send:        make(chan ws.Outbound, 256),
		UnitID:      unitIDValue,
		UserID:      user.ID.String(),
		Username:    strings.TrimSpace(user.FirstName + " " + user.LastName),
		Role:        user.Role,
		IsActive:    true,
		ChannelType: channelType,
		ChannelID:   channelID,
		IPAddress:   c.ClientIP(),
		UserAgent:   c.Request.UserAgent(),
	}

	hub.RegisterClient(client)

	writeClientJSON(client, map[string]any{
		"type":        "welcome",
		"message":     "Connected to walkie-talkie",
		"room":        client.Room,
		"clientId":    clientID,
		"role":        user.Role,
		"channelType": channelType,
		"channelId":   channelID,
	})
	writeClientJSON(client, ws.PTTStatePayload("ptt.state", ptt.Snapshot(client.Room)))

	go clientWritePump(client)
	clientReadPump(client, hub, ptt)
}

func canJoinTaskForceChannel(user *models.User, taskForce *models.TaskForce) bool {
	if user == nil || taskForce == nil || taskForce.Status != models.TaskForceStatusActive {
		return false
	}
	if user.Role == models.RoleSuperAdmin || user.IsSuperAdmin {
		return true // listen-only unless command override is enabled
	}
	if user.Role != models.RoleOfficer && user.Role != models.RoleUnitAdmin {
		return false
	}
	if user.UnitID == nil {
		return false
	}
	return taskForceHasActiveMember(taskForce.ID, *user.UnitID)
}

func canJoinUnitChannel(user *models.User, unitID uuid.UUID) bool {
	if user.Role == models.RoleSuperAdmin || user.IsSuperAdmin {
		return true // listen-only unless command override is enabled
	}
	if user.Role != models.RoleOfficer && user.Role != models.RoleUnitAdmin {
		return false
	}
	return user.UnitID != nil && *user.UnitID == unitID
}

func clientReadPump(client *ws.Client, hub *ws.Hub, ptt *ws.PTTManager) {
	defer func() {
		ptt.Leave(client)
		hub.UnregisterClient(client)
		client.Conn.Close()
	}()

	client.Conn.SetReadLimit(maxWebSocketMessageSize)
	client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	client.Conn.SetPongHandler(func(string) error {
		client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		messageType, message, err := client.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Binary frames are live PCM audio. Only the current floor holder may
		// transmit; everyone else's bytes are dropped.
		if messageType == websocket.BinaryMessage {
			if len(message) > 0 && len(message) <= maxAudioFrameSize && ptt.IsSpeaker(client) {
				hub.SendToRoomExcept(client.Room, client.ID, message, websocket.BinaryMessage)
			}
			continue
		}

		var msg ws.Message
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("Failed to parse message: %v", err)
			continue
		}

		msg.SenderID = client.ID
		msg.Timestamp = time.Now()

		switch msg.Type {
		case ws.MessageTypePTTRequest:
			ptt.RequestFloor(client)

		case ws.MessageTypePTTRelease:
			ptt.ReleaseFloor(client, "released")

		case ws.MessageTypePTTState:
			writeClientJSON(client, ws.PTTStatePayload("ptt.state", ptt.Snapshot(client.Room)))

		case ws.MessageTypePTTCommandOverride:
			handlePTTCommandOverride(client, msg)

		case ws.MessageTypeChat, ws.MessageTypeLocation:
			encoded, _ := json.Marshal(msg)
			hub.SendToRoom(client.Room, encoded)

		case ws.MessageTypeSignal, ws.MessageTypeOffer, ws.MessageTypeAnswer, ws.MessageTypeIceCandidate:
			if msg.TargetID != "" {
				encoded, _ := json.Marshal(msg)
				hub.SendToClient(msg.TargetID, encoded)
			}
		}
	}
}

// handlePTTCommandOverride lets a super admin opt into transmitting. Every
// change is audited, including the previous value.
func handlePTTCommandOverride(client *ws.Client, msg ws.Message) {
	if client.Role != models.RoleSuperAdmin {
		writeClientJSON(client, map[string]any{"type": "ptt.error", "reason": "forbidden"})
		return
	}

	var input struct {
		Enabled bool `json:"enabled"`
	}
	if len(msg.Data) > 0 {
		if err := json.Unmarshal(msg.Data, &input); err != nil {
			writeClientJSON(client, map[string]any{"type": "ptt.error", "reason": "invalid_override"})
			return
		}
	}

	previous := client.CommandOverride.Swap(input.Enabled)
	if config.DB != nil {
		if userID, err := uuid.Parse(client.UserID); err == nil {
			entityType := client.ChannelType
			entityID := client.ChannelID
			if entityType == "" {
				entityType = "unit"
				entityID = client.Room
			}
			_ = services.NewAuditService().LogAction(
				userID,
				"walkie.command_override",
				entityType,
				entityID,
				map[string]any{"enabled": previous},
				map[string]any{"enabled": input.Enabled},
				client.IPAddress,
				client.UserAgent,
			)
		}
	}

	writeClientJSON(client, map[string]any{
		"type":    "ptt.override",
		"enabled": input.Enabled,
	})
}

func writeClientJSON(client *ws.Client, payload any) {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return
	}
	select {
	case client.Send <- ws.Outbound{MessageType: websocket.TextMessage, Data: encoded}:
	default:
	}
}

func clientWritePump(client *ws.Client) {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		client.Conn.Close()
	}()

	for {
		select {
		case outbound, ok := <-client.Send:
			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := client.Conn.WriteMessage(outbound.MessageType, outbound.Data); err != nil {
				return
			}

		case <-ticker.C:
			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
