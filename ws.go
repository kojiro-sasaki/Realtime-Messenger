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
			conn.WriteMessage(websocket.TextMessage, []byte("that name is taken"))
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
	joinMsg := []byte("[SYSTEM] " + client.name + " joined the chat")

	var conns []*Client
	for c := range clients {
		conns = append(conns, c)
	}
	mu.Unlock()

	for _, c := range conns {
		if c == client {
			continue
		}
		c.conn.WriteMessage(websocket.TextMessage, joinMsg)
	}

	defer func() {
		leaveMsg := []byte("[SYSTEM] " + client.name + " left the chat")
		mu.Lock()
		delete(clients, client)

		var conns []*Client
		for c := range clients {
			conns = append(conns, c)
		}
		mu.Unlock()

		for _, c := range conns {
			c.conn.WriteMessage(websocket.TextMessage, leaveMsg)
		}
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
		if len(text) > 500 {
			conn.WriteMessage(websocket.TextMessage, []byte("[SYSTEM] Message too long"))
			continue
		}
		fullMsg := []byte(client.name + ": " + text)

		mu.Lock()
		var conns []*Client
		for c := range clients {
			conns = append(conns, c)
		}
		mu.Unlock()

		if text == "/users" {
			mu.Lock()
			var names []string
			for c := range clients {
				names = append(names, c.name)
			}
			mu.Unlock()

			response := "[SYSTEM] Users: " + strings.Join(names, ", ")
			conn.WriteMessage(websocket.TextMessage, []byte(response))
			continue
		}

		for _, c := range conns {
			err := c.conn.WriteMessage(websocket.TextMessage, fullMsg)
			if err != nil {
				mu.Lock()
				delete(clients, c)
				mu.Unlock()
				c.conn.Close()
			}
		}
	}
}
