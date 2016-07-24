package websockets

import (
	"fmt"
	"sync"

	"github.com/bakape/meguca/util"
)

// Clients stores all synchronised websocket clients in a theread-safe map
var Clients = ClientMap{
	clients: make(map[string]clientContainer),
}

type clientContainer struct {
	syncID string  // Board or thread the Client is syncronised to
	client *Client // Pointer to Client instance
}

// ClientMap is a thread-safe store for all connected clients. You also perform
// multiclient message dispatches etc., by calling its methods.
type ClientMap struct {
	clients map[string]clientContainer
	sync.RWMutex
}

// Add adds a client to the map
func (c *ClientMap) Add(cl *Client, syncID string) (err error) {
	c.Lock()
	defer c.Unlock()

	// Dedup client ID
	var id string
	for {
		id, err = util.RandomID(32)
		if err != nil {
			return err
		}
		if _, ok := c.clients[id]; !ok {
			break
		}
	}

	cl.ID = id
	c.clients[id] = clientContainer{
		syncID: syncID,
		client: cl,
	}
	cl.synced = true

	return nil
}

// ChangeSync changes the thread or board ID the client is synchronised to
func (c *ClientMap) ChangeSync(clientID, syncID string) {
	c.Lock()
	defer c.Unlock()
	cont := c.clients[clientID]
	cont.syncID = syncID
	c.clients[clientID] = cont
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

// Get returns a *Client of the passed ID or an error, if there is none
func (c *ClientMap) Get(id string) (*Client, error) {
	c.RLock()
	defer c.RUnlock()
	cl, ok := c.clients[id]
	if !ok {
		return nil, fmt.Errorf("no client found: %s", id)
	}
	return cl.client, nil
}

// CountByIP returns the number of unique IPs synchronised with the server
func (c *ClientMap) CountByIP() int {
	c.RLock()
	defer c.RUnlock()
	ips := make(map[string]bool, len(c.clients))
	for _, cl := range c.clients {
		ips[cl.client.IP] = true
	}
	return len(ips)
}

// Clear removes all clients from the map
func (c *ClientMap) Clear() {
	c.Lock()
	defer c.Unlock()
	c.clients = make(map[string]clientContainer)
}
