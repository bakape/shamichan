package feeds

import (
	"meguca/common"
	"sync"
)

// Clients stores all synchronized websocket clients in a thread-safe map
var clients = ClientMap{
	// Start with 128 to avoid reallocations on server start
	clients: make(map[common.Client]syncID, 128),
	ips:     make(map[string]int, 128),
}

func init() {
	common.GetByIPAndBoard = GetByIPAndBoard
	common.GetClientsByIp = GetByIP
}

// ClientMap is a thread-safe store for all clients connected to this server
// instance
type ClientMap struct {
	// Map of clients to the threads or boards they are synced to
	clients map[common.Client]syncID
	// Count of clients by IP
	ips map[string]int
	sync.RWMutex
}

// syncID contains the board and thread the client are currently synced to. If
// the client is on the board page, thread = 0.
type syncID struct {
	op    uint64
	board string
}

// SyncClient adds a client to a the global client map and synchronizes to an
// update feed, if any. If the client was already synced to another feed, it is
// automatically unsubscribed.
func SyncClient(cl common.Client, op uint64, board string) (*Feed, error) {
	clients.Lock()
	old, ok := clients.clients[cl]
	clients.clients[cl] = syncID{op, board}
	if !ok {
		clients.ips[cl.IP()]++
	}
	clients.Unlock()

	if old.op != 0 {
		removeFromFeed(old.op, cl)
	}
	if op == 0 {
		return nil, nil
	}
	return addToFeed(op, cl)
}

// RemoveClient removes a client from the global client map and any subscribed
// to feed
func RemoveClient(cl common.Client) {
	clients.Lock()

	old := clients.clients[cl]
	delete(clients.clients, cl)

	ip := cl.IP()
	clients.ips[ip]--
	if clients.ips[ip] == 0 {
		delete(clients.ips, ip)
	}

	clients.Unlock()

	if old.op != 0 {
		removeFromFeed(old.op, cl)
	}
}

// GetSync returns if the client is synced and the thread and board it is
// synced to
func GetSync(cl common.Client) (synced bool, op uint64, board string) {
	clients.RLock()
	defer clients.RUnlock()

	sync, synced := clients.clients[cl]
	op = sync.op
	board = sync.board
	return
}

// Return number of unique connected IPs
func IPCount() int {
	clients.RLock()
	defer clients.RUnlock()
	return len(clients.ips)
}

// GetByIPAndBoard retrieves all Clients that match the passed IP on a board
func GetByIPAndBoard(ip, board string) []common.Client {
	clients.RLock()
	defer clients.RUnlock()

	cls := make([]common.Client, 0, 16)
	for cl, sync := range clients.clients {
		if cl.IP() == ip && (board == "all" || sync.board == board) {
			cls = append(cls, cl)
		}
	}
	return cls
}

// Returns all clients matching IP
func GetByIP(ip string) []common.Client {
	clients.RLock()
	defer clients.RUnlock()

	cls := make([]common.Client, 0, 16)
	for cl, _ := range clients.clients {
		if cl.IP() == ip {
			cls = append(cls, cl)
		}
	}
	return cls
}

// Get all synced to a thread
func GetByThread(id uint64) []common.Client {
	clients.RLock()
	defer clients.RUnlock()

	cls := make([]common.Client, 0, 16)
	for cl, sync := range clients.clients {
		if sync.op == id {
			cls = append(cls, cl)
		}
	}
	return cls
}

// All returns all currently connected clients
func All() []common.Client {
	clients.RLock()
	defer clients.RUnlock()

	cls := make([]common.Client, 0, len(clients.clients))
	for cl := range clients.clients {
		cls = append(cls, cl)
	}
	return cls
}
