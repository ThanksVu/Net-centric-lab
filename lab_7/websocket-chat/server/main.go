package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// --- Message & Notification types ---
type Message struct {
	Type     string `json:"type"` // "join","leave","chat","system"
	Room     string `json:"room"`
	Username string `json:"username"`
	Text     string `json:"text"`
	Time     string `json:"time"`
}

const (
	MsgChat     = "chat"
	MsgSystem   = "system"
	MsgUserList = "user_list"
	MsgStats    = "stats"
	MsgCommand  = "command"
)

type Notification struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"` // info, warning, error, success
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Target    string    `json:"target"` // all, room:name, user:username
}

// --- Client ---
type Client struct {
	conn     *websocket.Conn
	send     chan Message
	username string
	room     string
	hub      *Hub
}

// --- Room & Hub ---
type Room struct {
	Name    string
	Clients map[*Client]bool
	mu      sync.RWMutex
}

type Hub struct {
	rooms      map[string]*Room
	register   chan *Client
	unregister chan *Client
	broadcast  chan Message
	mu         sync.RWMutex

	notifMu sync.RWMutex
	history []Notification
}

// --- New Hub ---
func NewHub() *Hub {
	return &Hub{
		rooms:      make(map[string]*Room),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan Message, 256),
		history:    make([]Notification, 0, 50),
	}
}

// --- Hub run loop ---
func (h *Hub) run() {
	for {
		select {
		case c := <-h.register:
			h.addClientToRoom(c)
		case c := <-h.unregister:
			h.removeClientFromRoom(c)
		case msg := <-h.broadcast:
			h.broadcastToRoom(msg.Room, msg)
		}
	}
}

// --- Get or create room ---
func (h *Hub) getOrCreateRoom(name string) *Room {
	h.mu.Lock()
	defer h.mu.Unlock()
	if r, ok := h.rooms[name]; ok {
		return r
	}
	r := &Room{Name: name, Clients: make(map[*Client]bool)}
	h.rooms[name] = r
	return r
}

// --- Add client to room ---
func (h *Hub) addClientToRoom(client *Client) {
	r := h.getOrCreateRoom(client.room)
	r.mu.Lock()
	r.Clients[client] = true
	r.mu.Unlock()

	// join notification
	join := Message{
		Type:     "join",
		Room:     r.Name,
		Username: client.username,
		Text:     fmt.Sprintf("%s joined", client.username),
		Time:     time.Now().Format(time.RFC3339),
	}
	h.broadcastToRoom(r.Name, join)

	// send filtered notification history
	h.notifMu.RLock()
	history := make([]Notification, len(h.history))
	copy(history, h.history)
	h.notifMu.RUnlock()

	for _, n := range history {
		send := false
		switch {
		case n.Target == "all":
			send = true
		case strings.HasPrefix(n.Target, "room:"):
			roomName := strings.TrimPrefix(n.Target, "room:")
			if roomName == client.room {
				send = true
			}
		case strings.HasPrefix(n.Target, "user:"):
			user := strings.TrimPrefix(n.Target, "user:")
			if user == client.username {
				send = true
			}
		}
		if send {
			nj := Message{
				Type:     MsgSystem,
				Room:     client.room,
				Username: "SYSTEM",
				Text:     fmt.Sprintf("[NOTIF] %s: %s", n.Title, n.Message),
				Time:     n.Timestamp.Format(time.RFC3339),
			}
			client.send <- nj
		}
	}

	// send current user list
	h.sendUserListToRoom(r.Name)
}

// --- Remove client ---
func (h *Hub) removeClientFromRoom(client *Client) {
	h.mu.RLock()
	r, ok := h.rooms[client.room]
	h.mu.RUnlock()
	if !ok {
		return
	}

	r.mu.Lock()
	if _, present := r.Clients[client]; present {
		delete(r.Clients, client)
	}
	remaining := len(r.Clients)
	r.mu.Unlock()

	// leave notification
	leave := Message{
		Type:     "leave",
		Room:     r.Name,
		Username: client.username,
		Text:     fmt.Sprintf("%s left", client.username),
		Time:     time.Now().Format(time.RFC3339),
	}
	h.broadcastToRoom(r.Name, leave)

	h.sendUserListToRoom(r.Name)

	if remaining == 0 {
		h.mu.Lock()
		delete(h.rooms, r.Name)
		h.mu.Unlock()
	}
}

// --- Broadcast to room ---
func (h *Hub) broadcastToRoom(roomName string, msg Message) {
	h.mu.RLock()
	r, ok := h.rooms[roomName]
	h.mu.RUnlock()
	if !ok {
		return
	}

	r.mu.RLock()
	for c := range r.Clients {
		select {
		case c.send <- msg:
		default:
		}
	}
	r.mu.RUnlock()
}

// --- Send user list ---
func (h *Hub) sendUserListToRoom(roomName string) {
	h.mu.RLock()
	r, ok := h.rooms[roomName]
	h.mu.RUnlock()
	if !ok {
		return
	}

	users := make([]string, 0)
	r.mu.RLock()
	for c := range r.Clients {
		users = append(users, c.username)
	}
	r.mu.RUnlock()

	text := fmt.Sprintf("Users in '%s' (%d):\n", roomName, len(users))
	for _, u := range users {
		text += "- " + u + "\n"
	}

	msg := Message{Type: MsgUserList, Room: roomName, Username: "SYSTEM", Text: text, Time: time.Now().Format(time.RFC3339)}
	h.broadcastToRoom(roomName, msg)
}

