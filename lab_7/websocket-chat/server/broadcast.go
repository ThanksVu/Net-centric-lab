package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)



type Client struct {
	ID   string
	Conn *websocket.Conn
	Send chan []byte
}



type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

func newHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}



func (h *Hub) run() {
	for {
		select {

		case client := <-h.register:
			// Add new client
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Println("Client registered:", client.ID)

		case client := <-h.unregister:
			// Remove client
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
				log.Println("Client unregistered:", client.ID)
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			// Send message to ALL clients
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}



func (c *Client) readPump(hub *Hub) {
	defer func() {
		hub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(512)
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))

	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, msg, err := c.Conn.ReadMessage()
		if err != nil {
			log.Println("Read error:", err)
			break
		}

		// Broadcast to hub
		hub.broadcast <- []byte(fmt.Sprintf("%s: %s", c.ID, msg))
	}
}



func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer ticker.Stop()

	for {
		select {

		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

			if !ok {
				// Channel closed
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}

			_, _ = w.Write(message)
			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			// Send ping
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, []byte("ping")); err != nil {
				return
			}
		}
	}
}



var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var hub = newHub()

func handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}

	clientID := fmt.Sprintf("User-%d", time.Now().UnixNano())

	client := &Client{
		ID:   clientID,
		Conn: conn,
		Send: make(chan []byte, 256),
	}

	hub.register <- client

	go client.writePump()
	go client.readPump(hub)

	log.Println("New client connected:", client.ID)
}



func main() {
	go hub.run()

	http.HandleFunc("/ws", handleWS)

	fmt.Println("🚀 Broadcast chat server running at :8080/ws")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
