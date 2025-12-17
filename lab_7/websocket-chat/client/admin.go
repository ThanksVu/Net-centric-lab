// ================= client/admin.go =================
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run admin.go <cmd> [args]")
		fmt.Println(`Commands:
	broadcast "message"
	room <room> "message"
	user <username> "message"
	announce "title" "message"`)
		return
	}
	cmd := os.Args[1]
	host := "http://localhost:8080"

	n := make(map[string]interface{})
	n["id"] = ""
	n["timestamp"] = time.Now()

	switch cmd {
	case "broadcast":
		if len(os.Args) < 3 { log.Fatal("message required") }
		n["title"] = "Broadcast"
		n["message"] = os.Args[2]
		n["type"] = "info"
		n["target"] = "all"
	case "room":
		if len(os.Args) < 4 { log.Fatal("room and message required") }
		n["title"] = "Room Message"
		n["message"] = os.Args[3]
		n["type"] = "info"
		n["target"] = "room:" + os.Args[2]
	case "user":
		if len(os.Args) < 4 { log.Fatal("username and message required") }
		n["title"] = "Private Message"
		n["message"] = os.Args[3]
		n["type"] = "info"
		n["target"] = "user:" + os.Args[2]
	case "announce":
		if len(os.Args) < 4 { log.Fatal("title and message required") }
		n["title"] = os.Args[2]
		n["message"] = os.Args[3]
		n["type"] = "warning"
		n["target"] = "all"
	default:
		log.Fatal("unknown command")
	}

	b, _ := json.Marshal(n)
	resp, err := http.Post(host+"/api/notify", "application/json", bytes.NewBuffer(b))
	if err != nil { log.Fatal(err) }
	defer resp.Body.Close()
	var out map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&out)
	fmt.Println("server response:", out)
}