// --- Handle commands ---
func (h *Hub) handleCommand(client *Client, cmd string) {
	switch cmd {
	case "/users":
		h.sendUserListToRoom(client.room)
	case "/stats":
		h.mu.RLock()
		roomDetails := make(map[string]int)
		for name, room := range h.rooms {
			room.mu.RLock()
			roomDetails[name] = len(room.Clients)
			room.mu.RUnlock()
		}
		h.mu.RUnlock()
		totalUsers := 0
		for _, v := range roomDetails {
			totalUsers += v
		}
		totalRooms := len(roomDetails)
		stats := map[string]interface{}{"total_users": totalUsers, "total_rooms": totalRooms, "room_details": roomDetails}
		b, _ := json.MarshalIndent(stats, "", "  ")
		client.send <- Message{Type: MsgStats, Room: client.room, Username: "SYSTEM", Text: string(b), Time: time.Now().Format(time.RFC3339)}
	case "/rooms":
		h.mu.RLock()
		names := make([]string, 0, len(h.rooms))
		for name := range h.rooms {
			names = append(names, name)
		}
		h.mu.RUnlock()
		text := "Rooms:\n"
		for _, n := range names {
			text += "- " + n + "\n"
		}
		client.send <- Message{Type: MsgSystem, Room: client.room, Username: "SYSTEM", Text: text, Time: time.Now().Format(time.RFC3339)}
	default:
		client.send <- Message{Type: MsgSystem, Room: client.room, Username: "SYSTEM", Text: "Unknown command", Time: time.Now().Format(time.RFC3339)}
	}
}

// --- Notifications ---
func (h *Hub) addNotification(n Notification) {
	h.notifMu.Lock()
	defer h.notifMu.Unlock()
	if n.ID == "" {
		n.ID = uuid.NewString()
	}
	h.history = append(h.history, n)
	if len(h.history) > 50 {
		h.history = h.history[len(h.history)-50:]
	}
}

func (h *Hub) routeNotification(n Notification) {
	h.addNotification(n)
	switch {
	case n.Target == "all":
		h.mu.RLock()
		for name := range h.rooms {
			system := Message{Type: MsgSystem, Room: name, Username: "ADMIN", Text: fmt.Sprintf("%s: %s", n.Title, n.Message), Time: n.Timestamp.Format(time.RFC3339)}
			h.broadcastToRoom(name, system)
		}
		h.mu.RUnlock()
	case strings.HasPrefix(n.Target, "room:"):
		room := strings.TrimPrefix(n.Target, "room:")
		system := Message{Type: MsgSystem, Room: room, Username: "ADMIN", Text: fmt.Sprintf("%s: %s", n.Title, n.Message), Time: n.Timestamp.Format(time.RFC3339)}
		h.broadcastToRoom(room, system)
	case strings.HasPrefix(n.Target, "user:"):
		user := strings.TrimPrefix(n.Target, "user:")
		h.mu.RLock()
		for _, room := range h.rooms {
			room.mu.RLock()
			for c := range room.Clients {
				if c.username == user {
					c.send <- Message{Type: MsgSystem, Room: room.Name, Username: "ADMIN", Text: fmt.Sprintf("%s: %s", n.Title, n.Message), Time: n.Timestamp.Format(time.RFC3339)}
				}
			}
			room.mu.RUnlock()
		}
		h.mu.RUnlock()
	}
}

// --- Websocket ---
var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

func serveWs(hub *Hub, c *gin.Context) {
	username := c.Query("username")
	room := c.Query("room")
	if username == "" || room == "" {
		c.String(400, "username and room query params required")
		return
	}

	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("upgrade:", err)
		return
	}

	client := &Client{conn: ws, send: make(chan Message, 256), username: username, room: room, hub: hub}
	hub.register <- client

	go client.writePump()
	client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	for {
		var msg Message
		if err := c.conn.ReadJSON(&msg); err != nil {
			log.Println("read error:", err)
			break
		}
		if msg.Type == MsgCommand {
			c.hub.handleCommand(c, msg.Text)
			continue
		}
		msg.Username = c.username
		msg.Room = c.room
		msg.Time = time.Now().Format(time.RFC3339)
		c.hub.broadcast <- msg
	}
}

func (c *Client) writePump() {
	defer c.conn.Close()
	for m := range c.send {
		if err := c.conn.WriteJSON(m); err != nil {
			log.Println("write error:", err)
			break
		}
	}
}

// --- HTTP Admin ---
func handleNotification(h *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		var notif Notification
		if err := c.BindJSON(&notif); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		if notif.Timestamp.IsZero() {
			notif.Timestamp = time.Now()
		}
		if notif.ID == "" {
			notif.ID = uuid.NewString()
		}
		h.routeNotification(notif)
		c.JSON(200, gin.H{"status": "sent", "id": notif.ID})
	}
}

func getStats(h *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		h.mu.RLock()
		roomDetails := make(map[string]int)
		for name, room := range h.rooms {
			room.mu.RLock()
			roomDetails[name] = len(room.Clients)
			room.mu.RUnlock()
		}
		h.mu.RUnlock()
		totalUsers := 0
		for _, v := range roomDetails {
			totalUsers += v
		}
		res := StatsMessage{TotalUsers: totalUsers, TotalRooms: len(roomDetails), RoomDetails: roomDetails}
		c.JSON(200, res)
	}
}

// --- StatsMessage for API ---
type StatsMessage struct {
	TotalUsers  int            `json:"total_users"`
	TotalRooms  int            `json:"total_rooms"`
	RoomDetails map[string]int `json:"room_details"`
}

// --- main ---
func main() {
	hub := NewHub()
	go hub.run()

	r := gin.Default()
	r.GET("/ws", func(c *gin.Context) { serveWs(hub, c) })
	r.POST("/api/notify", handleNotification(hub))
	r.GET("/api/stats", getStats(hub))

	log.Println("Server running on :8080")
	r.Run(":8080")
}
