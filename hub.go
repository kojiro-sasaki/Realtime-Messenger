package main

import (
	"strings"
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
			removeClient(c)
		}
	}
}
func sendToClient(c *Client, msg []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.WriteMessage(websocket.TextMessage, msg)
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

func removeClient(c *Client) {
	mu.Lock()
	delete(clients, c)
	mu.Unlock()
	c.conn.Close()
}

func sendPrivateMessage(sender *Client, recipientName string, msg string) {
	var target *Client
	mu.Lock()
	for c := range clients {
		if c.name == recipientName {
			target = c
			break
		}
	}
	mu.Unlock()
	if target == nil {
		sendToClient(sender, []byte("[SYSTEM] User not found"))
		return
	}
	sendToClient(target, []byte("[DM] "+sender.name+":"+msg))
	sendToClient(sender, []byte("[DM to "+target.name+"] "+msg))
}

func handleCommand(c *Client, command string) bool {
	if command == "/users" {
		names := getUsernames()
		sendToClient(c, []byte("[SYSTEM] Users: "+strings.Join(names, ", ")))
		return true
	}

	if strings.HasPrefix(command, "/msg ") {
		parts := strings.SplitN(command, " ", 3)

		if len(parts) < 3 {
			sendToClient(c, []byte("[SYSTEM] Usage: /msg <user> <message>"))
			return true
		}

		sendPrivateMessage(c, parts[1], parts[2])
		return true
	}

	return false
}
