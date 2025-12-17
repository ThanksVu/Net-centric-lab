package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// 1. Upgrade HTTP → WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	log.Println("New client connected!")
	defer func() {
		log.Println("Client disconnected")
		conn.Close()
	}()

	// 2. Read messages loop
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("Read error:", err)
			break
		}

		log.Println("Received:", string(msg))

		// 3. Echo message back
		err = conn.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			log.Println("Write error:", err)
			break
		}
	}
}

func main() {
	http.HandleFunc("/ws", handleWebSocket)
	fmt.Println(" Echo server running at :8080/ws")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
