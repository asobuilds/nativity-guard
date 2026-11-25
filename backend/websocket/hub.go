package websocket

import (
	"encoding/json"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// Message types
const (
	MessageTypeJoin               = "join"
	MessageTypeLeave              = "leave"
	MessageTypeSignal             = "signal"
	MessageTypeOffer              = "offer"
	MessageTypeAnswer             = "answer"
	MessageTypeIceCandidate       = "ice-candidate"
	MessageTypeChat               = "chat"
	MessageTypeLocation           = "location"
	MessageTypeStatus             = "status"
	MessageTypePTTRequest         = "ptt.request"
	MessageTypePTTRelease         = "ptt.release"
	MessageTypePTTState           = "ptt.state"
	MessageTypePTTCommandOverride = "ptt.command_override"
)

// Message represents a WebSocket message
type Message struct {
	Type      string          `json:"type"`
	Room      string          `json:"room"`
	SenderID  string          `json:"senderId"`
	TargetID  string          `json:"targetId,omitempty"`
	Data      json.RawMessage `json:"data"`
	Timestamp time.Time       `json:"timestamp"`
}

// Outbound is a frame queued for a client. MessageType is a gorilla/websocket
// message type (TextMessage or BinaryMessage) so audio can travel as binary
// while control frames remain JSON text.
type Outbound struct {
	MessageType int
	Data        []byte
}

// Client represents a connected WebSocket client
type Client struct {
	ID       string
	Room     string
	Conn     *websocket.Conn
	Send     chan Outbound
	UnitID   string
	UserID   string
	Username string
	Role     string
	IsActive bool

	ChannelType string
	ChannelID   string

	// CommandOverride lets a super admin transmit on a unit channel. It can
	// only be flipped by an authenticated super admin via ptt.command_override.
	CommandOverride atomic.Bool

	// Captured at handshake time so floor-control audit rows can record the
	// origin without reaching back into the upgraded HTTP request.
	IPAddress string
	UserAgent string
}

// Hub maintains the set of active clients
type Hub struct {
	clients    map[string]*Client
	rooms      map[string]map[string]*Client
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mutex      sync.RWMutex
}

var hubInstance *Hub
var once sync.Once

// GetHub returns the singleton Hub instance
func GetHub() *Hub {
	once.Do(func() {
		hubInstance = &Hub{
			clients:    make(map[string]*Client),
			rooms:      make(map[string]map[string]*Client),
			broadcast:  make(chan []byte),
			register:   make(chan *Client),
			unregister: make(chan *Client),
		}
		go hubInstance.run()
	})
	return hubInstance
}

// RegisterClient adds a client to the hub
func (h *Hub) RegisterClient(client *Client) {
	h.register <- client
}

// UnregisterClient removes a client from the hub
func (h *Hub) UnregisterClient(client *Client) {
	h.unregister <- client
}

// Broadcast sends a text message to every connected client.
func (h *Hub) Broadcast(message []byte) {
	h.broadcast <- message
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			h.clients[client.ID] = client
			if h.rooms[client.Room] == nil {
				h.rooms[client.Room] = make(map[string]*Client)
			}
			h.rooms[client.Room][client.ID] = client
			h.mutex.Unlock()
			log.Printf("✅ Client %s joined room %s", client.ID, client.Room)
			h.broadcastRoomStatus(client.Room)

		case client := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[client.ID]; ok {
				delete(h.clients, client.ID)
				if h.rooms[client.Room] != nil {
					delete(h.rooms[client.Room], client.ID)
					if len(h.rooms[client.Room]) == 0 {
						delete(h.rooms, client.Room)
					}
				}
				close(client.Send)
			}
			h.mutex.Unlock()
			log.Printf("❌ Client %s left room %s", client.ID, client.Room)
			h.broadcastRoomStatus(client.Room)

		case message := <-h.broadcast:
			out := Outbound{MessageType: websocket.TextMessage, Data: message}
			h.mutex.RLock()
			for _, client := range h.clients {
				select {
				case client.Send <- out:
				default:
				}
			}
			h.mutex.RUnlock()
		}
	}
}

func (h *Hub) broadcastRoomStatus(room string) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	if h.rooms[room] == nil {
		return
	}

	status := struct {
		Type  string   `json:"type"`
		Room  string   `json:"room"`
		Count int      `json:"count"`
		Users []string `json:"users"`
	}{
		Type:  "room-status",
		Room:  room,
		Count: len(h.rooms[room]),
		Users: make([]string, 0),
	}

	for _, client := range h.rooms[room] {
		status.Users = append(status.Users, client.Username)
	}

	msg, _ := json.Marshal(status)
	out := Outbound{MessageType: websocket.TextMessage, Data: msg}
	for _, client := range h.rooms[room] {
		select {
		case client.Send <- out:
		default:
		}
	}
}

// SendToClient sends a text message to a specific client
func (h *Hub) SendToClient(clientID string, message []byte) {
	h.SendToClientOutbound(clientID, Outbound{MessageType: websocket.TextMessage, Data: message})
}

// SendToClientOutbound sends one frame to a specific client.
func (h *Hub) SendToClientOutbound(clientID string, out Outbound) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	if client, ok := h.clients[clientID]; ok {
		select {
		case client.Send <- out:
		default:
		}
	}
}

// SendToRoom sends a text message to all clients in a room
func (h *Hub) SendToRoom(room string, message []byte) {
	h.SendToRoomExcept(room, "", message, websocket.TextMessage)
}

// SendToRoomBinary sends a binary frame to all clients in a room.
func (h *Hub) SendToRoomBinary(room string, message []byte) {
	h.SendToRoomExcept(room, "", message, websocket.BinaryMessage)
}

// SendToRoomExcept sends a frame to every client in a room except excludeID.
func (h *Hub) SendToRoomExcept(room string, excludeID string, message []byte, messageType int) {
	out := Outbound{MessageType: messageType, Data: message}
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	if h.rooms[room] == nil {
		return
	}
	for id, client := range h.rooms[room] {
		if id == excludeID {
			continue
		}
		select {
		case client.Send <- out:
		default:
		}
	}
}

// RoomSize returns the number of connected clients in a room.
func (h *Hub) RoomSize(room string) int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return len(h.rooms[room])
}
