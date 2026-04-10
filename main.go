package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/ws", wsHandler)
	http.Handle("/", http.FileServer(http.Dir("./static")))
	fmt.Println("server started")

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("server error :", err)
	}
}
