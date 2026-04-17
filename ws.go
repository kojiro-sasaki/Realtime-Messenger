package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		return origin == "http://localhost:8080"
	},
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("upgrade error:", err)
		return
	}

	conn.SetReadLimit(1024)
	conn.SetReadDeadline(time.Now().Add(120 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(120 * time.Second))
		return nil
	})

	_, name, err := conn.ReadMessage()
	if err != nil {
		conn.Close()
		return
	}

	nameStr := strings.ToLower(strings.TrimSpace(string(name)))
	if nameStr == "" {
		conn.Close()
		return
	}
	if strings.Contains(nameStr, " ") {
		data, _ := json.Marshal(Message{
			Type:    "system",
			Message: "Name cannot contain spaces",
		})
		conn.WriteMessage(websocket.TextMessage, data)
		conn.Close()
		return
	}
	if len(nameStr) > 20 {
		data, _ := json.Marshal(Message{
			Type:    "system",
			Message: "Name too long",
		})
		conn.WriteMessage(websocket.TextMessage, data)
		conn.Close()
		return
	}
	if isNameTaken(nameStr) {
		data, _ := json.Marshal(Message{
			Type:    "system",
			Message: "Name already taken",
		})
		conn.WriteMessage(websocket.TextMessage, data)
		conn.Close()
		return
	}

	role := roleUser
	if nameStr == "admin123" {
		role = roleAdmin
	}

	id, err := getUserID(nameStr)
	if err != nil {
		conn.Close()
		return
	}

	client := &Client{
		conn: conn,
		id:   id,
		name: nameStr,
		room: "general",
		role: role,
	}

	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()

		for {
			<-ticker.C

			client.mu.Lock()
			err := conn.WriteMessage(websocket.PingMessage, nil)
			client.mu.Unlock()

			if err != nil {
				return
			}
		}
	}()

	mu.Lock()
	clients[client] = true
	mu.Unlock()

	fmt.Println("connected:", client.name)

	broadcastJSONtoRoom(client.room, Message{
		Type:    "system",
		Message: client.name + " joined the chat",
	})

	defer func() {
		removeClient(client)
		fmt.Println("disconnected:", client.name)

		broadcastJSONtoRoom(client.room, Message{
			Type:    "system",
			Message: client.name + " left the chat",
		})
	}()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("read error:", err)
			break
		}

		text := strings.TrimSpace(string(msg))
		fmt.Println("message:", client.name, text)

		if text == "" {
			continue
		}

		client.mu.Lock()
		if time.Since(client.lastMessage) < 200*time.Millisecond {
			client.mu.Unlock()
			continue
		}
		client.lastMessage = time.Now()
		client.mu.Unlock()

		if strings.HasPrefix(text, "/") {
			fmt.Println("command:", client.name, "->", text)

			if handleCommand(client, text) {
				continue
			}
		}

		if len(text) > 500 {
			sendJSON(client, Message{
				Type:    "system",
				Message: "Message too long",
			})
			continue
		}

		_, err = db.Exec(
			"INSERT INTO messages (sender, text) VALUES (?, ?)",
			client.name,
			text,
		)
		if err != nil {
			fmt.Println("db error:", err)
		}

		broadcastJSONtoRoom(client.room, Message{
			Type:    "message",
			Sender:  "[" + client.room + "] " + "[" + client.role + "] " + client.name,
			Message: text,
		})
	}
}
