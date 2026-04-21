package chat

import (
	"encoding/json"
	"log"
	"net/http"
	"realtime-messenger/internal/auth"
	"realtime-messenger/internal/db"

	"github.com/gorilla/websocket"

	"strings"
	"time"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		return origin == "http://localhost:8080"
	},
}

func WsHandler(h *Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("upgrade error:", err)
			return
		}
		cookie, err := r.Cookie("session")
		if err != nil {
			conn.Close()
			return
		}
		nameStr, err := auth.ParseToken(cookie.Value)
		if err != nil {
			log.Println("ws: bad token:", err)

			conn.Close()
			return
		}
		nameStr = strings.ToLower(strings.TrimSpace(nameStr))
		if nameStr == "" {

			conn.Close()
			return
		}
		if strings.Contains(nameStr, " ") {
			data, _ := json.Marshal(Message{
				Type:    "system",
				Message: "Name cannot contain spaces",
			})
			conn.WriteMessage(websocket.TextMessage, data)
			conn.Close()
			return
		}
		if len(nameStr) > 20 {
			data, _ := json.Marshal(Message{
				Type:    "system",
				Message: "Name too long",
			})
			conn.WriteMessage(websocket.TextMessage, data)
			conn.Close()
			return
		}
		if h.isNameTaken(nameStr) {
			data, _ := json.Marshal(Message{
				Type:    "system",
				Message: "Name already taken",
			})
			conn.WriteMessage(websocket.TextMessage, data)
			log.Println("ws: name taken:", nameStr)
			conn.Close()
			return
		}

		role, err := db.GetUserRole(nameStr)
		if err != nil {
			log.Println("ws: no role for:", nameStr, err)

			conn.Close()
			return
		}

		id, err := db.GetUserID(nameStr)
		if err != nil {
			log.Println("ws: no id for:", nameStr, err)
			conn.Close()
			return
		}

		client := &Client{
			Conn: conn,
			Id:   id,
			Name: nameStr,
			Room: "general",
			Role: role,
			Send: make(chan []byte, 256),
		}
		go client.writeConn()
		go client.readConn(h)
		go func() {
			ticker := time.NewTicker(60 * time.Second)
			defer ticker.Stop()

			for {
				<-ticker.C

				client.Mu.Lock()
				err := conn.WriteMessage(websocket.PingMessage, nil)
				client.Mu.Unlock()

				if err != nil {
					return
				}
			}
		}()

		h.register <- client

		log.Println("connected:", client.Name)

		h.broadcastJSONtoRoom(client.Room, Message{
			Type:    "system",
			Message: client.Name + " joined the chat",
		})
	}
}
