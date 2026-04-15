package main

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	conn        *websocket.Conn
	name        string
	mu          sync.Mutex
	lastMessage time.Time
	room        string
	role        string
}

const (
	roleUser  = "user"
	roleAdmin = "admin"
	roleMod   = "mod"
)
