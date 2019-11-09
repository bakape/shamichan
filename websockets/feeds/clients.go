package feeds

import (
	"github.com/bakape/meguca/common"
	"sync"
)

var (
	// Clients stores all synchronized websocket clients in a thread-safe map
	clients = struct {
		sync.RWMutex
		clients map[common.Client]syncID
	}{
		// Start with 128 to avoid reallocations on server start
		clients: make(map[common.Client]syncID, 128),
	}
	ips = struct {
		sync.RWMutex
		ips map[string]int
	}{
		ips: make(map[string]int, 128),
	}
)

func init() {
	common.GetByIPAndBoard = GetByIPAndBoard
	common.GetClientsByIP = GetByIP
}

// syncID contains the board and thread the client are currently synced to. If
// the client is on the board page, thread = 0.
type syncID struct {
	op    uint64
	board string
}

// Regiter IP as conencted
func RegisterIP(ip string) (err error) {
	ips.Lock()
	defer ips.Unlock()

	online := ips.ips[ip]
	if online >= 16 {
		return common.ErrTooManyConnections
	}

	ips.ips[ip]++
	return nil
}

// Unregister IP as conencted
func UnregisterIP(ip string) {
	ips.Lock()
	defer ips.Unlock()

	ips.ips[ip]--
	if ips.ips[ip] == 0 {
		delete(ips.ips, ip)
	}
}

// IPCount returns number of unique connected IPs
func IPCount() int {
	ips.RLock()
	defer ips.RUnlock()
	return len(ips.ips)
}

// SyncClient adds a client to a the global client map and synchronizes to an
// update feed, if any. If the client was already synced to another feed, it is
// automatically unsubscribed.
func SyncClient(cl common.Client, op uint64, board string) (*Feed, error) {
	clients.Lock()
	old, ok := clients.clients[cl]
	clients.clients[cl] = syncID{op, board}
	clients.Unlock()

	if ok {
		removeFromFeed(old.op, old.board, cl)
	}
	return addToFeed(op, board, cl)
}

// RemoveClient removes a client from the global client map and any subscribed
// to feed
func RemoveClient(cl common.Client) {
	clients.Lock()
	old, ok := clients.clients[cl]
	if ok {
		delete(clients.clients, cl)
	}
	clients.Unlock()

	if ok {
		removeFromFeed(old.op, old.board, cl)
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

// GetByIP returns all clients matching the specified IP
func GetByIP(ip string) []common.Client {
	clients.RLock()
	defer clients.RUnlock()

	cls := make([]common.Client, 0, 16)
	for cl := range clients.clients {
		if cl.IP() == ip {
			cls = append(cls, cl)
		}
	}
	return cls
}

// GetByThread gets all synced to a thread
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
