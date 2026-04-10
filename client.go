package main

import (
	"sync"

	"github.com/gorilla/websocket"
)

type Client struct {
	conn *websocket.Conn
	name string
	mu   sync.Mutex
}
