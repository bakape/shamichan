// FrontEnds for using the inbuilt post cache

package server

import (
	"errors"
	"fmt"
	"meguca/cache"
	"meguca/common"
	"meguca/config"
	"meguca/db"
	"meguca/templates"
	"net/http"
	"strconv"
	"strings"
)

// Contains data of a board page
type pageStore struct {
	pageNumber, pageTotal int
	json                  []byte
	data                  common.Board
}

var errPageOverflow = errors.New("page not found")

var threadCache = cache.FrontEnd{
	GetCounter: func(k cache.Key) (uint64, error) {
		return db.ThreadCounter(k.ID)
	},

	GetFresh: func(k cache.Key) (interface{}, error) {
		return db.GetThread(k.ID, int(k.LastN))
	},

	RenderHTML: func(data interface{}, json []byte) []byte {
		return []byte(templates.ThreadPosts(data.(common.Thread), json))
	},
}

var catalogCache = cache.FrontEnd{
	GetCounter: func(k cache.Key) (uint64, error) {
		if k.Board == "all" {
			return db.AllBoardCounter()
		}
		return db.BoardCounter(k.Board)
	},

	GetFresh: func(k cache.Key) (interface{}, error) {
		if k.Board == "all" {
			return db.GetAllBoardCatalog()
		}
		return db.GetBoardCatalog(k.Board)
	},

	RenderHTML: func(data interface{}, json []byte) []byte {
		return []byte(templates.CatalogThreads(data.(common.Board), json))
	},
}

var boardCache = cache.FrontEnd{
	GetCounter: func(k cache.Key) (uint64, error) {
		if k.Board == "all" {
			return db.AllBoardCounter()
		}
		return db.BoardCounter(k.Board)
	},

	// Board pages are built as a list of individually fetched and cached
	// threads with up to 5 replies each
	GetFresh: func(k cache.Key) (interface{}, error) {
		// Get thread IDs in the right order
		var (
			ids []uint64
			err error
		)
		if k.Board == "all" {
			ids, err = db.GetAllThreadsIDs()
		} else {
			ids, err = db.GetThreadIDs(k.Board)
		}
		if err != nil {
			return nil, err
		}

		// Get data and JSON for these views and paginate
		var (
			pages = make([]pageStore, 0, len(ids)/15+1)
			page  pageStore
		)
		closePage := func() {
			if page.data != nil {
				page.json = append(page.json, ']')
				pages = append(pages, page)
			}
		}

		// Hide threads from NSFW boards, if enabled
		var (
			confs    map[string]config.BoardConfContainer
			hideNSFW bool
		)
		if k.Board == "all" && config.Get().HideNSFW {
			hideNSFW = true
			confs = config.GetAllBoardConfigs()
		}

		for i, id := range ids {
			// Start a new page
			if i%15 == 0 {
				closePage()
				page = pageStore{
					pageNumber: len(pages),
					json:       append(make([]byte, 0, 1<<10), '['),
					data:       make(common.Board, 0, 15),
				}
			}

			k := cache.ThreadKey(id, 5)
			json, data, _, err := cache.GetJSONAndData(k, threadCache)
			if err != nil {
				return nil, err
			}
			t := data.(common.Thread)

			if hideNSFW && confs[t.Board].NSFW {
				continue
			}

			if len(page.json) != 1 {
				page.json = append(page.json, ',')
			}
			page.json = append(page.json, json...)
			page.data = append(page.data, t)
		}
		closePage()

		// Record total page count in all stores
		l := len(pages)
		if l == 0 { // Empty board
			l = 1
			pages = []pageStore{
				{
					json: []byte("[]"),
				},
			}
		}
		for i := range pages {
			pages[i].pageTotal = l
		}

		return pages, nil
	},

	Size: func(data interface{}, _, _ []byte) (s int) {
		for _, p := range data.([]pageStore) {
			s += len(p.json) * 2
		}
		return
	},
}

// For individual pages of a board index
var boardPageCache = cache.FrontEnd{
	GetCounter: func(k cache.Key) (uint64, error) {
		// Get the counter of the parent board
		k.Page = -1
		_, _, ctr, err := cache.GetJSONAndData(k, boardCache)
		return ctr, err
	},

	GetFresh: func(k cache.Key) (interface{}, error) {
		i := int(k.Page)
		k.Page = -1
		_, data, _, err := cache.GetJSONAndData(k, boardCache)
		if err != nil {
			return nil, err
		}

		pages := data.([]pageStore)
		if i > len(pages)-1 {
			return nil, errPageOverflow
		}
		return pages[i], nil
	},

	EncodeJSON: func(data interface{}) ([]byte, error) {
		return data.(pageStore).json, nil
	},

	RenderHTML: func(data interface{}, json []byte) []byte {
		return []byte(templates.IndexThreads(data.(pageStore).data, json))
	},

	Size: func(_ interface{}, _, html []byte) int {
		// Only the HTML is owned by this store. All other data is just
		// borrowed from boardCache.
		return len(html)
	},
}

// Returns arguments for accessing the board page JSON/HTML cache
func boardCacheArgs(r *http.Request, board string, catalog bool) (
	k cache.Key, f cache.FrontEnd,
) {
	var page int64
	if !catalog {
		p, err := strconv.ParseUint(r.URL.Query().Get("page"), 10, 64)
		if err == nil {
			page = int64(p)
		}
	}

	k = cache.BoardKey(board, page, !catalog)
	if catalog {
		f = catalogCache
	} else {
		f = boardPageCache
	}
	return
}

// Start cache upkeep proccesses. Requires a ready DB connection.
func listenToThreadDeletion() error {
	return db.Listen("thread_deleted", func(msg string) (err error) {
		split := strings.Split(msg, ":")
		if len(split) != 2 {
			return fmt.Errorf("unparsable thread deletion message: '%s'", msg)
		}
		board := split[0]
		id, err := strconv.ParseUint(split[1], 10, 64)
		if err != nil {
			return
		}

		// Clear all cache records associated with a thread
		for _, i := range [...]int{0, 5, 100} {
			cache.Delete(cache.ThreadKey(id, i))
		}
		cache.DeleteByBoard(board)
		cache.DeleteByBoard("all")

		return nil
	})
}
