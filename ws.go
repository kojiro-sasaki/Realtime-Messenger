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
	CheckOrigin: func(r *http.Request) bool { return true },
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

	nameStr := strings.TrimSpace(string(name))
	if nameStr == "" {
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

	client := &Client{conn: conn, name: nameStr}

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

	broadcastJSON(Message{
		Type:    "system",
		Message: client.name + " joined the chat",
	})

	defer func() {
		removeClient(client)
		fmt.Println("disconnected:", client.name)

		broadcastJSON(Message{
			Type:    "system",
			Message: client.name + " left the chat",
		})
	}()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}

		text := strings.TrimSpace(string(msg))
		if text == "" {
			continue
		}

		if handleCommand(client, text) {
			continue
		}

		if len(text) > 500 {
			sendJSON(client, Message{
				Type:    "system",
				Message: "Message too long",
			})
			continue
		}

		broadcastJSON(Message{
			Type:    "message",
			Sender:  client.name,
			Message: text,
		})
	}
}
