// ================= client/room_client.go =================
// Simple terminal client that connects via websocket and reads stdin
package main


import (
    "bufio"
    "encoding/json"
    "fmt"
    "log"
    "net/url"
    "os"
    "strings"

	
    "github.com/gorilla/websocket"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run room_client.go <username> <room>")
		return
	}
	username := os.Args[1]
	room := os.Args[2]

	u := url.URL{Scheme: "ws", Host: "localhost:8080", Path: "/ws", RawQuery: "username=" + username + "&room=" + room}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	// read incoming
	go func() {
		for {
			var msg map[string]interface{}
			if err := c.ReadJSON(&msg); err != nil {
				log.Println("read error:", err)
				return
			}
			// pretty print
			if t, ok := msg["type"].(string); ok && (t == "chat" || t == "join" || t == "leave" || t == "system" || t == "user_list" || t == "stats") {
				fmt.Printf("[%s] %s: %s\n", msg["time"], msg["username"], msg["text"])
			} else {
				b, _ := jsonMarshal(msg)
				fmt.Println(string(b))
			}
		}
	}()

	// send input
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		text := scanner.Text()
		if strings.TrimSpace(text) == "" {
			continue
		}
		// special commands start with '/'
		if strings.HasPrefix(text, "/") {
			c.WriteJSON(map[string]string{"type": "command", "text": text})
			continue
		}
		c.WriteJSON(map[string]string{"type": "chat", "text": text})
	}
}

func jsonMarshal(v interface{}) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}
