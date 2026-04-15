package main

import (
	"encoding/json"
	"sort"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

var mu sync.Mutex

var clients = make(map[*Client]bool)

var allowedRooms = map[string]bool{
	"general": true,
	"dev":     true,
	"gaming":  true,
	"sport":   true,
}

type Message struct {
	Type    string `json:"type"`
	Sender  string `json:"sender,omitempty"`
	Message string `json:"message,omitempty"`
}

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
func findClient(name string) *Client {
	mu.Lock()
	defer mu.Unlock()

	for c := range clients {
		if c.name == name {
			return c
		}
	}
	return nil
}
func sendPrivateMessage(sender *Client, recipient string, msg string) {
	target := findClient(recipient)

	if target == nil {
		sendJSON(sender, Message{
			Type:    "system",
			Message: "User not found",
		})
		return
	}

	sendJSON(target, Message{
		Type:    "dm",
		Sender:  sender.name,
		Message: msg,
	})

	sendJSON(sender, Message{
		Type:    "dm",
		Sender:  sender.name,
		Message: "(to " + target.name + ") " + msg,
	})
}

func handleCommand(c *Client, text string) bool {

	if text == "/help" {
		sendJSON(c, Message{
			Type: "system",
			Message: "Commands:\n" +
				"/users - list users\n" +
				"/msg <user> <message>\n" +
				"/name <newname>\n" +
				"/help",
		})
		return true
	}

	if text == "/users" {
		names := getUsernames()
		sendJSON(c, Message{
			Type:    "system",
			Message: "Users: " + strings.Join(names, ", "),
		})
		return true
	}

	if strings.HasPrefix(text, "/msg ") {
		parts := strings.SplitN(text, " ", 3)

		if len(parts) < 3 {
			sendJSON(c, Message{
				Type:    "system",
				Message: "Usage: /msg <user> <message>",
			})
			return true
		}

		sendPrivateMessage(c, parts[1], parts[2])
		return true
	}

	if strings.HasPrefix(text, "/name ") {
		parts := strings.SplitN(text, " ", 2)

		if len(parts) < 2 {
			sendJSON(c, Message{
				Type:    "system",
				Message: "Usage: /name <newname>",
			})
			return true
		}

		newname := strings.TrimSpace(parts[1])

		if newname == "" {
			sendJSON(c, Message{
				Type:    "system",
				Message: "Invalid name",
			})
			return true
		}

		if isNameTaken(newname) {
			sendJSON(c, Message{
				Type:    "system",
				Message: "Name already taken",
			})
			return true
		}
		c.mu.Lock()
		old := c.name
		c.name = newname
		c.mu.Unlock()

		sendJSON(c, Message{
			Type:    "system",
			Message: "Name changed to " + newname,
		})

		broadcastJSON(Message{
			Type:    "system",
			Message: old + " changed name to " + newname,
		})

		return true
	}
	if strings.HasPrefix(text, "/join ") {
		parts := strings.SplitN(text, " ", 2)
		if len(parts) < 2 {
			sendJSON(c, Message{
				Type:    "system",
				Message: "Usage : /join <room>",
			})
			return true
		}
		newRoom := strings.ToLower(strings.TrimSpace(parts[1]))
		if newRoom == "" {
			sendJSON(c, Message{
				Type:    "system",
				Message: "Invalid room",
			})
			return true
		}
		if !allowedRooms[newRoom] {
			sendJSON(c, Message{
				Type:    "system",
				Message: "Room doesnt exist",
			})
			return true
		}
		c.mu.Lock()
		oldroom := c.room
		c.room = newRoom
		c.mu.Unlock()
		sendJSON(c, Message{
			Type:    "system",
			Message: "You joined  room:" + newRoom,
		})
		broadcastJSONtoRoom(oldroom, Message{
			Type:    "system",
			Message: c.name + " left the room",
		})
		broadcastJSONtoRoom(newRoom, Message{
			Type:    "system",
			Message: c.name + " join the room",
		})
		return true
	}
	if strings.HasPrefix(text, "/rooms") {
		sendJSON(c, Message{
			Type:    "system",
			Message: "List of available rooms : " + strings.Join(getRooms(), ", "),
		})
		return true
	}
	if text == "/rusers" {
		sendJSON(c, Message{
			Type:    "system",
			Message: "List of user in this room" + strings.Join(getusersfromRoom(c.room), ", "),
		})
		return true
	}
	return false
}
func broadcastJSON(v any) {
	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	broadcast(data)
}
func sendJSON(c *Client, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.WriteMessage(websocket.TextMessage, data)
}
func broadcasttoRoom(room string, msg []byte) {
	mu.Lock()
	var conns []*Client
	for c := range clients {
		if c.room == room {
			conns = append(conns, c)
		}
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
func broadcastJSONtoRoom(room string, v any) {
	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	broadcasttoRoom(room, data)
}
func getRooms() []string {
	var rooms []string
	for r := range allowedRooms {
		rooms = append(rooms, r)
	}
	sort.Strings(rooms)
	return rooms
}
func getusersfromRoom(room string) []string {
	mu.Lock()
	defer mu.Unlock()
	var users []string
	for c := range clients {
		if c.room == room {
			users = append(users, c.name)
		}
	}
	return users
}
