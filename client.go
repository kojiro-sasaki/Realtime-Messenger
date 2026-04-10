package main

import "github.com/gorilla/websocket"

type Client struct {
	conn *websocket.Conn
	name string
}
