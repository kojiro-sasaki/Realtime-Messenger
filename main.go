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
var clients = make(map[*websocket.Conn]bool)

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Upgrager error", err)
		return
	}
	mu.Lock()
	clients[conn] = true
	mu.Unlock()
	defer func() {
		mu.Lock()
		delete(clients, conn)
		mu.Unlock()
		conn.Close()
	}()
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}
		mu.Lock()
		var conns []*websocket.Conn
		for c := range clients {
			conns = append(conns, c)
		}
		mu.Unlock()
		for _, client := range conns {
			err := client.WriteMessage(websocket.TextMessage, msg)
			if err != nil {
				mu.Lock()
				client.Close()
				delete(clients, client)
				mu.Unlock()
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
