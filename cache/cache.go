// Package cache provides an in-memory LRU cache for reducing the duplicate
// workload of database requests and post JSON generation
package cache

import (
	"context"
	"encoding/gob"
	"net/http"
	"time"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/bakape/recache/v6"
)

// TODO: Evict from cache on post and thread updates

const evictionTimer = time.Second * 10

var (
	cache *recache.Cache

	// Cache frontend for retreiving thread page JSON
	threadFrontend *recache.Frontend

	// Cache frontend for retrieving the thread index.
	// Contrains only one record.
	indexFrontend *recache.Frontend

	// Stores the threads IDs pf all threads.
	// Contrains only one record.
	threadIDFrontend *recache.Frontend

	// List of currently used tags
	usedTagsFrontend *recache.Frontend
)

// Key for identifying thread pages
type threadKey struct {
	id   uint64
	page int
}

// Init cache with specified max memory usage
func Init() (err error) {
	cache = recache.NewCache(recache.CacheOptions{
		MemoryLimit: uint(config.Server.CacheSize * (1 << 20)),
		LRULimit:    time.Hour,
	})

	threadFrontend = cache.NewFrontend(func(
		k recache.Key,
		rw *recache.RecordWriter,
	) (err error) {
		key := k.(threadKey)
		buf, err := db.GetThread(key.id, key.page)
		if err != nil {
			return
		}
		rw.Write(buf)
		return
	})

	threadIDFrontend = cache.NewFrontend(func(
		_ recache.Key,
		rw *recache.RecordWriter,
	) (err error) {
		ids, err := db.GetThreadIDs()
		if err != nil {
			return
		}
		return gob.NewEncoder(rw).Encode(ids)
	})

	indexFrontend = cache.NewFrontend(func(
		_ recache.Key,
		rw *recache.RecordWriter,
	) (err error) {
		var ids []uint64
		s, err := rw.Bind(threadIDFrontend, struct{}{})
		if err != nil {
			return
		}
		err = gob.NewDecoder(s.Decompress()).Decode(&ids)
		if err != nil {
			return
		}

		_, err = rw.Write([]byte{'['})
		if err != nil {
			return
		}
		for i, id := range ids {
			if i != 0 {
				_, err = rw.Write([]byte{','})
				if err != nil {
					return
				}
			}
			err = rw.Include(threadFrontend, threadKey{
				id:   id,
				page: -5,
			})
			if err != nil {
				return
			}
		}
		_, err = rw.Write([]byte{']'})
		return
	})

	usedTagsFrontend = cache.NewFrontend(func(
		_ recache.Key,
		rw *recache.RecordWriter,
	) (err error) {
		_, err = rw.Bind(threadIDFrontend, struct{}{})
		if err != nil {
			return
		}
		buf, err := db.GetTagList(context.Background())
		if err != nil {
			return
		}
		_, err = rw.Write(buf)
		return
	})

	return
}

// Evict entire cache
func EvictAll() {
	cache.EvictAll(evictionTimer)
}

// Evict all stored data for a thread
func EvictThread(id uint64) {
	threadFrontend.EvictByFunc(
		evictionTimer,
		func(k recache.Key) (bool, error) {
			return k.(threadKey).id == id, nil
		},
	)
}

// Evict a single page of a thread
func EvictThreadPage(id uint64, page uint32) {
	threadFrontend.Evict(evictionTimer, threadKey{
		id:   id,
		page: int(page),
	})

	// Always evict last 5 posts as the change is most likely to happen in those
	// anyway. We can omit cheking this page actually includes them.
	threadFrontend.Evict(evictionTimer, threadKey{
		id:   id,
		page: -5,
	})
}

// Call this to evict caches on new thread creation or old thread deletion
func EvictThreadList() {
	threadIDFrontend.EvictAll(0)
}

// Write thread page JSON to w
//
// page: page of the thread to write;
// 		 -1 to get the last page;
// 		 -5 to get last 5 post variant for the thread index;
func WriteThread(
	w http.ResponseWriter, r *http.Request,
	id uint64,
	page int,
) (err error) {
	// Normalize -1 to not produce duplicate cache entries
	if page == -1 {
		page, err = db.GetLastPage(id)
		if err != nil {
			return
		}
	}

	_, err = threadFrontend.WriteHTTP(threadKey{id, page}, w, r)
	return
}

// Write thread index JSON to w
func WriteIndex(w http.ResponseWriter, r *http.Request) (err error) {
	_, err = indexFrontend.WriteHTTP(struct{}{}, w, r)
	return
}

// Write List of currently used thread tags
func WriteUsedTags(w http.ResponseWriter, r *http.Request) (err error) {
	_, err = usedTagsFrontend.WriteHTTP(struct{}{}, w, r)
	return
}
