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
	conn.SetReadLimit(1024)
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
	fmt.Println("client connected:", client.name)
	broadcast([]byte("[SYSTEM] " + client.name + " joined the chat"))

	defer func() {
		fmt.Println("client disconnected:", client.name)
		removeClient(client)
		broadcast([]byte("[SYSTEM] " + client.name + " left the chat"))
	}()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("read error:", err)
			break
		}

		text := strings.TrimSpace(string(msg))
		if text == "" {
			continue
		}
		if text == "/users" {
			names := getUsernames()
			if err := sendToClient(client, []byte("[SYSTEM] Users: "+strings.Join(names, ", "))); err != nil {
				return
			}
			continue
		}
		if len(text) > 500 {
			if err := sendToClient(client, []byte("[SYSTEM] Message too long")); err != nil {
				return
			}
			continue
		}
		broadcast([]byte(client.name + ": " + text))
	}
}
