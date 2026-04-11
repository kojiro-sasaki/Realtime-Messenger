package main

import (
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
		fmt.Println("Upgrader error", err)
		return
	}
	conn.SetReadDeadline(time.Now().Add(120 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(120 * time.Second))
		return nil
	})
	conn.SetReadLimit(1024)
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
		if strings.HasPrefix(text, "/msg ") {
			parts := strings.SplitN(text, " ", 3)
			if len(parts) < 3 {
				sendToClient(client, []byte("[SYSTEM] Usage : /msg <user> <message>"))
			}
			reciever := parts[1]
			message := parts[2]
			sendPrivateMessage(client, reciever, message)
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
