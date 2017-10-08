// Package cache provides an in-memory LRU cache for reducing the duplicate
// workload of database requests and post HTML and JSON generation
package cache

import (
	"container/list"
	"sync"
	"time"
)

// Time for the cache to expire and need counter comparison
const expiryTime = time.Second

var (
	cache     = make(map[Key]*list.Element, 10)
	ll        = list.New()
	totalUsed int
	mu        sync.Mutex

	// Size sets the maximum size of the cache before evicting unread data in MB
	Size float64 = 1 << 7
)

// Key stores the ID of either a thread or board page
type Key struct {
	LastN uint8
	Board string
	ID    uint64
	Page  int64
}

// Single cache entry
type store struct {
	// Controls general access to the contents of the struct, except for size
	sync.RWMutex
	key           Key
	updateCounter uint64
	lastChecked   time.Time
	data          interface{}
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

	ll = ll.Init()
	cache = make(map[Key]*list.Element, 10)
	totalUsed = 0
}

// Update the total used memory counter and evict, if over limit
func updateUsedSize(k Key, delta int) {
	mu.Lock()
	defer mu.Unlock()

	// Guard against asynchronous double eviction
	if _, ok := cache[k]; !ok {
		return
	}
	totalUsed += delta

	for totalUsed > int(Size)*(1<<20) {
		if last := ll.Back(); last != nil {
			removeEntry(last)
		}
	}
}

// Return, if the data can still be considered fresh, without querying the DB
func (s *store) isFresh() bool {
	return time.Now().Sub(s.lastChecked) < expiryTime
}

// Stores the new values of s. Calculates and stores the new size. Passes the
// delta to the central cache to fire eviction checks.
func (s *store) update(data interface{}, json, html []byte, f FrontEnd) {
	var newSize int
	if f.Size == nil {
		newSize = computeSize(data, json, html)
	} else {
		newSize = f.Size(data, json, html)
	}

	s.data = data
	s.json = json
	s.html = html

	s.sizeMu.Lock()
	delta := newSize - s.size
	s.size = newSize
	s.sizeMu.Unlock()

	// In a separate goroutine, to ensure there is never any lock intersection
	go updateUsedSize(s.key, delta)
}

// Calculating the actual memory footprint of the stored post data is expensive.
// Assume it is as big as the JSON. Most probably it's far less than that.
func computeSize(data interface{}, json, html []byte) int {
	newSize := len(json) + len(html)
	if data != nil {
		newSize += len(json)
	}
	return newSize
}

// Delete an entry by key. If no entry found, this is a NOP.
func Delete(k Key) {
	mu.Lock()
	defer mu.Unlock()

	if el := cache[k]; el != nil {
		removeEntry(el)
	}
}

// Remove entry from cache. Requires lock of mu.
func removeEntry(el *list.Element) {
	s := ll.Remove(el).(*store)
	delete(cache, s.key)

	s.sizeMu.Lock()
	totalUsed -= s.size
	s.sizeMu.Unlock()
}

// Delete all entries by the board property of Key.
// If no entries found, this is a NOP.
func DeleteByBoard(board string) {
	mu.Lock()
	defer mu.Unlock()

	for k, el := range cache {
		if k.Board == board {
			removeEntry(el)
		}
	}
}
