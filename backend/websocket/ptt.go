package websocket

import (
	"encoding/json"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	gws "github.com/gorilla/websocket"

	"security-solution/config"
	"security-solution/models"
	"security-solution/services"
)

// DefaultTransmissionTimeout is the hard ceiling for a single transmission.
const DefaultTransmissionTimeout = 60 * time.Second

// PTTState is the point-in-time floor state for a unit channel.
type PTTState struct {
	Status           string     `json:"status"`
	Room             string     `json:"room"`
	SpeakerID        string     `json:"speakerId,omitempty"`
	SpeakerUserID    string     `json:"speakerUserId,omitempty"`
	SpeakerName      string     `json:"speakerName,omitempty"`
	ExpiresAt        *time.Time `json:"expiresAt,omitempty"`
	RemainingSeconds int        `json:"remainingSeconds,omitempty"`
}

// FloorResult reports the outcome of a floor request or release.
type FloorResult struct {
	Granted bool
	Busy    bool
	Denied  bool
	Reason  string
	State   PTTState
}

// PTTRoom is the floor state for one unit channel (keyed by UnitID).
type PTTRoom struct {
	Mutex        sync.Mutex
	Room         string
	Speaker      *Client
	SpeakerSince time.Time
	ExpiresAt    time.Time
	GrantID      uint64
	Timer        *time.Timer
}

// PTTManager owns every unit's floor state. Only one client per unit may hold
// the floor at a time.
type PTTManager struct {
	mu      sync.Mutex
	rooms   map[string]*PTTRoom
	timeout time.Duration
	hub     *Hub
	now     func() time.Time
}

// NewPTTManager builds a manager with the given transmission ceiling. A
// non-positive timeout falls back to DefaultTransmissionTimeout.
func NewPTTManager(hub *Hub, timeout time.Duration) *PTTManager {
	if timeout <= 0 {
		timeout = DefaultTransmissionTimeout
	}
	return &PTTManager{
		rooms:   make(map[string]*PTTRoom),
		timeout: timeout,
		hub:     hub,
		now:     time.Now,
	}
}

var (
	pttInstance *PTTManager
	pttOnce     sync.Once
)

// GetPTTManager returns the process-wide PTT manager. PTT_TRANSMISSION_TIMEOUT
// exists so the 60-second ceiling can be shortened in tests and local runs.
func GetPTTManager() *PTTManager {
	pttOnce.Do(func() {
		timeout := DefaultTransmissionTimeout
		if raw := os.Getenv("PTT_TRANSMISSION_TIMEOUT"); raw != "" {
			if parsed, err := time.ParseDuration(raw); err == nil && parsed > 0 {
				timeout = parsed
			}
		}
		pttInstance = NewPTTManager(GetHub(), timeout)
	})
	return pttInstance
}

func (m *PTTManager) room(id string) *PTTRoom {
	m.mu.Lock()
	defer m.mu.Unlock()
	room := m.rooms[id]
	if room == nil {
		room = &PTTRoom{Room: id}
		m.rooms[id] = room
	}
	return room
}

func (r *PTTRoom) stateLocked(now time.Time) PTTState {
	state := PTTState{Room: r.Room, Status: "idle"}
	if r.Speaker == nil || !now.Before(r.ExpiresAt) {
		return state
	}
	expiresAt := r.ExpiresAt
	remaining := int((r.ExpiresAt.Sub(now) + time.Second - 1) / time.Second)
	if remaining < 0 {
		remaining = 0
	}
	state.Status = "busy"
	state.SpeakerID = r.Speaker.ID
	state.SpeakerUserID = r.Speaker.UserID
	state.SpeakerName = r.Speaker.Username
	state.ExpiresAt = &expiresAt
	state.RemainingSeconds = remaining
	return state
}

// RequestFloor grants the floor when the unit is idle. A super admin must opt
// in with ptt.command_override before transmitting.
func (m *PTTManager) RequestFloor(client *Client) FloorResult {
	if client == nil {
		return FloorResult{Denied: true, Reason: "invalid_client"}
	}
	if client.Role == models.RoleSuperAdmin && !client.CommandOverride.Load() {
		m.sendJSON(client, map[string]any{
			"type":   "ptt.error",
			"reason": "command_override_required",
		})
		return FloorResult{Denied: true, Reason: "command_override_required"}
	}

	room := m.room(client.Room)
	now := m.now()

	room.Mutex.Lock()
	if room.Speaker != nil && now.Before(room.ExpiresAt) {
		state := room.stateLocked(now)
		room.Mutex.Unlock()
		m.sendJSON(client, statePayload("ptt.busy", state))
		return FloorResult{Busy: true, State: state}
	}

	// Clear any expired state or dangling timer before granting.
	if room.Timer != nil {
		room.Timer.Stop()
		room.Timer = nil
	}
	room.GrantID++
	grantID := room.GrantID
	room.Speaker = client
	room.SpeakerSince = now
	room.ExpiresAt = now.Add(m.timeout)
	room.Timer = time.AfterFunc(m.timeout, func() {
		m.Timeout(room.Room, grantID)
	})
	state := room.stateLocked(now)
	expiresAt := room.ExpiresAt
	room.Mutex.Unlock()

	m.broadcastState("ptt.granted", state)
	m.audit(client, "walkie.floor_granted", map[string]any{
		"speakerId":      client.ID,
		"speakerName":    client.Username,
		"expiresAt":      expiresAt.UTC().Format(time.RFC3339Nano),
		"timeoutSeconds": int(m.timeout.Seconds()),
	})
	return FloorResult{Granted: true, State: state}
}

