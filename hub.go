package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/gorilla/websocket"
)

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
}

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

func newHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
		case msg := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- msg:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}

	}
}

func (h *Hub) getUsernames() []string {
	var names []string
	for c := range h.clients {
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

func (h *Hub) removeClient(c *Client) {
	delete(h.clients, c)
}
func (h *Hub) findClient(name string) *Client {
	for c := range h.clients {
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
		changeRoom(c, newRoom)
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
	if text == "/leave" {
		changeRoom(c, "general")
		return true
	}
	if strings.HasPrefix(text, "/kick ") {
		if !hasPermission(c, roleMod) {
			sendJSON(c, Message{
				Type:    "system",
				Message: "No permission",
			})
			return true
		}
		parts := strings.SplitN(text, " ", 2)
		targetName := strings.TrimSpace(parts[1])
		target := findClient(targetName)
		if target == nil {
			sendJSON(c, Message{
				Type:    "system",
				Message: "User not found",
			})
			return true
		}
		removeClient(target)
		broadcastJSON(Message{
			Type:    "system",
			Message: target.name + " was kicked by " + c.name,
		})
		return true
	}
	if strings.HasPrefix(text, "/role ") {
		if !hasPermission(c, roleAdmin) {
			sendJSON(c, Message{
				Type:    "system",
				Message: "No permission",
			})
			return true
		}

		parts := strings.SplitN(text, " ", 3)
		if len(parts) < 3 {
			sendJSON(c, Message{
				Type:    "system",
				Message: "Usage: /role <username> <role>",
			})
			return true
		}

		targetName := strings.TrimSpace(parts[1])
		newRole := strings.TrimSpace(parts[2])

		target := findClient(targetName)
		if target == nil {
			sendJSON(c, Message{
				Type:    "system",
				Message: "User not found",
			})
			return true
		}

		target.mu.Lock()
		switch newRole {
		case "user":
			target.role = roleUser
		case "mod":
			target.role = roleMod
		case "admin":
			target.role = roleAdmin
		default:
			target.mu.Unlock()
			sendJSON(c, Message{
				Type:    "system",
				Message: "Invalid role (user/mod/admin)",
			})
			return true
		}
		updatedRole := target.role
		target.mu.Unlock()

		sendJSON(target, Message{
			Type:    "system",
			Message: "Your role is now " + updatedRole,
		})

		sendJSON(c, Message{
			Type:    "system",
			Message: target.name + " is now " + updatedRole,
		})

		return true
	}
	if strings.HasPrefix(text, "/whois ") {
		parts := strings.SplitN(text, " ", 2)
		if len(parts) < 2 {
			sendJSON(c, Message{
				Type:    "system",
				Message: "Usage : /whois <name>",
			})
			return true
		}
		name := strings.TrimSpace(parts[1])
		target := findClient(name)
		if target == nil {
			sendJSON(c, Message{
				Type:    "system",
				Message: "Client not found",
			})
			return true
		}
		sendJSON(c, Message{
			Type: "system",
			Message: fmt.Sprintf("Name:%s\nRole:%s\nRoom%s\n",
				target.name, target.role, target.room),
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
func changeRoom(c *Client, newroom string) {
	c.mu.Lock()
	oldRoom := c.room
	if oldRoom == newroom {
		c.mu.Unlock()
		sendJSON(c, Message{
			Type:    "system",
			Message: "You are already in room" + newroom,
		})
		return
	}
	c.room = newroom
	c.mu.Unlock()
	if oldRoom != "" {
		broadcastJSONtoRoom(oldRoom, Message{
			Type:    "system",
			Message: c.name + " left the room",
		})
	}
	broadcastJSONtoRoom(newroom, Message{
		Type:    "system",
		Message: c.name + " joined the room",
	})
	sendJSON(c, Message{
		Type:    "system",
		Message: "You moved to " + newroom,
	})
}
func hasPermission(c *Client, required string) bool {
	roles := map[string]int{
		roleUser:  1,
		roleMod:   2,
		roleAdmin: 3,
	}

	return roles[c.role] >= roles[required]
}
