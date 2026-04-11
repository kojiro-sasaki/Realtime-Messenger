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

	if isNameTaken(nameStr) {
		conn.WriteMessage(websocket.TextMessage, []byte("[SYSTEM] Name already taken"))
		conn.Close()
		return
	}

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
			names := getUsernames()
			sendToClient(client, []byte("[SYSTEM] Users: "+strings.Join(names, ", ")))
			continue
		}
		if len(text) > 500 {
			sendToClient(client, []byte("[SYSTEM] Message too long"))
			continue
		}
		broadcast([]byte(client.name + ": " + text))
	}
}
