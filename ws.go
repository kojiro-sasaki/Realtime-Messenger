package main

import (
	"fmt"
	"net/http"

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

	client := &Client{
		conn: conn,
		name: string(name),
	}

	mu.Lock()
	clients[client] = true
	mu.Unlock()

	defer func() {
		mu.Lock()
		delete(clients, client)
		mu.Unlock()
		conn.Close()
	}()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}

		fullMsg := []byte(client.name + ": " + string(msg))

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
