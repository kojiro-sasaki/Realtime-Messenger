package main

import "sync"

var mu sync.Mutex

var clients = make(map[*Client]bool)
