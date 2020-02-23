package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/bakape/meguca/cache"
	"github.com/bakape/meguca/common"
	"github.com/jackc/pgx/v4"
)

func setJSONHeaders(w http.ResponseWriter) {
	head := w.Header()
	for key, val := range vanillaHeaders {
		head.Set(key, val)
	}
	head.Set("Content-Type", "application/json")
}

// // Serve a single post as JSON
// func servePost(w http.ResponseWriter, r *http.Request) {
// 	httpError(w, r, func() (err error) {
// 		id, err := strconv.ParseUint(extractParam(r, "post"), 10, 64)
// 		if err != nil {
// 			httpError(w, r, common.StatusError{
// 				Err:  err,
// 				Code: 400,
// 			})
// 			return
// 		}

// 		post, err := db.GetPost(id)
// 		if err != nil {
// 			httpError(w, r, err)
// 			return
// 		}
// 		serveJSON(w, r, "", post)
// 	}())
// }

// Serves thread page JSON
func serveThread(w http.ResponseWriter, r *http.Request) {
	handleError(w, r, func() (err error) {
		var (
			page   int
			thread uint64
		)
		err = func() (err error) {
			thread, err = strconv.ParseUint(extractParam(r, "thread"), 10, 64)
			if err != nil {
				return
			}

			page, err = strconv.Atoi(extractParam(r, "page"))
			if err != nil {
				return
			}
			if page < 0 {
				switch page {
				case -1, -5:
				default:
					return fmt.Errorf("invalid page number: %d", page)
				}
			}

			return
		}()
		if err != nil {
			return common.StatusError{
				Err:  err,
				Code: 400,
			}
		}

		setJSONHeaders(w)
		err = cache.WriteThread(w, r, thread, page)
		if err != pgx.ErrNoRows {
			err = common.StatusError{
				Err:  err,
				Code: 404,
			}
		}
		return
	})
}

// Serves thread index page JSON
func serveIndex(w http.ResponseWriter, r *http.Request) {
	serverJSONFromCache(w, r, cache.WriteIndex)
}

func serverJSONFromCache(
	w http.ResponseWriter, r *http.Request,
	src func(w http.ResponseWriter, r *http.Request) error,
) {
	handleError(w, r, func() (err error) {
		setJSONHeaders(w)
		return src(w, r)
	})
}

// Serve list of currently used thread tags
func serverUsedTags(w http.ResponseWriter, r *http.Request) {
	serverJSONFromCache(w, r, cache.WriteUsedTags)
}

// func serveThreadUpdates(w http.ResponseWriter, r *http.Request) {
// 	err := func() (err error) {
// 		var data map[uint64]uint64
// 		err = decodeJSON(r, &data)
// 		if err != nil {
// 			return
// 		}

// 		diff, err := db.DiffThreadPostCounts(data)
// 		if err != nil {
// 			return
// 		}
// 		serveJSON(w, r, "", diff)
// 		return
// 	}()
// 	if err != nil {
// 		httpError(w, r, err)
// 	}
// }
