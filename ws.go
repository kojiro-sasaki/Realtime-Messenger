package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Upgrader error", err)
		return
	}

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

	mu.Lock()
	for c := range clients {
		if c.name == nameStr {
			mu.Unlock()
			conn.WriteMessage(websocket.TextMessage, []byte("[SYSTEM] Name already taken"))
			conn.Close()
			return
		}
	}
	mu.Unlock()

	client := &Client{
		conn: conn,
		name: nameStr,
	}

	mu.Lock()
	clients[client] = true
	mu.Unlock()

	broadcast([]byte("[SYSTEM] " + client.name + " joined the chat"))

	defer func() {
		mu.Lock()
		delete(clients, client)
		mu.Unlock()

		broadcast([]byte("[SYSTEM] " + client.name + " left the chat"))
		conn.Close()
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

		if text == "/users" {
			mu.Lock()
			var names []string
			for c := range clients {
				names = append(names, c.name)
			}
			mu.Unlock()

			client.mu.Lock()
			conn.WriteMessage(
				websocket.TextMessage,
				[]byte("[SYSTEM] Users: "+strings.Join(names, ", ")),
			)
			client.mu.Unlock()
			continue
		}

		if len(text) > 500 {
			client.mu.Lock()
			conn.WriteMessage(websocket.TextMessage, []byte("[SYSTEM] Message too long"))
			client.mu.Unlock()
			continue
		}

		broadcast([]byte(client.name + ": " + text))
	}
}
