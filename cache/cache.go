// Package cache provides an in-memory LRU cache for reducing the duplicate
// workload of database requests and post JSON generation
package cache

import (
	"net/http"
	"strconv"
	"time"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/bakape/pg_util"
	"github.com/bakape/recache"
)

var (
	cache *recache.Cache

	// Cache frontend for retreiving thread page JSON
	threadFrontend *recache.Frontend
)

// Key for identifying board index pages
type boardKey struct {
	page  uint32
	board string
}

// Key for identifying thread pages
type threadKey struct {
	id   uint64
	page int
}

// Init cache with specified max memory usage
func Init() (err error) {
	cache = recache.NewCache(recache.Options{
		MemoryLimit: uint(config.Server.CacheSize * (1 << 20)),
		LRULimit:    time.Hour,
	})

	// TODO: Global post index frontend

	threadFrontend = cache.NewFrontend(
		func(k recache.Key, rw *recache.RecordWriter) (err error) {
			key := k.(threadKey)
			buf, err := db.GetThread(key.id, key.page)
			if err != nil {
				return
			}
			rw.Write(buf)
			return
		},
	)

	listen := func(ch string, handler func(string) error) error {
		return db.Listen(pg_util.ListenOpts{
			DebounceInterval: time.Second,
			Channel:          "thread.updated",
			OnMsg:            handler,
			OnConnectionLoss: cache.EvictAll,
		})
	}

	err = listen("thread.updated", func(msg string) (err error) {
		ints, err := db.SplitUint64s(msg, 2)
		if err != nil {
			return
		}
		thread := ints[0]
		page := int(ints[1])

		threadFrontend.EvictByFunc(func(k recache.Key) (bool, error) {
			key := k.(threadKey)
			if key.id == thread {
				switch page {
				case -2, key.page:
					return true, nil
				}
			}
			return false, nil
		})

		return
	})
	if err != nil {
		return
	}
	return listen("thread.deleted", func(msg string) (err error) {
		thread, err := strconv.ParseUint(msg, 10, 64)
		if err != nil {
			return
		}

		threadFrontend.EvictByFunc(func(k recache.Key) (bool, error) {
			return k.(threadKey).id == thread, nil
		})

		return
	})
}

// Clear entire cache
func Clear() {
	cache.EvictAll()
}

// Write thread page JSON to w
// page: page of the thread to fetch. -1 to fetch the last page.
func Thread(
	w http.ResponseWriter, r *http.Request,
	id uint64,
	page int,
) (err error) {
	_, err = threadFrontend.WriteHTTP(threadKey{id, page}, w, r)
	return
}
