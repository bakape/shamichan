package imager

import (
	"sync"
	"sync/atomic"
	"time"
)

type keyStore struct {
	lastUsed int64 // Only access this using atomics
	val      interface{}
}

// Thread-safe self-expiring map for caching
type cacheMap struct {
	m sync.Map
}

// Create new CacheMap and start eviction scheduler.
//
// Must never become unreachable or will result in a memory leak.
func newCacheMap() (c *cacheMap) {
	c = new(cacheMap)

	// Clean up unused keys
	go func() {
		for {
			time.Sleep(time.Minute * 10)

			var (
				// A non-monotonic clock is fine here, because shifts in the
				// system clock will simply cause a cache rebuild at worst
				threshold = time.Now().Add(-time.Hour).Unix()
				toDel     []interface{}
			)
			c.m.Range(func(k, v interface{}) bool {
				if atomic.LoadInt64(&v.(*keyStore).lastUsed) < threshold {
					toDel = append(toDel, k)
				}
				return true
			})
			for _, k := range toDel {
				c.m.Delete(k)
			}
		}
	}()

	return
}

// Insert key into the map
func (c *cacheMap) Insert(key, val interface{}) {
	c.m.Store(key, &keyStore{
		lastUsed: time.Now().Unix(),
		val:      val,
	})
}

// Load key from the map and return it, if found
func (c *cacheMap) Get(key interface{}) (val interface{}, ok bool) {
	v, ok := c.m.Load(key)
	if !ok {
		return
	}
	_v := v.(*keyStore)
	atomic.StoreInt64(&_v.lastUsed, time.Now().Unix())
	val = _v.val
	return
}

// Get a value from the map or generate a new one using gen() and store it
func (c *cacheMap) GetOrGen(
	key interface{},
	gen func() (val interface{}, err error),
) (
	val interface{},
	err error,
) {
	val, ok := c.Get(key)
	if ok {
		return
	}

	val, err = gen()
	if err != nil {
		return
	}
	c.Insert(key, val)
	return
}

// Remove key from the map
func (c *cacheMap) Delete(key interface{}) {
	c.m.Delete(key)
}
