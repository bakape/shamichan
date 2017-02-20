// Package cache provides an in-memory LRU cache for reducing the duplicate
// workload of database requests and post HTML and JSON generation
package cache

import (
	"container/list"
	"sync"
	"time"

	"github.com/mailru/easyjson"
)

var (
	cache     = make(map[Key]*list.Element, 10)
	ll        = list.New()
	totalUsed int
	mu        sync.Mutex

	// Time in seconds for the cache to expire and need counter comparison.
	// Mutable for quicker testing.
	expiryTime int64 = 25

	// Size sets the maximum size of cache before evicting unread data in MB
	Size float64 = 1 << 7
)

// Key stores the ID of either a thread or board page
type Key struct {
	LastN uint8
	Board string
	ID    uint64
}

// Single cache entry
type store struct {
	// Controls general access to the contents of the struct, except for size
	sync.Mutex
	key           Key
	lastChecked   int64
	updateCounter uint64
	data          easyjson.Marshaler
	html, json    []byte

	// Separate mutex, because accessed both from get requests and cache
	// eviction calls
	size   int
	sizeMu sync.Mutex
}

// Retrieve a store from the cache or create a new one
func getStore(k Key) (s *store) {
	mu.Lock()
	defer mu.Unlock()

	el := cache[k]
	if el == nil {
		s = &store{key: k}
		cache[k] = ll.PushFront(s)
	} else {
		ll.MoveToFront(el)
		s = el.Value.(*store)
	}
	return s
}

// Clear the cache. Only used for testing.
func Clear() {
	mu.Lock()
	defer mu.Unlock()

	ll = list.New()
	cache = make(map[Key]*list.Element, 10)
}

// Update the total used memory counter and evict, if over limit
func updateUsedSize(delta int) {
	mu.Lock()
	defer mu.Unlock()

	totalUsed += delta

	if totalUsed > int(Size)*(1<<20) {
		s := ll.Remove(ll.Back()).(*store)
		delete(cache, s.key)

		s.sizeMu.Lock()
		totalUsed -= s.size
		s.sizeMu.Unlock()
	}
}

// Return, if the data can still be considered fresh, without querying the DB
func (s *store) isFresh() bool {
	return time.Now().Unix()-s.lastChecked < expiryTime
}

// Stores the new values of s. Calculates and stores the new size. Passes the
// delta to the central cache tp fire eviction checks.
func (s *store) update(data easyjson.Marshaler, json, html []byte) {
	// Calculating the actual memory footprint of the stored post data is
	// expensive. Assume it is as big as the JSON. Most probably it's far less
	// than that.
	newSize := len(json) + len(html)
	if data != nil {
		newSize += len(json)
	}

	s.data = data
	s.json = json
	s.html = html

	s.sizeMu.Lock()
	delta := newSize - s.size
	s.size = newSize
	s.sizeMu.Unlock()

	// Technically it is possible to update the size even when the store is
	// already evicted, but that should never happen, unless you have a very
	// small cache, very large threads and a lot of traffic.
	updateUsedSize(delta)
}
