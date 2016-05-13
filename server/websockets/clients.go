package websockets

import (
	"github.com/bakape/meguca/util"
	"sync"
)

// Clients stores all synchronised websocket clients in a theread-safe map
var Clients = ClientMap{
	clients: make(map[string]*Client),
}

// ClientMap is a threadsame store for *clients
type ClientMap struct {
	clients map[string]*Client
	sync.RWMutex
}

// Add adds a client to the map
func (c *ClientMap) Add(cl *Client) {
	c.Lock()
	defer c.Unlock()

	// Dedup client ID
	var id string
	for {
		id = util.RandomID(16)
		if _, ok := c.clients[id]; !ok {
			break
		}
	}

	cl.ID = id
	c.clients[id] = cl
}

// Remove removes a client from the map
func (c *ClientMap) Remove(id string) {
	c.Lock()
	defer c.Unlock()
	delete(c.clients, id)
}

// Has checks if a client exists already by id
func (c *ClientMap) Has(id string) bool {
	c.RLock()
	defer c.RUnlock()
	_, ok := c.clients[id]
	return ok
}

// CountByIP returns the number of unique IPs synchronised with the server
func (c *ClientMap) CountByIP() int {
	c.RLock()
	defer c.RUnlock()
	ips := make(map[string]bool, len(c.clients))
	for _, cl := range c.clients {
		ips[cl.ident.IP] = true
	}
	return len(ips)
}

// SendAll sends a message to all  synchronised websocket clients
func (c *ClientMap) SendAll(msg []byte) {
	c.RLock()
	defer c.RUnlock()
	for _, cl := range c.clients {
		cl.Send <- msg
	}
}
