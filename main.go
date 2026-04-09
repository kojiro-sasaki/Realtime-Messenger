package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var clients = make(map[*websocket.Conn]bool)

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Upgrager error")
		return
	}
	clients[conn] = true

	defer func() {
		delete(clients, conn)
		conn.Close()
	}()
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}
		for client := range clients {
			err := client.WriteMessage(websocket.TextMessage, msg)
			if err != nil {
				client.Close()
				delete(clients, client)
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
