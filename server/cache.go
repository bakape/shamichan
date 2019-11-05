// FrontEnds for using the inbuilt post cache

package server

import (
	"net/http"
	"strconv"

	"github.com/Chiiruno/meguca/cache"
	"github.com/Chiiruno/meguca/db"
)

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
		f = cache.CatalogFE
	} else {
		f = cache.BoardPageFE
	}
	return
}

// Start cache upkeep proccesses. Requires a ready DB connection.
func listenToThreadDeletion() error {
	return db.Listen("thread_deleted", func(msg string) (err error) {
		board, id, err := db.SplitBoardAndID(msg)
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