// ReleaseFloor clears the floor when the caller is the current speaker.
func (m *PTTManager) ReleaseFloor(client *Client, reason string) FloorResult {
	if client == nil {
		return FloorResult{}
	}
	room := m.room(client.Room)
	now := m.now()

	room.Mutex.Lock()
	if room.Speaker == nil || room.Speaker.ID != client.ID {
		state := room.stateLocked(now)
		room.Mutex.Unlock()
		return FloorResult{State: state}
	}
	speaker := room.Speaker
	duration := now.Sub(room.SpeakerSince)
	m.clearLocked(room)
	state := room.stateLocked(now)
	room.Mutex.Unlock()

	m.broadcastMap(releasedPayload(speaker, reason, duration))
	m.audit(speaker, "walkie.floor_released", map[string]any{
		"speakerId":       speaker.ID,
		"reason":          reason,
		"durationSeconds": int(duration.Seconds()),
	})
	return FloorResult{State: state}
}

// Timeout releases the floor when the 60-second ceiling elapses. grantID makes
// late timers harmless after a release/re-grant.
func (m *PTTManager) Timeout(roomID string, grantID uint64) {
	room := m.room(roomID)
	now := m.now()

	room.Mutex.Lock()
	if room.Speaker == nil || room.GrantID != grantID {
		room.Mutex.Unlock()
		return
	}
	speaker := room.Speaker
	duration := now.Sub(room.SpeakerSince)
	m.clearLocked(room)
	room.Mutex.Unlock()

	m.broadcastMap(releasedPayload(speaker, "timeout", duration))
	m.audit(speaker, "walkie.floor_timeout", map[string]any{
		"speakerId":       speaker.ID,
		"reason":          "timeout",
		"durationSeconds": int(duration.Seconds()),
	})
}

// Leave releases the floor when the speaker disconnects.
func (m *PTTManager) Leave(client *Client) {
	if client == nil {
		return
	}
	room := m.room(client.Room)
	now := m.now()

	room.Mutex.Lock()
	if room.Speaker == nil || room.Speaker.ID != client.ID {
		room.Mutex.Unlock()
		return
	}
	speaker := room.Speaker
	duration := now.Sub(room.SpeakerSince)
	m.clearLocked(room)
	room.Mutex.Unlock()

	m.broadcastMap(releasedPayload(speaker, "disconnected", duration))
	m.audit(speaker, "walkie.floor_released", map[string]any{
		"speakerId":       speaker.ID,
		"reason":          "disconnected",
		"durationSeconds": int(duration.Seconds()),
	})
}

// IsSpeaker reports whether client currently holds a live floor.
func (m *PTTManager) IsSpeaker(client *Client) bool {
	if client == nil {
		return false
	}
	room := m.room(client.Room)
	now := m.now()
	room.Mutex.Lock()
	defer room.Mutex.Unlock()
	return room.Speaker != nil && room.Speaker.ID == client.ID && now.Before(room.ExpiresAt)
}

// Snapshot returns the current floor state for a unit channel.
func (m *PTTManager) Snapshot(roomID string) PTTState {
	room := m.room(roomID)
	now := m.now()
	room.Mutex.Lock()
	defer room.Mutex.Unlock()
	return room.stateLocked(now)
}

func (m *PTTManager) clearLocked(room *PTTRoom) {
	if room.Timer != nil {
		room.Timer.Stop()
		room.Timer = nil
	}
	room.Speaker = nil
	room.SpeakerSince = time.Time{}
	room.ExpiresAt = time.Time{}
	// Invalidate any timer callback that was already dequeued.
	room.GrantID++
}

func (m *PTTManager) broadcastState(messageType string, state PTTState) {
	m.broadcastMap(statePayload(messageType, state))
}

func (m *PTTManager) broadcastMap(payload map[string]any) {
	if m.hub == nil {
		return
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return
	}
	room, _ := payload["room"].(string)
	if room == "" {
		return
	}
	m.hub.SendToRoom(room, encoded)
}

func (m *PTTManager) sendJSON(client *Client, payload map[string]any) {
	if client == nil || client.Send == nil {
		return
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return
	}
	select {
	case client.Send <- Outbound{MessageType: gws.TextMessage, Data: encoded}:
	default:
	}
}

// PTTStatePayload renders a control frame for a floor state.
func PTTStatePayload(messageType string, state PTTState) map[string]any {
	return statePayload(messageType, state)
}

func statePayload(messageType string, state PTTState) map[string]any {
	payload := map[string]any{
		"type":   messageType,
		"room":   state.Room,
		"status": state.Status,
	}
	if state.SpeakerID != "" {
		payload["speakerId"] = state.SpeakerID
		payload["speakerUserId"] = state.SpeakerUserID
		payload["speakerName"] = state.SpeakerName
	}
	if state.ExpiresAt != nil {
		payload["expiresAt"] = state.ExpiresAt.UTC().Format(time.RFC3339Nano)
	}
	if state.RemainingSeconds > 0 {
		payload["remainingSeconds"] = state.RemainingSeconds
	}
	return payload
}

func releasedPayload(speaker *Client, reason string, duration time.Duration) map[string]any {
	return map[string]any{
		"type":            "ptt.released",
		"room":            speaker.Room,
		"status":          "idle",
		"speakerId":       speaker.ID,
		"speakerUserId":   speaker.UserID,
		"speakerName":     speaker.Username,
		"reason":          reason,
		"durationSeconds": int(duration.Seconds()),
	}
}

func (m *PTTManager) audit(client *Client, action string, newValue map[string]any) {
	if client == nil || config.DB == nil {
		return
	}
	userID, err := uuid.Parse(client.UserID)
	if err != nil {
		return
	}
	_ = services.NewAuditService().LogAction(
		userID,
		action,
		"unit",
		client.Room,
		nil,
		newValue,
		client.IPAddress,
		client.UserAgent,
	)
}
