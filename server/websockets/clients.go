package websockets

import "sync"

// Clients stores all synchronised websocket clients in a theread-safe map
var Clients = ClientMap{
	clients: make(map[*Client]SyncID),
}

// ClientMap is a thread-safe store for all clients connected to this server
// instance
type ClientMap struct {
	// Map of clients to the threads or boards they are synced to
	clients map[*Client]SyncID
	sync.RWMutex
}

// SyncID contains the board and thread the client are currently synced to. If
// the client is on the board page, thread = 0.
type SyncID struct {
	OP    int64
	Board string
}

// Add adds a client to the map
func (c *ClientMap) Add(cl *Client, syncID SyncID) {
	c.Lock()
	defer c.Unlock()
	c.clients[cl] = syncID
	cl.synced = true
}

// ChangeSync changes the thread or board ID the client is synchronised to
func (c *ClientMap) ChangeSync(cl *Client, syncID SyncID) {
	c.Lock()
	defer c.Unlock()
	c.clients[cl] = syncID
}

// Remove removes a client from the map
func (c *ClientMap) Remove(cl *Client) {
	c.Lock()
	defer c.Unlock()
	delete(c.clients, cl)
}

// CountByIP returns the number of unique IPs synchronised with the server
func (c *ClientMap) CountByIP() int {
	c.RLock()
	ips := make(map[string]bool, len(c.clients))
	for cl := range c.clients {
		ips[cl.IP] = true
	}
	c.RUnlock()
	return len(ips)
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
