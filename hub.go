package main

import (
	"sync"

	"github.com/gorilla/websocket"
)

var mu sync.Mutex

var clients = make(map[*Client]bool)

func broadcast(msg []byte) {
	mu.Lock()
	var conns []*Client
	for c := range clients {
		conns = append(conns, c)
	}
	mu.Unlock()
	for _, c := range conns {
		c.mu.Lock()
		err := c.conn.WriteMessage(websocket.TextMessage, msg)
		c.mu.Unlock()
		if err != nil {
			mu.Lock()
			delete(clients, c)
			mu.Unlock()
			c.conn.Close()
		}
	}
}
func sendToClient(c *Client, msg []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.conn.WriteMessage(websocket.TextMessage, msg)
}

func getUsernames() []string {
	mu.Lock()
	defer mu.Unlock()
	var names []string
	for c := range clients {
		names = append(names, c.name)
	}
	return names
}

func isNameTaken(name string) bool {
	mu.Lock()
	defer mu.Unlock()
	for c := range clients {
		if c.name == name {
			return true
		}
	}
	return false
}
