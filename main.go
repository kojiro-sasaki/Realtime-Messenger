package main

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}
var mu sync.Mutex

type Client struct {
	conn *websocket.Conn
	name string
}

var clients = make(map[*Client]bool)

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Upgrager error", err)
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
		fullMsg := []byte(client.name + ":" + string(msg))

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
				delete(clients, client)
				mu.Unlock()
				c.conn.Close()
			}
		}

	}
}
func main() {
	http.HandleFunc("/ws", wsHandler)
	http.Handle("/", http.FileServer(http.Dir("./static")))
	fmt.Println("server started")

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("server error :", err)
	}
}
