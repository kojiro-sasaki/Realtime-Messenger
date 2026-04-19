package chat

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	Conn        *websocket.Conn
	Name        string
	Send        chan []byte
	Id          int
	Mu          sync.Mutex
	LastMessage time.Time
	Room        string
	Role        string
}

const (
	RoleUser  = "user"
	RoleAdmin = "admin"
	RoleMod   = "mod"
)
