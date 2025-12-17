package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/gorilla/websocket"
)

func main() {
	// 1. Connect to WebSocket server
	conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8080/ws", nil)
	if err != nil {
		log.Fatal("Connection error:", err)
	}
	fmt.Println(" Connected to server!")
	defer conn.Close()

	// 2. Graceful shutdown (Ctrl + C)
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	// 3. Goroutine: read messages from server
	go func() {
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				log.Println("Server closed:", err)
				return
			}
			fmt.Println("Echo:", string(msg))
		}
	}()

	// 4. Read from stdin and send to server
	reader := bufio.NewReader(os.Stdin)

	for {
		select {
		case <-interrupt:
			fmt.Println("\n👋 Client exiting...")
			conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "bye"))
			return

		default:
			fmt.Print("> ")
			text, _ := reader.ReadString('\n')
			text = strings.TrimSpace(text)

			if text == "" {
				continue
			}

			err := conn.WriteMessage(websocket.TextMessage, []byte(text))
			if err != nil {
				log.Println("Write error:", err)
				return
			}
		}
	}
}
