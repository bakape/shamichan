// Package cache provides an in-memory LRU cache for reducing the duplicate
// workload of database requests and post JSON generation
package cache

import (
	"net/http"
	"time"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/util"
	"github.com/bakape/pg_util"
	"github.com/bakape/recache"
)

var (
	cache *recache.Cache

	// Cache frontend for retreiving board catalog page JSON
	catalogFrontend *recache.Frontend

	// Cache frontend for retreiving board index page JSON
	boardFrontend *recache.Frontend

	// Cache frontend for retreiving thread page JSON
	threadFrontend *recache.Frontend

	// Index page HTML frontend
	indexFrontend *recache.Frontend
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

	catalogFrontend = cache.NewFrontend(
		func(k recache.Key, rw *recache.RecordWriter) (err error) {
			board := k.(string)
			var buf []byte
			if board == "all" {
				buf, err = db.GetAllBoardCatalog()
			} else {
				buf, err = db.GetBoardCatalog(board)
			}
			if err != nil {
				return
			}
			rw.Write(buf)
			return
		},
	)
	boardFrontend = cache.NewFrontend(
		func(k recache.Key, rw *recache.RecordWriter) (err error) {
			key := k.(boardKey)
			var buf []byte
			if key.board == "all" {
				buf, err = db.GetAllBoard(key.page)
			} else {
				buf, err = db.GetBoard(key.board, key.page)
			}
			if err != nil {
				return
			}
			rw.Write(buf)
			return
		},
	)
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

	util.Hook("config.changed", func() error {
		indexFrontend.EvictAll()
		return nil
	})

	evictByBoard := func(board string) {
		catalogFrontend.Evict(board)
		catalogFrontend.Evict("all")

		boardFrontend.EvictByFunc(func(k recache.Key) (bool, error) {
			switch k.(boardKey).board {
			case "all", board:
				return true, nil
			default:
				return false, nil
			}
		})
	}

	listen := func(ch string, handler func(string) error) error {
		return db.Listen(pg_util.ListenOpts{
			DebounceInterval: time.Second,
			Channel:          "thread.updated",
			OnMsg:            handler,
			OnConnectionLoss: func() error {
				cache.EvictAll()
				return nil
			},
		})
	}

	err = listen("thread.updated", func(msg string) (err error) {
		board, ints, err := db.SplitBoardAndInts(msg, 2)
		if err != nil {
			return
		}
		thread := uint64(ints[0])
		page := int(ints[1])

		evictByBoard(board)
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
		board, ints, err := db.SplitBoardAndInts(msg, 1)
		if err != nil {
			return
		}
		thread := uint64(ints[0])

		evictByBoard(board)
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

// Write catalog page JSON to w
func Catalog(w http.ResponseWriter, r *http.Request, board string) (err error) {
	_, err = catalogFrontend.WriteHTTP(board, w, r)
	return
}

// Write board index page JSON to w
func Board(
	w http.ResponseWriter, r *http.Request,
	board string,
	page uint32,
) (err error) {
	_, err = boardFrontend.WriteHTTP(boardKey{page, board}, w, r)
	return
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

// Write index page HTML to w
func IndexHTML(
	w http.ResponseWriter, r *http.Request,
	pos common.ModerationLevel,
) (err error) {
	_, err = indexFrontend.WriteHTTP(pos, w, r)
	return
}
