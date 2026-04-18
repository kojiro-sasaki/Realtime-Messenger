package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

type findRequest struct {
	name string
	resp chan *Client
}

type userRequest struct {
	resp chan []string
}
type roomUserRequest struct {
	room string
	resp chan []string
}
type nameTakenRequest struct {
	name string
	resp chan bool
}
type Hub struct {
	clients       map[*Client]bool
	broadcast     chan []byte
	roombroadcast chan RoomMessage
	register      chan *Client
	unregister    chan *Client
	findReq       chan findRequest
	usersReq      chan userRequest
	roomUsersReq  chan roomUserRequest
	nameReq       chan nameTakenRequest
}
type RoomMessage struct {
	room string
	data []byte
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
		clients:       make(map[*Client]bool),
		broadcast:     make(chan []byte),
		roombroadcast: make(chan RoomMessage),
		register:      make(chan *Client),
		unregister:    make(chan *Client),
		findReq:       make(chan findRequest),
		usersReq:      make(chan userRequest),
		roomUsersReq:  make(chan roomUserRequest),
		nameReq:       make(chan nameTakenRequest),
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

		case rm := <-h.roombroadcast:
			for client := range h.clients {
				if client.room == rm.room {
					select {
					case client.send <- rm.data:
					default:
						close(client.send)
						delete(h.clients, client)
					}
				}
			}
		case req := <-h.findReq:
			var res *Client
			for c := range h.clients {
				if c.name == req.name {
					res = c
					break
				}
			}
			req.resp <- res
		case req := <-h.usersReq:
			var names []string
			for c := range h.clients {
				names = append(names, c.name)
			}
			req.resp <- names

		case req := <-h.roomUsersReq:
			var users []string
			for c := range h.clients {
				if c.room == req.room {
					users = append(users, c.name)
				}
			}
			req.resp <- users

		case req := <-h.nameReq:
			taken := false
			for c := range h.clients {
				if c.name == req.name {
					taken = true
					break
				}
			}
			req.resp <- taken
		}
	}
}

func (c *Client) readConn(h *Hub) {
	defer func() {
		h.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512)

	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
		h.broadcast <- msg
	}
}
func (c *Client) writeConn() {
	for msg := range c.send {
		c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		err := c.conn.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			return
		}
	}
}
func (h *Hub) sendPrivateMessage(sender *Client, recipient string, msg string) {
	target := h.findClient(recipient)

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

func handleCommand(h *Hub, c *Client, text string) bool {

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
		names := h.getUsernames()
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

		h.sendPrivateMessage(c, parts[1], parts[2])
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

		if h.isNameTaken(newname) {
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

		h.broadcastJSON(Message{
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
		changeRoom(h, c, newRoom)
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
			Message: "List of user in this room" + strings.Join(h.getusersfromRoom(c.room), ", "),
		})
		return true
	}
	if text == "/leave" {
		changeRoom(h, c, "general")
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
		target := h.findClient(targetName)
		if target == nil {
			sendJSON(c, Message{
				Type:    "system",
				Message: "User not found",
			})
			return true
		}
		h.unregister <- target
		h.broadcastJSON(Message{
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

		target := h.findClient(targetName)
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
		target := h.findClient(name)
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
func (h *Hub) broadcastJSON(v any) {
	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	h.broadcast <- data
}
func sendJSON(c *Client, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	select {
	case c.send <- data:
	default:
		return fmt.Errorf("client overloaded")
	}
	return nil
}
func (h *Hub) broadcastJSONtoRoom(room string, v any) {
	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	h.roombroadcast <- RoomMessage{
		room: room,
		data: data,
	}
}
func getRooms() []string {
	var rooms []string
	for r := range allowedRooms {
		rooms = append(rooms, r)
	}
	sort.Strings(rooms)
	return rooms
}

func changeRoom(h *Hub, c *Client, newroom string) {
	c.mu.Lock()
	oldRoom := c.room
	if oldRoom == newroom {
		c.mu.Unlock()
		sendJSON(c, Message{
			Type:    "system",
			Message: "You are already in room " + newroom,
		})
		return
	}
	c.room = newroom
	c.mu.Unlock()

	if oldRoom != "" {
		h.broadcastJSONtoRoom(oldRoom, Message{
			Type:    "system",
			Message: c.name + " left the room",
		})
	}

	h.broadcastJSONtoRoom(newroom, Message{
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

func (h *Hub) findClient(name string) *Client {
	resp := make(chan *Client, 1)
	h.findReq <- findRequest{name: name, resp: resp}
	return <-resp
}
func (h *Hub) getUsernames() []string {
	resp := make(chan []string, 1)
	h.usersReq <- userRequest{resp: resp}
	return <-resp
}
func (h *Hub) getusersfromRoom(room string) []string {
	resp := make(chan []string, 1)
	h.roomUsersReq <- roomUserRequest{room: room, resp: resp}
	return <-resp
}
func (h *Hub) isNameTaken(name string) bool {
	resp := make(chan bool, 1)
	h.nameReq <- nameTakenRequest{name: name, resp: resp}
	return <-resp
}
