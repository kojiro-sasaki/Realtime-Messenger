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
	joinMsg := []byte(client.name + " joined the chat")

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
		leaveMsg := []byte(client.name + " left the chat")
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

		fullMsg := []byte(client.name + ": " + text)

		mu.Lock()
		var conns []*Client
		for c := range clients {
			conns = append(conns, c)
		}
		mu.Unlock()

		for _, c := range conns {
			if c == client {
				continue
			}
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
