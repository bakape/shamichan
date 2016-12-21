package websockets

import "sync"

// Clients stores all synchronized websocket clients in a thread-safe map
var Clients = ClientMap{
	// Start with 100 to avoid reallocations on server start
	clients: make(map[*Client]SyncID, 100),
	ips:     make(map[string]int, 100),
}

// ClientMap is a thread-safe store for all clients connected to this server
// instance
type ClientMap struct {
	// Map of clients to the threads or boards they are synced to
	clients map[*Client]SyncID
	// Map of connected IPs to their client count
	ips map[string]int
	sync.RWMutex
}

// SyncID contains the board and thread the client are currently synced to. If
// the client is on the board page, thread = 0.
type SyncID struct {
	OP    uint64
	Board string
}

// Add adds a client to the map
func (c *ClientMap) add(cl *Client, syncID SyncID) {
	c.Lock()
	c.clients[cl] = syncID
	cl.synced = true
	newIP := c.ips[cl.ip] == 0
	c.ips[cl.ip]++
	c.Unlock()

	if newIP {
		c.sendIPCount()
	}
}

// Send current IP count to all synchronized clients
func (c *ClientMap) sendIPCount() {
	c.RLock()
	defer c.RUnlock()

	msg, _ := EncodeMessage(MessageSyncCount, len(c.ips))
	for cl := range c.clients {
		cl.Send(msg)
	}
}

// ChangeSync changes the thread or board ID the client is synchronized to
func (c *ClientMap) changeSync(cl *Client, syncID SyncID) {
	c.Lock()
	defer c.Unlock()
	c.clients[cl] = syncID
}

// Remove removes a client from the map
func (c *ClientMap) remove(cl *Client) {
	c.Lock()
	delete(c.clients, cl)
	c.ips[cl.ip]--
	removedIP := c.ips[cl.ip] == 0
	if removedIP {
		delete(c.ips, cl.ip)
	}
	c.Unlock()

	if removedIP {
		c.sendIPCount()
	}
}

// Clear removes all clients from the map
func (c *ClientMap) Clear() {
	c.Lock()
	defer c.Unlock()
	c.clients = make(map[*Client]SyncID)
}

// GetSync returns if the current client is synced and  the thread and board it
// is synced to.
func (c *ClientMap) GetSync(cl *Client) (bool, SyncID) {
	c.RLock()
	defer c.RUnlock()
	sync, ok := c.clients[cl]
	return ok, sync
}
