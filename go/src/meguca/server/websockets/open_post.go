package websockets

import (
	"bytes"
	"sync"
)

// Contains an open body and functions for editing it
type bodyBuffer struct {
	sync.RWMutex
	bytes.Buffer
}

// Data of a post currently being written to by a Client
type openPost struct {
	hasImage bool
	bodyBuffer
	len, lines int
	id, op     uint64
	time       int64
	board      string
}
